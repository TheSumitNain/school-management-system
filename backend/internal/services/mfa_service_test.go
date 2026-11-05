package services

import (
	"encoding/hex"
	"context"
	"testing"

	"school-management-system/internal/repository"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	codes, hashes, err := GenerateRecoveryCodes(10)
	if err != nil { t.Fatalf("GenerateRecoveryCodes: %v", err) }
	if len(codes) != 10 || len(hashes) != 10 { t.Fatalf("got %d codes and %d hashes; want 10", len(codes), len(hashes)) }
	seenCodes := map[string]struct{}{}
	seenHashes := map[string]struct{}{}
	for i, code := range codes {
		if len(code) != 26 { t.Fatalf("code %d length=%d; want 26", i, len(code)) }
		if _, ok := seenCodes[code]; ok { t.Fatal("duplicate recovery code") }
		seenCodes[code] = struct{}{}
		if len(hashes[i]) != 64 { t.Fatalf("hash %d length=%d; want 64", i, len(hashes[i])) }
		if _, err := hex.DecodeString(hashes[i]); err != nil { t.Fatalf("hash %d is not hex: %v", i, err) }
		if hashes[i] != HashRefreshToken(code) { t.Fatalf("hash %d does not match its recovery code", i) }
		if _, ok := seenHashes[hashes[i]]; ok { t.Fatal("duplicate recovery hash") }
		seenHashes[hashes[i]] = struct{}{}
	}
}

func TestGenerateRecoveryCodesRejectsUnsafeCounts(t *testing.T) {
	for _, count := range []int{0, 1, 7, 13, 100} {
		if _, _, err := GenerateRecoveryCodes(count); err == nil { t.Fatalf("count %d should be rejected", count) }
	}
}

func TestParseMFAChallengeTokenVersion(t *testing.T) {
	for _, test := range []struct {
		token string
		want  int64
		ok    bool
	}{
		{token: "opaque-random-token.7", want: 7, ok: true},
		{token: "opaque-random-token.1", want: 1, ok: true},
		{token: "opaque-random-token", ok: false},
		{token: "opaque-random-token.0", ok: false},
		{token: "opaque-random-token.-1", ok: false},
		{token: ".7", ok: false},
		{token: "opaque-random-token.not-a-version", ok: false},
	} {
		got, err := parseMFAChallengeTokenVersion(test.token)
		if test.ok {
			if err != nil || got != test.want {
				t.Fatalf("parseMFAChallengeTokenVersion(%q) = %d, %v; want %d, nil", test.token, got, err, test.want)
			}
		} else if err == nil {
			t.Fatalf("parseMFAChallengeTokenVersion(%q) unexpectedly accepted version %d", test.token, got)
		}
	}
}

func TestMFAServiceDecryptsPreviousEncryptionKeyVersions(t *testing.T) {
	oldKey := make([]byte, 32)
	oldKey[0] = 7
	currentKey := make([]byte, 32)
	currentKey[0] = 9
	secret := []byte("a test authenticator secret")
	ciphertext, err := EncryptTOTPSecret(secret, oldKey)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret: %v", err)
	}
	service := &MFAService{
		encryptionKey: currentKey,
		keyVersion: 3,
		previousEncryptionKeys: map[int][]byte{2: oldKey},
	}
	got, err := service.decryptCredential(&repository.MFACredential{
		Ciphertext: ciphertext,
		KeyVersion: 2,
	})
	if err != nil {
		t.Fatalf("decryptCredential with previous key: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("decrypted secret = %q, want %q", got, secret)
	}
}

func TestMFAServiceRejectsUnavailablePreviousKeyVersion(t *testing.T) {
	service := &MFAService{
		encryptionKey: make([]byte, 32),
		keyVersion: 3,
		previousEncryptionKeys: map[int][]byte{2: make([]byte, 32)},
	}
	_, err := service.decryptCredential(&repository.MFACredential{
		Ciphertext: make([]byte, 32),
		KeyVersion: 1,
	})
	if err != ErrMFAKeyVersionUnavailable {
		t.Fatalf("decryptCredential error = %v, want ErrMFAKeyVersionUnavailable", err)
	}
}

type mfaRotationTestStore struct {
	oldCiphertext []byte
	oldVersion int
	newCiphertext []byte
	newVersion int
}

func (s *mfaRotationTestStore) IsMandatoryForPrincipal(context.Context, int64) (bool, error) { return true, nil }
func (s *mfaRotationTestStore) GetCredential(context.Context, int64) (*repository.MFACredential, error) { return nil, nil }
func (s *mfaRotationTestStore) SavePendingCredential(context.Context, int64, []byte, int) error { return nil }
func (s *mfaRotationTestStore) ActivateEnrollment(context.Context, int64, string, int64, int64, []string) error { return nil }
func (s *mfaRotationTestStore) ConsumeLoginChallenge(context.Context, int64, string, int64, int64) error { return nil }
func (s *mfaRotationTestStore) ConsumeLoginRecoveryCode(context.Context, int64, string, int64, string) error { return nil }
func (s *mfaRotationTestStore) ReplaceRecoveryCodeHashes(context.Context, int64, string, int64, int64, []string) error { return nil }
func (s *mfaRotationTestStore) ReencryptCredential(_ context.Context, _ int64, oldCiphertext []byte, oldVersion int, newCiphertext []byte, newVersion int) error {
	s.oldCiphertext = append([]byte(nil), oldCiphertext...)
	s.oldVersion = oldVersion
	s.newCiphertext = append([]byte(nil), newCiphertext...)
	s.newVersion = newVersion
	return nil
}

func TestMFAServiceRotatesCredentialToCurrentKey(t *testing.T) {
	oldKey := make([]byte, 32)
	oldKey[0] = 4
	currentKey := make([]byte, 32)
	currentKey[0] = 8
	secret := []byte("rotation test TOTP secret")
	oldCiphertext, err := EncryptTOTPSecret(secret, oldKey)
	if err != nil { t.Fatalf("EncryptTOTPSecret(old): %v", err) }

	store := &mfaRotationTestStore{}
	service := &MFAService{
		credentials: store,
		encryptionKey: currentKey,
		keyVersion: 5,
	}
	credential := &repository.MFACredential{Ciphertext: oldCiphertext, KeyVersion: 4}
	if err := service.rotateCredential(context.Background(), 42, credential, secret); err != nil {
		t.Fatalf("rotateCredential: %v", err)
	}
	if store.oldVersion != 4 || store.newVersion != 5 {
		t.Fatalf("rotation versions = %d -> %d, want 4 -> 5", store.oldVersion, store.newVersion)
	}
	if string(store.oldCiphertext) != string(oldCiphertext) {
		t.Fatal("rotation did not compare against the original ciphertext")
	}
	decrypted, err := DecryptTOTPSecret(store.newCiphertext, currentKey)
	if err != nil { t.Fatalf("decrypt rotated ciphertext with current key: %v", err) }
	if string(decrypted) != string(secret) {
		t.Fatalf("rotated secret = %q, want %q", decrypted, secret)
	}
}
