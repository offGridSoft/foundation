package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	OffgridBugDeployRootPath         = "/" + core.ContractVersionToken + "/" + core.ProductTokenBug + "/releases/deploy"
	OffgridBugDeployPreparePath      = OffgridBugDeployRootPath + "/prepare"
	OffgridBugDeployFinalizePath     = OffgridBugDeployRootPath + "/finalize"
	OffgridWitnessDeployRootPath     = "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/releases/deploy"
	OffgridWitnessDeployPreparePath  = OffgridWitnessDeployRootPath + "/prepare"
	OffgridWitnessDeployFinalizePath = OffgridWitnessDeployRootPath + "/finalize"
)

type DeployEndpoints struct {
	Prepare    core.APIEndpoint
	Finalize   core.APIEndpoint
	StatusRoot core.APIEndpoint
}

func BugDeployEndpointsForBaseURL(baseURL string) (DeployEndpoints, error) {
	return deployEndpointsForBaseURL(baseURL, OffgridBugDeployRootPath)
}

func WitnessDeployEndpointsForBaseURL(baseURL string) (DeployEndpoints, error) {
	return deployEndpointsForBaseURL(baseURL, OffgridWitnessDeployRootPath)
}

func deployEndpointsForBaseURL(baseURL, rootPath string) (DeployEndpoints, error) {
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
	endpoints := DeployEndpoints{Prepare: prepare, Finalize: finalize, StatusRoot: statusRoot}
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
	for _, endpoint := range []core.APIEndpoint{e.Prepare, e.Finalize, e.StatusRoot} {
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
		OffgridBugDeployRootPath == OffgridWitnessDeployRootPath {
		return fmt.Errorf(ErrFmtDeployEndpoints, core.ErrReleaseContract)
	}
	return nil
}
