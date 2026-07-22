package hostresource

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	DiskPressureNameUnknown  = "unknown"
	DiskPressureNameDisabled = "disabled"
	DiskPressureNameHealthy  = "healthy"
	DiskPressureNameReached  = "reached"

	MissingPathNameUnknown = "unknown"
	MissingPathNameReject  = "reject"
	MissingPathNameIsEmpty = "is-empty"

	MemoryPressureNameUnknown = "unknown"
	MemoryPressureNameHealthy = "healthy"
	MemoryPressureNameReached = "reached"

	RuntimeOOMNameUnknown     = "unknown"
	RuntimeOOMNameNone        = "none"
	RuntimeOOMNameGoAllocator = "go-allocator"
	RuntimeOOMNameGoGC        = "go-gc"

	ErrFmtDiskPressureState   = "hostresource.DiskPressureState: %w"
	ErrFmtMissingPathPolicy   = "hostresource.MissingPathPolicy: %w"
	ErrFmtMemoryPressureState = "hostresource.MemoryPressureState: %w"
	ErrFmtRuntimeOOMKind      = "hostresource.RuntimeOOMKind: %w"
)

func (s DiskPressureState) IsValid() bool {
	return s == DiskPressureDisabled || s == DiskPressureHealthy || s == DiskPressureReached
}

func (s DiskPressureState) String() string {
	switch s {
	case DiskPressureDisabled:
		return DiskPressureNameDisabled
	case DiskPressureHealthy:
		return DiskPressureNameHealthy
	case DiskPressureReached:
		return DiskPressureNameReached
	default:
		return DiskPressureNameUnknown
	}
}

func ParseDiskPressureState(token string) (DiskPressureState, error) {
	switch token {
	case DiskPressureNameDisabled:
		return DiskPressureDisabled, nil
	case DiskPressureNameHealthy:
		return DiskPressureHealthy, nil
	case DiskPressureNameReached:
		return DiskPressureReached, nil
	default:
		return DiskPressureUnknown, fmt.Errorf(ErrFmtDiskPressureState, core.ErrHostResourceContract)
	}
}

func (s DiskPressureState) MarshalJSON() ([]byte, error) {
	return marshalHostEnumJSON(s.String(), s.Validate(), ErrFmtDiskPressureState)
}

func (s *DiskPressureState) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf(ErrFmtDiskPressureState, core.ErrHostResourceContract)
	}
	parsed, err := unmarshalHostEnumJSON(data, ParseDiskPressureState, ErrFmtDiskPressureState)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (p MissingPathPolicy) IsValid() bool {
	return p == MissingPathReject || p == MissingPathIsEmpty
}

func (p MissingPathPolicy) String() string {
	switch p {
	case MissingPathReject:
		return MissingPathNameReject
	case MissingPathIsEmpty:
		return MissingPathNameIsEmpty
	default:
		return MissingPathNameUnknown
	}
}

func ParseMissingPathPolicy(token string) (MissingPathPolicy, error) {
	switch token {
	case MissingPathNameReject:
		return MissingPathReject, nil
	case MissingPathNameIsEmpty:
		return MissingPathIsEmpty, nil
	default:
		return MissingPathUnknown, fmt.Errorf(ErrFmtMissingPathPolicy, core.ErrHostResourceContract)
	}
}

func (p MissingPathPolicy) MarshalJSON() ([]byte, error) {
	return marshalHostEnumJSON(p.String(), p.Validate(), ErrFmtMissingPathPolicy)
}

func (p *MissingPathPolicy) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf(ErrFmtMissingPathPolicy, core.ErrHostResourceContract)
	}
	parsed, err := unmarshalHostEnumJSON(data, ParseMissingPathPolicy, ErrFmtMissingPathPolicy)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

func (s MemoryPressureState) IsValid() bool {
	return s == MemoryPressureHealthy || s == MemoryPressureReached
}

func (s MemoryPressureState) String() string {
	switch s {
	case MemoryPressureHealthy:
		return MemoryPressureNameHealthy
	case MemoryPressureReached:
		return MemoryPressureNameReached
	default:
		return MemoryPressureNameUnknown
	}
}

func ParseMemoryPressureState(token string) (MemoryPressureState, error) {
	switch token {
	case MemoryPressureNameHealthy:
		return MemoryPressureHealthy, nil
	case MemoryPressureNameReached:
		return MemoryPressureReached, nil
	default:
		return MemoryPressureUnknown, fmt.Errorf(ErrFmtMemoryPressureState, core.ErrHostResourceContract)
	}
}

func (s MemoryPressureState) MarshalJSON() ([]byte, error) {
	return marshalHostEnumJSON(s.String(), s.Validate(), ErrFmtMemoryPressureState)
}

func (s *MemoryPressureState) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf(ErrFmtMemoryPressureState, core.ErrHostResourceContract)
	}
	parsed, err := unmarshalHostEnumJSON(data, ParseMemoryPressureState, ErrFmtMemoryPressureState)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (k RuntimeOOMKind) IsValid() bool {
	return k == RuntimeOOMNone || k == RuntimeOOMGoAllocator || k == RuntimeOOMGoGC
}

func (k RuntimeOOMKind) String() string {
	switch k {
	case RuntimeOOMNone:
		return RuntimeOOMNameNone
	case RuntimeOOMGoAllocator:
		return RuntimeOOMNameGoAllocator
	case RuntimeOOMGoGC:
		return RuntimeOOMNameGoGC
	default:
		return RuntimeOOMNameUnknown
	}
}

func ParseRuntimeOOMKind(token string) (RuntimeOOMKind, error) {
	switch token {
	case RuntimeOOMNameNone:
		return RuntimeOOMNone, nil
	case RuntimeOOMNameGoAllocator:
		return RuntimeOOMGoAllocator, nil
	case RuntimeOOMNameGoGC:
		return RuntimeOOMGoGC, nil
	default:
		return RuntimeOOMUnknown, fmt.Errorf(ErrFmtRuntimeOOMKind, core.ErrHostResourceContract)
	}
}

func (k RuntimeOOMKind) MarshalJSON() ([]byte, error) {
	return marshalHostEnumJSON(k.String(), k.Validate(), ErrFmtRuntimeOOMKind)
}

func (k *RuntimeOOMKind) UnmarshalJSON(data []byte) error {
	if k == nil {
		return fmt.Errorf(ErrFmtRuntimeOOMKind, core.ErrHostResourceContract)
	}
	parsed, err := unmarshalHostEnumJSON(data, ParseRuntimeOOMKind, ErrFmtRuntimeOOMKind)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

func marshalHostEnumJSON(token string, validationErr error, format string) ([]byte, error) {
	if validationErr != nil {
		return nil, fmt.Errorf(format, validationErr)
	}
	return json.Marshal(token)
}

func unmarshalHostEnumJSON[T any](data []byte, parse func(string) (T, error), format string) (T, error) {
	var zero T
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return zero, fmt.Errorf(format, errors.Join(core.ErrHostResourceContract, err))
	}
	return parse(token)
}
