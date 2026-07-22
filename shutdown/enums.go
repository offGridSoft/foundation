package shutdown

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	StepOutcomeNameUnknown             = "unknown"
	StepOutcomeNameCompleted           = "completed"
	StepOutcomeNameFailed              = "failed"
	StepOutcomeNameTimedOut            = "timed-out"
	StepOutcomeNamePanicked            = "panicked"
	StepOutcomeNameTotalBudgetExceeded = "total-budget-exceeded"
	ErrFmtStepOutcome                  = "shutdown.StepOutcome: %w"
)

func (o StepOutcome) IsValid() bool {
	return o >= StepOutcomeCompleted && o <= StepOutcomeTotalBudgetExceeded
}

func (o StepOutcome) String() string {
	switch o {
	case StepOutcomeCompleted:
		return StepOutcomeNameCompleted
	case StepOutcomeFailed:
		return StepOutcomeNameFailed
	case StepOutcomeTimedOut:
		return StepOutcomeNameTimedOut
	case StepOutcomePanicked:
		return StepOutcomeNamePanicked
	case StepOutcomeTotalBudgetExceeded:
		return StepOutcomeNameTotalBudgetExceeded
	default:
		return StepOutcomeNameUnknown
	}
}

func ParseStepOutcome(token string) (StepOutcome, error) {
	switch token {
	case StepOutcomeNameCompleted:
		return StepOutcomeCompleted, nil
	case StepOutcomeNameFailed:
		return StepOutcomeFailed, nil
	case StepOutcomeNameTimedOut:
		return StepOutcomeTimedOut, nil
	case StepOutcomeNamePanicked:
		return StepOutcomePanicked, nil
	case StepOutcomeNameTotalBudgetExceeded:
		return StepOutcomeTotalBudgetExceeded, nil
	default:
		return StepOutcomeUnknown, fmt.Errorf(ErrFmtStepOutcome, core.ErrShutdownContract)
	}
}

func (o StepOutcome) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf(ErrFmtStepOutcome, err)
	}
	return json.Marshal(o.String())
}

func (o *StepOutcome) UnmarshalJSON(data []byte) error {
	if o == nil {
		return fmt.Errorf(ErrFmtStepOutcome, core.ErrShutdownContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtStepOutcome, errors.Join(core.ErrShutdownContract, err))
	}
	parsed, err := ParseStepOutcome(token)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}
