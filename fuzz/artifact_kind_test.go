package fuzz

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestArtifactKindHostileRoundTripTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		want    ArtifactKind
		wantErr bool
	}{
		{name: "corpus", raw: `"fuzz-corpus"`, want: ArtifactKindCorpus},
		{name: "crasher", raw: `"fuzz-crasher"`, want: ArtifactKindCrasher},
		{name: "unknown token", raw: `"unknown"`, wantErr: true},
		{name: "empty token", raw: `""`, wantErr: true},
		{name: "case mutation", raw: `"Fuzz-Corpus"`, wantErr: true},
		{name: "number", raw: `1`, wantErr: true},
		{name: "null", raw: `null`, wantErr: true},
		{name: "object", raw: `{}`, wantErr: true},
		{name: "trailing data", raw: `"fuzz-corpus" true`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := ArtifactKindCrasher
			err := candidate.UnmarshalJSON([]byte(tc.raw))
			if tc.wantErr {
				if !errors.Is(err, core.ErrFuzzContract) || candidate != ArtifactKindCrasher {
					t.Fatalf("UnmarshalJSON(%s) = (%v, %v), want unchanged crasher and ErrFuzzContract", tc.raw, candidate, err)
				}
				return
			}
			if err != nil || candidate != tc.want {
				t.Fatalf("UnmarshalJSON(%s) = (%v, %v), want (%v, nil)", tc.raw, candidate, err, tc.want)
			}
			encoded, marshalErr := json.Marshal(candidate)
			if marshalErr != nil || string(encoded) != tc.raw {
				t.Fatalf("MarshalJSON() = (%s, %v), want (%s, nil)", encoded, marshalErr, tc.raw)
			}
		})
	}

	var nilKind *ArtifactKind
	if err := nilKind.UnmarshalJSON([]byte(`"fuzz-corpus"`)); !errors.Is(err, core.ErrFuzzContract) {
		t.Fatalf("nil UnmarshalJSON() error = %v, want ErrFuzzContract", err)
	}
	if _, err := json.Marshal(ArtifactKindUnknown); !errors.Is(err, core.ErrFuzzContract) {
		t.Fatalf("MarshalJSON(unknown) error = %v, want ErrFuzzContract", err)
	}
}
