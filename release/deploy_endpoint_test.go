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
			endpoints, err := DeployEndpointsForBaseURL(tc.baseURL)
			if tc.wantErr {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("DeployEndpointsForBaseURL() error = %v, want %v", err, core.ErrReleaseContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("DeployEndpointsForBaseURL() error = %v", err)
			}
			if err := endpoints.Validate(); err != nil {
				t.Fatalf("DeployEndpoints.Validate() error = %v", err)
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
	releaseID := mustBugReleaseID(t)
	status, err := endpoints.Status(releaseID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := status.String(), core.OffgridAPIBaseURL+OffgridBugDeployRootPath+"/"+releaseID.String(); got != want {
		t.Fatalf("status endpoint = %q, want %q", got, want)
	}
}
