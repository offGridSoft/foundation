package exchange

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type proxyBehavior uint8

const (
	proxyBehaviorHTMLBody proxyBehavior = iota + 1
	proxyBehaviorGzipEncoding
	proxyBehaviorUndecodableJSON
)

func TestSendJSONRetryableStatusWithNonProtocolBodyRetriesThroughProxy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantErr        error
		name           string
		proxyResponses uint64
		maxAttempts    uint64
		wantAttempts   uint64
		status         core.HTTPStatusCode
		behavior       proxyBehavior
		wantRetryClass bool
	}{
		{name: "http 503 html error page retries and reaches origin", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusServiceUnavailable, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 502 html error page retries and reaches origin", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusBadGateway, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 504 gzip encoded error retries and reaches origin", behavior: proxyBehaviorGzipEncoding, status: core.HTTPStatusGatewayTimeout, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 429 undecodable json body retries and reaches origin", behavior: proxyBehaviorUndecodableJSON, status: core.HTTPStatusTooManyRequests, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 408 html error page retries and reaches origin", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusRequestTimeout, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 500 html error page retries and reaches origin", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusInternalServerError, proxyResponses: 1, maxAttempts: 3, wantAttempts: 2},
		{name: "http 503 html forever exhausts exact retry budget", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusServiceUnavailable, proxyResponses: 99, maxAttempts: 2, wantAttempts: 2, wantErr: core.ErrExchangeRetryExhausted, wantRetryClass: true},
		{name: "http 404 html error page stays terminal on first attempt", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusNotFound, proxyResponses: 99, maxAttempts: 3, wantAttempts: 1, wantErr: core.ErrExchangeContentType},
		{name: "http 501 html error page stays terminal despite server class", behavior: proxyBehaviorHTMLBody, status: mustHTTPStatus(t, http.StatusNotImplemented), proxyResponses: 99, maxAttempts: 3, wantAttempts: 1, wantErr: core.ErrExchangeContentType},
		{name: "http 200 html body stays terminal on first attempt", behavior: proxyBehaviorHTMLBody, status: core.HTTPStatusOK, proxyResponses: 99, maxAttempts: 3, wantAttempts: 1, wantErr: core.ErrExchangeContentType},
		{name: "http 200 undecodable json body stays terminal on first attempt", behavior: proxyBehaviorUndecodableJSON, status: core.HTTPStatusOK, proxyResponses: 99, maxAttempts: 3, wantAttempts: 1, wantErr: core.ErrExchangeResponse},
		{name: "http 400 undecodable json body stays terminal on first attempt", behavior: proxyBehaviorUndecodableJSON, status: mustHTTPStatus(t, http.StatusBadRequest), proxyResponses: 99, maxAttempts: 3, wantAttempts: 1, wantErr: core.ErrExchangeResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Uint64
			server := httptest.NewServer(proxyThenOriginHandler(t, tc.behavior, tc.status, tc.proxyResponses, &requests))
			defer server.Close()

			client := mustTestClientRuntime(t, server.Client(), fixedClock{now: core.UnixNanoTimeFromInt64(1)}, fixedJitter{fraction: 1}, &recordingWaiter{})
			got, gotErr := SendJSON[receiveFixture, responseFixture](context.Background(), client, idempotentClientRequest(t, server.URL), clientTestPolicy(tc.maxAttempts))
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("SendJSON() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantRetryClass && !errors.Is(gotErr, core.ErrExchangeContentType) {
				t.Fatalf("SendJSON() exhausted error = %v, want cause %v preserved", gotErr, core.ErrExchangeContentType)
			}
			if got.Attempts != tc.wantAttempts {
				t.Fatalf("SendJSON() attempts = %d, want %d", got.Attempts, tc.wantAttempts)
			}
			if gotRequests := requests.Load(); gotRequests != tc.wantAttempts {
				t.Fatalf("server requests = %d, want %d", gotRequests, tc.wantAttempts)
			}
			if tc.wantErr == nil && (got.Status != core.HTTPStatusOK || got.Envelope.Data == nil || got.Envelope.Data.Value != "accepted") {
				t.Fatalf("SendJSON() recovered response = %+v, want accepted origin envelope", got)
			}
		})
	}
}

func proxyThenOriginHandler(t *testing.T, behavior proxyBehavior, status core.HTTPStatusCode, proxyResponses uint64, requests *atomic.Uint64) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) > proxyResponses {
			writeTypedNetworkResponse(t, writer, core.HTTPStatusOK)
			return
		}
		switch behavior {
		case proxyBehaviorHTMLBody:
			writeRawResponse(writer, status, "text/html", "<html>busy</html>")
		case proxyBehaviorGzipEncoding:
			writer.Header().Set(core.HTTPHeaderContentEncoding, "gzip")
			writeRawResponse(writer, status, core.HTTPContentTypeJSON, `{"request_id":"a","data":null,"error":null}`)
		case proxyBehaviorUndecodableJSON:
			writeRawResponse(writer, status, core.HTTPContentTypeJSON, "<html>disguised</html>")
		default:
			t.Errorf("unknown proxy behavior = %d", behavior)
		}
	})
}

type earlyRejectTransport struct {
	bodyBytesToRead int
	status          int
}

func (tr earlyRejectTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	buffer := make([]byte, tr.bodyBytesToRead)
	if _, err := io.ReadFull(request.Body, buffer); err != nil {
		return nil, err
	}
	if err := request.Body.Close(); err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: tr.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    request,
	}, nil
}

func TestSendStreamEarlyServerRejectionPreservesStatusError(t *testing.T) {
	t.Parallel()

	endpoint, err := core.ParseAPIEndpoint("https://api.offgridsoftware.ca/v1/upload")
	if err != nil {
		t.Fatalf("ParseAPIEndpoint() error = %v, want nil", err)
	}
	client := mustTestClient(t, &http.Client{Transport: earlyRejectTransport{bodyBytesToRead: 2, status: http.StatusForbidden}})
	_, gotErr := SendStream(t.Context(), client, StreamUploadRequest[core.APIEndpoint]{
		Target: endpoint, Body: strings.NewReader("artifact"), ContentLength: core.NewByteLength(8),
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt},
		ExpectedStatus: core.HTTPStatusOK,
	}, streamTestPolicy())
	statusErr, hasStatus := errors.AsType[StatusError](gotErr)
	if !hasStatus || statusErr.Status != mustHTTPStatus(t, http.StatusForbidden) {
		t.Fatalf("SendStream(early rejection) error = %v status=%+v/%v, want StatusError with 403 preserved", gotErr, statusErr, hasStatus)
	}
	if !errors.Is(gotErr, io.ErrUnexpectedEOF) || !errors.Is(gotErr, core.ErrExchangeRequest) {
		t.Fatalf("SendStream(early rejection) error = %v, want short-count facts %v and %v preserved alongside status", gotErr, io.ErrUnexpectedEOF, core.ErrExchangeRequest)
	}
}

func TestReceiveJSONRejectsRoutesWhoseMethodNeverCarriesABody(t *testing.T) {
	t.Parallel()

	policy := ServerPolicy{RequestBodyLimit: core.NewByteCount(receiveFixtureBodyLimit)}
	cases := []struct {
		wantErr error
		name    string
		method  string
		key     string
		route   core.HTTPRouteSemantics
	}{
		{name: "get safe route rejects smuggled json body", method: http.MethodGet, route: core.HTTPRouteSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySafe}, wantErr: core.ErrExchangeContract},
		{name: "head safe route rejects smuggled json body", method: http.MethodHead, route: core.HTTPRouteSemantics{Method: core.HTTPMethodHead, Replay: core.HTTPReplaySafe}, wantErr: core.ErrExchangeContract},
		{name: "options safe route rejects smuggled json body", method: http.MethodOptions, route: core.HTTPRouteSemantics{Method: core.HTTPMethodOptions, Replay: core.HTTPReplaySafe}, wantErr: core.ErrExchangeContract},
		{name: "delete idempotent route rejects smuggled json body", method: http.MethodDelete, route: core.HTTPRouteSemantics{Method: core.HTTPMethodDelete, Replay: core.HTTPReplayIdempotent}, wantErr: core.ErrExchangeContract},
		{name: "post single attempt route accepts body", method: http.MethodPost, route: core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt}},
		{name: "post idempotent route accepts body with key", method: http.MethodPost, key: "review-key-0001", route: core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent}},
		{name: "put idempotent route accepts body", method: http.MethodPut, route: core.HTTPRouteSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplayIdempotent}},
		{name: "patch idempotent route accepts body with key", method: http.MethodPatch, key: "review-key-0002", route: core.HTTPRouteSemantics{Method: core.HTTPMethodPatch, Replay: core.HTTPReplayIdempotent}},
		{name: "patch single attempt route accepts body", method: http.MethodPatch, route: core.HTTPRouteSemantics{Method: core.HTTPMethodPatch, Replay: core.HTTPReplaySingleAttempt}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(tc.method, "https://api.offgridsoftware.ca/v1/exchange", strings.NewReader(`{"name":"a","count":1}`))
			request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
			if tc.key != "" {
				request.Header.Set(core.HTTPHeaderIdempotencyKey, tc.key)
			}
			got, gotErr := ReceiveJSON[receiveFixture](request, tc.route, policy)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("ReceiveJSON() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr != nil && got != (Received[receiveFixture]{}) {
				t.Fatalf("ReceiveJSON() rejected value = %+v, want zero value", got)
			}
			if tc.wantErr == nil && got.Body != (receiveFixture{Name: "a", Count: 1}) {
				t.Fatalf("ReceiveJSON() body = %+v, want decoded fixture", got.Body)
			}
		})
	}
}
