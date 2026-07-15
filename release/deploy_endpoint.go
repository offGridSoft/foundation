package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	OffgridBugDeployRootPath         = "/" + core.ContractVersionToken + "/" + core.ProductTokenBug + "/releases/deploy"
	OffgridBugDeployPreparePath      = OffgridBugDeployRootPath + "/prepare"
	OffgridBugDeployFinalizePath     = OffgridBugDeployRootPath + "/finalize"
	OffgridBugReleaseLatestPath      = "/" + core.ContractVersionToken + "/" + core.ProductTokenBug + "/releases/latest"
	OffgridWitnessDeployRootPath     = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/releases/deploy"
	OffgridWitnessDeployPreparePath  = OffgridWitnessDeployRootPath + "/prepare"
	OffgridWitnessDeployFinalizePath = OffgridWitnessDeployRootPath + "/finalize"
	OffgridWitnessReleaseLatestPath  = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/releases/latest"
)

type DeployEndpoints struct {
	Prepare    core.APIEndpoint
	Finalize   core.APIEndpoint
	StatusRoot core.APIEndpoint
	Latest     core.APIEndpoint
	Product    core.Product
}

func BugDeployEndpointsForBaseURL(baseURL string) (DeployEndpoints, error) {
	return deployEndpointsForBaseURL(baseURL, OffgridBugDeployRootPath, OffgridBugReleaseLatestPath, core.ProductBug)
}

func WitnessDeployEndpointsForBaseURL(baseURL string) (DeployEndpoints, error) {
	return deployEndpointsForBaseURL(baseURL, OffgridWitnessDeployRootPath, OffgridWitnessReleaseLatestPath, core.ProductWitness)
}

func deployEndpointsForBaseURL(baseURL, rootPath, latestPath string, product core.Product) (DeployEndpoints, error) {
	prepare, err := core.APIEndpointForBaseURL(baseURL, rootPath+"/prepare")
	if err != nil {
		return DeployEndpoints{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	finalize, err := core.APIEndpointForBaseURL(baseURL, rootPath+"/finalize")
	if err != nil {
		return DeployEndpoints{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	statusRoot, err := core.APIEndpointForBaseURL(baseURL, rootPath)
	if err != nil {
		return DeployEndpoints{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	latest, err := core.APIEndpointForBaseURL(baseURL, latestPath)
	if err != nil {
		return DeployEndpoints{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	endpoints := DeployEndpoints{Prepare: prepare, Finalize: finalize, StatusRoot: statusRoot, Latest: latest, Product: product}
	if err := endpoints.Validate(); err != nil {
		return DeployEndpoints{}, err
	}
	return endpoints, nil
}

func BugDeployEndpoints() (DeployEndpoints, error) {
	return BugDeployEndpointsForBaseURL(core.OffgridAPIBaseURL)
}

func WitnessDeployEndpoints() (DeployEndpoints, error) {
	return WitnessDeployEndpointsForBaseURL(core.OffgridAPIBaseURL)
}

func (e DeployEndpoints) Validate() error {
	if err := e.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	for _, endpoint := range []core.APIEndpoint{e.Prepare, e.Finalize, e.StatusRoot, e.Latest} {
		if err := endpoint.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtDeployEndpoints, err)
		}
	}
	return nil
}

func (e DeployEndpoints) Status(releaseID ReleaseID) (core.APIEndpoint, error) {
	if err := e.Validate(); err != nil {
		return core.APIEndpoint{}, err
	}
	if err := releaseID.Validate(); err != nil {
		return core.APIEndpoint{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	endpoint, err := core.ParseAPIEndpoint(e.StatusRoot.String() + "/" + releaseID.String())
	if err != nil {
		return core.APIEndpoint{}, wrapReleaseContract(ErrFmtDeployEndpoints, err)
	}
	return endpoint, nil
}

func validateDeployEndpointPaths() error {
	if OffgridBugDeployPreparePath == OffgridBugDeployFinalizePath || OffgridBugDeployRootPath == "" ||
		OffgridWitnessDeployPreparePath == OffgridWitnessDeployFinalizePath || OffgridWitnessDeployRootPath == "" ||
		OffgridBugDeployRootPath == OffgridWitnessDeployRootPath ||
		OffgridBugReleaseLatestPath == OffgridWitnessReleaseLatestPath {
		return fmt.Errorf(ErrFmtDeployEndpoints, core.ErrReleaseContract)
	}
	return nil
}
