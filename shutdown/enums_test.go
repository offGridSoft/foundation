package shutdown

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestStepOutcomeExhaustiveEnumAndHostileJSONTable(t *testing.T) {
	t.Parallel()

	valid := []StepOutcome{StepOutcomeCompleted, StepOutcomeFailed, StepOutcomeTimedOut, StepOutcomePanicked, StepOutcomeTotalBudgetExceeded}
	for _, value := range valid {
		if !value.IsValid() || value.Validate() != nil || value.String() == StepOutcomeNameUnknown {
			t.Fatalf("StepOutcome(%d) = valid:%t string:%q error:%v, want valid named state", value, value.IsValid(), value.String(), value.Validate())
		}
		raw, err := json.Marshal(value)
		got := StepOutcomeCompleted
		unmarshalErr := json.Unmarshal(raw, &got)
		if err != nil || unmarshalErr != nil || got != value || string(raw) != `"`+value.String()+`"` {
			t.Fatalf("StepOutcome(%d) round trip = raw:%q got:%d errors:%v/%v", value, raw, got, err, unmarshalErr)
		}
	}
	invalidStates := []StepOutcome{StepOutcomeUnknown, StepOutcomeTotalBudgetExceeded + 1, 127, 128, 255}
	for _, value := range invalidStates {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != StepOutcomeNameUnknown {
			t.Fatalf("StepOutcome(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(StepOutcome(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}
	invalidJSON := [][]byte{nil, {}, []byte(`""`), []byte(`"unknown"`), []byte(`"Completed"`), []byte(`0`), []byte(`true`), []byte(`null`), []byte(`[]`), []byte(`{}`), []byte(`"completed" false`), []byte(`"completed`)}
	for index, data := range invalidJSON {
		got := StepOutcomeCompleted
		if err := got.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || got != StepOutcomeCompleted {
			t.Fatalf("invalid JSON %d = value:%d error:%v, want unchanged completed and ErrShutdownContract", index, got, err)
		}
	}
}
