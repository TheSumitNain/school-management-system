package handlers

import "testing"

func TestParseStaffAttendanceDate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
	}{
		{name: "valid date", value: "2026-11-23", wantOK: true},
		{name: "year one", value: "0001-01-01", wantOK: true},
		{name: "year zero", value: "0000-01-01"},
		{name: "invalid calendar date", value: "2026-02-30"},
		{name: "non canonical date", value: "2026-1-01"},
		{name: "whitespace", value: " 2026-11-23 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseStaffAttendanceDate(tt.value)
			if ok != tt.wantOK {
				t.Fatalf("valid=%v want=%v", ok, tt.wantOK)
			}
		})
	}
}

func TestParseStaffAttendanceMonth(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantOK  bool
	}{
		{name: "valid month", value: "2026-11", wantOK: true},
		{name: "year one", value: "0001-01", wantOK: true},
		{name: "year zero", value: "0000-01"},
		{name: "invalid month", value: "2026-13"},
		{name: "non canonical month", value: "2026-1"},
		{name: "whitespace", value: " 2026-11 "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseStaffAttendanceMonth(tt.value)
			if ok != tt.wantOK {
				t.Fatalf("valid=%v want=%v", ok, tt.wantOK)
			}
		})
	}
}
