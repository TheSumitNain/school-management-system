package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

var (
	ErrMFARequired = errors.New("multi-factor authentication is required")
	ErrMFAInvalid = errors.New("MFA challenge or code is invalid")
	ErrMFAKeyVersionUnavailable = errors.New("MFA encryption key version is unavailable")
)

type mfaCredentialStore interface {
	IsMandatoryForPrincipal(context.Context,int64) (bool,error)
	GetCredential(context.Context,int64) (*repository.MFACredential,error)
	SavePendingCredential(context.Context,int64,[]byte,int) error
	ActivateEnrollment(context.Context,int64,string) error
}

type mfaChallengeStore interface {
	Create(context.Context,int64,repository.MFAChallengePurpose,string,time.Time) error
	Get(context.Context,string) (*repository.MFAChallenge,error)
	RecordFailure(context.Context,string) error
	Consume(context.Context,string) (int64,repository.MFAChallengePurpose,error)
}

type MFAService struct {
	credentials mfaCredentialStore
	challenges mfaChallengeStore
	encryptionKey []byte
	keyVersion int
	issuer string
}

type MFAStart struct {
	Required bool
	EnrollmentRequired bool
	ChallengeToken string
	Secret string
	ProvisioningURI string
	ExpiresAt time.Time
}

func NewMFAService(credentials mfaCredentialStore, challenges mfaChallengeStore, key []byte, keyVersion int, issuer string) (*MFAService,error) {
	if credentials==nil || challenges==nil { return nil,fmt.Errorf("MFA repositories are required") }
	if len(key)!=32 || keyVersion<1 { return nil,fmt.Errorf("valid 32-byte TOTP encryption key and positive version are required") }
	if strings.TrimSpace(issuer)=="" { return nil,fmt.Errorf("MFA issuer is required") }
	return &MFAService{credentials:credentials,challenges:challenges,encryptionKey:append([]byte(nil),key...),keyVersion:keyVersion,issuer:strings.TrimSpace(issuer)},nil
}

// BeginAuthentication returns a short-lived challenge for mandatory MFA.
// An un-enrolled administrator receives a one-time provisioning secret and
// must complete enrollment before any application session can be issued.
func (s *MFAService) BeginAuthentication(ctx context.Context, principalID int64, accountName string) (*MFAStart,error) {
	required,err:=s.credentials.IsMandatoryForPrincipal(ctx,principalID)
	if err!=nil { return nil,err }
	result:=&MFAStart{Required:required}
	if !required { return result,nil }
	credential,err:=s.credentials.GetCredential(ctx,principalID)
	if err!=nil { return nil,err }
	if credential==nil || credential.VerifiedAt==nil {
		secret,err:=GenerateTOTPSecret(); if err!=nil { return nil,err }
		decoded,err:=base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret); if err!=nil { return nil,fmt.Errorf("decode generated TOTP secret: %w",err) }
		ciphertext,err:=EncryptTOTPSecret(decoded,s.encryptionKey); if err!=nil { return nil,err }
		if err=s.credentials.SavePendingCredential(ctx,principalID,ciphertext,s.keyVersion); err!=nil { return nil,err }
		uri,err:=TOTPProvisioningURI(s.issuer,accountName,secret); if err!=nil { return nil,err }
		challenge,expiry,err:=s.newChallenge(ctx,principalID,repository.MFAChallengeEnrollment); if err!=nil { return nil,err }
		result.EnrollmentRequired=true
		result.ChallengeToken=challenge
		result.Secret=secret
		result.ProvisioningURI=uri
		result.ExpiresAt=expiry
		return result,nil
	}
	challenge,expiry,err:=s.newChallenge(ctx,principalID,repository.MFAChallengeLogin)
	if err!=nil { return nil,err }
	result.ChallengeToken=challenge
	result.ExpiresAt=expiry
	return result,nil
}

func (s *MFAService) CompleteEnrollment(ctx context.Context, principalID int64, challengeToken, code string) error {
	hash:=HashRefreshToken(strings.TrimSpace(challengeToken))
	challenge,err:=s.challenges.Get(ctx,hash)
	if err!=nil { return ErrMFAInvalid }
	if challenge.PrincipalID!=principalID || challenge.Purpose!=repository.MFAChallengeEnrollment { return ErrMFAInvalid }
	credential,err:=s.credentials.GetCredential(ctx,principalID)
	if err!=nil { return err }
	if credential==nil || credential.VerifiedAt!=nil { return ErrMFAInvalid }
	secret,err:=s.decryptCredential(credential); if err!=nil { return err }
	if !VerifyTOTP(secret,strings.TrimSpace(code),time.Now()) {
		if err:=s.challenges.RecordFailure(ctx,hash); err!=nil && !errors.Is(err,repository.ErrMFAChallengeUnavailable) { return err }
		return ErrMFAInvalid
	}
	if err:=s.credentials.ActivateEnrollment(ctx,principalID,hash); err!=nil {
		if errors.Is(err,repository.ErrMFAEnrollmentInvalid) { return ErrMFAInvalid }
		return err
	}
	return nil
}

// VerifyLoginChallenge consumes a valid login challenge exactly once. The
// caller may issue a session only after this method returns the principal ID.
func (s *MFAService) VerifyLoginChallenge(ctx context.Context, challengeToken, code string) (int64,error) {
	hash:=HashRefreshToken(strings.TrimSpace(challengeToken))
	challenge,err:=s.challenges.Get(ctx,hash)
	if err!=nil { return 0,ErrMFAInvalid }
	if challenge.Purpose!=repository.MFAChallengeLogin { return 0,ErrMFAInvalid }
	credential,err:=s.credentials.GetCredential(ctx,challenge.PrincipalID)
	if err!=nil { return 0,err }
	if credential==nil || credential.VerifiedAt==nil { return 0,ErrMFAInvalid }
	secret,err:=s.decryptCredential(credential); if err!=nil { return 0,err }
	if !VerifyTOTP(secret,strings.TrimSpace(code),time.Now()) {
		if err:=s.challenges.RecordFailure(ctx,hash); err!=nil && !errors.Is(err,repository.ErrMFAChallengeUnavailable) { return 0,err }
		return 0,ErrMFAInvalid
	}
	principalID,purpose,err:=s.challenges.Consume(ctx,hash)
	if err!=nil || principalID!=challenge.PrincipalID || purpose!=repository.MFAChallengeLogin { return 0,ErrMFAInvalid }
	return principalID,nil
}

func (s *MFAService) newChallenge(ctx context.Context, principalID int64, purpose repository.MFAChallengePurpose) (string,time.Time,error) {
	raw:=make([]byte,32)
	if _,err:=rand.Read(raw); err!=nil { return "",time.Time{},fmt.Errorf("generate MFA challenge: %w",err) }
	token:=base64.RawURLEncoding.EncodeToString(raw)
	expires:=time.Now().Add(5*time.Minute)
	if err:=s.challenges.Create(ctx,principalID,purpose,HashRefreshToken(token),expires); err!=nil { return "",time.Time{},err }
	return token,expires,nil
}

func (s *MFAService) decryptCredential(c *repository.MFACredential) ([]byte,error) {
	if c.KeyVersion!=s.keyVersion { return nil,ErrMFAKeyVersionUnavailable }
	secret,err:=DecryptTOTPSecret(c.Ciphertext,s.encryptionKey)
	if err!=nil { return nil,fmt.Errorf("decrypt stored MFA credential: %w",err) }
	return secret,nil
}

// GenerateRecoveryCodes returns high-entropy one-time codes. The caller must
// persist only hashes and display the raw values once after successful setup.
func GenerateRecoveryCodes(count int) ([]string,[]string,error) {
	if count<8 || count>12 { return nil,nil,fmt.Errorf("recovery code count must be between 8 and 12") }
	codes:=make([]string,0,count)
	hashes:=make([]string,0,count)
	for i:=0;i<count;i++ {
		raw:=make([]byte,16)
		if _,err:=rand.Read(raw); err!=nil { return nil,nil,fmt.Errorf("generate recovery code: %w",err) }
		code:=strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
		sum:=HashRefreshToken(code)
		decoded,err:=hex.DecodeString(sum); if err!=nil { return nil,nil,err }
		codes=append(codes,code)
		hashes=append(hashes,hex.EncodeToString(decoded))
	}
	return codes,hashes,nil
}
