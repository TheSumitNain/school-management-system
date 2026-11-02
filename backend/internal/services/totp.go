package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpSecretBytes = 20
	totpPeriod      = 30 * time.Second
	totpDigits      = 6
)

// GenerateTOTPSecret creates a cryptographically random 160-bit TOTP secret.
// The returned value is unpadded Base32, as expected by authenticator apps.
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, totpSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

// EncryptTOTPSecret encrypts a decoded TOTP secret using AES-256-GCM.
// The 12-byte nonce is prepended to the authenticated ciphertext.
func EncryptTOTPSecret(secret, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("TOTP encryption key must be exactly 32 bytes")
	}
	if len(secret) < 16 || len(secret) > 64 {
		return nil, fmt.Errorf("TOTP secret must contain between 16 and 64 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize TOTP encryption: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize TOTP authenticated encryption: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate TOTP encryption nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, secret, []byte("school-management-system:totp:v1")), nil
}

// DecryptTOTPSecret authenticates and decrypts a value produced by
// EncryptTOTPSecret. Callers must obtain key material from a secret manager.
func DecryptTOTPSecret(ciphertext, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("TOTP encryption key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize TOTP decryption: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize TOTP authenticated decryption: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize()+gcm.Overhead()+16 {
		return nil, fmt.Errorf("invalid encrypted TOTP secret")
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	secret, err := gcm.Open(nil, nonce, sealed, []byte("school-management-system:totp:v1"))
	if err != nil {
		return nil, fmt.Errorf("decrypt TOTP secret: authentication failed")
	}
	if len(secret) < 16 || len(secret) > 64 {
		return nil, fmt.Errorf("decrypted TOTP secret has invalid length")
	}
	return secret, nil
}

// VerifyTOTP accepts a six-digit SHA-1 TOTP code in the current 30-second
// time step, with one adjacent step of clock-skew tolerance in either direction.
func VerifyTOTP(secret []byte, code string, now time.Time) bool {
	if len(secret) < 16 || len(secret) > 64 || len(code) != totpDigits || now.Unix() < 0 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	counter := now.Unix() / int64(totpPeriod/time.Second)
	for offset := int64(-1); offset <= 1; offset++ {
		step := counter + offset
		if step < 0 {
			continue
		}
		expected := totpCode(secret, uint64(step))
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func totpCode(secret []byte, counter uint64) string {
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)
	modulus := uint32(1)
	for i := 0; i < totpDigits; i++ {
		modulus *= 10
	}
	return fmt.Sprintf("%0*d", totpDigits, value%modulus)
}

// TOTPProvisioningURI builds an otpauth URI for authenticator applications.
// The caller is responsible for displaying the secret only during enrollment.
func TOTPProvisioningURI(issuer, accountName, secret string) (string, error) {
	issuer = strings.TrimSpace(issuer)
	accountName = strings.TrimSpace(accountName)
	secret = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if issuer == "" || accountName == "" || secret == "" {
		return "", fmt.Errorf("issuer, account name, and TOTP secret are required")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil || len(decoded) < 16 || len(decoded) > 64 {
		return "", fmt.Errorf("invalid Base32 TOTP secret")
	}
	label := url.PathEscape(issuer + ":" + accountName)
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", issuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", strconv.Itoa(totpDigits))
	values.Set("period", strconv.Itoa(int(totpPeriod/time.Second)))
	return "otpauth://totp/" + label + "?" + values.Encode(), nil
}
