package peachfuzz

import (
	"errors"
	"testing"
)

// FuzzParseRunOutcomeBoundary attacks the single wire-to-domain enum owner
// consumed by persisted run records, report state, and signed evidence.
func FuzzParseRunOutcomeBoundary(f *testing.F) {
	for outcome := RunOutcomeCompleted; outcome <= RunOutcomeUnsupported; outcome++ {
		f.Add(outcome.String())
	}
	f.Add("")
	f.Add("Completed")
	f.Add("completed ")
	f.Add("candidate_found")
	f.Add("completed\x00")
	f.Fuzz(func(t *testing.T, spelling string) {
		outcome, err := ParseRunOutcome(spelling)
		if err != nil {
			if !errors.Is(err, ErrContract) {
				t.Fatalf("ParseRunOutcome(%q) error = %v, want errors.Is(..., %v)", spelling, err, ErrContract)
			}
			if outcome != RunOutcomeUnknown {
				t.Fatalf("ParseRunOutcome(%q) rejected outcome = %v, want %v", spelling, outcome, RunOutcomeUnknown)
			}
			return
		}
		if !outcome.IsValid() {
			t.Fatalf("ParseRunOutcome(%q) outcome = %d, want valid", spelling, outcome)
		}
		if got := outcome.String(); got != spelling {
			t.Fatalf("ParseRunOutcome(%q).String() = %q, want %q", spelling, got, spelling)
		}
	})
}
