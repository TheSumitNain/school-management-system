// Command lgdvalidate checks a normalized, reviewed LGD hierarchy snapshot.
// It is deliberately read-only: it never connects to PostgreSQL or imports data.
package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type record struct {
	code, name, parent, igod string
}

type snapshot struct {
	states      map[string]record
	districts   map[string]record
	subdistricts map[string]record
	localities  map[string]record
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

	s, err := validateCSV(f)
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
	for _, count := range counts {
		fmt.Printf("%s: %d\n", count.name, count.got)
	}
	fmt.Println("Validation: PASS (read-only; no database connection or writes)")
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
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
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
		if !validLocalityType(row[9]) {
			return nil, fmt.Errorf("CSV line %d: unsupported locality_type %q", line, row[9])
		}
		if err := addRecord(s.states, row[0], row[1], "", row[2]); err != nil {
			return nil, fmt.Errorf("CSV line %d state: %w", line, err)
		}
		if err := addRecord(s.districts, row[3], row[4], row[0], ""); err != nil {
			return nil, fmt.Errorf("CSV line %d district: %w", line, err)
		}
		if err := addRecord(s.subdistricts, row[5], row[6], row[3], ""); err != nil {
			return nil, fmt.Errorf("CSV line %d sub-district: %w", line, err)
		}
		if err := addRecord(s.localities, row[7], row[8], row[5], ""); err != nil {
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

func addRecord(target map[string]record, code, name, parent, igod string) error {
	if code == "" || name == "" {
		return errors.New("source code and name must not be empty")
	}
	if len(code) > 50 || len(name) > 120 {
		return errors.New("source code exceeds 50 or name exceeds 120 characters")
	}
	candidate := record{code: code, name: name, parent: parent, igod: igod}
	if previous, exists := target[code]; exists {
		if previous != candidate {
			return fmt.Errorf("source code %q has conflicting name or parent mapping", code)
		}
		return nil
	}
	target[code] = candidate
	return nil
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
