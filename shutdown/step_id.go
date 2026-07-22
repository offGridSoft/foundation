package shutdown

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const ErrFmtStepID = "shutdown.StepID: %w"

type StepID string

func ParseStepID(value string) (StepID, error) {
	if err := core.ValidateOpaqueToken(value, core.ShutdownStepIDMaxRunes); err != nil {
		return "", fmt.Errorf(ErrFmtStepID, errors.Join(core.ErrShutdownContract, err))
	}
	return StepID(value), nil
}

func (id StepID) Validate() error {
	_, err := ParseStepID(string(id))
	return err
}

func (id StepID) String() string {
	return string(id)
}

func (id StepID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.String())
}

func (id *StepID) UnmarshalJSON(data []byte) error {
	if id == nil {
		return fmt.Errorf(ErrFmtStepID, core.ErrShutdownContract)
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtStepID, errors.Join(core.ErrShutdownContract, err))
	}
	parsed, err := ParseStepID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
