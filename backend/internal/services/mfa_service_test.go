package services

import (
	"encoding/hex"
	"testing"
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
		if _, ok := seenHashes[hashes[i]]; ok { t.Fatal("duplicate recovery hash") }
		seenHashes[hashes[i]] = struct{}{}
	}
}

func TestGenerateRecoveryCodesRejectsUnsafeCounts(t *testing.T) {
	for _, count := range []int{0, 1, 7, 13, 100} {
		if _, _, err := GenerateRecoveryCodes(count); err == nil { t.Fatalf("count %d should be rejected", count) }
	}
}
