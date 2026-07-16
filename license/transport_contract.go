package license

import "github.com/offGridSoft/foundation/v2026/core"

var (
	_ core.Validatable = (*BugCheckIn)(nil)
	_ core.Validatable = (*WitnessCheckIn)(nil)
	_ core.APIBody     = BugCheckInResponse{}
	_ core.APIBody     = WitnessCheckInResponse{}
)
