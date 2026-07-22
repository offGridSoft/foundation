package peachfuzz

import (
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestOffgridProductionEndpointsBindTypedBaseAndPathsTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		get  func() (core.APIEndpoint, error)
		name string
		path string
	}{
		{name: "stats endpoint", path: OffgridRunStatsPath, get: OffgridRunStatsEndpoint},
		{name: "evidence upload endpoint", path: OffgridRunEvidenceUploadPath, get: OffgridRunEvidenceUploadEndpoint},
		{name: "evidence finalize endpoint", path: OffgridRunEvidenceFinalizePath, get: OffgridRunEvidenceFinalizeEndpoint},
		{name: "evidence materialize endpoint", path: OffgridRunEvidenceMaterializePath, get: OffgridRunEvidenceMaterializeEndpoint},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := test.get()
			if err != nil {
				t.Fatalf("production endpoint error = %v, want nil", err)
			}
			want, err := core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, test.path)
			if err != nil {
				t.Fatalf("APIEndpointForBaseURL() error = %v, want nil", err)
			}
			if got != want {
				t.Fatalf("production endpoint = %s, want %s", got.String(), want.String())
			}
		})
	}
}
