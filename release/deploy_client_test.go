package release

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestDeployClientPrepareHostileHTTPTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate      func(*testing.T, *deployTransportChain)
		body        func(*testing.T, deployTransportChain) []byte
		name        string
		contentType string
		status      int
		wantAPI     bool
		accept      bool
	}{
		{name: "valid signed response", contentType: core.HTTPContentTypeJSON, status: core.HTTPStatusOK, body: prepareSuccessEnvelope, accept: true},
		{name: "foreign plan signature", contentType: core.HTTPContentTypeJSON, status: core.HTTPStatusOK, body: prepareSuccessEnvelope, mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.prepareResponse.Plan = signDeployBody(t, chain.foreignSigner, chain.prepareResponse.Plan.Body)
		}},
		{name: "wrong media type", contentType: "text/plain", status: core.HTTPStatusOK, body: prepareSuccessEnvelope},
		{name: "trailing json", contentType: core.HTTPContentTypeJSON, status: core.HTTPStatusOK, body: func(t *testing.T, chain deployTransportChain) []byte {
			return append(prepareSuccessEnvelope(t, chain), []byte("{}")...)
		}},
		{name: "duplicate envelope field", contentType: core.HTTPContentTypeJSON, status: core.HTTPStatusOK, body: duplicatePrepareEnvelopeField},
		{name: "oversized response", contentType: core.HTTPContentTypeJSON, status: core.HTTPStatusOK, body: func(*testing.T, deployTransportChain) []byte {
			return bytes.Repeat([]byte{' '}, core.StrictJSONMaxBytes+1)
		}},
		{name: "typed api refusal", contentType: core.HTTPContentTypeJSON, status: http.StatusBadRequest, body: prepareFailureEnvelope, wantAPI: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validDeployTransportChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != OffgridBugDeployPreparePath {
					t.Errorf("request = %s %s, want POST %s", request.Method, request.URL.Path, OffgridBugDeployPreparePath)
				}
				if got := request.Header.Get(core.HTTPHeaderContentType); got != core.HTTPContentTypeJSON {
					t.Errorf("content type = %q, want %q", got, core.HTTPContentTypeJSON)
				}
				w.Header().Set(core.HTTPHeaderContentType, tc.contentType)
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.body(t, chain))
			}))
			defer server.Close()
			endpoints, err := BugDeployEndpointsForBaseURL(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := DeployClient{
				HTTP: server.Client(), Endpoints: endpoints,
				ReleaseKeys: chain.releaseSigner.keyring, ServerKeys: chain.serverSigner.keyring,
			}
			_, err = client.Prepare(context.Background(), chain.prepareRequest)
			if tc.accept {
				if err != nil {
					t.Fatalf("DeployClient.Prepare() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployClient.Prepare() error = %v, want release/foundation contract", err)
			}
			var apiErr DeployAPIError
			if got := errors.As(err, &apiErr); got != tc.wantAPI {
				t.Fatalf("errors.As(DeployAPIError) = %v, want %v; error=%v", got, tc.wantAPI, err)
			}
		})
	}
}

func TestDeployClientFinalizeHostileHTTPTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *deployTransportChain)
		body   func(*testing.T, deployTransportChain) []byte
		name   string
		accept bool
	}{
		{name: "valid signed response", body: finalizeSuccessEnvelope, accept: true},
		{name: "foreign receipt signature", body: finalizeSuccessEnvelope, mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Receipt = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Receipt.Body)
		}},
		{name: "malformed envelope", body: func(*testing.T, deployTransportChain) []byte { return []byte(`{`) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validDeployTransportChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != OffgridBugDeployFinalizePath {
					t.Errorf("request = %s %s, want POST %s", request.Method, request.URL.Path, OffgridBugDeployFinalizePath)
				}
				w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
				_, _ = w.Write(tc.body(t, chain))
			}))
			defer server.Close()
			endpoints, err := BugDeployEndpointsForBaseURL(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := DeployClient{
				HTTP: server.Client(), Endpoints: endpoints,
				ReleaseKeys: chain.releaseSigner.keyring, ServerKeys: chain.serverSigner.keyring,
			}
			_, err = client.Finalize(context.Background(), chain.finalizeRequest)
			if tc.accept {
				if err != nil {
					t.Fatalf("DeployClient.Finalize() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployClient.Finalize() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func prepareSuccessEnvelope(t *testing.T, chain deployTransportChain) []byte {
	t.Helper()
	envelope := core.APIEnvelope[DeployPrepareResponse]{
		Data: &chain.prepareResponse, RequestID: core.NewAPIRequestID("deploy-http-test"),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func prepareFailureEnvelope(t *testing.T, _ deployTransportChain) []byte {
	t.Helper()
	envelope := core.APIEnvelope[DeployPrepareResponse]{
		Error:     &core.APIErrorBody{Code: core.APICodeInvalidInput, Message: "deploy refused"},
		RequestID: core.NewAPIRequestID("deploy-http-refusal"),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func finalizeSuccessEnvelope(t *testing.T, chain deployTransportChain) []byte {
	t.Helper()
	envelope := core.APIEnvelope[DeployFinalizeResponse]{
		Data: &chain.finalizeResponse, RequestID: core.NewAPIRequestID("deploy-finalize-http-test"),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func duplicatePrepareEnvelopeField(t *testing.T, chain deployTransportChain) []byte {
	t.Helper()
	data := prepareSuccessEnvelope(t, chain)
	if len(data) < 2 || data[len(data)-1] != '}' {
		t.Fatalf("prepare envelope = %q, want JSON object", data)
	}
	duplicate := append([]byte(nil), data[:len(data)-1]...)
	duplicate = append(duplicate, []byte(`,"request_id":"deploy-http-duplicate"}`)...)
	return duplicate
}
