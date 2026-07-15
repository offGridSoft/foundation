package release

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestDeployEndpointsHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{name: "production", baseURL: core.OffgridAPIBaseURL},
		{name: "loopback", baseURL: "http://127.0.0.1:40123"},
		{name: "production http rejected", baseURL: "http://api.offgridsoftware.ca", wantErr: true},
		{name: "base path rejected", baseURL: core.OffgridAPIBaseURL + "/api", wantErr: true},
		{name: "query rejected", baseURL: core.OffgridAPIBaseURL + "?x=1", wantErr: true},
		{name: "userinfo rejected", baseURL: "https://user@api.offgridsoftware.ca", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoints, err := BugDeployEndpointsForBaseURL(tc.baseURL)
			if tc.wantErr {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("BugDeployEndpointsForBaseURL() error = %v, want %v", err, core.ErrReleaseContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("BugDeployEndpointsForBaseURL() error = %v", err)
			}
			if err := endpoints.Validate(); err != nil {
				t.Fatalf("DeployEndpoints.Validate() error = %v", err)
			}
		})
	}
}

func TestProductDeployEndpointsUseOneSharedContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		build        func(string) (DeployEndpoints, error)
		buildDefault func() (DeployEndpoints, error)
		latest       string
		name         string
		root         string
		product      core.Product
	}{
		{name: core.ProductTokenBug, product: core.ProductBug, root: OffgridBugDeployRootPath, latest: OffgridBugReleaseLatestPath, build: BugDeployEndpointsForBaseURL, buildDefault: BugDeployEndpoints},
		{name: core.ProductTokenWitness, product: core.ProductWitness, root: OffgridWitnessDeployRootPath, latest: OffgridWitnessReleaseLatestPath, build: WitnessDeployEndpointsForBaseURL, buildDefault: WitnessDeployEndpoints},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoints, err := tc.build(core.OffgridAPIBaseURL)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := endpoints.Prepare.String(), core.OffgridAPIBaseURL+tc.root+"/prepare"; got != want {
				t.Fatalf("prepare endpoint = %q, want %q", got, want)
			}
			if got, want := endpoints.Finalize.String(), core.OffgridAPIBaseURL+tc.root+"/finalize"; got != want {
				t.Fatalf("finalize endpoint = %q, want %q", got, want)
			}
			if got, want := endpoints.Latest.String(), core.OffgridAPIBaseURL+tc.latest; got != want {
				t.Fatalf("latest endpoint = %q, want %q", got, want)
			}
			if endpoints.Product != tc.product {
				t.Fatalf("product = %v, want %v", endpoints.Product, tc.product)
			}
			defaults, err := tc.buildDefault()
			if err != nil {
				t.Fatal(err)
			}
			if defaults != endpoints {
				t.Fatalf("default endpoints = %#v, want %#v", defaults, endpoints)
			}
		})
	}
}

func TestDeployEndpointPathsAreCompilerOwned(t *testing.T) {
	t.Parallel()
	if err := validateDeployEndpointPaths(); err != nil {
		t.Fatal(err)
	}
	endpoints, err := BugDeployEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := endpoints.Prepare.String(), core.OffgridAPIBaseURL+OffgridBugDeployPreparePath; got != want {
		t.Fatalf("prepare endpoint = %q, want %q", got, want)
	}
	if got, want := endpoints.Finalize.String(), core.OffgridAPIBaseURL+OffgridBugDeployFinalizePath; got != want {
		t.Fatalf("finalize endpoint = %q, want %q", got, want)
	}
	if got, want := endpoints.Latest.String(), core.OffgridAPIBaseURL+OffgridBugReleaseLatestPath; got != want {
		t.Fatalf("latest endpoint = %q, want %q", got, want)
	}
	releaseID := mustBugReleaseID(t)
	status, err := endpoints.Status(releaseID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := status.String(), core.OffgridAPIBaseURL+OffgridBugDeployRootPath+"/"+releaseID.String(); got != want {
		t.Fatalf("status endpoint = %q, want %q", got, want)
	}
}
