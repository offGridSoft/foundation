package custody

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func custodyTestEndpoints(t testFatalHelper, baseURL string) Endpoints {
	t.Helper()
	endpoints, err := WitnessCustodyEndpointsForBaseURL(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	return endpoints
}

func custodyEnvelopeJSON[T core.Validatable](t testFatalHelper, data T) []byte {
	t.Helper()
	body, err := json.Marshal(core.APIEnvelope[T]{Data: &data, RequestID: core.NewAPIRequestID("req-1")})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func custodyFailureEnvelopeJSON[T core.Validatable](t testFatalHelper, code core.APICode, message string) []byte {
	t.Helper()
	body, err := json.Marshal(core.APIEnvelope[T]{
		Error:     &core.APIErrorBody{Code: code, Message: message},
		RequestID: core.NewAPIRequestID("req-1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func writeCustodyJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func validUploadRequiredResponse(t testFatalHelper) SessionOpenResponse {
	t.Helper()
	return SessionOpenResponse{
		Schema:     core.SchemaCustodySessionOpenResponse,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Upload: &SessionUploadGrant{
			Session:   mustSessionID(t),
			Targets:   validUploadTargets(t),
			Retention: mustRetention(),
			ExpiresAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		},
		Disposition: SessionOpenDispositionUploadRequired,
	}
}

func validFinalizeRequest(t testFatalHelper) FinalizeRequest {
	t.Helper()
	return FinalizeRequest{
		Schema:     core.SchemaCustodyFinalizeRequest,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Session:    mustSessionID(t),
		Objects:    mustUploadedObjects(t),
	}
}

func witnessObjectPathFor(t testFatalHelper, artifact string) ObjectPath {
	t.Helper()
	object, err := ParseObjectPath(core.WitnessCustodyPathRoot + "/" + mustCustomerID(t).String() + "/2026/07/" + mustBundleRoot(t).String() + "/" + artifact)
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func TestCustodyClientValidateFailClosedTable(t *testing.T) {
	t.Parallel()
	_, keyring := verifiedSignedReceipt(t)
	endpoints := custodyTestEndpoints(t, "https://api.example.com")
	cases := []struct {
		name   string
		client Client
	}{
		{name: "nil http client is rejected", client: Client{ServerKeys: keyring, Endpoints: endpoints}},
		{name: "zero endpoints are rejected", client: Client{HTTP: http.DefaultClient, ServerKeys: keyring}},
		{name: "duplicate endpoint routes are rejected", client: Client{HTTP: http.DefaultClient, ServerKeys: keyring, Endpoints: Endpoints{Open: endpoints.Open, Finalize: endpoints.Open, Download: endpoints.Download}}},
		{name: "empty server keyring is rejected", client: Client{HTTP: http.DefaultClient, Endpoints: endpoints}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.client.Validate(); !errors.Is(err, core.ErrCustodyContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Validate() error = %v, want custody/foundation contract violation", err)
			}
			if _, err := tc.client.OpenSession(context.Background(), validOpenRequest(t)); err == nil {
				t.Fatal("OpenSession() error = nil, want fail-closed client rejection")
			}
		})
	}
}

func TestCustodyErrorsUnwrapToCustodyContract(t *testing.T) {
	t.Parallel()
	httpErr := CustodyHTTPError{StatusCode: 502, Cause: io.ErrUnexpectedEOF}
	if !errors.Is(httpErr, core.ErrCustodyContract) || !errors.Is(httpErr, io.ErrUnexpectedEOF) {
		t.Fatalf("CustodyHTTPError.Unwrap() = %v, want custody contract and cause", httpErr)
	}
	apiErr := CustodyAPIError{StatusCode: 403, Body: core.APIErrorBody{Code: core.APICodeForbidden, Message: "no"}}
	if !errors.Is(apiErr, core.ErrCustodyContract) {
		t.Fatalf("CustodyAPIError.Unwrap() = %v, want custody contract", apiErr)
	}
}

func TestOpenSessionUploadRequiredBindsExactWireExchange(t *testing.T) {
	t.Parallel()
	request := validOpenRequest(t)
	wantBody, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	_, keyring := verifiedSignedReceipt(t)
	response := validUploadRequiredResponse(t)
	server := httptest.NewServer(openExchangeHandler(t, wantBody, custodyEnvelopeJSON(t, response)))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	got, err := client.OpenSession(context.Background(), request)
	if err != nil {
		t.Fatalf("OpenSession() error = %v, want nil", err)
	}
	if got.Disposition != SessionOpenDispositionUploadRequired || got.Upload == nil {
		t.Fatalf("OpenSession() disposition = %v upload=%v, want upload_required grant", got.Disposition, got.Upload)
	}
	if got.Upload.Session != response.Upload.Session || len(got.Upload.Targets) != 1 {
		t.Fatalf("OpenSession() grant = %+v, want session %v with one target", got.Upload, response.Upload.Session)
	}
}

func openExchangeHandler(t *testing.T, wantBody, responseBody []byte) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != OffgridWitnessCustodyOpenPath {
			t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, OffgridWitnessCustodyOpenPath)
		}
		if got := r.Header.Get(core.HTTPHeaderContentType); got != core.HTTPContentTypeJSON {
			t.Errorf("content type = %q, want %q", got, core.HTTPContentTypeJSON)
		}
		gotBody, readErr := io.ReadAll(r.Body)
		if readErr != nil || !bytes.Equal(gotBody, wantBody) {
			t.Errorf("request body = %s (err %v), want %s", gotBody, readErr, wantBody)
		}
		writeCustodyJSON(w, http.StatusOK, responseBody)
	}
}

func TestOpenSessionReceiptReusedVerifiesServerSignature(t *testing.T) {
	t.Parallel()
	signed, keyring := verifiedSignedReceipt(t)
	response := SessionOpenResponse{
		Schema:          core.SchemaCustodySessionOpenResponse,
		Customer:        signed.Body.Customer,
		BundleRoot:      signed.Body.BundleRoot,
		ExistingReceipt: &signed,
		Disposition:     SessionOpenDispositionReceiptReused,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, response))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	got, err := client.OpenSession(context.Background(), validOpenRequest(t))
	if err != nil {
		t.Fatalf("OpenSession() error = %v, want nil", err)
	}
	if got.Disposition != SessionOpenDispositionReceiptReused || got.ExistingReceipt == nil {
		t.Fatalf("OpenSession() = %+v, want reused signed receipt", got)
	}
	if got.ExistingReceipt.Signature != signed.Signature {
		t.Fatalf("reused receipt signature = %v, want %v", got.ExistingReceipt.Signature, signed.Signature)
	}
}

func TestOpenSessionRejectsInvalidRequestWithoutHTTPCall(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	_, keyring := verifiedSignedReceipt(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, validUploadRequiredResponse(t)))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	request := validOpenRequest(t)
	request.Schema = core.SchemaCustodyFinalizeRequest
	if _, err := client.OpenSession(context.Background(), request); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("OpenSession() error = %v, want %v", err, core.ErrCustodyContract)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("HTTP calls = %d, want 0 for invalid request", got)
	}
}

func TestOpenSessionHostileTransportTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		handler     func(t *testing.T, w http.ResponseWriter)
		makeContext func() (context.Context, context.CancelFunc)
		name        string
		wantAPIErr  bool
	}{
		{
			name: "forbidden failure envelope surfaces typed api error",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusForbidden, custodyFailureEnvelopeJSON[SessionOpenResponse](t, core.APICodeForbidden, "custody standing refused"))
			},
			wantAPIErr: true,
		},
		{
			name: "server fault failure envelope surfaces typed api error",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusInternalServerError, custodyFailureEnvelopeJSON[SessionOpenResponse](t, core.APICodeInternal, "ledger unavailable"))
			},
			wantAPIErr: true,
		},
		{
			name: "wrong content type is rejected",
			handler: func(t *testing.T, w http.ResponseWriter) {
				w.Header().Set(core.HTTPHeaderContentType, "text/plain")
				_, _ = w.Write(custodyEnvelopeJSON(t, validUploadRequiredResponse(t)))
			},
		},
		{
			name: "empty body is rejected",
			handler: func(_ *testing.T, w http.ResponseWriter) {
				w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
			},
		},
		{
			name: "oversized body is rejected",
			handler: func(_ *testing.T, w http.ResponseWriter) {
				w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
				_, _ = w.Write(bytes.Repeat([]byte("a"), core.StrictJSONMaxBytes+1))
			},
		},
		{
			name: "garbage json is rejected",
			handler: func(_ *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusOK, []byte(`{"data":`))
			},
		},
		{
			name: "success status with error-only envelope is rejected",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusOK, custodyFailureEnvelopeJSON[SessionOpenResponse](t, core.APICodeInternal, "half envelope"))
			},
		},
		{
			name: "failure status with data-only envelope is rejected",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusForbidden, custodyEnvelopeJSON(t, validUploadRequiredResponse(t)))
			},
		},
		{
			name: "nil context fails closed before transport",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, validUploadRequiredResponse(t)))
			},
			makeContext: func() (context.Context, context.CancelFunc) { return nil, func() {} },
		},
		{
			name: "canceled context aborts the exchange",
			handler: func(t *testing.T, w http.ResponseWriter) {
				writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, validUploadRequiredResponse(t)))
			},
			makeContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, keyring := verifiedSignedReceipt(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				tc.handler(t, w)
			}))
			defer server.Close()
			client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
			ctx := context.Context(context.Background())
			if tc.makeContext != nil {
				madeCtx, cancel := tc.makeContext()
				defer cancel()
				ctx = madeCtx
			}
			_, err := client.OpenSession(ctx, validOpenRequest(t))
			if !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("OpenSession() error = %v, want %v", err, core.ErrCustodyContract)
			}
			var apiErr CustodyAPIError
			if gotAPI := errors.As(err, &apiErr); gotAPI != tc.wantAPIErr {
				t.Fatalf("errors.As(CustodyAPIError) = %v, want %v (err %v)", gotAPI, tc.wantAPIErr, err)
			}
		})
	}
}

func TestOpenSessionHostileResponseBindingTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mutate func(t *testing.T, response *SessionOpenResponse)
		name   string
	}{
		{
			name: "customer drift is rejected",
			mutate: func(t *testing.T, response *SessionOpenResponse) {
				foreign, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZ0")
				if err != nil {
					t.Fatal(err)
				}
				response.Customer = foreign
			},
		},
		{
			name: "bundle root drift is rejected",
			mutate: func(t *testing.T, response *SessionOpenResponse) {
				response.BundleRoot = mustBLAKE3(t, "0")
			},
		},
		{
			name: "grant target for foreign artifact is rejected",
			mutate: func(t *testing.T, response *SessionOpenResponse) {
				response.Upload.Targets[0].Artifact = mustArtifactNameValue(t, "bundle2.tar")
				response.Upload.Targets[0].Object = witnessObjectPathFor(t, "bundle2.tar")
			},
		},
		{
			name: "reused receipt with forged signature is rejected",
			mutate: func(t *testing.T, response *SessionOpenResponse) {
				signed := mustSignedReceipt(t)
				keyID, _, _ := receiptSigningKey(t)
				signed.KeyID = keyID
				*response = SessionOpenResponse{
					Schema:          core.SchemaCustodySessionOpenResponse,
					Customer:        signed.Body.Customer,
					BundleRoot:      signed.Body.BundleRoot,
					ExistingReceipt: &signed,
					Disposition:     SessionOpenDispositionReceiptReused,
				}
			},
		},
		{
			name: "reused receipt signed by unknown key is rejected",
			mutate: func(t *testing.T, response *SessionOpenResponse) {
				signed := foreignSignedReceipt(t)
				*response = SessionOpenResponse{
					Schema:          core.SchemaCustodySessionOpenResponse,
					Customer:        signed.Body.Customer,
					BundleRoot:      signed.Body.BundleRoot,
					ExistingReceipt: &signed,
					Disposition:     SessionOpenDispositionReceiptReused,
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, keyring := verifiedSignedReceipt(t)
			response := validUploadRequiredResponse(t)
			tc.mutate(t, &response)
			body, err := json.Marshal(core.APIEnvelope[SessionOpenResponse]{Data: &response, RequestID: core.NewAPIRequestID("req-1")})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCustodyJSON(w, http.StatusOK, body)
			}))
			defer server.Close()
			client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
			if _, err := client.OpenSession(context.Background(), validOpenRequest(t)); !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("OpenSession() error = %v, want %v", err, core.ErrFoundationContract)
			}
		})
	}
}

func foreignSignedReceipt(t testFatalHelper) core.Signed[ReceiptBody] {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	keyID, err := core.ParseSigningKeyID("custody-foreign-key-2026")
	if err != nil {
		t.Fatal(err)
	}
	body := validReceipt(t)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[ReceiptBody]{Body: body, KeyID: keyID, Signature: signature}
}

func TestFinalizeReturnsBoundSignedReceipt(t *testing.T) {
	t.Parallel()
	signed, keyring := verifiedSignedReceipt(t)
	request := validFinalizeRequest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != OffgridWitnessCustodyFinalizePath {
			t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, OffgridWitnessCustodyFinalizePath)
		}
		writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, signed))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	got, err := client.Finalize(context.Background(), request)
	if err != nil {
		t.Fatalf("Finalize() error = %v, want nil", err)
	}
	if got.Body.ReceiptID != signed.Body.ReceiptID || got.Signature != signed.Signature {
		t.Fatalf("Finalize() receipt = %v/%v, want %v/%v", got.Body.ReceiptID, got.Signature, signed.Body.ReceiptID, signed.Signature)
	}
}

func TestFinalizeHostileReceiptTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mutateRequest func(t *testing.T, request *FinalizeRequest)
		respond       func(t *testing.T) core.Signed[ReceiptBody]
		name          string
	}{
		{
			name: "forged receipt signature is rejected",
			respond: func(t *testing.T) core.Signed[ReceiptBody] {
				signed, _ := verifiedSignedReceipt(t)
				forged, err := core.ParseEd25519SignatureHex(strings.Repeat("b", 128))
				if err != nil {
					t.Fatal(err)
				}
				signed.Signature = forged
				return signed
			},
		},
		{
			name:    "foreign-key receipt is rejected",
			respond: func(t *testing.T) core.Signed[ReceiptBody] { return foreignSignedReceipt(t) },
		},
		{
			name: "session drift between request and receipt is rejected",
			mutateRequest: func(t *testing.T, request *FinalizeRequest) {
				session, err := ParseSessionID("01J00000000000000000000009")
				if err != nil {
					t.Fatal(err)
				}
				request.Session = session
			},
			respond: func(t *testing.T) core.Signed[ReceiptBody] {
				signed, _ := verifiedSignedReceipt(t)
				return signed
			},
		},
		{
			name: "reported object drift is rejected",
			mutateRequest: func(t *testing.T, request *FinalizeRequest) {
				generation, err := ParseGeneration("1710000000000000999")
				if err != nil {
					t.Fatal(err)
				}
				request.Objects[0].Generation = generation
			},
			respond: func(t *testing.T) core.Signed[ReceiptBody] {
				signed, _ := verifiedSignedReceipt(t)
				return signed
			},
		},
		{
			name: "object count drift is rejected",
			mutateRequest: func(t *testing.T, request *FinalizeRequest) {
				second := mustUploadedObject(t)
				second.Artifact = mustArtifactNameValue(t, "bundle2.tar")
				second.Object = witnessObjectPathFor(t, "bundle2.tar")
				request.Objects = append(request.Objects, second)
			},
			respond: func(t *testing.T) core.Signed[ReceiptBody] {
				signed, _ := verifiedSignedReceipt(t)
				return signed
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, keyring := verifiedSignedReceipt(t)
			signed := tc.respond(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, signed))
			}))
			defer server.Close()
			client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
			request := validFinalizeRequest(t)
			if tc.mutateRequest != nil {
				tc.mutateRequest(t, &request)
			}
			if _, err := client.Finalize(context.Background(), request); !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Finalize() error = %v, want %v", err, core.ErrFoundationContract)
			}
		})
	}
}

func uploadFixture(t testFatalHelper, payload []byte, serverURL string) UploadArtifactInput {
	t.Helper()
	url, err := core.ParseSignedUploadURL(serverURL + "/offgrid-custody/bundle.tar?sig=abc")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	return UploadArtifactInput{
		Body: bytes.NewReader(payload),
		Target: UploadTarget{
			Artifact: mustArtifactNameValue(t, "bundle.tar"),
			Object:   mustObjectPath(t),
			URL:      url,
			Headers: []core.UploadHeader{
				{Name: core.HTTPHeaderContentType, Value: "application/octet-stream"},
				{Name: "x-goog-if-generation-match", Value: "0"},
			},
			Provider: core.StorageProviderGCS,
			Method:   core.UploadMethodSignedPUT,
		},
		Artifact: ArtifactDescriptor{
			Name:   mustArtifactNameValue(t, "bundle.tar"),
			SHA256: core.NewSHA256Hex(sum),
			BLAKE3: mustBLAKE3(t, "d"),
			Size:   core.NewByteCount(uint64(len(payload))),
		},
	}
}

func uploadTestClient(t testFatalHelper, server *httptest.Server) Client {
	t.Helper()
	_, keyring := verifiedSignedReceipt(t)
	return Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, "https://api.example.com")}
}

func TestUploadArtifactStreamsBytesAndBindsGeneration(t *testing.T) {
	t.Parallel()
	payload := []byte("custody bundle payload bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("upload method = %s, want %s", r.Method, http.MethodPut)
		}
		if got := r.Header.Get("x-goog-if-generation-match"); got != "0" {
			t.Errorf("create-only header = %q, want %q", got, "0")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(body, payload) {
			t.Errorf("streamed body = %q (err %v), want %q", body, err, payload)
		}
		w.Header().Set(GCSGenerationHeaderName, "1710000000000000")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := uploadTestClient(t, server)
	got, err := client.UploadArtifact(context.Background(), uploadFixture(t, payload, server.URL))
	if err != nil {
		t.Fatalf("UploadArtifact() error = %v, want nil", err)
	}
	want := UploadedObject{
		Artifact:   mustArtifactNameValue(t, "bundle.tar"),
		Object:     mustObjectPath(t),
		Generation: mustGeneration(t),
		SHA256:     core.NewSHA256Hex(sha256.Sum256(payload)),
		BLAKE3:     mustBLAKE3(t, "d"),
		Size:       core.NewByteCount(uint64(len(payload))),
		Provider:   core.StorageProviderGCS,
	}
	if got != want {
		t.Fatalf("UploadArtifact() = %+v, want %+v", got, want)
	}
}

func TestUploadArtifactHostileTable(t *testing.T) {
	t.Parallel()
	payload := []byte("custody bundle payload bytes")
	cases := []struct {
		mutate      func(t *testing.T, input *UploadArtifactInput)
		handler     func(w http.ResponseWriter, r *http.Request)
		makeContext func() (context.Context, context.CancelFunc)
		name        string
		wantStorage bool
	}{
		{
			name: "digest drift between stream and descriptor is refused",
			mutate: func(t *testing.T, input *UploadArtifactInput) {
				input.Artifact.SHA256 = mustSHA256(t, "c")
			},
			wantStorage: true,
		},
		{
			name: "truncated stream is refused",
			mutate: func(_ *testing.T, input *UploadArtifactInput) {
				input.Body = bytes.NewReader(payload[:len(payload)-3])
			},
		},
		{
			name: "oversized stream is refused",
			mutate: func(_ *testing.T, input *UploadArtifactInput) {
				input.Body = bytes.NewReader(append(append([]byte{}, payload...), "extra"...))
			},
		},
		{
			name: "storage rejection status is surfaced",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusForbidden)
			},
		},
		{
			name: "missing generation header is refused",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name: "hostile generation header is refused",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set(GCSGenerationHeaderName, "gen\x01eration")
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name:   "nil body reader fails closed",
			mutate: func(_ *testing.T, input *UploadArtifactInput) { input.Body = nil },
		},
		{
			name: "target artifact mismatch fails closed",
			mutate: func(t *testing.T, input *UploadArtifactInput) {
				input.Target.Artifact = mustArtifactNameValue(t, "bundle2.tar")
			},
		},
		{
			name: "resumable target method is refused by the signed-put client",
			mutate: func(_ *testing.T, input *UploadArtifactInput) {
				input.Target.Method = core.UploadMethodResumableURI
			},
		},
		{
			name:        "nil context fails closed",
			makeContext: func() (context.Context, context.CancelFunc) { return nil, func() {} },
		},
		{
			name: "canceled context aborts the transfer",
			makeContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := tc.handler
			if handler == nil {
				handler = func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					w.Header().Set(GCSGenerationHeaderName, "1710000000000000")
					w.WriteHeader(http.StatusOK)
				}
			}
			server := httptest.NewTLSServer(http.HandlerFunc(handler))
			defer server.Close()
			client := uploadTestClient(t, server)
			input := uploadFixture(t, payload, server.URL)
			if tc.mutate != nil {
				tc.mutate(t, &input)
			}
			ctx := context.Context(context.Background())
			if tc.makeContext != nil {
				madeCtx, cancel := tc.makeContext()
				defer cancel()
				ctx = madeCtx
			}
			_, err := client.UploadArtifact(ctx, input)
			if !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("UploadArtifact() error = %v, want %v", err, core.ErrCustodyContract)
			}
			if tc.wantStorage && !errors.Is(err, core.ErrStorageVerification) {
				t.Fatalf("UploadArtifact() error = %v, want %v", err, core.ErrStorageVerification)
			}
		})
	}
}
