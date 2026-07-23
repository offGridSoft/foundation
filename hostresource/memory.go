package hostresource

import (
	"bytes"
	"math"
	"runtime"

	"github.com/offGridSoft/foundation/v2026/core"
)

type MemorySnapshot struct {
	ManagedBytes uint64
}

func (s MemorySnapshot) Validate() error {
	if s.ManagedBytes > math.MaxInt64 {
		return core.ErrHostResourceContract
	}
	return nil
}

func readMemorySnapshot() (MemorySnapshot, error) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	managed := stats.Sys
	if stats.HeapReleased <= managed {
		managed -= stats.HeapReleased
	} else {
		managed = 0
	}
	snapshot := MemorySnapshot{ManagedBytes: managed}
	return snapshot, snapshot.Validate()
}

type MemoryLimit struct {
	LimitBytes     uint64
	TriggerPercent uint8
}

func (l MemoryLimit) Validate() error {
	if l.LimitBytes == 0 || l.LimitBytes > math.MaxInt64 || l.TriggerPercent == 0 || l.TriggerPercent > 100 {
		return core.ErrHostResourceContract
	}
	return nil
}

func (l MemoryLimit) TriggerBytes() (uint64, error) {
	if err := l.Validate(); err != nil {
		return 0, err
	}
	percent := uint64(l.TriggerPercent)
	whole := l.LimitBytes / 100 * percent
	remainder := l.LimitBytes % 100 * percent
	return whole + (remainder+99)/100, nil
}

type MemoryPressureState uint8

const (
	MemoryPressureUnknown MemoryPressureState = iota
	MemoryPressureHealthy
	MemoryPressureReached
)

func (s MemoryPressureState) Validate() error {
	if !s.IsValid() {
		return core.ErrHostResourceContract
	}
	return nil
}

type MemoryAssessment struct {
	Snapshot MemorySnapshot
	Limit    MemoryLimit
	State    MemoryPressureState
}

type MemoryAssessmentRequest struct {
	Limit MemoryLimit
}

func (r MemoryAssessmentRequest) Validate() error {
	return r.Limit.Validate()
}

func AssessMemory(request MemoryAssessmentRequest) (MemoryAssessment, error) {
	if err := request.Validate(); err != nil {
		return MemoryAssessment{}, err
	}
	snapshot, err := readMemorySnapshot()
	if err != nil {
		return MemoryAssessment{}, err
	}
	assessment, err := newMemoryAssessment(snapshot, request.Limit)
	if err != nil {
		return MemoryAssessment{}, err
	}
	return assessment, memoryAssessmentError(assessment)
}

func newMemoryAssessment(snapshot MemorySnapshot, limit MemoryLimit) (MemoryAssessment, error) {
	if err := snapshot.Validate(); err != nil {
		return MemoryAssessment{}, err
	}
	trigger, err := limit.TriggerBytes()
	if err != nil {
		return MemoryAssessment{}, err
	}
	state := MemoryPressureHealthy
	if snapshot.ManagedBytes >= trigger {
		state = MemoryPressureReached
	}
	return MemoryAssessment{Snapshot: snapshot, Limit: limit, State: state}, nil
}

func (a MemoryAssessment) Validate() error {
	expected, err := newMemoryAssessment(a.Snapshot, a.Limit)
	if err != nil {
		return err
	}
	if err := a.State.Validate(); err != nil {
		return err
	}
	if expected.State != a.State {
		return core.ErrHostResourceContract
	}
	return nil
}

func memoryAssessmentError(assessment MemoryAssessment) error {
	if err := assessment.Validate(); err != nil {
		return err
	}
	if assessment.State == MemoryPressureReached {
		return core.ErrMemoryLimitReached
	}
	return nil
}

type RuntimeOOMKind uint8

const (
	RuntimeOOMUnknown RuntimeOOMKind = iota
	RuntimeOOMNone
	RuntimeOOMGoAllocator
	RuntimeOOMGoGC
)

func (k RuntimeOOMKind) Validate() error {
	if !k.IsValid() {
		return core.ErrHostResourceContract
	}
	return nil
}

type RuntimeOOMEvidence struct {
	Kind RuntimeOOMKind
}

func (e RuntimeOOMEvidence) Validate() error {
	return e.Kind.Validate()
}

func DetectRuntimeOOM(stderr []byte) RuntimeOOMEvidence {
	if bytes.Contains(stderr, []byte(core.GoRuntimeOOMAllocatorBanner)) {
		return RuntimeOOMEvidence{Kind: RuntimeOOMGoAllocator}
	}
	if bytes.Contains(stderr, []byte(core.GoRuntimeOOMGCBanner)) {
		return RuntimeOOMEvidence{Kind: RuntimeOOMGoGC}
	}
	return RuntimeOOMEvidence{Kind: RuntimeOOMNone}
}
