package peachfuzz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/workloadidentity"
)

const clientTestToken = "header.payload.signature"

type staticTokenSource struct{ token workloadidentity.Token }

func (s staticTokenSource) Validate() error { return s.token.Validate() }

func (s staticTokenSource) Token(context.Context) (workloadidentity.Token, error) {
	return s.token, nil
}

func TestClientRecordOGSExactTypedBoundary(t *testing.T) {
	t.Parallel()
	stats := validRunStats(t)
	receipt := RunStatsReceipt{RunID: stats.RunID, Disposition: RecordDispositionRecorded}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", request.Method, http.MethodPost)
		}
		if got := request.Header.Get(core.HTTPHeaderContentType); got != core.HTTPContentTypeJSON {
			t.Errorf("content type = %q, want %q", got, core.HTTPContentTypeJSON)
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writeClientEnvelope(t, w, receipt)
	}))
	t.Cleanup(server.Close)

	got, err := validRecordClient(t, server).Record(t.Context(), stats)
	if err != nil {
		t.Fatalf("RecordClient.Record() error = %v", err)
	}
	if got != receipt {
		t.Fatalf("RecordClient.Record() = %#v, want %#v", got, receipt)
	}
}

func TestClientRecordOGSRejectsSubstitutedRunIdentity(t *testing.T) {
	t.Parallel()
	stats := validRunStats(t)
	otherID, err := ParseRunID(strings.Repeat("d", RunIDTextBytes))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writeClientEnvelope(t, w, RunStatsReceipt{RunID: otherID, Disposition: RecordDispositionRecorded})
	}))
	t.Cleanup(server.Close)

	_, err = validRecordClient(t, server).Record(t.Context(), stats)
	if !errors.Is(err, ErrContract) {
		t.Fatalf("RecordClient.Record() error = %v, want %v", err, ErrContract)
	}
}

func TestClientRecordOGSRejectsTrailingProtocolValue(t *testing.T) {
	t.Parallel()
	stats := validRunStats(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writeClientEnvelope(t, w, RunStatsReceipt{RunID: stats.RunID, Disposition: RecordDispositionRecorded})
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(server.Close)

	_, err := validRecordClient(t, server).Record(t.Context(), stats)
	var transportError HTTPError
	if !errors.As(err, &transportError) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RecordClient.Record() error = %v, want typed unavailable HTTPError", err)
	}
}

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

func validRecordClient(t *testing.T, server *httptest.Server) RecordClient {
	t.Helper()
	endpoint, err := core.APIEndpointForBaseURL(server.URL, OffgridRunStatsPath)
	if err != nil {
		t.Fatal(err)
	}
	token, err := workloadidentity.ParseToken(clientTestToken)
	if err != nil {
		t.Fatal(err)
	}
	return RecordClient{HTTP: server.Client(), Endpoint: endpoint, Identity: staticTokenSource{token: token}}
}

func validSnapshotClient(t *testing.T, server *httptest.Server) SnapshotClient {
	t.Helper()
	endpoint, err := core.APIEndpointForBaseURL(server.URL, OffgridRunStatsPath)
	if err != nil {
		t.Fatal(err)
	}
	return SnapshotClient{HTTP: server.Client(), Endpoint: endpoint}
}

func writeClientEnvelope[T core.APIBody](t *testing.T, w http.ResponseWriter, body T) {
	t.Helper()
	envelope := core.APIEnvelope[T]{Data: &body, RequestID: core.NewAPIRequestID("peachfuzz-client-test")}
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		t.Errorf("Encode() error = %v", err)
	}
}
