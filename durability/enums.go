package durability

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	InstallNameUnknown = "unknown"
	InstallNameReplace = "replace"
	InstallNameCreate  = "create"

	ActivationNameUnknown               = "unknown"
	ActivationNameNotActivated          = "not-activated"
	ActivationNameDirectorySyncRequired = "directory-sync-required"
	ActivationNameDurable               = "durable"

	TemporaryNameUnknown             = "unknown"
	TemporaryNameRetained            = "retained"
	TemporaryNameRemovalSyncRequired = "removal-sync-required"
	TemporaryNameRemoved             = "removed"

	ErrFmtInstallMode     = "durability.InstallMode: %w"
	ErrFmtActivationState = "durability.ActivationState: %w"
	ErrFmtTemporaryState  = "durability.TemporaryState: %w"
)

func (m InstallMode) IsValid() bool {
	return m == InstallReplace || m == InstallCreate
}

func (m InstallMode) String() string {
	switch m {
	case InstallReplace:
		return InstallNameReplace
	case InstallCreate:
		return InstallNameCreate
	default:
		return InstallNameUnknown
	}
}

func ParseInstallMode(token string) (InstallMode, error) {
	switch token {
	case InstallNameReplace:
		return InstallReplace, nil
	case InstallNameCreate:
		return InstallCreate, nil
	default:
		return InstallUnknown, fmt.Errorf(ErrFmtInstallMode, core.ErrDurabilityContract)
	}
}

func (m InstallMode) MarshalJSON() ([]byte, error) {
	return marshalEnumJSON(m.String(), m.Validate(), ErrFmtInstallMode)
}

func (m *InstallMode) UnmarshalJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf(ErrFmtInstallMode, core.ErrDurabilityContract)
	}
	parsed, err := unmarshalEnumJSON(data, ParseInstallMode, ErrFmtInstallMode)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (s ActivationState) IsValid() bool {
	return s == ActivationNotActivated || s == ActivationDirectorySyncRequired || s == ActivationDurable
}

func (s ActivationState) String() string {
	switch s {
	case ActivationNotActivated:
		return ActivationNameNotActivated
	case ActivationDirectorySyncRequired:
		return ActivationNameDirectorySyncRequired
	case ActivationDurable:
		return ActivationNameDurable
	default:
		return ActivationNameUnknown
	}
}

func ParseActivationState(token string) (ActivationState, error) {
	switch token {
	case ActivationNameNotActivated:
		return ActivationNotActivated, nil
	case ActivationNameDirectorySyncRequired:
		return ActivationDirectorySyncRequired, nil
	case ActivationNameDurable:
		return ActivationDurable, nil
	default:
		return ActivationUnknown, fmt.Errorf(ErrFmtActivationState, core.ErrDurabilityContract)
	}
}

func (s ActivationState) MarshalJSON() ([]byte, error) {
	return marshalEnumJSON(s.String(), s.Validate(), ErrFmtActivationState)
}

func (s *ActivationState) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf(ErrFmtActivationState, core.ErrDurabilityContract)
	}
	parsed, err := unmarshalEnumJSON(data, ParseActivationState, ErrFmtActivationState)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s TemporaryState) IsValid() bool {
	return s == TemporaryRetained || s == TemporaryRemovalSyncRequired || s == TemporaryRemoved
}

func (s TemporaryState) String() string {
	switch s {
	case TemporaryRetained:
		return TemporaryNameRetained
	case TemporaryRemovalSyncRequired:
		return TemporaryNameRemovalSyncRequired
	case TemporaryRemoved:
		return TemporaryNameRemoved
	default:
		return TemporaryNameUnknown
	}
}

func ParseTemporaryState(token string) (TemporaryState, error) {
	switch token {
	case TemporaryNameRetained:
		return TemporaryRetained, nil
	case TemporaryNameRemovalSyncRequired:
		return TemporaryRemovalSyncRequired, nil
	case TemporaryNameRemoved:
		return TemporaryRemoved, nil
	default:
		return TemporaryUnknown, fmt.Errorf(ErrFmtTemporaryState, core.ErrDurabilityContract)
	}
}

func (s TemporaryState) MarshalJSON() ([]byte, error) {
	return marshalEnumJSON(s.String(), s.Validate(), ErrFmtTemporaryState)
}

func (s *TemporaryState) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf(ErrFmtTemporaryState, core.ErrDurabilityContract)
	}
	parsed, err := unmarshalEnumJSON(data, ParseTemporaryState, ErrFmtTemporaryState)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func marshalEnumJSON(token string, validationErr error, format string) ([]byte, error) {
	if validationErr != nil {
		return nil, fmt.Errorf(format, validationErr)
	}
	return json.Marshal(token)
}

func unmarshalEnumJSON[T any](data []byte, parse func(string) (T, error), format string) (T, error) {
	var zero T
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return zero, fmt.Errorf(format, errors.Join(core.ErrDurabilityContract, err))
	}
	return parse(token)
}
