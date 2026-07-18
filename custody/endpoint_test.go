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
