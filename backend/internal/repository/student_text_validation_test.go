package repository

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeAddressText(t *testing.T) {
	t.Run("nil remains nil", func(t *testing.T) {
		got, err := normalizeAddressText(nil, 3)
		if err != nil || got != nil {
			t.Fatalf("normalizeAddressText(nil): got %v, err %v; want nil, nil", got, err)
		}
	})

	t.Run("blank becomes nil", func(t *testing.T) {
		value := " \t\n "
		got, err := normalizeAddressText(&value, 3)
		if err != nil || got != nil {
			t.Fatalf("normalizeAddressText(blank): got %v, err %v; want nil, nil", got, err)
		}
	})

	t.Run("trims before counting Unicode code points", func(t *testing.T) {
		value := "  " + strings.Repeat("अ", 3) + "  "
		got, err := normalizeAddressText(&value, 3)
		if err != nil {
			t.Fatalf("normalizeAddressText(exact Unicode boundary): %v", err)
		}
		if got == nil || *got != strings.Repeat("अ", 3) {
			t.Fatalf("normalized value = %v; want exactly three Hindi code points", got)
		}
	})

	t.Run("rejects one code point over the limit", func(t *testing.T) {
		value := strings.Repeat("अ", 4)
		got, err := normalizeAddressText(&value, 3)
		if got != nil || !errors.Is(err, ErrInvalidAcademicInput) {
			t.Fatalf("normalizeAddressText(over limit): got %v, err %v; want nil, ErrInvalidAcademicInput", got, err)
		}
	})
}

func TestValidStudentFullName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "", want: false},
		{name: " ", want: false},
		{name: "a", want: false},
		{name: "अ", want: false},
		{name: "AB", want: true},
		{name: " आरव ", want: true},
		{name: strings.Repeat("अ", 200), want: true},
		{name: strings.Repeat("अ", 201), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validStudentFullName(test.name); got != test.want {
				t.Fatalf("validStudentFullName(%q) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}
