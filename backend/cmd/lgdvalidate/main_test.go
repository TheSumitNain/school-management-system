package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
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
	_, err := validateCSV(strings.NewReader("state,name,igod_state_code,district_code,district_name,subdistrict_code,subdistrict_name,locality_code,locality_name,locality_type\n06,Haryana,HR,701,Ambala,12001,Ambala,900001,Ambala,village\n"))
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
	if err == nil || !strings.Contains(err.Error(), "conflicting name, parent, or locality type") {
		t.Fatalf("validateCSV() error = %v, want conflicting locality-parent error", err)
	}
}

func TestValidateCSVRejectsMalformedIGODCrosswalkCode(t *testing.T) {
	input := strings.ReplaceAll(validSnapshot, ",HR,", ",hr,")
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "igod_state_code") {
		t.Fatalf("validateCSV() error = %v, want malformed IGOD crosswalk error", err)
	}
}

func TestValidateCSVPreservesLeadingZeroCodesAndBOM(t *testing.T) {
	input := "\uFEFF" + validSnapshot
	got, err := validateCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("validateCSV() error = %v", err)
	}
	if _, ok := got.states["06"]; !ok {
		t.Fatalf("state source code lost leading zero: keys=%v", got.states)
	}
	if _, ok := got.localities["900001"]; !ok {
		t.Fatalf("locality source code was not retained as a string: keys=%v", got.localities)
	}
}

func TestValidateCSVRejectsConflictingLocalityTypeForSameSourceCode(t *testing.T) {
	input := validSnapshot + "06,Haryana,HR,701,Ambala,12001,Ambala,900001,Ambala,city\n"
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "conflicting name, parent, or locality type") {
		t.Fatalf("validateCSV() error = %v, want conflicting locality-type error", err)
	}
}

func TestValidateCSVRejectsUnknownIGODCrosswalkCode(t *testing.T) {
	input := strings.ReplaceAll(validSnapshot, ",HR,", ",ZZ,")
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "not in the reviewed V40 IGOD") {
		t.Fatalf("validateCSV() error = %v, want unknown IGOD crosswalk error", err)
	}
}

func TestValidateCSVRejectsDuplicateLocalityRow(t *testing.T) {
	input := validSnapshot + "06,Haryana,HR,701,Ambala,12001,Ambala,900001,Ambala,village\n"
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "duplicate locality row") {
		t.Fatalf("validateCSV() error = %v, want duplicate locality row error", err)
	}
}

func TestAddRecordCountsUnicodeCharactersLikePostgresVarchar(t *testing.T) {
	target := make(map[string]record)
	name120 := strings.Repeat("अ", 120)
	if err := addRecord(target, "01", name120, "", "HR", ""); err != nil {
		t.Fatalf("addRecord() rejected 120 Unicode characters: %v", err)
	}
	if err := addRecord(target, "02", strings.Repeat("अ", 121), "", "HR", ""); err == nil {
		t.Fatal("addRecord() accepted 121 Unicode characters; want VARCHAR(120) limit")
	}
}

func TestValidateCSVAllowsSameSourceCodeAcrossEntityNamespaces(t *testing.T) {
	// V42 source_system values are entity-specific (LGD_STATE, LGD_DISTRICT,
	// etc.), so identical code strings in different entity classes are distinct.
	input := strings.ReplaceAll(validSnapshot, ",701,Ambala,", ",06,Ambala,")
	got, err := validateCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("validateCSV() rejected code reuse across separate entity namespaces: %v", err)
	}
	if _, ok := got.states["06"]; !ok {
		t.Fatal("expected state source code 06 to remain present")
	}
	if _, ok := got.districts["06"]; !ok {
		t.Fatal("expected district source code 06 to remain independently present")
	}
}

func TestValidateCSVRejectsInvalidUTF8(t *testing.T) {
	input := strings.Replace(validSnapshot, "Ambala", "Amb\xffala", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("validateCSV() error = %v, want invalid UTF-8 error", err)
	}
}

func TestValidateCSVRejectsPostgresUnsupportedNUL(t *testing.T) {
	input := strings.Replace(validSnapshot, "Ambala", "Amb\x00ala", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "contains a NUL byte") {
		t.Fatalf("validateCSV() error = %v, want NUL byte error", err)
	}
}

func TestSHA256FileHashesOriginalBytesWithoutParsing(t *testing.T) {
	content := []byte{0x00, 0x01, 0x02, 0xff, '\n'}
	path := filepath.Join(t.TempDir(), "official-source.xls")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	got, err := sha256File(path)
	if err != nil {
		t.Fatalf("sha256File() error = %v", err)
	}
	wantBytes := sha256.Sum256(content)
	want := hex.EncodeToString(wantBytes[:])
	if got != want {
		t.Fatalf("sha256File() = %q, want %q", got, want)
	}
}

func TestValidateCSVRejectsMissingStateForDistrict(t *testing.T) {
	input := validSnapshot + "07,Other State,OD,703,Orphan District,12003,Orphan Tehsil,900004,Orphan Village,village\n"
	input = strings.Replace(input, "07,Other State,OD,703,Orphan District", "08,Other State,OD,703,Orphan District", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "references missing state code") {
		t.Fatalf("validateCSV() error = %v, want missing-state hierarchy error", err)
	}
}

func TestValidateCSVRejectsMissingDistrictForSubdistrict(t *testing.T) {
	input := validSnapshot + "06,Haryana,HR,703,Other District,12003,Orphan Tehsil,900004,Orphan Village,village\n"
	input = strings.Replace(input, "703,Other District,12003", "704,Other District,12003", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "references missing district code") {
		t.Fatalf("validateCSV() error = %v, want missing-district hierarchy error", err)
	}
}

func TestValidateCSVRejectsMissingSubdistrictForLocality(t *testing.T) {
	input := validSnapshot + "06,Haryana,HR,701,Ambala,12003,Other Tehsil,900004,Orphan Village,village\n"
	input = strings.Replace(input, "12003,Other Tehsil,900004", "12004,Other Tehsil,900004", 1)
	_, err := validateCSV(strings.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "references missing sub-district code") {
		t.Fatalf("validateCSV() error = %v, want missing-subdistrict hierarchy error", err)
	}
}

func TestValidateCSVRejectsHeaderOnlySnapshot(t *testing.T) {
	_, err := validateCSV(strings.NewReader(strings.Join(expectedHeader, ",") + "\n"))
	if err == nil || !strings.Contains(err.Error(), "contains no data rows") {
		t.Fatalf("validateCSV() error = %v, want empty-snapshot error", err)
	}
}
