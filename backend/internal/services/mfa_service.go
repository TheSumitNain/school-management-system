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
	"strconv"
	"time"

	"school-management-system/internal/models"

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
	ActivateEnrollment(context.Context,int64,string,int64,int64,[]string) error
	ConsumeLoginChallenge(context.Context,int64,string,int64,int64) error
	ConsumeLoginRecoveryCode(context.Context,int64,string,int64,string) error
	ReplaceRecoveryCodeHashes(context.Context,int64,string,int64,int64,[]string) error
}

type mfaCredentialRotator interface {
	ReencryptCredential(context.Context, int64, []byte, int, []byte, int) error
}

type mfaChallengeStore interface {
	Create(context.Context,int64,int64,repository.MFAChallengePurpose,string,time.Time) error
	Get(context.Context,string) (*repository.MFAChallenge,error)
	RecordFailure(context.Context,string) error
}

type MFAService struct {
	credentials mfaCredentialStore
	challenges mfaChallengeStore
	encryptionKey []byte
	keyVersion int
	previousEncryptionKeys map[int][]byte
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
	return NewMFAServiceWithKeyRing(credentials, challenges, key, keyVersion, nil, issuer)
}

// NewMFAServiceWithKeyRing encrypts new credentials with the current key and
// retains previous versions solely for decrypting and lazily re-encrypting
// existing credentials after successful TOTP verification.
func NewMFAServiceWithKeyRing(credentials mfaCredentialStore, challenges mfaChallengeStore, key []byte, keyVersion int, previous map[int][]byte, issuer string) (*MFAService,error) {
	if credentials==nil || challenges==nil { return nil,fmt.Errorf("MFA repositories are required") }
	if len(key)!=32 || keyVersion<1 { return nil,fmt.Errorf("valid 32-byte TOTP encryption key and positive version are required") }
	if strings.TrimSpace(issuer)=="" { return nil,fmt.Errorf("MFA issuer is required") }
	copied := make(map[int][]byte, len(previous))
	for version, previousKey := range previous {
		if version < 1 || version == keyVersion || len(previousKey) != 32 {
			return nil, fmt.Errorf("previous TOTP encryption keys require distinct positive versions and exactly 32 bytes")
		}
		if _, exists := copied[version]; exists { return nil, fmt.Errorf("duplicate previous TOTP encryption key version") }
		copied[version] = append([]byte(nil), previousKey...)
	}
	return &MFAService{credentials:credentials,challenges:challenges,encryptionKey:append([]byte(nil),key...),keyVersion:keyVersion,previousEncryptionKeys:copied,issuer:strings.TrimSpace(issuer)},nil
}

// BeginAuthentication returns a short-lived challenge for mandatory MFA.
// An un-enrolled administrator receives a one-time provisioning secret and
// must complete enrollment before any application session can be issued.
func (s *MFAService) BeginAuthentication(ctx context.Context, principal *models.AuthPrincipal, accountName string) (*MFAStart,error) {
	if principal == nil || principal.ID <= 0 || principal.TokenVersion <= 0 || principal.Status != "active" {
		return nil, ErrMFAInvalid
	}
	principalID := principal.ID
	tokenVersion := principal.TokenVersion
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
		challenge,expiry,err:=s.newChallenge(ctx,principalID,tokenVersion,repository.MFAChallengeEnrollment); if err!=nil { return nil,err }
		result.EnrollmentRequired=true
		result.ChallengeToken=challenge
		result.Secret=secret
		result.ProvisioningURI=uri
		result.ExpiresAt=expiry
		return result,nil
	}
	challenge,expiry,err:=s.newChallenge(ctx,principalID,tokenVersion,repository.MFAChallengeLogin)
	if errors.Is(err, repository.ErrMFALoginLocked) {
		return nil, ErrLoginLocked
	}
	if err!=nil { return nil,err }
	result.ChallengeToken=challenge
	result.ExpiresAt=expiry
	return result,nil
}

func (s *MFAService) CompleteEnrollment(ctx context.Context, challengeToken, code string) (int64, int64, []string, error) {
	tokenVersion, tokenErr := parseMFAChallengeTokenVersion(challengeToken)
	if tokenErr != nil { return 0, 0, nil, ErrMFAInvalid }
	hash := HashRefreshToken(strings.TrimSpace(challengeToken))
	challenge, err := s.challenges.Get(ctx, hash)
	if errors.Is(err, repository.ErrMFALoginLocked) { return 0, 0, nil, ErrLoginLocked }
	if err != nil { return 0, 0, nil, ErrMFAInvalid }
	principalID := challenge.PrincipalID
	if challenge.Purpose != repository.MFAChallengeEnrollment { return 0, 0, nil, ErrMFAInvalid }
	credential, err := s.credentials.GetCredential(ctx, principalID)
	if err != nil { return 0, 0, nil, err }
	if credential == nil || credential.VerifiedAt != nil { return 0, 0, nil, ErrMFAInvalid }
	secret, err := s.decryptCredential(credential)
	if err != nil { return 0, 0, nil, err }
	step, valid := VerifyTOTPAndGetStep(secret, strings.TrimSpace(code), time.Now())
	if !valid {
		if recordErr := s.challenges.RecordFailure(ctx, hash); errors.Is(recordErr, repository.ErrMFALoginLocked) {
			return 0, 0, nil, ErrLoginLocked
		} else if recordErr != nil && !errors.Is(recordErr, repository.ErrMFAChallengeUnavailable) {
			return 0, 0, nil, recordErr
		}
		return 0, 0, nil, ErrMFAInvalid
	}
	if err := s.rotateCredential(ctx, principalID, credential, secret); err != nil { return 0, 0, nil, err }
	codes, hashes, err := GenerateRecoveryCodes(10)
	if err != nil { return 0, 0, nil, err }
	if err := s.credentials.ActivateEnrollment(ctx, principalID, hash, tokenVersion, step, hashes); err != nil {
		if errors.Is(err, repository.ErrMFAEnrollmentInvalid) { return 0, 0, nil, ErrMFAInvalid }
		return 0, 0, nil, err
	}
	return principalID, tokenVersion + 1, codes, nil
}

// VerifyLoginChallenge consumes a valid login challenge exactly once. The
// caller may issue a session only after this method returns the principal ID.
func (s *MFAService) VerifyLoginChallenge(ctx context.Context, challengeToken, code string) (int64, int64, error) {
	tokenVersion, tokenErr := parseMFAChallengeTokenVersion(challengeToken)
	if tokenErr != nil { return 0, 0, ErrMFAInvalid }
	hash := HashRefreshToken(strings.TrimSpace(challengeToken))
	challenge, err := s.challenges.Get(ctx, hash)
	if errors.Is(err, repository.ErrMFALoginLocked) {
		return 0, 0, ErrLoginLocked
	}
	if err != nil { return 0, 0, ErrMFAInvalid }
	if challenge.Purpose != repository.MFAChallengeLogin { return 0, 0, ErrMFAInvalid }
	credential, err := s.credentials.GetCredential(ctx, challenge.PrincipalID)
	if err != nil { return 0, 0, err }
	if credential == nil || credential.VerifiedAt == nil { return 0, 0, ErrMFAInvalid }
	normalizedCode := strings.TrimSpace(code)
	secret, decryptErr := s.decryptCredential(credential)
	if decryptErr == nil {
		step, valid := VerifyTOTPAndGetStep(secret, normalizedCode, time.Now())
		if valid {
			if err := s.rotateCredential(ctx, challenge.PrincipalID, credential, secret); err != nil { return 0, 0, err }
			if err := s.credentials.ConsumeLoginChallenge(ctx, challenge.PrincipalID, hash, tokenVersion, step); err != nil {
				if errors.Is(err, repository.ErrMFAEnrollmentInvalid) { return 0, 0, ErrMFAInvalid }
				return 0, 0, err
			}
			return challenge.PrincipalID, tokenVersion, nil
		}
	}
	// Recovery-code use is independent of the encrypted TOTP secret and is
	// committed together with challenge consumption.
	recoveryHash := HashRefreshToken(strings.ToUpper(normalizedCode))
	err = s.credentials.ConsumeLoginRecoveryCode(ctx, challenge.PrincipalID, hash, tokenVersion, recoveryHash)
	if err == nil { return challenge.PrincipalID, tokenVersion, nil }
	if !errors.Is(err, repository.ErrMFARecoveryCodeInvalid) && !errors.Is(err, repository.ErrMFAEnrollmentInvalid) { return 0, 0, err }
	if recordErr := s.challenges.RecordFailure(ctx, hash); errors.Is(recordErr, repository.ErrMFALoginLocked) {
		return 0, 0, ErrLoginLocked
	} else if recordErr != nil && !errors.Is(recordErr, repository.ErrMFAChallengeUnavailable) {
		return 0, 0, recordErr
	}
	return 0, 0, ErrMFAInvalid
}

// BeginRecoveryCodeRegeneration creates a short-lived, five-attempt step-up challenge.
func (s *MFAService) BeginRecoveryCodeRegeneration(ctx context.Context, principalID, tokenVersion int64) (*MFAStart, error) {
	required, err := s.credentials.IsMandatoryForPrincipal(ctx, principalID)
	if err != nil { return nil, err }
	if !required { return nil, ErrMFAInvalid }
	credential, err := s.credentials.GetCredential(ctx, principalID)
	if err != nil { return nil, err }
	if credential == nil || credential.VerifiedAt == nil { return nil, ErrMFAInvalid }
	token, expiry, err := s.newChallenge(ctx, principalID, tokenVersion, repository.MFAChallengeRecoveryRegeneration)
	if err != nil { return nil, err }
	return &MFAStart{Required: true, ChallengeToken: token, ExpiresAt: expiry}, nil
}

// RegenerateRecoveryCodes accepts a challenge-bound fresh TOTP and atomically
// consumes the challenge, old codes, and the matched authenticator time-step.
func (s *MFAService) RegenerateRecoveryCodes(ctx context.Context, challengeToken, code string) ([]string, error) {
	tokenVersion, tokenErr := parseMFAChallengeTokenVersion(challengeToken)
	if tokenErr != nil { return nil, ErrMFAInvalid }
	challengeHash := HashRefreshToken(strings.TrimSpace(challengeToken))
	challenge, err := s.challenges.Get(ctx, challengeHash)
	if errors.Is(err, repository.ErrMFALoginLocked) { return nil, ErrLoginLocked }
	if err != nil || challenge.Purpose != repository.MFAChallengeRecoveryRegeneration { return nil, ErrMFAInvalid }
	credential, err := s.credentials.GetCredential(ctx, challenge.PrincipalID)
	if err != nil { return nil, err }
	if credential == nil || credential.VerifiedAt == nil { return nil, ErrMFAInvalid }
	secret, err := s.decryptCredential(credential)
	if err != nil { return nil, err }
	step, valid := VerifyTOTPAndGetStep(secret, strings.TrimSpace(code), time.Now())
	if !valid {
		if recordErr := s.challenges.RecordFailure(ctx, challengeHash); errors.Is(recordErr, repository.ErrMFALoginLocked) {
			return nil, ErrLoginLocked
		} else if recordErr != nil && !errors.Is(recordErr, repository.ErrMFAChallengeUnavailable) {
			return nil, recordErr
		}
		return nil, ErrMFAInvalid
	}
	if err := s.rotateCredential(ctx, challenge.PrincipalID, credential, secret); err != nil { return nil, err }
	codes, hashes, err := GenerateRecoveryCodes(10)
	if err != nil { return nil, err }
	if err := s.credentials.ReplaceRecoveryCodeHashes(ctx, challenge.PrincipalID, challengeHash, tokenVersion, step, hashes); err != nil {
		if errors.Is(err, repository.ErrMFAEnrollmentInvalid) { return nil, ErrMFAInvalid }
		return nil, err
	}
	return codes, nil
}

func (s *MFAService) newChallenge(ctx context.Context, principalID, tokenVersion int64, purpose repository.MFAChallengePurpose) (string,time.Time,error) {
	if principalID <= 0 || tokenVersion <= 0 { return "",time.Time{},ErrMFAInvalid }
	raw:=make([]byte,32)
	if _,err:=rand.Read(raw); err!=nil { return "",time.Time{},fmt.Errorf("generate MFA challenge: %w",err) }
	token:=base64.RawURLEncoding.EncodeToString(raw)+"."+strconv.FormatInt(tokenVersion,10)
	expires:=time.Now().Add(5*time.Minute)
	if err:=s.challenges.Create(ctx,principalID,tokenVersion,purpose,HashRefreshToken(token),expires); err!=nil {
		if errors.Is(err, repository.ErrPrincipalVersionChanged) { return "",time.Time{},ErrSessionPrincipalVersionChanged }
		if errors.Is(err, repository.ErrMFALoginLocked) { return "",time.Time{},ErrLoginLocked }
		return "",time.Time{},err
	}
	return token,expires,nil
}

func parseMFAChallengeTokenVersion(token string) (int64,error) {
	parts:=strings.Split(strings.TrimSpace(token),".")
	if len(parts)!=2 || parts[0]=="" { return 0,ErrMFAInvalid }
	version,err:=strconv.ParseInt(parts[1],10,64)
	if err!=nil || version<=0 { return 0,ErrMFAInvalid }
	return version,nil
}

func (s *MFAService) decryptCredential(c *repository.MFACredential) ([]byte,error) {
	var key []byte
	if c == nil { return nil, ErrMFAKeyVersionUnavailable }
	if c.KeyVersion == s.keyVersion {
		key = s.encryptionKey
	} else {
		key = s.previousEncryptionKeys[c.KeyVersion]
	}
	if len(key) != 32 { return nil,ErrMFAKeyVersionUnavailable }
	secret,err:=DecryptTOTPSecret(c.Ciphertext,key)
	if err!=nil { return nil,fmt.Errorf("decrypt stored MFA credential: %w",err) }
	return secret,nil
}

// rotateCredential upgrades a successfully decrypted credential using an
// optimistic compare-and-swap update. A concurrent rotation is harmless.
func (s *MFAService) rotateCredential(ctx context.Context, principalID int64, c *repository.MFACredential, secret []byte) error {
	if c == nil || c.KeyVersion == s.keyVersion { return nil }
	rotator, ok := s.credentials.(mfaCredentialRotator)
	if !ok { return nil }
	ciphertext, err := EncryptTOTPSecret(secret, s.encryptionKey)
	if err != nil { return fmt.Errorf("encrypt rotated MFA credential: %w", err) }
	return rotator.ReencryptCredential(ctx, principalID, c.Ciphertext, c.KeyVersion, ciphertext, s.keyVersion)
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
