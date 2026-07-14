package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	OffgridBugCheckInPath     = "/" + core.ContractVersionToken + "/" + core.ProductTokenBug + "/check_in"
	OffgridWitnessCheckInPath = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/check_in"
	OffgridAPICallKeyHeader   = "X-Offgrid-Api-Key"
)

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

func CheckInEndpointForBaseURL(baseURL string, product core.Product) (core.APIEndpoint, error) {
	path, err := CheckInPath(product)
	if err != nil {
		return core.APIEndpoint{}, err
	}
	endpoint, err := core.APIEndpointForBaseURL(baseURL, path)
	if err != nil {
		return core.APIEndpoint{}, fmt.Errorf(ErrFmtCheckInEndpoint, errors.Join(core.ErrLicenseContract, err))
	}
	return endpoint, nil
}

func ParseCheckInEndpoint(value string) (core.APIEndpoint, error) {
	endpoint, err := core.ParseAPIEndpoint(value)
	if err != nil {
		return core.APIEndpoint{}, fmt.Errorf(ErrFmtCheckInEndpoint, errors.Join(core.ErrLicenseContract, err))
	}
	return endpoint, nil
}
