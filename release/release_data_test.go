package release

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/workloadidentity"
)

const (
	releaseDataTestCommit         = "cccccccccccccccccccccccccccccccccccccccc"
	releaseDataTestRequestID      = "1111111111111111111111111111111111111111111111111111111111111111"
	releaseDataAlternateRequestID = "2222222222222222222222222222222222222222222222222222222222222222"
	releaseDataTestToken          = "header.payload.signature"
	releaseDataTestResponsePath   = "/release-data"
)

func TestReleaseDataContractsOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	valid := validReleaseDataResponse(t)
	alternateGarbleVersion, err := ParseToolVersion("v0.15.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mutate  func(*ReleaseDataResponse)
		name    string
		wantErr bool
	}{
		{name: "valid"},
		{name: "request schema", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.Request.Schema = core.SchemaReleaseDataResponse }},
		{name: "response schema", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.Schema = core.SchemaReleaseDataRequest }},
		{name: "request product", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.Request.Product = core.ProductWitness }},
		{name: "release signing key", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.ReleaseSigningKey.PublicKeyHex = v.ServerPublicKey }},
		{name: "custody seed", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.GarbleCustodySeed = core.GarbleCustodySeed{} }},
		{name: "server public key", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.ServerPublicKey = core.Ed25519PublicKeyHex{} }},
		{name: "garble version", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.GarbleVersion = ToolVersion{} }},
		{name: "unpinned garble version", wantErr: true, mutate: func(v *ReleaseDataResponse) { v.GarbleVersion = alternateGarbleVersion }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := valid
			if tc.mutate != nil {
				tc.mutate(&value)
			}
			err := value.Validate()
			if tc.wantErr && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ReleaseDataResponse.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ReleaseDataResponse.Validate() error = %v", err)
			}
		})
	}
}

type staticReleaseTokenSource struct{ token workloadidentity.Token }

func (s staticReleaseTokenSource) Validate() error { return s.token.Validate() }
func (s staticReleaseTokenSource) Token(context.Context) (workloadidentity.Token, error) {
	return s.token, nil
}

func TestReleaseDataSourceOGSComposesIdentityAndProductEndpoint(t *testing.T) {
	t.Parallel()
	response := validReleaseDataResponse(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != OffgridBugReleaseDataPath {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if got := request.Header.Get(core.HTTPHeaderAccept); got != core.HTTPContentTypeJSON {
			t.Errorf("accept = %q, want %q", got, core.HTTPContentTypeJSON)
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		envelope := core.APIEnvelope[ReleaseDataResponse]{Data: &response, RequestID: core.NewAPIRequestID(releaseDataTestRequestID)}
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Errorf("Encode() error = %v", err)
		}
	}))
	t.Cleanup(server.Close)
	endpoints, err := BugDeployEndpointsForBaseURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	token, err := workloadidentity.ParseToken(releaseDataTestToken)
	if err != nil {
		t.Fatal(err)
	}
	source := ReleaseDataSource{HTTP: server.Client(), Identity: staticReleaseTokenSource{token: token}, Endpoints: endpoints}
	got, err := source.Fetch(t.Context(), response.Request)
	if err != nil {
		t.Fatalf("ReleaseDataSource.Fetch() error = %v", err)
	}
	if got.Request != response.Request {
		t.Fatalf("ReleaseDataSource.Fetch() request binding changed")
	}
}

func TestReleaseDataClientOGSExactRequestAndResponseBinding(t *testing.T) {
	t.Parallel()
	response := validReleaseDataResponse(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != releaseDataTestResponsePath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		wantAuthorization := core.HTTPAuthorizationBearerPrefix + releaseDataTestToken
		if got := r.Header.Get(core.HTTPHeaderAuthorization); got != wantAuthorization {
			t.Errorf("authorization = %q, want %q", got, wantAuthorization)
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, core.StrictJSONMaxBytes+1))
		if err != nil {
			t.Errorf("ReadAll() error = %v", err)
		}
		request, err := core.DecodeStrictJSON[ReleaseDataRequest](body)
		if err != nil {
			t.Errorf("DecodeStrictJSON() error = %v", err)
		}
		if request != response.Request {
			t.Errorf("request = %#v, want %#v", request, response.Request)
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		envelope := core.APIEnvelope[ReleaseDataResponse]{Data: &response, RequestID: core.NewAPIRequestID(releaseDataTestRequestID)}
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Errorf("Encode() error = %v", err)
		}
	}))
	t.Cleanup(server.Close)
	endpoint, err := core.APIEndpointForBaseURL(server.URL, releaseDataTestResponsePath)
	if err != nil {
		t.Fatal(err)
	}
	token, err := workloadidentity.ParseToken(releaseDataTestToken)
	if err != nil {
		t.Fatal(err)
	}
	client := ReleaseDataClient{HTTP: server.Client(), Endpoint: endpoint, Token: token, Product: core.ProductBug}
	got, err := client.Fetch(t.Context(), response.Request)
	if err != nil {
		t.Fatalf("ReleaseDataClient.Fetch() error = %v", err)
	}
	if got.Request != response.Request || got.ReleaseSigningKey != response.ReleaseSigningKey {
		t.Fatalf("ReleaseDataClient.Fetch() response does not match broker response")
	}
}

func TestReleaseDataClientOGSRefusesRedirect(t *testing.T) {
	t.Parallel()
	redirectTargetReached := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirectTargetReached = true
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirect.Close)
	endpoint, err := core.APIEndpointForBaseURL(redirect.URL, releaseDataTestResponsePath)
	if err != nil {
		t.Fatal(err)
	}
	token, err := workloadidentity.ParseToken(releaseDataTestToken)
	if err != nil {
		t.Fatal(err)
	}
	response := validReleaseDataResponse(t)
	client := ReleaseDataClient{HTTP: redirect.Client(), Endpoint: endpoint, Token: token, Product: core.ProductBug}
	if _, err := client.Fetch(t.Context(), response.Request); err == nil {
		t.Fatal("ReleaseDataClient.Fetch() error = nil, want redirect refusal")
	}
	if redirectTargetReached {
		t.Fatalf("redirect target reached = %t, want false", redirectTargetReached)
	}
}

func TestReleaseDataClientOGSRejectsResponseRequestSubstitution(t *testing.T) {
	t.Parallel()
	response := validReleaseDataResponse(t)
	alternateRequestID := mustDeployRequestID(t, releaseDataAlternateRequestID)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response.Request.RequestID = alternateRequestID
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		envelope := core.APIEnvelope[ReleaseDataResponse]{Data: &response, RequestID: core.NewAPIRequestID(releaseDataTestRequestID)}
		if err := json.NewEncoder(w).Encode(envelope); err != nil {
			t.Errorf("Encode() error = %v", err)
		}
	}))
	t.Cleanup(server.Close)
	endpoint, err := core.APIEndpointForBaseURL(server.URL, releaseDataTestResponsePath)
	if err != nil {
		t.Fatal(err)
	}
	token, err := workloadidentity.ParseToken(releaseDataTestToken)
	if err != nil {
		t.Fatal(err)
	}
	client := ReleaseDataClient{HTTP: server.Client(), Endpoint: endpoint, Token: token, Product: core.ProductBug}
	if _, err := client.Fetch(t.Context(), validReleaseDataResponse(t).Request); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("ReleaseDataClient.Fetch() error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func mustDeployRequestID(t *testing.T, value string) DeployRequestID {
	t.Helper()
	requestID, err := ParseDeployRequestID(value)
	if err != nil {
		t.Fatal(err)
	}
	return requestID
}

func validReleaseDataResponse(t *testing.T) ReleaseDataResponse {
	t.Helper()
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := core.ParseBuildCommit(releaseDataTestCommit)
	if err != nil {
		t.Fatal(err)
	}
	requestID, err := ParseDeployRequestID(releaseDataTestRequestID)
	if err != nil {
		t.Fatal(err)
	}
	request, err := BuildReleaseDataRequest(core.ProductBug, version, commit, requestID)
	if err != nil {
		t.Fatal(err)
	}
	releaseKey := generatedSigningKey(t, 1)
	serverKey := generatedSigningKey(t, 2)
	garbleVersion, err := GarbleToolVersion()
	if err != nil {
		t.Fatal(err)
	}
	response := ReleaseDataResponse{
		Request: request, ReleaseSigningKey: releaseKey, GarbleCustodySeed: mustCustodySeed(t),
		ServerPublicKey: serverKey.PublicKeyHex, GarbleVersion: garbleVersion, Schema: core.SchemaReleaseDataResponse,
	}
	if err := response.Validate(); err != nil {
		t.Fatal(err)
	}
	return response
}

func generatedSigningKey(t *testing.T, seedByte byte) core.GeneratedSigningKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = seedByte + byte(index)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public, err := core.NewEd25519PublicKeyHex(private.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	key := core.GeneratedSigningKey{PrivateKeyBase64: base64.StdEncoding.EncodeToString(private), PublicKeyHex: public}
	if err := key.Validate(); err != nil {
		t.Fatal(err)
	}
	return key
}
