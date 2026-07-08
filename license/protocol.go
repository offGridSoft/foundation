package license

import (
	"errors"
	"fmt"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

const (
	OffgridAPIBaseURL         = "https://api.offgridsoftware.ca"
	OffgridBugCheckInPath     = "/v1/" + core.ProductTokenBug + "/check_in"
	OffgridWitnessCheckInPath = "/v1/" + core.ProductTokenWitness + "/check_in"
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
	if err := core.ValidateHTTPSURL(value, core.HTTPSURLPolicy{
		MaxRunes:    core.HTTPSURLDefaultMaxRunes,
		RequirePath: true,
	}); err != nil {
		return CheckInEndpoint{}, fmt.Errorf(ErrFmtCheckInEndpoint, errors.Join(core.ErrLicenseContract, err))
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
