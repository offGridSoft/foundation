package custody

import (
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestWitnessCustodyEndpointContractsTable(t *testing.T) {
	t.Parallel()
	wantRoot := "/" + core.ContractVersionToken + "/" + core.ProductTokenWitness + "/custody"
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "root", got: OffgridWitnessCustodyRootPath, want: wantRoot},
		{name: "open", got: OffgridWitnessCustodyOpenPath, want: wantRoot + "/open"},
		{name: "finalize", got: OffgridWitnessCustodyFinalizePath, want: wantRoot + "/finalize"},
		{name: "download", got: OffgridWitnessCustodyDownloadPath, want: wantRoot + "/download"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("endpoint = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestWitnessCustodyDefaultEndpointsBindProductionBaseAndClosedPaths(t *testing.T) {
	t.Parallel()

	got, err := WitnessCustodyEndpoints()
	if err != nil {
		t.Fatalf("WitnessCustodyEndpoints() error = %v, want nil", err)
	}
	want, err := WitnessCustodyEndpointsForBaseURL(core.OffgridAPIBaseURL)
	if err != nil {
		t.Fatalf("WitnessCustodyEndpointsForBaseURL() error = %v, want nil", err)
	}
	if got != want {
		t.Fatalf("WitnessCustodyEndpoints() = %+v, want %+v", got, want)
	}
}
