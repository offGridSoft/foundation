package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestFuzzCorpusEntryNameBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "minimum lowercase hex", value: "0123abcd"},
		{name: "maximum lowercase hex", value: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		{name: "short", value: "0123abc", wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "too long", value: strings.Repeat("a", FuzzCorpusEntryNameMaxBytes+1), wantErr: true},
		{name: "uppercase", value: "0123ABCd", wantErr: true},
		{name: "separator", value: "0123/abc", wantErr: true},
		{name: "dot", value: "0123.abc", wantErr: true},
		{name: "newline", value: "0123abc\n", wantErr: true},
		{name: "nul", value: "0123abc\x00", wantErr: true},
		{name: "non ascii", value: "0123abcé", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := ParseFuzzCorpusEntryName(tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseFuzzCorpusEntryName(%q) error = %v, wantErr %t", tc.value, err, tc.wantErr)
			}
			if tc.wantErr {
				if !errors.Is(err, ErrContract) {
					t.Fatalf("ParseFuzzCorpusEntryName(%q) error = %v, want errors.Is ErrContract", tc.value, err)
				}
				return
			}
			encoded, marshalErr := json.Marshal(value)
			var decoded FuzzCorpusEntryName
			unmarshalErr := json.Unmarshal(encoded, &decoded)
			if marshalErr != nil || unmarshalErr != nil || decoded != value {
				t.Fatalf("corpus name round trip = (%s, %+v, %v, %v), want stable", encoded, decoded, marshalErr, unmarshalErr)
			}
		})
	}
}

func TestFuzzCorpusEntryNameRejectsMalformedJSONAndNilReceiver(t *testing.T) {
	t.Parallel()

	original, err := ParseFuzzCorpusEntryName("0123abcd")
	if err != nil {
		t.Fatalf("ParseFuzzCorpusEntryName(original) error = %v", err)
	}
	tests := []struct {
		name       string
		data       string
		wantSyntax bool
	}{
		{name: "number", data: `7`},
		{name: "boolean", data: `true`},
		{name: "null", data: `null`},
		{name: "array", data: `[]`},
		{name: "object", data: `{}`},
		{name: "truncated", data: `"0123abcd`, wantSyntax: true},
		{name: "invalid identity", data: `"0123ABCd"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := original
			err := json.Unmarshal([]byte(tc.data), &candidate)
			if tc.wantSyntax {
				var syntaxError *json.SyntaxError
				if !errors.As(err, &syntaxError) {
					t.Fatalf("json.Unmarshal(%s) error = %v, want json.SyntaxError", tc.data, err)
				}
			} else if !errors.Is(err, ErrContract) {
				t.Fatalf("json.Unmarshal(%s) error = %v, want errors.Is ErrContract", tc.data, err)
			}
			if candidate != original {
				t.Fatalf("json.Unmarshal(%s) candidate = %s, want unchanged %s", tc.data, candidate, original)
			}
		})
	}
	var target *FuzzCorpusEntryName
	if err := target.UnmarshalJSON([]byte(`"0123abcd"`)); !errors.Is(err, ErrContract) {
		t.Fatalf("nil UnmarshalJSON() error = %v, want errors.Is ErrContract", err)
	}
	if _, err := json.Marshal(FuzzCorpusEntryName{}); !errors.Is(err, ErrContract) {
		t.Fatalf("json.Marshal(zero) error = %v, want errors.Is ErrContract", err)
	}
}

func TestFuzzCorpusSelectionRetainsCanonicalBoundedPrefix(t *testing.T) {
	t.Parallel()

	var selection FuzzCorpusSelection
	for position := FuzzArtifactIndexMaxEntries + 1; position >= 0; position-- {
		name, err := ParseFuzzCorpusEntryName(fmt.Sprintf("%08x", position))
		if err != nil {
			t.Fatalf("ParseFuzzCorpusEntryName(%d) error = %v", position, err)
		}
		if err := selection.Observe(name); err != nil {
			t.Fatalf("Observe(%s) error = %v", name, err)
		}
	}
	duplicate, err := ParseFuzzCorpusEntryName("00000000")
	if err != nil {
		t.Fatalf("ParseFuzzCorpusEntryName(duplicate) error = %v", err)
	}
	if err := selection.Observe(duplicate); err != nil {
		t.Fatalf("Observe(duplicate) error = %v", err)
	}
	entries := selection.Entries()
	if err := selection.Validate(); err != nil {
		t.Fatalf("selection.Validate() error = %v", err)
	}
	if len(entries) != FuzzArtifactIndexMaxEntries || entries[0].String() != "00000000" || entries[len(entries)-1].String() != "0000007f" {
		t.Fatalf("selection = %d entries %s..%s, want %d entries 00000000..0000007f", len(entries), entries[0], entries[len(entries)-1], FuzzArtifactIndexMaxEntries)
	}
	if selection.Dropped() != 2 {
		t.Fatalf("selection.Dropped() = %d, want two over-bound candidates", selection.Dropped())
	}
	entries[0] = FuzzCorpusEntryName{}
	if got := selection.Entries()[0].String(); got != "00000000" {
		t.Fatalf("selection.Entries()[0] after caller mutation = %q, want defensive-copy value %q", got, "00000000")
	}
}

func TestFuzzCorpusSelectionBoundaryAndOrderMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		positions   []int
		wantEntries int
		wantDropped uint64
		wantLast    string
	}{
		{name: "neutral empty"},
		{name: "below bound", positions: corpusPositions(127, false), wantEntries: 127, wantLast: "0000007e"},
		{name: "exact bound", positions: corpusPositions(128, true), wantEntries: 128, wantLast: "0000007f"},
		{name: "one over ascending", positions: corpusPositions(129, false), wantEntries: 128, wantDropped: 1, wantLast: "0000007f"},
		{name: "one over descending", positions: corpusPositions(129, true), wantEntries: 128, wantDropped: 1, wantLast: "0000007f"},
		{name: "adversarial interleave", positions: []int{200, 0, 199, 1, 198, 2, 197, 3}, wantEntries: 8, wantLast: "000000c8"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var selection FuzzCorpusSelection
			for _, position := range tc.positions {
				name, err := ParseFuzzCorpusEntryName(fmt.Sprintf("%08x", position))
				if err != nil {
					t.Fatalf("ParseFuzzCorpusEntryName(%d) error = %v", position, err)
				}
				if err := selection.Observe(name); err != nil {
					t.Fatalf("Observe(%s) error = %v", name, err)
				}
			}
			if err := selection.Validate(); err != nil {
				t.Fatalf("selection.Validate() error = %v", err)
			}
			entries := selection.Entries()
			if len(entries) != tc.wantEntries || selection.Dropped() != tc.wantDropped {
				t.Fatalf("selection = (%d entries, %d dropped), want (%d, %d)", len(entries), selection.Dropped(), tc.wantEntries, tc.wantDropped)
			}
			if tc.wantEntries != 0 && entries[len(entries)-1].String() != tc.wantLast {
				t.Fatalf("selection last = %s, want %s", entries[len(entries)-1], tc.wantLast)
			}
		})
	}
}

func TestFuzzCorpusSelectionRejectsHostileStateTable(t *testing.T) {
	t.Parallel()

	valid := corpusNames(t, 2)
	tests := []struct {
		name      string
		selection FuzzCorpusSelection
	}{
		{name: "entry above bound", selection: FuzzCorpusSelection{entries: corpusNames(t, FuzzArtifactIndexMaxEntries+1)}},
		{name: "zero entry", selection: FuzzCorpusSelection{entries: []FuzzCorpusEntryName{{}}}},
		{name: "duplicate entry", selection: FuzzCorpusSelection{entries: []FuzzCorpusEntryName{valid[0], valid[0]}}},
		{name: "descending entries", selection: FuzzCorpusSelection{entries: []FuzzCorpusEntryName{valid[1], valid[0]}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.selection.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("selection.Validate() error = %v, want errors.Is ErrContract", err)
			}
		})
	}

	var nilSelection *FuzzCorpusSelection
	if err := nilSelection.Observe(valid[0]); !errors.Is(err, ErrContract) {
		t.Fatalf("nil Observe() error = %v, want errors.Is ErrContract", err)
	}
	var selection FuzzCorpusSelection
	if err := selection.Observe(FuzzCorpusEntryName{}); !errors.Is(err, ErrContract) {
		t.Fatalf("Observe(zero) error = %v, want errors.Is ErrContract", err)
	}
}

func TestFuzzCorpusSelectionDuplicateAndSaturationRatchets(t *testing.T) {
	t.Parallel()

	var selection FuzzCorpusSelection
	for _, name := range corpusNames(t, FuzzArtifactIndexMaxEntries) {
		if err := selection.Observe(name); err != nil {
			t.Fatalf("Observe(%s) error = %v", name, err)
		}
	}
	retained := selection.Entries()[FuzzArtifactIndexMaxEntries/2]
	if err := selection.Observe(retained); err != nil {
		t.Fatalf("Observe(retained duplicate) error = %v", err)
	}
	if selection.Dropped() != 0 {
		t.Fatalf("Dropped() after retained duplicate = %d, want zero", selection.Dropped())
	}

	selection.dropped = math.MaxUint64
	over, err := ParseFuzzCorpusEntryName("ffffffff")
	if err != nil {
		t.Fatalf("ParseFuzzCorpusEntryName(over) error = %v", err)
	}
	if err := selection.Observe(over); err != nil {
		t.Fatalf("Observe(over) error = %v", err)
	}
	if selection.Dropped() != math.MaxUint64 {
		t.Fatalf("Dropped() = %d, want saturated MaxUint64", selection.Dropped())
	}
}

func corpusPositions(count int, reverse bool) []int {
	positions := make([]int, count)
	for position := range count {
		positions[position] = position
	}
	if reverse {
		slices.Reverse(positions)
	}
	return positions
}

func corpusNames(t *testing.T, count int) []FuzzCorpusEntryName {
	t.Helper()
	names := make([]FuzzCorpusEntryName, count)
	for position := range count {
		name, err := ParseFuzzCorpusEntryName(fmt.Sprintf("%08x", position))
		if err != nil {
			t.Fatalf("ParseFuzzCorpusEntryName(%d) error = %v", position, err)
		}
		names[position] = name
	}
	return names
}
