package repository

import (
	"strings"
	"testing"
	"time"
)

func TestStaffAuditEventDoesNotExposeRawMetadata(t *testing.T) {
	event := StaffAuditEvent{
		ID: 7,
		Action: "staff.profile.updated",
		ResourceType: "staff_principal",
		Outcome: "success",
		ActorUsername: "school-admin",
		CreatedAt: time.Date(2026, 11, 23, 7, 0, 0, 0, time.UTC),
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal staff audit event: %v", err)
	}
	body := string(encoded)
	for _, forbidden := range []string{"metadata", "must-not-leak", "sensitive-value", "password"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("staff audit API JSON exposed %q: %s", forbidden, body)
		}
	}
	for _, expected := range []string{"staff.profile.updated", "school-admin", "success"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("staff audit API JSON omitted expected safe field %q: %s", expected, body)
		}
	}
}
