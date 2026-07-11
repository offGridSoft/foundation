package license

import "github.com/offGridSoft/foundation/v2026/core"

// TransportResponseBody is the structural response socket required by
// lfw/api.Body. APIBody prevents primitives, strings, and loose maps from
// becoming transport responses by accident.
type TransportResponseBody interface {
	core.Validatable
	APIBody()
}

var (
	_ core.Validatable      = (*BugCheckIn)(nil)
	_ core.Validatable      = (*WitnessCheckIn)(nil)
	_ TransportResponseBody = CheckInResponse[SeatLeaseBody]{}
	_ TransportResponseBody = CheckInResponse[SubscriptionLeaseBody]{}
)
