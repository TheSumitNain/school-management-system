package repository

import (
	"context"
	"testing"
	"time"
)

// TestStudentLocationIGODStateReferenceV40PostgresIntegration verifies that
// V40 seeds the current 36 State/UT names from the Government of India IGOD
// directory and preserves their source codes for future reference-data updates.
func TestStudentLocationIGODStateReferenceV40PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	expected := map[string]string{
		"AN": "Andaman and Nicobar Islands",
		"AP": "Andhra Pradesh",
		"AR": "Arunachal Pradesh",
		"AS": "Assam",
		"BR": "Bihar",
		"CH": "Chandigarh",
		"CG": "Chhattisgarh",
		"ND": "Dadra and Nagar Haveli and Daman and Diu",
		"DL": "Delhi",
		"GA": "Goa",
		"GJ": "Gujarat",
		"HR": "Haryana",
		"HP": "Himachal Pradesh",
		"JK": "Jammu and Kashmir",
		"JH": "Jharkhand",
		"KA": "Karnataka",
		"KL": "Kerala",
		"LA": "Ladakh",
		"LD": "Lakshadweep",
		"MP": "Madhya Pradesh",
		"MH": "Maharashtra",
		"MN": "Manipur",
		"ML": "Meghalaya",
		"MZ": "Mizoram",
		"NL": "Nagaland",
		"OD": "Odisha",
		"PY": "Puducherry",
		"PB": "Punjab",
		"RJ": "Rajasthan",
		"SK": "Sikkim",
		"TN": "Tamil Nadu",
		"TS": "Telangana",
		"TR": "Tripura",
		"UP": "Uttar Pradesh",
		"UK": "Uttarakhand",
		"WB": "West Bengal",
	}

	rows, err := pool.Query(ctx, `
		SELECT source_code, name
		FROM student_locations
		WHERE school_id IS NULL
		AND country_code = 'IN'
		AND level = 'state'
		AND source_system = 'IGOD'
		ORDER BY source_code
	`)
	if err != nil {
		t.Fatalf("query V40 IGOD State/UT reference rows: %v", err)
	}
	defer rows.Close()

	found := make(map[string]string, len(expected))
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			t.Fatalf("scan V40 IGOD State/UT reference row: %v", err)
		}
		found[code] = name
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate V40 IGOD State/UT reference rows: %v", err)
	}
	if len(found) != len(expected) {
		t.Fatalf("V40 seeded %d State/UT rows, want %d", len(found), len(expected))
	}
	for code, name := range expected {
		if found[code] != name {
			t.Errorf("IGOD code %s maps to %q, want %q", code, found[code], name)
		}
	}
}
