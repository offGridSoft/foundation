package peachfuzz

import (
	"crypto/sha256"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
	foundationfuzz "github.com/offGridSoft/foundation/v2026/fuzz"
)

func TestFuzzSidecarRefHostileStateTable(t *testing.T) {
	t.Parallel()
	digest := foundationcore.NewSHA256Hex(sha256.Sum256([]byte("retained evidence")))
	captured, err := NewCapturedFuzzSidecarRef(FuzzSidecarCapture{Kind: FuzzSidecarKindStdout, Digest: digest, RetainedBytes: 17, DroppedBytes: 9, Lines: 1})
	if err != nil {
		t.Fatalf("NewCapturedFuzzSidecarRef() error = %v", err)
	}
	tests := []struct {
		mutate func(*FuzzSidecarRef)
		name   string
	}{
		{name: "unknown kind", mutate: func(ref *FuzzSidecarRef) { ref.Kind = FuzzSidecarKindUnknown }},
		{name: "unknown state", mutate: func(ref *FuzzSidecarRef) { ref.State = FuzzSidecarStateUnknown }},
		{name: "missing digest", mutate: func(ref *FuzzSidecarRef) { ref.Digest = foundationcore.SHA256Hex{} }},
		{name: "retained total mismatch", mutate: func(ref *FuzzSidecarRef) { ref.OriginalBytes++ }},
		{name: "truncation lie", mutate: func(ref *FuzzSidecarRef) { ref.Truncated = false }},
		{name: "offset lies about prefix retention", mutate: func(ref *FuzzSidecarRef) { ref.RetainedOffsetBytes = 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := captured
			tc.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("Validate() error = %v, want errors.Is(..., %v)", err, ErrContract)
			}
		})
	}
}

func TestFuzzSidecarConstructorsPinPositiveNeutralAndBoundaryStates(t *testing.T) {
	t.Parallel()
	digest := foundationcore.NewSHA256Hex(sha256.Sum256([]byte("x")))
	absent, absentErr := NewAbsentFuzzSidecarRef(FuzzSidecarKindCorpusIndex)
	empty, emptyErr := NewEmptyFuzzSidecarRef(FuzzSidecarKindStderr)
	captured, capturedErr := NewCapturedFuzzSidecarRef(FuzzSidecarCapture{Kind: FuzzSidecarKindStdout, Digest: digest, RetainedBytes: 1, Lines: 1})
	lost, lostErr := NewLostFuzzSidecarRef(FuzzSidecarKindStderr, 7)
	unmeasured, unmeasuredErr := NewUnmeasuredLostFuzzSidecarRef(FuzzSidecarKindCorpusIndex)
	constructors := []struct {
		err  error
		name string
		ref  FuzzSidecarRef
	}{
		{name: "absent", ref: absent, err: absentErr},
		{name: "empty", ref: empty, err: emptyErr},
		{name: "captured", ref: captured, err: capturedErr},
		{name: "lost", ref: lost, err: lostErr},
		{name: "unmeasured lost", ref: unmeasured, err: unmeasuredErr},
	}
	for _, tc := range constructors {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.err != nil {
				t.Fatalf("constructor error = %v", tc.err)
			}
			if err := tc.ref.Validate(); err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			encoded, err := foundationcore.EncodeValidatedJSON(tc.ref)
			if err != nil {
				t.Fatalf("EncodeValidatedJSON() error = %v", err)
			}
			decoded, err := foundationcore.DecodeStrictJSON[FuzzSidecarRef](encoded)
			if err != nil || decoded != tc.ref {
				t.Fatalf("DecodeStrictJSON() = (%+v, %v), want %+v/nil", decoded, err, tc.ref)
			}
		})
	}
	if _, err := NewCapturedFuzzSidecarRef(FuzzSidecarCapture{Kind: FuzzSidecarKindStdout, Digest: digest, RetainedBytes: math.MaxUint64, DroppedBytes: 1}); !errors.Is(err, ErrContract) {
		t.Fatalf("overflow constructor error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
}

func TestFuzzEvidenceRejectsCrossWiredKinds(t *testing.T) {
	t.Parallel()
	evidence, err := NewEmptyFuzzEvidence()
	if err != nil {
		t.Fatalf("NewEmptyFuzzEvidence() error = %v", err)
	}
	evidence.Stdout.Kind = FuzzSidecarKindStderr
	if err := evidence.Validate(); !errors.Is(err, ErrContract) {
		t.Fatalf("Validate() error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
}

func TestFuzzArtifactIndexCanonicalizesAndRoundTrips(t *testing.T) {
	t.Parallel()
	first := FuzzArtifact{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("first"))), Bytes: 11}
	second := FuzzArtifact{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("second"))), Bytes: 13}
	index, err := NewFuzzArtifactIndex(foundationfuzz.ArtifactKindCorpus, []FuzzArtifact{second, first}, 0, false)
	if err != nil {
		t.Fatalf("NewFuzzArtifactIndex() error = %v", err)
	}
	wantEntries := []FuzzArtifact{first, second}
	slices.SortFunc(wantEntries, func(left, right FuzzArtifact) int {
		return strings.Compare(left.Digest.String(), right.Digest.String())
	})
	if !slices.Equal(index.Entries, wantEntries) || index.TotalBytes != 24 || index.Count != 2 || index.State != FuzzArtifactIndexStateComplete {
		t.Fatalf("NewFuzzArtifactIndex() = %+v, want canonical complete 2-entry index", index)
	}
	encoded, err := foundationcore.EncodeValidatedJSON(index)
	if err != nil {
		t.Fatalf("EncodeValidatedJSON() error = %v", err)
	}
	decoded, err := foundationcore.DecodeStrictJSON[FuzzArtifactIndex](encoded)
	if err != nil || !slices.Equal(decoded.Entries, index.Entries) || decoded.TotalBytes != index.TotalBytes || decoded.Count != index.Count || decoded.Dropped != index.Dropped || decoded.Kind != index.Kind || decoded.State != index.State {
		t.Fatalf("DecodeStrictJSON() = (%+v, %v), want %+v/nil", decoded, err, index)
	}
}

func TestFuzzArtifactIndexRejectsHostileContradictions(t *testing.T) {
	t.Parallel()
	first := FuzzArtifact{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("first"))), Bytes: 11}
	second := FuzzArtifact{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("second"))), Bytes: 13}
	valid, err := NewFuzzArtifactIndex(foundationfuzz.ArtifactKindCrasher, []FuzzArtifact{first, second}, 1, false)
	if err != nil {
		t.Fatalf("NewFuzzArtifactIndex() error = %v", err)
	}
	tests := []struct {
		mutate func(*FuzzArtifactIndex)
		name   string
	}{
		{name: "unknown kind", mutate: func(index *FuzzArtifactIndex) { index.Kind = foundationfuzz.ArtifactKindUnknown }},
		{name: "count lie", mutate: func(index *FuzzArtifactIndex) { index.Count++ }},
		{name: "byte total lie", mutate: func(index *FuzzArtifactIndex) { index.TotalBytes++ }},
		{name: "completeness lie", mutate: func(index *FuzzArtifactIndex) { index.State = FuzzArtifactIndexStateComplete }},
		{name: "duplicate digest", mutate: func(index *FuzzArtifactIndex) { index.Entries[1].Digest = index.Entries[0].Digest }},
		{name: "zero byte artifact", mutate: func(index *FuzzArtifactIndex) { index.Entries[0].Bytes = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			candidate.Entries = append([]FuzzArtifact(nil), valid.Entries...)
			tc.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("Validate() error = %v, want errors.Is(..., %v)", err, ErrContract)
			}
		})
	}
}

func TestFuzzArtifactIndexPinsEntryAndArithmeticBounds(t *testing.T) {
	t.Parallel()
	entries := make([]FuzzArtifact, foundationfuzz.ArtifactIndexMaxEntries+1)
	for position := range entries {
		entries[position] = FuzzArtifact{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte{byte(position), byte(position >> 8)})), Bytes: 1}
	}
	if _, err := NewFuzzArtifactIndex(foundationfuzz.ArtifactKindCorpus, entries, 0, false); !errors.Is(err, ErrContract) {
		t.Fatalf("entry-limit constructor error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
	overflow := []FuzzArtifact{
		{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("large"))), Bytes: math.MaxUint64},
		{Digest: foundationcore.NewSHA256Hex(sha256.Sum256([]byte("overflow"))), Bytes: 1},
	}
	if _, err := NewFuzzArtifactIndex(foundationfuzz.ArtifactKindCorpus, overflow, 0, false); !errors.Is(err, ErrContract) {
		t.Fatalf("total-overflow constructor error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
}

func TestFuzzArtifactIndexMakesEnumerationFailureExplicit(t *testing.T) {
	t.Parallel()
	index, err := NewFuzzArtifactIndex(foundationfuzz.ArtifactKindCorpus, nil, 0, true)
	if err != nil {
		t.Fatalf("NewFuzzArtifactIndex() error = %v", err)
	}
	if index.State != FuzzArtifactIndexStateEnumerationFailed || index.Count != 0 || index.Dropped != 0 {
		t.Fatalf("NewFuzzArtifactIndex() = %+v, want explicit enumeration-failed empty index", index)
	}
}
