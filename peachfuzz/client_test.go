package peachfuzz

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSnapshotClientOGSPublicBoundary(t *testing.T) {
	t.Parallel()
	snapshot := validProjectSnapshot(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Query().Get(QueryProject) != snapshot.Project.String() {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get(core.HTTPHeaderAuthorization); got != "" {
			t.Errorf("public authorization header = %q, want empty", got)
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writeClientEnvelope(t, w, snapshot)
	}))
	t.Cleanup(server.Close)
	client := validSnapshotClient(t, server)
	got, err := client.Snapshot(t.Context(), snapshot.Project)
	if err != nil {
		t.Fatalf("SnapshotClient.Snapshot() error = %v", err)
	}
	if got.Project != snapshot.Project || got.CoreYearsHumanized != snapshot.CoreYearsHumanized {
		t.Fatalf("SnapshotClient.Snapshot() = %#v, want project-bound snapshot", got)
	}
}

func TestSnapshotClientOGSRejectsSubstitutedProject(t *testing.T) {
	t.Parallel()
	snapshot := validProjectSnapshot(t)
	otherProject, err := ParseProjectID("witness")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		snapshot.Project = otherProject
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writeClientEnvelope(t, w, snapshot)
	}))
	t.Cleanup(server.Close)
	_, err = validSnapshotClient(t, server).Snapshot(t.Context(), validProjectSnapshot(t).Project)
	if !errors.Is(err, ErrContract) {
		t.Fatalf("SnapshotClient.Snapshot() error = %v, want %v", err, ErrContract)
	}
}

func TestSnapshotClientOGSTimeoutContractTable(t *testing.T) {
	t.Parallel()
	endpoint, err := core.APIEndpointForBaseURL("https://api.offgridsoftware.ca", OffgridRunStatsPath)
	if err != nil {
		t.Fatalf("APIEndpointForBaseURL() error = %v", err)
	}
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{name: "exact contract", timeout: HTTPClientTimeout, valid: true},
		{name: "unbounded", timeout: 0},
		{name: "drifted shorter", timeout: HTTPClientTimeout - time.Second},
		{name: "drifted longer", timeout: HTTPClientTimeout + time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := SnapshotClient{HTTP: &http.Client{Timeout: tc.timeout}, Endpoint: endpoint}
			err := client.Validate()
			if tc.valid && err != nil {
				t.Fatalf("SnapshotClient.Validate() error = %v", err)
			}
			if !tc.valid && !errors.Is(err, ErrContract) {
				t.Fatalf("SnapshotClient.Validate() error = %v, want %v", err, ErrContract)
			}
		})
	}
}

func TestSnapshotClientOGSPreservesStrictDecodeIdentity(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_, _ = w.Write([]byte(`{"unexpected":true}`))
	}))
	t.Cleanup(server.Close)
	_, err := validSnapshotClient(t, server).Snapshot(t.Context(), validProjectSnapshot(t).Project)
	var httpErr HTTPError
	if !errors.As(err, &httpErr) || !errors.Is(err, core.ErrJSONContract) || !errors.Is(err, ErrContract) {
		t.Fatalf("SnapshotClient.Snapshot() error = %v, want HTTPError wrapping ErrJSONContract and ErrContract", err)
	}
}

func TestOffgridRunStatsPathUsesPublicAPIVersionContract(t *testing.T) {
	t.Parallel()
	want := "/" + core.APIVersionToken + "/peachfuzz/stats"
	if OffgridRunStatsPath != want {
		t.Fatalf("OffgridRunStatsPath = %q, want %q", OffgridRunStatsPath, want)
	}
	if core.APIVersionToken == core.ContractVersionToken {
		t.Fatalf("public API version %q must remain independent from schema generation %q", core.APIVersionToken, core.ContractVersionToken)
	}
}

func TestOffgridRunEvidenceUploadPathUsesPublicAPIVersionContract(t *testing.T) {
	t.Parallel()
	want := "/" + core.APIVersionToken + "/peachfuzz/evidence/uploads"
	if OffgridRunEvidenceUploadPath != want {
		t.Fatalf("OffgridRunEvidenceUploadPath = %q, want %q", OffgridRunEvidenceUploadPath, want)
	}
	endpoint, err := OffgridRunEvidenceUploadEndpoint()
	if err != nil {
		t.Fatalf("OffgridRunEvidenceUploadEndpoint() error = %v, want nil", err)
	}
	if endpoint.String() != core.OffgridAPIBaseURL+want {
		t.Fatalf("OffgridRunEvidenceUploadEndpoint() = %q, want %q", endpoint.String(), core.OffgridAPIBaseURL+want)
	}
}

func TestOffgridRunEvidenceFoldPathsUsePublicAPIVersionContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		endpoint func() (core.APIEndpoint, error)
		path     string
		want     string
	}{
		{path: OffgridRunEvidenceFinalizePath, want: "/" + core.APIVersionToken + "/peachfuzz/evidence/finalize", endpoint: OffgridRunEvidenceFinalizeEndpoint},
		{path: OffgridRunEvidenceMaterializePath, want: "/" + core.APIVersionToken + "/peachfuzz/evidence/materialize", endpoint: OffgridRunEvidenceMaterializeEndpoint},
	} {
		if tc.path != tc.want {
			t.Fatalf("fold path = %q, want %q", tc.path, tc.want)
		}
		endpoint, err := tc.endpoint()
		if err != nil || endpoint.String() != core.OffgridAPIBaseURL+tc.want {
			t.Fatalf("fold endpoint = %q, error=%v, want %q", endpoint.String(), err, core.OffgridAPIBaseURL+tc.want)
		}
	}
}

func validSnapshotClient(t *testing.T, server *httptest.Server) SnapshotClient {
	t.Helper()
	endpoint, err := core.APIEndpointForBaseURL(server.URL, OffgridRunStatsPath)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	client.Timeout = HTTPClientTimeout
	return SnapshotClient{HTTP: client, Endpoint: endpoint}
}

func writeClientEnvelope[T core.APIBody](t *testing.T, w http.ResponseWriter, body T) {
	t.Helper()
	envelope := core.APIEnvelope[T]{Data: &body, RequestID: core.NewAPIRequestID("peachfuzz-client-test")}
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		t.Errorf("Encode() error = %v", err)
	}
}
