package core

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBugRepositoryIDHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	var entropy [BugRepositoryIDEntropyBytes]byte
	entropy[len(entropy)-1] = 1
	valid, err := NewBugRepositoryID(entropy)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		value string
		valid bool
	}{
		{name: "canonical", value: valid.String(), valid: true},
		{name: "missing prefix", value: strings.Repeat("a", 64)},
		{name: "uppercase digest", value: BugRepositoryIDPrefix + strings.Repeat("A", 64)},
		{name: "short digest", value: BugRepositoryIDPrefix + strings.Repeat("a", 63)},
		{name: "long digest", value: BugRepositoryIDPrefix + strings.Repeat("a", 65)},
		{name: "non hex", value: BugRepositoryIDPrefix + strings.Repeat("g", 64)},
		{name: "zero identity", value: BugRepositoryIDPrefix + strings.Repeat("0", 64)},
		{name: "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, parseErr := ParseBugRepositoryID(tc.value)
			if tc.valid {
				if parseErr != nil || got != valid {
					t.Fatalf("ParseBugRepositoryID(%q) = %v, %v", tc.value, got, parseErr)
				}
				return
			}
			if !errors.Is(parseErr, ErrFoundationContract) {
				t.Fatalf("ParseBugRepositoryID(%q) error = %v, want %v", tc.value, parseErr, ErrFoundationContract)
			}
		})
	}

	var zero [BugRepositoryIDEntropyBytes]byte
	if _, err := NewBugRepositoryID(zero); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("NewBugRepositoryID(zero) error = %v, want %v", err, ErrFoundationContract)
	}
}

func TestBugRepositoryIDJSONRoundTrip(t *testing.T) {
	t.Parallel()

	var entropy [BugRepositoryIDEntropyBytes]byte
	entropy[0] = 1
	id, err := NewBugRepositoryID(entropy)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	var decoded BugRepositoryID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != id {
		t.Fatalf("decoded = %v, want %v", decoded, id)
	}
	before := decoded
	if err := json.Unmarshal([]byte(`"wrong"`), &decoded); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("UnmarshalJSON(hostile) error = %v, want %v", err, ErrFoundationContract)
	}
	if decoded != before {
		t.Fatalf("rejected input mutated receiver: got %v want %v", decoded, before)
	}
}
