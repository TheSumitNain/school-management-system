package main

import (
	"strings"
	"testing"
)

const validSnapshot = `state_code,state_name,igod_state_code,district_code,district_name,subdistrict_code,subdistrict_name,locality_code,locality_name,locality_type
06,Haryana,HR,701,Ambala,12001,Ambala,900001,Ambala,village
06,Haryana,HR,701,Ambala,12001,Ambala,900002,Barara,village
06,Haryana,HR,702,Other District,12002,Other Tehsil,900003,Example Town,town
`

func TestValidateCSVBuildsUniqueHierarchy(t *testing.T) {
	got, err := validateCSV(strings.NewReader(validSnapshot))
	if err != nil {
		t.Fatalf("validateCSV() error = %v", err)
	}
	if len(got.states) != 1 || len(got.districts) != 2 ||
		len(got.subdistricts) != 2 || len(got.localities) != 3 {
		t.Fatalf("unexpected distinct hierarchy counts: states=%d districts=%d subdistricts=%d localities=%d",
			len(got.states), len(got.districts), len(got.subdistricts), len(got.localities))
	}
}

func TestValidateCSVRejectsWrongHeader(t *testing.T) {
	_, err := validateCSV(strings.NewReader("state,name\n06,Haryana\n"))
	if err == nil || !strings.Contains(err.Error(), "CSV column") {
		t.Fatalf("validateCSV() error = %v, want header error", err)
	}
}

func TestValidateCSVRejectsConflictingDistrictParent(t *testing.T) {
	input := validSnapshot + "07,Other State,OD,701,Ambala,12003,Other Tehsil,900004,Fourth Place,village\n"
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "conflicting name or parent mapping") {
		t.Fatalf("validateCSV() error = %v, want conflicting district-parent error", err)
	}
}

func TestValidateCSVRejectsConflictingSourceCode(t *testing.T) {
	input := validSnapshot + "06,Different State,HR,703,Third District,12003,Third Tehsil,900004,Fourth Place,village\n"
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "conflicting name or parent mapping") {
		t.Fatalf("validateCSV() error = %v, want conflicting source-code error", err)
	}
}

func TestValidateCSVRejectsDuplicateIGODCrosswalk(t *testing.T) {
	input := validSnapshot + "07,Another State,HR,704,Fourth District,12004,Fourth Tehsil,900005,Fifth Place,village\n"
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "IGOD state code") {
		t.Fatalf("validateCSV() error = %v, want duplicate IGOD crosswalk error", err)
	}
}

func TestValidateCSVRejectsUnsupportedLocalityType(t *testing.T) {
	input := strings.Replace(validSnapshot, "900001,Ambala,village", "900001,Ambala,block", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "unsupported locality_type") {
		t.Fatalf("validateCSV() error = %v, want locality type error", err)
	}
}

func TestValidateCSVRejectsConflictingSubdistrictParent(t *testing.T) {
	input := strings.Replace(validSnapshot, "06,Haryana,HR,701,Ambala,12001", "06,Haryana,HR,703,New District,12001", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "conflicting name or parent mapping") {
		t.Fatalf("validateCSV() error = %v, want conflicting sub-district-parent error", err)
	}
}

func TestValidateCSVRejectsConflictingLocalityParent(t *testing.T) {
	input := strings.Replace(validSnapshot, "900003,Example Town,town", "900001,Ambala,village", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "conflicting name or parent mapping") {
		t.Fatalf("validateCSV() error = %v, want conflicting locality-parent error", err)
	}
}

func TestValidateCSVRejectsMalformedIGODCrosswalkCode(t *testing.T) {
	input := strings.Replace(validSnapshot, ",HR,", ",hr,", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "igod_state_code") {
		t.Fatalf("validateCSV() error = %v, want malformed IGOD crosswalk error", err)
	}
}
