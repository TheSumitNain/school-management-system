package services

import (
	"testing"
)

func TestParentOTPLoginServiceRequiresDelivery(t *testing.T) {
	if _, err := NewParentOTPLoginService(nil, nil, nil); err == nil {
		t.Fatal("expected constructor to reject missing dependencies")
	}
}
