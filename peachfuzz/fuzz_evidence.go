package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
	foundationfuzz "github.com/offGridSoft/foundation/v2026/fuzz"
)

// FuzzSidecarKind is the closed identity of one fuzz-evidence sidecar. The
// contract is lifted from Witness's proven sidecar model; Peachfuzz uses the
// same shape with content-addressed object custody instead of bundle paths.
type FuzzSidecarKind uint8

const (
	FuzzSidecarKindUnknown FuzzSidecarKind = iota
	FuzzSidecarKindStdout
	FuzzSidecarKindStderr
	FuzzSidecarKindCorpusIndex
	FuzzSidecarKindCrasherIndex
)

const (
	fuzzSidecarKindNameStdout       = "stdout"
	fuzzSidecarKindNameStderr       = "stderr"
	fuzzSidecarKindNameCorpusIndex  = "fuzz-corpus-index"
	fuzzSidecarKindNameCrasherIndex = "fuzz-crasher-index"
	fuzzSidecarUnknownName          = "unknown"
	// FuzzEvidenceSidecarCount is the closed cardinality of FuzzEvidence.
	FuzzEvidenceSidecarCount = 4
)

func (k FuzzSidecarKind) String() string {
	switch k {
	case FuzzSidecarKindStdout:
		return fuzzSidecarKindNameStdout
	case FuzzSidecarKindStderr:
		return fuzzSidecarKindNameStderr
	case FuzzSidecarKindCorpusIndex:
		return fuzzSidecarKindNameCorpusIndex
	case FuzzSidecarKindCrasherIndex:
		return fuzzSidecarKindNameCrasherIndex
	default:
		return fuzzSidecarUnknownName
	}
}

func (k FuzzSidecarKind) Validate() error {
	if !k.IsValid() {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func (k FuzzSidecarKind) IsValid() bool {
	return k >= FuzzSidecarKindStdout && k <= FuzzSidecarKindCrasherIndex
}

func (k FuzzSidecarKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.String())
}

func (k *FuzzSidecarKind) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, errors.Join(ErrContract, err))
	}
	switch value {
	case fuzzSidecarKindNameStdout:
		*k = FuzzSidecarKindStdout
	case fuzzSidecarKindNameStderr:
		*k = FuzzSidecarKindStderr
	case fuzzSidecarKindNameCorpusIndex:
		*k = FuzzSidecarKindCorpusIndex
	case fuzzSidecarKindNameCrasherIndex:
		*k = FuzzSidecarKindCrasherIndex
	default:
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

type FuzzSidecarState uint8

const (
	FuzzSidecarStateUnknown FuzzSidecarState = iota
	FuzzSidecarStateAbsent
	FuzzSidecarStateEmpty
	FuzzSidecarStateCaptured
	FuzzSidecarStateLost
)

const (
	fuzzSidecarStateNameAbsent   = "absent"
	fuzzSidecarStateNameEmpty    = "empty"
	fuzzSidecarStateNameCaptured = "captured"
	fuzzSidecarStateNameLost     = "lost"
)

func (s FuzzSidecarState) String() string {
	switch s {
	case FuzzSidecarStateAbsent:
		return fuzzSidecarStateNameAbsent
	case FuzzSidecarStateEmpty:
		return fuzzSidecarStateNameEmpty
	case FuzzSidecarStateCaptured:
		return fuzzSidecarStateNameCaptured
	case FuzzSidecarStateLost:
		return fuzzSidecarStateNameLost
	default:
		return fuzzSidecarUnknownName
	}
}

func (s FuzzSidecarState) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func (s FuzzSidecarState) IsValid() bool {
	return s >= FuzzSidecarStateAbsent && s <= FuzzSidecarStateLost
}

func (s FuzzSidecarState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *FuzzSidecarState) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, errors.Join(ErrContract, err))
	}
	switch value {
	case fuzzSidecarStateNameAbsent:
		*s = FuzzSidecarStateAbsent
	case fuzzSidecarStateNameEmpty:
		*s = FuzzSidecarStateEmpty
	case fuzzSidecarStateNameCaptured:
		*s = FuzzSidecarStateCaptured
	case fuzzSidecarStateNameLost:
		*s = FuzzSidecarStateLost
	default:
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

// FuzzSidecarRef is bounded, authenticated fuzz evidence. OriginalBytes is
// the exact stream size; RetainedBytes is the immutable object's size; and
// DroppedBytes makes truncation explicit instead of silently claiming the
// retained prefix is the complete process stream.
type FuzzSidecarRef struct {
	Digest              core.SHA256Hex   `json:"digest"`
	OriginalBytes       uint64           `json:"original_bytes"`
	RetainedBytes       uint64           `json:"retained_bytes"`
	DroppedBytes        uint64           `json:"dropped_bytes"`
	RetainedOffsetBytes uint64           `json:"retained_offset_bytes"`
	Lines               uint64           `json:"lines"`
	Kind                FuzzSidecarKind  `json:"kind"`
	State               FuzzSidecarState `json:"state"`
	OriginalBytesKnown  bool             `json:"original_bytes_known"`
	Truncated           bool             `json:"truncated"`
}

func NewAbsentFuzzSidecarRef(kind FuzzSidecarKind) (FuzzSidecarRef, error) {
	ref := FuzzSidecarRef{Kind: kind, State: FuzzSidecarStateAbsent}
	return ref, ref.Validate()
}

func NewEmptyFuzzSidecarRef(kind FuzzSidecarKind) (FuzzSidecarRef, error) {
	ref := FuzzSidecarRef{Kind: kind, State: FuzzSidecarStateEmpty, OriginalBytesKnown: true}
	return ref, ref.Validate()
}

type FuzzSidecarCapture struct {
	Digest        core.SHA256Hex
	RetainedBytes uint64
	DroppedBytes  uint64
	Lines         uint64
	Kind          FuzzSidecarKind
}

func (c FuzzSidecarCapture) Validate() error {
	if !c.Kind.IsValid() || c.Digest.Validate() != nil || c.RetainedBytes == 0 || c.RetainedBytes > math.MaxUint64-c.DroppedBytes {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func NewCapturedFuzzSidecarRef(capture FuzzSidecarCapture) (FuzzSidecarRef, error) {
	if err := capture.Validate(); err != nil {
		return FuzzSidecarRef{}, fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	ref := FuzzSidecarRef{
		Digest: capture.Digest, OriginalBytes: capture.RetainedBytes + capture.DroppedBytes,
		RetainedBytes: capture.RetainedBytes, DroppedBytes: capture.DroppedBytes, Lines: capture.Lines,
		Kind: capture.Kind, State: FuzzSidecarStateCaptured, OriginalBytesKnown: true, Truncated: capture.DroppedBytes != 0,
	}
	return ref, ref.Validate()
}

func NewLostFuzzSidecarRef(kind FuzzSidecarKind, originalBytes uint64) (FuzzSidecarRef, error) {
	ref := FuzzSidecarRef{OriginalBytes: originalBytes, DroppedBytes: originalBytes, Kind: kind, State: FuzzSidecarStateLost, OriginalBytesKnown: true, Truncated: true}
	return ref, ref.Validate()
}

func NewUnmeasuredLostFuzzSidecarRef(kind FuzzSidecarKind) (FuzzSidecarRef, error) {
	ref := FuzzSidecarRef{Kind: kind, State: FuzzSidecarStateLost, Truncated: true}
	return ref, ref.Validate()
}

func (r FuzzSidecarRef) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	type fuzzSidecarRefJSON struct {
		Digest              *core.SHA256Hex  `json:"digest,omitempty"`
		OriginalBytes       uint64           `json:"original_bytes"`
		RetainedBytes       uint64           `json:"retained_bytes"`
		DroppedBytes        uint64           `json:"dropped_bytes"`
		RetainedOffsetBytes uint64           `json:"retained_offset_bytes"`
		Lines               uint64           `json:"lines"`
		Kind                FuzzSidecarKind  `json:"kind"`
		State               FuzzSidecarState `json:"state"`
		OriginalBytesKnown  bool             `json:"original_bytes_known"`
		Truncated           bool             `json:"truncated"`
	}
	var digest *core.SHA256Hex
	if !r.Digest.IsZero() {
		value := r.Digest
		digest = &value
	}
	return json.Marshal(fuzzSidecarRefJSON{
		Digest: digest, OriginalBytes: r.OriginalBytes, RetainedBytes: r.RetainedBytes,
		DroppedBytes: r.DroppedBytes, RetainedOffsetBytes: r.RetainedOffsetBytes,
		Lines: r.Lines, Kind: r.Kind, State: r.State, OriginalBytesKnown: r.OriginalBytesKnown, Truncated: r.Truncated,
	})
}

func (r FuzzSidecarRef) Validate() error {
	if err := errors.Join(r.Kind.Validate(), r.State.Validate()); err != nil {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, errors.Join(ErrContract, err))
	}
	if !r.byteAccountingValid() {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	var err error
	switch r.State {
	case FuzzSidecarStateAbsent, FuzzSidecarStateEmpty:
		err = r.validateZeroByteState()
	case FuzzSidecarStateCaptured:
		err = r.validateCapturedState()
	case FuzzSidecarStateLost:
		err = r.validateLostState()
	}
	return err
}

func (r FuzzSidecarRef) byteAccountingValid() bool {
	return r.OriginalBytes >= r.RetainedBytes && r.OriginalBytes-r.RetainedBytes == r.DroppedBytes
}

func (r FuzzSidecarRef) validateZeroByteState() error {
	bytesKnown := r.State == FuzzSidecarStateEmpty
	if !r.zeroByteFields() || r.OriginalBytesKnown != bytesKnown {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func (r FuzzSidecarRef) zeroByteFields() bool {
	return r.Digest.IsZero() && r.OriginalBytes == 0 && r.RetainedBytes == 0 && r.DroppedBytes == 0 && r.RetainedOffsetBytes == 0 && r.Lines == 0 && !r.Truncated
}

func (r FuzzSidecarRef) validateCapturedState() error {
	if r.Digest.Validate() != nil || r.RetainedBytes == 0 || r.RetainedOffsetBytes != 0 || !r.OriginalBytesKnown || r.Truncated != (r.DroppedBytes != 0) {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func (r FuzzSidecarRef) validateLostState() error {
	if !r.lostFieldsValid() || r.OriginalBytesKnown != (r.OriginalBytes != 0) {
		return fmt.Errorf(ErrFmtFuzzSidecarRef, ErrContract)
	}
	return nil
}

func (r FuzzSidecarRef) lostFieldsValid() bool {
	return r.Digest.IsZero() && r.RetainedBytes == 0 && r.DroppedBytes == r.OriginalBytes && r.RetainedOffsetBytes == 0 && r.Lines == 0 && r.Truncated
}

// FuzzEvidence is the fixed, compiler-visible fuzz sidecar set. Index
// sidecars remain absent until a producer has corpus/crasher index bytes; an
// absent typed ref is evidence of absence, never an omitted convention.
type FuzzEvidence struct {
	Stdout       FuzzSidecarRef `json:"stdout"`
	Stderr       FuzzSidecarRef `json:"stderr"`
	CorpusIndex  FuzzSidecarRef `json:"corpus_index"`
	CrasherIndex FuzzSidecarRef `json:"crasher_index"`
}

type FuzzArtifactIndexState uint8

const (
	FuzzArtifactIndexStateUnknown FuzzArtifactIndexState = iota
	FuzzArtifactIndexStateComplete
	FuzzArtifactIndexStatePartial
	FuzzArtifactIndexStateEnumerationFailed
)

const (
	fuzzArtifactIndexStateNameComplete          = "complete"
	fuzzArtifactIndexStateNamePartial           = "partial"
	fuzzArtifactIndexStateNameEnumerationFailed = "enumeration-failed"
)

func (s FuzzArtifactIndexState) String() string {
	switch s {
	case FuzzArtifactIndexStateComplete:
		return fuzzArtifactIndexStateNameComplete
	case FuzzArtifactIndexStatePartial:
		return fuzzArtifactIndexStateNamePartial
	case FuzzArtifactIndexStateEnumerationFailed:
		return fuzzArtifactIndexStateNameEnumerationFailed
	default:
		return fuzzSidecarUnknownName
	}
}

func (s FuzzArtifactIndexState) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	return nil
}

func (s FuzzArtifactIndexState) IsValid() bool {
	return s >= FuzzArtifactIndexStateComplete && s <= FuzzArtifactIndexStateEnumerationFailed
}

func (s FuzzArtifactIndexState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *FuzzArtifactIndexState) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, errors.Join(ErrContract, err))
	}
	switch value {
	case fuzzArtifactIndexStateNameComplete:
		*s = FuzzArtifactIndexStateComplete
	case fuzzArtifactIndexStateNamePartial:
		*s = FuzzArtifactIndexStatePartial
	case fuzzArtifactIndexStateNameEnumerationFailed:
		*s = FuzzArtifactIndexStateEnumerationFailed
	default:
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	return nil
}

type FuzzArtifact struct {
	Digest core.SHA256Hex `json:"digest"`
	Bytes  uint64         `json:"bytes"`
}

const FuzzArtifactMinimumBytes uint64 = 1

func (a FuzzArtifact) Validate() error {
	if a.Bytes < FuzzArtifactMinimumBytes {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	if err := a.Digest.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, errors.Join(ErrContract, err))
	}
	return nil
}

// FuzzArtifactIndex is Witness's bounded typed corpus/crasher index shape,
// expressed with content identities rather than bundle filesystem metadata.
// Entries are canonical digest order so identical evidence has identical JSON.
type FuzzArtifactIndex struct {
	Entries    []FuzzArtifact              `json:"entries"`
	TotalBytes uint64                      `json:"total_bytes"`
	Dropped    uint64                      `json:"dropped"`
	Count      uint64                      `json:"count"`
	Kind       foundationfuzz.ArtifactKind `json:"kind"`
	State      FuzzArtifactIndexState      `json:"state"`
}

func NewFuzzArtifactIndex(kind foundationfuzz.ArtifactKind, entries []FuzzArtifact, dropped uint64, enumerationFailed bool) (FuzzArtifactIndex, error) {
	canonical := append([]FuzzArtifact(nil), entries...)
	slices.SortFunc(canonical, func(left, right FuzzArtifact) int {
		return strings.Compare(left.Digest.String(), right.Digest.String())
	})
	state := FuzzArtifactIndexStateComplete
	if dropped != 0 {
		state = FuzzArtifactIndexStatePartial
	}
	if enumerationFailed {
		state = FuzzArtifactIndexStateEnumerationFailed
	}
	index := FuzzArtifactIndex{Entries: canonical, Dropped: dropped, Kind: kind, State: state}
	for _, entry := range canonical {
		if index.TotalBytes > math.MaxUint64-entry.Bytes {
			return FuzzArtifactIndex{}, fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
		index.TotalBytes += entry.Bytes
		index.Count++
	}
	return index, index.Validate()
}

func (i FuzzArtifactIndex) Validate() error {
	if err := errors.Join(i.Kind.Validate(), i.State.Validate()); err != nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, errors.Join(ErrContract, err))
	}
	if !i.headerValid() {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	return i.validateEntries()
}

func (i FuzzArtifactIndex) headerValid() bool {
	var count uint64
	for range i.Entries {
		count++
	}
	bounded := len(i.Entries) <= foundationfuzz.ArtifactIndexMaxEntries && i.Count == count
	complete := i.State != FuzzArtifactIndexStateComplete || i.Dropped == 0
	partial := i.State != FuzzArtifactIndexStatePartial || i.Dropped != 0
	return bounded && complete && partial
}

func (i FuzzArtifactIndex) validateEntries() error {
	var total uint64
	for position, entry := range i.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		if position != 0 && strings.Compare(i.Entries[position-1].Digest.String(), entry.Digest.String()) >= 0 {
			return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
		if total > math.MaxUint64-entry.Bytes {
			return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
		total += entry.Bytes
	}
	if total != i.TotalBytes {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	return nil
}

func NewEmptyFuzzEvidence() (FuzzEvidence, error) {
	stdout, stdoutErr := NewAbsentFuzzSidecarRef(FuzzSidecarKindStdout)
	stderr, stderrErr := NewAbsentFuzzSidecarRef(FuzzSidecarKindStderr)
	corpus, corpusErr := NewAbsentFuzzSidecarRef(FuzzSidecarKindCorpusIndex)
	crashers, crashersErr := NewAbsentFuzzSidecarRef(FuzzSidecarKindCrasherIndex)
	evidence := FuzzEvidence{Stdout: stdout, Stderr: stderr, CorpusIndex: corpus, CrasherIndex: crashers}
	return evidence, errors.Join(stdoutErr, stderrErr, corpusErr, crashersErr, evidence.Validate())
}

func (e FuzzEvidence) Validate() error {
	checks := []struct {
		ref  FuzzSidecarRef
		kind FuzzSidecarKind
	}{
		{ref: e.Stdout, kind: FuzzSidecarKindStdout},
		{ref: e.Stderr, kind: FuzzSidecarKindStderr},
		{ref: e.CorpusIndex, kind: FuzzSidecarKindCorpusIndex},
		{ref: e.CrasherIndex, kind: FuzzSidecarKindCrasherIndex},
	}
	for _, check := range checks {
		if check.ref.Kind != check.kind {
			return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
		if err := check.ref.Validate(); err != nil {
			return fmt.Errorf(ErrFmtFuzzEvidence, errors.Join(ErrContract, err))
		}
	}
	return nil
}

func (e FuzzEvidence) ForEachCaptured(yield func(FuzzSidecarRef) error) error {
	if yield == nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	for _, ref := range [...]FuzzSidecarRef{e.Stdout, e.Stderr, e.CorpusIndex, e.CrasherIndex} {
		if ref.State == FuzzSidecarStateCaptured {
			if err := yield(ref); err != nil {
				return err
			}
		}
	}
	return nil
}
