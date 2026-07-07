package license

import (
	"fmt"
	"net/url"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

const (
	SchemaBugCheckIn          = "bug-license-check-in-v1"
	SchemaBugSeatLease        = "bug-license-lease-v1"
	SchemaWitnessCheckIn      = "witness-subscription-check-in-v1"
	SchemaWitnessSubscription = "witness-subscription-lease-v1"

	OffgridAPIBaseURL         = "https://api.offgridsoftware.ca"
	OffgridBugCheckInPath     = "/v1/bug/check_in"
	OffgridWitnessCheckInPath = "/v1/witness/check_in"
	OffgridAPICallKeyHeader   = "X-Offgrid-Api-Key"
)

type CheckInEndpoint struct {
	value string
}

func MustCheckInEndpoint(value string) CheckInEndpoint {
	endpoint, err := ParseCheckInEndpoint(value)
	if err != nil {
		panic(err)
	}
	return endpoint
}

func ParseCheckInEndpoint(value string) (CheckInEndpoint, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return CheckInEndpoint{}, fmt.Errorf(ErrFmtCheckInEndpoint, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || strings.TrimSpace(parsed.Path) == "" {
		return CheckInEndpoint{}, fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	return CheckInEndpoint{value: value}, nil
}

func (e CheckInEndpoint) String() string {
	return e.value
}

func (e CheckInEndpoint) Validate() error {
	_, err := ParseCheckInEndpoint(e.value)
	return err
}

func (e CheckInEndpoint) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e.value)
}

func (e *CheckInEndpoint) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	parsed, err := ParseCheckInEndpoint(value)
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

var (
	BugCheckInEndpoint     = MustCheckInEndpoint(OffgridAPIBaseURL + OffgridBugCheckInPath)
	WitnessCheckInEndpoint = MustCheckInEndpoint(OffgridAPIBaseURL + OffgridWitnessCheckInPath)
)
