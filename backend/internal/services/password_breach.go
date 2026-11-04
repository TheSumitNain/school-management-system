package services

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrCompromisedPassword = errors.New("password appears in known data breaches")
	ErrPasswordBreachCheckUnavailable = errors.New("unable to verify password against known data breaches")
)

// PasswordBreachChecker uses a privacy-preserving check and must never receive
// or transmit the cleartext password or its complete hash.
type PasswordBreachChecker interface {
	IsCompromised(ctx context.Context, password string) (bool, error)
}

type breachCacheEntry struct {
	suffixes map[string]struct{}
	expiresAt time.Time
}

type HIBPPasswordBreachChecker struct {
	client *http.Client
	mu sync.RWMutex
	cache map[string]breachCacheEntry
}

func NewHIBPPasswordBreachChecker(client *http.Client) *HIBPPasswordBreachChecker {
	if client == nil {
		client = &http.Client{Timeout: 4 * time.Second}
	}
	return &HIBPPasswordBreachChecker{
		client: client,
		cache: make(map[string]breachCacheEntry),
	}
}

func (c *HIBPPasswordBreachChecker) IsCompromised(ctx context.Context, password string) (bool, error) {
	if c == nil || c.client == nil {
		return false, fmt.Errorf("password breach checker is not configured")
	}
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hash[:5], hash[5:]

	now := time.Now()
	c.mu.RLock()
	entry, ok := c.cache[prefix]
	c.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		_, found := entry.suffixes[suffix]
		return found, nil
	}

	// HIBP's range API receives only the first five SHA-1 hex characters.
	// The remaining suffix is compared locally and is never sent over the wire.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pwnedpasswords.com/range/"+prefix, nil)
	if err != nil {
		return false, fmt.Errorf("build password range request: %w", err)
	}
	req.Header.Set("Add-Padding", "true")
	req.Header.Set("User-Agent", "school-management-system-password-policy")
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request password range: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("password range service returned HTTP %d", resp.StatusCode)
	}

	suffixes := make(map[string]struct{})
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 2<<20))
	scanner.Buffer(make([]byte, 4096), 2<<20)
	for scanner.Scan() {
		parts := strings.SplitN(strings.TrimSpace(scanner.Text()), ":", 2)
		if len(parts) != 2 || len(parts[0]) != 35 {
			return false, fmt.Errorf("password range service returned malformed data")
		}
		if _, err := fmt.Sscan(parts[1], new(int)); err != nil {
			return false, fmt.Errorf("password range service returned malformed count")
		}
		suffixes[strings.ToUpper(parts[0])] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read password range response: %w", err)
	}

	c.mu.Lock()
	if len(c.cache) >= 256 {
		for key, cached := range c.cache {
			if !now.Before(cached.expiresAt) || len(c.cache) >= 256 {
				delete(c.cache, key)
			}
		}
	}
	c.cache[prefix] = breachCacheEntry{suffixes: suffixes, expiresAt: now.Add(time.Hour)}
	c.mu.Unlock()

	_, found := suffixes[suffix]
	return found, nil
}

var defaultPasswordBreachChecker PasswordBreachChecker = NewHIBPPasswordBreachChecker(nil)

// HashPasswordContext enforces the local password policy and checks the
// Pwned Passwords range API before storing a bcrypt hash. The remote check is
// fail-closed: a service outage never silently bypasses the approved policy.
func HashPasswordContext(ctx context.Context, password string) (string, error) {
	return hashPasswordWithChecker(ctx, password, defaultPasswordBreachChecker)
}

func hashPasswordWithChecker(ctx context.Context, password string, checker PasswordBreachChecker) (string, error) {
	if !validNewPassword(password) {
		return "", ErrInvalidPassword
	}
	if checker == nil {
		return "", ErrPasswordBreachCheckUnavailable
	}
	compromised, err := checker.IsCompromised(ctx, password)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPasswordBreachCheckUnavailable, err)
	}
	if compromised {
		return "", ErrCompromisedPassword
	}
	return HashPassword(password)
}
