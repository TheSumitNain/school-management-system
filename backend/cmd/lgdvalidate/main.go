// Command lgdvalidate checks a normalized, reviewed LGD hierarchy snapshot.
// It is deliberately read-only: it never connects to PostgreSQL or imports data.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type record struct {
	code, name, parent, igod, localityType string
}

type snapshot struct {
	states      map[string]record
	districts   map[string]record
	subdistricts map[string]record
	localities  map[string]record
}

// supportedIGODStateCodes is the reviewed V40 State/UT crosswalk vocabulary.
// LGD state codes remain independent; this set only prevents unknown IGOD references.
var supportedIGODStateCodes = map[string]struct{}{
	"AN": {}, "AP": {}, "AR": {}, "AS": {}, "BR": {}, "CH": {},
	"CG": {}, "ND": {}, "DL": {}, "GA": {}, "GJ": {}, "HR": {},
	"HP": {}, "JK": {}, "JH": {}, "KA": {}, "KL": {}, "LA": {},
	"LD": {}, "MP": {}, "MH": {}, "MN": {}, "ML": {}, "MZ": {},
	"NL": {}, "OD": {}, "PY": {}, "PB": {}, "RJ": {}, "SK": {},
	"TN": {}, "TS": {}, "TR": {}, "UP": {}, "UK": {}, "WB": {},
}

var expectedHeader = []string{
	"state_code", "state_name", "igod_state_code",
	"district_code", "district_name",
	"subdistrict_code", "subdistrict_name",
	"locality_code", "locality_name", "locality_type",
}

func main() {
	file := flag.String("file", "", "normalized LGD hierarchy CSV file")
	snapshotDate := flag.String("snapshot-date", "", "official source snapshot date (YYYY-MM-DD)")
	sourceFile := flag.String("source-file", "", "untouched official source file to checksum only (optional; never parsed or modified)")
	minStates := flag.Int("min-states", 36, "minimum distinct State/UT rows expected")
	minDistricts := flag.Int("min-districts", 1, "minimum distinct districts expected")
	minSubdistricts := flag.Int("min-subdistricts", 1, "minimum distinct sub-districts expected")
	minLocalities := flag.Int("min-localities", 1, "minimum distinct village/city localities expected")
	flag.Parse()

	if *file == "" || *snapshotDate == "" {
		fail(errors.New("-file and -snapshot-date are required"))
	}
	if _, err := time.Parse("2006-01-02", *snapshotDate); err != nil {
		fail(fmt.Errorf("invalid -snapshot-date: %w", err))
	}
	if *minStates < 1 || *minDistricts < 1 || *minSubdistricts < 1 || *minLocalities < 1 {
		fail(errors.New("all minimum-count flags must be positive"))
	}
	f, err := os.Open(*file)
	if err != nil {
		fail(fmt.Errorf("open snapshot: %w", err))
	}
	defer f.Close()

	hash := sha256.New()
	s, err := validateCSV(io.TeeReader(f, hash))
	if err != nil {
		fail(err)
	}
	counts := []struct {
		name string
		got, min int
	}{
		{"states/UTs", len(s.states), *minStates},
		{"districts", len(s.districts), *minDistricts},
		{"sub-districts", len(s.subdistricts), *minSubdistricts},
		{"localities", len(s.localities), *minLocalities},
	}
	for _, count := range counts {
		if count.got < count.min {
			fail(fmt.Errorf("snapshot has %d %s; minimum required is %d", count.got, count.name, count.min))
		}
	}
	fmt.Printf("Snapshot date: %s\n", *snapshotDate)
	fmt.Printf("Normalized CSV SHA-256: %x\n", hash.Sum(nil))
	if *sourceFile != "" {
		sourceHash, err := sha256File(*sourceFile)
		if err != nil {
			fail(fmt.Errorf("checksum original source file: %w", err))
		}
		fmt.Printf("Original source SHA-256: %s\n", sourceHash)
	}
	for _, count := range counts {
		fmt.Printf("%s: %d\n", count.name, count.got)
	}
	fmt.Println("Validation: PASS (read-only; no database connection or writes)")
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func validateCSV(input io.Reader) (*snapshot, error) {
	r := csv.NewReader(input)
	r.FieldsPerRecord = len(expectedHeader)
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	header[0] = strings.TrimPrefix(header[0], "\uFEFF")
	for i := range expectedHeader {
		header[i] = strings.ToLower(strings.TrimSpace(header[i]))
		if header[i] != expectedHeader[i] {
			return nil, fmt.Errorf("CSV column %d is %q; want %q", i+1, header[i], expectedHeader[i])
		}
	}

	s := &snapshot{
		states: make(map[string]record), districts: make(map[string]record),
		subdistricts: make(map[string]record), localities: make(map[string]record),
	}
	line := 1
	for {
		row, readErr := r.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		line++
		if readErr != nil {
			return nil, fmt.Errorf("CSV line %d: %w", line, readErr)
		}
		for i, value := range row {
			if !utf8.ValidString(value) {
				return nil, fmt.Errorf("CSV line %d: %s is not valid UTF-8", line, expectedHeader[i])
			}
			if strings.ContainsRune(value, '\x00') {
				return nil, fmt.Errorf("CSV line %d: %s contains a NUL byte unsupported by PostgreSQL text", line, expectedHeader[i])
			}
			row[i] = strings.TrimSpace(value)
		}
		if row[0] == "" && row[1] == "" && row[2] == "" &&
			row[3] == "" && row[4] == "" && row[5] == "" &&
			row[6] == "" && row[7] == "" && row[8] == "" && row[9] == "" {
			continue
		}
		for i, value := range row {
			if value == "" {
				return nil, fmt.Errorf("CSV line %d: %s is required", line, expectedHeader[i])
			}
		}
		if len(row[2]) != 2 || row[2] != strings.ToUpper(row[2]) {
			return nil, fmt.Errorf("CSV line %d: igod_state_code must be the two-character V40 code", line)
		}
		if _, ok := supportedIGODStateCodes[row[2]]; !ok {
			return nil, fmt.Errorf("CSV line %d: igod_state_code %q is not in the reviewed V40 IGOD State/UT crosswalk", line, row[2])
		}
		if !validLocalityType(row[9]) {
			return nil, fmt.Errorf("CSV line %d: unsupported locality_type %q", line, row[9])
		}
		if err := addRecord(s.states, row[0], row[1], "", row[2], ""); err != nil {
			return nil, fmt.Errorf("CSV line %d state: %w", line, err)
		}
		if err := addRecord(s.districts, row[3], row[4], row[0], "", ""); err != nil {
			return nil, fmt.Errorf("CSV line %d district: %w", line, err)
		}
		if err := addRecord(s.subdistricts, row[5], row[6], row[3], "", ""); err != nil {
			return nil, fmt.Errorf("CSV line %d sub-district: %w", line, err)
		}
		if err := addLocalityRecord(s.localities, row[7], row[8], row[5], row[9]); err != nil {
			return nil, fmt.Errorf("CSV line %d locality: %w", line, err)
		}
	}
	if len(s.states) == 0 {
		return nil, errors.New("snapshot contains no data rows")
	}
	for code, district := range s.districts {
		if _, ok := s.states[district.parent]; !ok {
			return nil, fmt.Errorf("district %q references missing state code %q", code, district.parent)
		}
	}
	for code, subdistrict := range s.subdistricts {
		if _, ok := s.districts[subdistrict.parent]; !ok {
			return nil, fmt.Errorf("sub-district %q references missing district code %q", code, subdistrict.parent)
		}
	}
	for code, locality := range s.localities {
		if _, ok := s.subdistricts[locality.parent]; !ok {
			return nil, fmt.Errorf("locality %q references missing sub-district code %q", code, locality.parent)
		}
	}
	seenIGOD := make(map[string]string, len(s.states))
	for code, state := range s.states {
		if prior, exists := seenIGOD[state.igod]; exists {
			return nil, fmt.Errorf("IGOD state code %q is mapped to both LGD state %q and %q", state.igod, prior, code)
		}
		seenIGOD[state.igod] = code
	}
	return s, nil
}

func addRecord(target map[string]record, code, name, parent, igod, localityType string) error {
	if code == "" || name == "" {
		return errors.New("source code and name must not be empty")
	}
	if utf8.RuneCountInString(code) > 50 || utf8.RuneCountInString(name) > 120 {
		return errors.New("source code exceeds 50 or name exceeds 120 characters")
	}
	candidate := record{code: code, name: name, parent: parent, igod: igod, localityType: localityType}
	if previous, exists := target[code]; exists {
		if previous != candidate {
			return fmt.Errorf("source code %q has conflicting name or parent mapping", code)
		}
		return nil
	}
	target[code] = candidate
	return nil
}

func addLocalityRecord(target map[string]record, code, name, parent, localityType string) error {
	candidate := record{code: code, name: name, parent: parent, localityType: localityType}
	if previous, exists := target[code]; exists {
		if previous == candidate {
			return fmt.Errorf("duplicate locality row for source code %q", code)
		}
		return fmt.Errorf("source code %q has conflicting name, parent, or locality type", code)
	}
	return addRecord(target, code, name, parent, "", localityType)
}

func validLocalityType(value string) bool {
	switch value {
	case "village", "city", "town", "other":
		return true
	default:
		return false
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "LGD snapshot validation failed:", err)
	os.Exit(1)
}
