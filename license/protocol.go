package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
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

func CheckInPath(product core.Product) (string, error) {
	switch product {
	case core.ProductBug:
		return OffgridBugCheckInPath, nil
	case core.ProductWitness:
		return OffgridWitnessCheckInPath, nil
	default:
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
}

func CheckInEndpointForBaseURL(baseURL string, product core.Product) (CheckInEndpoint, error) {
	path, err := CheckInPath(product)
	if err != nil {
		return CheckInEndpoint{}, err
	}
	base, err := normalizeCheckInBaseURL(baseURL)
	if err != nil {
		return CheckInEndpoint{}, err
	}
	return ParseCheckInEndpoint(base + path)
}

func ParseCheckInEndpoint(value string) (CheckInEndpoint, error) {
	if err := validateCheckInEndpoint(value); err != nil {
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

func BugCheckInEndpoint() CheckInEndpoint {
	return CheckInEndpoint{value: OffgridAPIBaseURL + OffgridBugCheckInPath}
}

func WitnessCheckInEndpoint() CheckInEndpoint {
	return CheckInEndpoint{value: OffgridAPIBaseURL + OffgridWitnessCheckInPath}
}

func normalizeCheckInBaseURL(value string) (string, error) {
	if err := core.ValidateOpaqueToken(value, core.HTTPSURLDefaultMaxRunes); err != nil {
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	if parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	if strings.Trim(parsed.Path, "/") != "" {
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, core.ErrLicenseContract)
	}
	if err := validateCheckInEndpointScheme(parsed); err != nil {
		return "", fmt.Errorf(ErrFmtCheckInEndpoint, errors.Join(core.ErrLicenseContract, err))
	}
	parsed.Path = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func validateCheckInEndpoint(value string) error {
	if err := core.ValidateOpaqueToken(value, core.HTTPSURLDefaultMaxRunes); err != nil {
		return fmt.Errorf(core.ErrFmtHTTPSURL, core.ErrFoundationContract)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf(core.ErrFmtHTTPSURL, core.ErrFoundationContract)
	}
	if err := validateCheckInEndpointScheme(parsed); err != nil {
		return err
	}
	if parsed.User != nil || parsed.Host == "" {
		return fmt.Errorf(core.ErrFmtHTTPSURL, core.ErrFoundationContract)
	}
	if strings.TrimSpace(parsed.Path) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf(core.ErrFmtHTTPSURL, core.ErrFoundationContract)
	}
	return nil
}

func validateCheckInEndpointScheme(parsed *url.URL) error {
	switch parsed.Scheme {
	case core.URLSchemeHTTPS:
		return nil
	case core.URLSchemeHTTP:
		if localCheckInHost(parsed.Hostname()) {
			return nil
		}
	}
	return fmt.Errorf(core.ErrFmtHTTPSURL, core.ErrFoundationContract)
}

func localCheckInHost(host string) bool {
	if host == core.HostLocalhost {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
