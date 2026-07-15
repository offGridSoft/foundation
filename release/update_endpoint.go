package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	OffgridBugUpdateRootPath           = "/" + core.ContractVersionToken + "/" + core.ProductTokenBug + "/updates"
	OffgridBugUpdateCheckPath          = OffgridBugUpdateRootPath + "/check"
	OffgridBugUpdateDiagnosticPath     = OffgridBugUpdateRootPath + "/diagnostics"
	OffgridWitnessUpdateRootPath       = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/updates"
	OffgridWitnessUpdateCheckPath      = OffgridWitnessUpdateRootPath + "/check"
	OffgridWitnessUpdateDiagnosticPath = OffgridWitnessUpdateRootPath + "/diagnostics"
)

type UpdateEndpoints struct {
	Check      core.APIEndpoint
	Diagnostic core.APIEndpoint
	Product    core.Product
}

func BugUpdateEndpointsForBaseURL(baseURL string) (UpdateEndpoints, error) {
	return updateEndpointsForBaseURL(baseURL, OffgridBugUpdateCheckPath, OffgridBugUpdateDiagnosticPath, core.ProductBug)
}

func WitnessUpdateEndpointsForBaseURL(baseURL string) (UpdateEndpoints, error) {
	return updateEndpointsForBaseURL(baseURL, OffgridWitnessUpdateCheckPath, OffgridWitnessUpdateDiagnosticPath, core.ProductWitness)
}

func BugUpdateEndpoints() (UpdateEndpoints, error) {
	return BugUpdateEndpointsForBaseURL(core.OffgridAPIBaseURL)
}

func WitnessUpdateEndpoints() (UpdateEndpoints, error) {
	return WitnessUpdateEndpointsForBaseURL(core.OffgridAPIBaseURL)
}

func updateEndpointsForBaseURL(baseURL, checkPath, diagnosticPath string, product core.Product) (UpdateEndpoints, error) {
	check, err := core.APIEndpointForBaseURL(baseURL, checkPath)
	if err != nil {
		return UpdateEndpoints{}, wrapReleaseContract(ErrFmtUpdateEndpoints, err)
	}
	diagnostic, err := core.APIEndpointForBaseURL(baseURL, diagnosticPath)
	if err != nil {
		return UpdateEndpoints{}, wrapReleaseContract(ErrFmtUpdateEndpoints, err)
	}
	endpoints := UpdateEndpoints{Check: check, Diagnostic: diagnostic, Product: product}
	if err := endpoints.Validate(); err != nil {
		return UpdateEndpoints{}, err
	}
	return endpoints, nil
}

func (e UpdateEndpoints) Validate() error {
	if err := e.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateEndpoints, err)
	}
	if err := e.Check.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateEndpoints, err)
	}
	if err := e.Diagnostic.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateEndpoints, err)
	}
	if e.Check == e.Diagnostic {
		return fmt.Errorf(ErrFmtUpdateEndpoints, core.ErrReleaseContract)
	}
	return nil
}
