package custody

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	OffgridWitnessCustodyRootPath     = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/custody"
	OffgridWitnessCustodyOpenPath     = OffgridWitnessCustodyRootPath + "/open"
	OffgridWitnessCustodyFinalizePath = OffgridWitnessCustodyRootPath + "/finalize"
	OffgridWitnessCustodyDownloadPath = OffgridWitnessCustodyRootPath + "/download"
)

// Endpoints binds the three custody API routes to one base URL. Signed
// storage-transfer URLs are not endpoints here: they arrive inside grants.
type Endpoints struct {
	Open     core.APIEndpoint
	Finalize core.APIEndpoint
	Download core.APIEndpoint
}

func WitnessCustodyEndpoints() (Endpoints, error) {
	return WitnessCustodyEndpointsForBaseURL(core.OffgridAPIBaseURL)
}

func WitnessCustodyEndpointsForBaseURL(baseURL string) (Endpoints, error) {
	open, err := core.APIEndpointForBaseURL(baseURL, OffgridWitnessCustodyOpenPath)
	if err != nil {
		return Endpoints{}, fmt.Errorf(ErrFmtEndpoints, err)
	}
	finalize, err := core.APIEndpointForBaseURL(baseURL, OffgridWitnessCustodyFinalizePath)
	if err != nil {
		return Endpoints{}, fmt.Errorf(ErrFmtEndpoints, err)
	}
	download, err := core.APIEndpointForBaseURL(baseURL, OffgridWitnessCustodyDownloadPath)
	if err != nil {
		return Endpoints{}, fmt.Errorf(ErrFmtEndpoints, err)
	}
	endpoints := Endpoints{Open: open, Finalize: finalize, Download: download}
	if err := endpoints.Validate(); err != nil {
		return Endpoints{}, err
	}
	return endpoints, nil
}

func (e Endpoints) Validate() error {
	for _, endpoint := range []core.APIEndpoint{e.Open, e.Finalize, e.Download} {
		if err := endpoint.Validate(); err != nil {
			return fmt.Errorf(ErrFmtEndpoints, err)
		}
	}
	if e.Open == e.Finalize || e.Open == e.Download || e.Finalize == e.Download {
		return fmt.Errorf(ErrFmtEndpoints, core.ErrCustodyContract)
	}
	return nil
}
