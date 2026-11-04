package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type alwaysClearPasswordChecker struct{}
func (alwaysClearPasswordChecker) IsCompromised(context.Context,string) (bool,error) { return false,nil }

type fixedPasswordChecker struct { compromised bool; err error }
func (f fixedPasswordChecker) IsCompromised(context.Context,string) (bool,error) { return f.compromised,f.err }

func TestHIBPPasswordBreachCheckerUsesKAnonymityAndCachesPrefix(t *testing.T) {
	password := "A-unique-password-For-Testing-2026!"
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hash[:5], hash[5:]
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/range/"+prefix { t.Errorf("range request exposed unexpected path: %q",r.URL.Path) }
		if r.Header.Get("Add-Padding") != "true" { t.Errorf("Add-Padding header missing") }
		if strings.Contains(r.URL.Path,password) || strings.Contains(r.URL.Path,hash) { t.Errorf("request exposed password or complete hash") }
		_,_ = fmt.Fprintf(w,"%s:12\n%s:0\n",suffix,strings.Repeat("A",35))
	}))
	defer server.Close()

	checker := NewHIBPPasswordBreachChecker(server.Client())
	checker.endpoint = server.URL
	for i:=0;i<2;i++ {
		compromised,err:=checker.IsCompromised(context.Background(),password)
		if err!=nil { t.Fatalf("check breached password: %v",err) }
		if !compromised { t.Fatal("password present in range response must be marked compromised") }
	}
	if got:=requests.Load();got!=1 { t.Fatalf("expected cached range lookup after first request, got %d requests",got) }
}

func TestHashPasswordWithCheckerRejectsCompromisedPassword(t *testing.T) {
	_,err:=hashPasswordWithChecker(context.Background(),"Another-Strong-Password-2026!",fixedPasswordChecker{compromised:true})
	if !errors.Is(err,ErrCompromisedPassword) { t.Fatalf("expected compromised-password error, got %v",err) }
}

func TestHashPasswordWithCheckerFailsClosedWhenBreachCheckUnavailable(t *testing.T) {
	_,err:=hashPasswordWithChecker(context.Background(),"Another-Strong-Password-2026!",fixedPasswordChecker{err:errors.New("offline")})
	if !errors.Is(err,ErrPasswordBreachCheckUnavailable) { t.Fatalf("expected breach-check-unavailable error, got %v",err) }
}

func TestHashPasswordWithCheckerKeepsLocalPasswordPolicy(t *testing.T) {
	_,err:=hashPasswordWithChecker(context.Background(),"password1234",alwaysClearPasswordChecker{})
	if !errors.Is(err,ErrInvalidPassword) { t.Fatalf("expected local password policy error, got %v",err) }
}

func TestHIBPPasswordBreachCheckerIgnoresZeroCountPadding(t *testing.T) {
	password := "Password-that-is-not-breached-2026!"
	sum := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(sum[:]))
	suffix := hash[5:]
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		_,_ = fmt.Fprintf(w,"%s:0\\n",suffix)
	}))
	defer server.Close()
	checker := NewHIBPPasswordBreachChecker(server.Client())
	checker.endpoint = server.URL
	compromised,err:=checker.IsCompromised(context.Background(),password)
	if err!=nil { t.Fatalf("check zero-count padding entry: %v",err) }
	if compromised { t.Fatal("zero-count padding entry must not be treated as a breached password") }
}
