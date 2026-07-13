package license

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// LeaseGeneration is the server-issued monotonic generation signed into each
// lease. Zero is reserved for a device that has never received a lease.
type LeaseGeneration uint64

func (g LeaseGeneration) IsZero() bool { return g == 0 }

func (g LeaseGeneration) Validate() error {
	if g.IsZero() {
		return fmt.Errorf(ErrFmtLeaseGeneration, core.ErrLeaseGeneration)
	}
	return nil
}

func (g LeaseGeneration) ValidateOptional() error {
	if g.IsZero() {
		return nil
	}
	return g.Validate()
}

func (g LeaseGeneration) MarshalJSON() ([]byte, error) {
	if g.IsZero() {
		return []byte(core.JSONLiteralNull), nil
	}
	return json.Marshal(uint64(g))
}

//validate:unmarshal_ignore reason="Validation occurs before assigning the decoded generation to the receiver."
func (g *LeaseGeneration) UnmarshalJSON(data []byte) error {
	if string(data) == core.JSONLiteralNull {
		*g = 0
		return nil
	}
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtLeaseGeneration, core.ErrLeaseGeneration)
	}
	parsed := LeaseGeneration(value)
	if err := parsed.Validate(); err != nil {
		return err
	}
	*g = parsed
	return nil
}
