package peachfuzz

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestProjectIDOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "peachfuzz"},
		{name: "valid hyphen", value: "peach-fuzz-2026"},
		{name: "empty", wantErr: true},
		{name: "uppercase", value: "Peachfuzz", wantErr: true},
		{name: "leading digit", value: "2peachfuzz", wantErr: true},
		{name: "trailing hyphen", value: "peachfuzz-", wantErr: true},
		{name: "space", value: "peach fuzz", wantErr: true},
		{name: "over cap", value: "a" + strings.Repeat("b", ProjectIDMaxBytes), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseProjectID(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParseProjectID() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParseProjectID() error = %v", err)
			}
		})
	}
}

func TestPackageImportPathOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "github.com/offGridSoft/peachfuzz/internal/run"},
		{name: "empty", wantErr: true},
		{name: "empty segment", value: "example.com//run", wantErr: true},
		{name: "dot segment", value: "example.com/./run", wantErr: true},
		{name: "parent segment", value: "example.com/../run", wantErr: true},
		{name: "leading option root", value: "-run/example.com", wantErr: true},
		{name: "leading option nested", value: "example.com/-run", wantErr: true},
		{name: "space", value: "example.com/peach fuzz", wantErr: true},
		{name: "control byte", value: "example.com/peach\nfuzz", wantErr: true},
		{name: "over cap", value: strings.Repeat("a", PackageImportPathMaxBytes+1), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParsePackageImportPath(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParsePackageImportPath() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParsePackageImportPath() error = %v", err)
			}
		})
	}
}

func TestFuzzTargetNameOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	max := FuzzTargetPrefix + "A" + strings.Repeat("a", FuzzTargetNameMaxBytes-len(FuzzTargetPrefix)-1)
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "FuzzDecode"},
		{name: "valid max", value: max},
		{name: "empty", wantErr: true},
		{name: "bare prefix", value: FuzzTargetPrefix, wantErr: true},
		{name: "lowercase suffix", value: "Fuzzdecode", wantErr: true},
		{name: "wrong prefix", value: "TestDecode", wantErr: true},
		{name: "hyphen", value: "FuzzDecode-JSON", wantErr: true},
		{name: "over cap", value: max + "a", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseFuzzTargetName(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParseFuzzTargetName() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParseFuzzTargetName() error = %v", err)
			}
		})
	}
}

func TestFixedIdentityOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		parse   func(string) error
		name    string
		valid   string
		invalid string
	}{
		{name: "run id", valid: strings.Repeat("a", RunIDTextBytes), invalid: strings.Repeat("a", RunIDTextBytes-1), parse: func(value string) error { _, err := ParseRunID(value); return err }},
		{name: "machine id", valid: strings.Repeat("b", MachineIDTextBytes), invalid: strings.Repeat("B", MachineIDTextBytes), parse: func(value string) error { _, err := ParseMachineID(value); return err }},
		{name: "commit sha", valid: strings.Repeat("c", CommitSHATextBytes), invalid: strings.Repeat("g", CommitSHATextBytes), parse: func(value string) error { _, err := ParseCommitSHA(value); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.parse(tc.valid); err != nil {
				t.Fatalf("valid identity error = %v", err)
			}
			if err := tc.parse(tc.invalid); !errors.Is(err, ErrContract) {
				t.Fatalf("invalid identity error = %v, want %v", err, ErrContract)
			}
		})
	}
}

func TestIdentityJSONOGSRejectsWithoutMutation(t *testing.T) {
	t.Parallel()
	original, err := ParseProjectID("peachfuzz")
	if err != nil {
		t.Fatal(err)
	}
	candidate := original
	if err := json.Unmarshal([]byte(`"Peachfuzz"`), &candidate); !errors.Is(err, ErrContract) {
		t.Fatalf("UnmarshalJSON() error = %v, want %v", err, ErrContract)
	}
	if candidate != original {
		t.Fatalf("rejected JSON mutated ProjectID")
	}
	var nilProject *ProjectID
	if err := nilProject.UnmarshalJSON([]byte(`"peachfuzz"`)); !errors.Is(err, ErrContract) {
		t.Fatalf("nil UnmarshalJSON() error = %v, want %v", err, ErrContract)
	}
}
