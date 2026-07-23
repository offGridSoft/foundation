package exchange

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type networkBehavior uint8

const (
	networkBehaviorTyped networkBehavior = iota + 1
	networkBehaviorEmpty
	networkBehaviorWhitespace
	networkBehaviorMalformed
	networkBehaviorDuplicate
	networkBehaviorUnknown
	networkBehaviorWrongContentType
	networkBehaviorContentEncoding
	networkBehaviorDeclaredOversize
	networkBehaviorStreamedOversize
	networkBehaviorTruncated
	networkBehaviorFailureAtSuccessStatus
	networkBehaviorSuccessAtFailureStatus
	networkBehaviorAbort
	networkBehaviorRedirect307
	networkBehaviorRedirect302
	networkBehaviorCrossOriginRedirect
)

func TestSendJSONRealNetworkFailureTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantErr        error
		name           string
		behavior       networkBehavior
		status         core.HTTPStatusCode
		acceptedStatus core.HTTPStatusCode
		redirect       core.HTTPRedirectPolicy
		wantAttempts   uint64
		wantRequests   uint64
	}{
		{name: "http 200 typed success completes first attempt", behavior: networkBehaviorTyped, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantAttempts: 1},
		{name: "http 201 typed success honors endpoint status", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusCreated), acceptedStatus: mustHTTPStatus(t, http.StatusCreated), redirect: core.HTTPRedirectReject, wantAttempts: 1},
		{name: "http 408 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusRequestTimeout, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 425 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusTooEarly, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 429 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusTooManyRequests, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 500 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusInternalServerError, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 502 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusBadGateway, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 503 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusServiceUnavailable, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 504 retries exact budget then exhausts", behavior: networkBehaviorTyped, status: core.HTTPStatusGatewayTimeout, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "http 400 is terminal after one attempt", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusBadRequest), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 401 is terminal after one attempt", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusUnauthorized), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 403 is terminal after one attempt", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusForbidden), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 404 is terminal after one attempt", behavior: networkBehaviorTyped, status: core.HTTPStatusNotFound, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 409 is terminal after one attempt", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusConflict), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 422 is terminal after one attempt", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusUnprocessableEntity), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 501 is terminal despite server class", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusNotImplemented), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "http 505 is terminal despite server class", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusHTTPVersionNotSupported), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "empty response rejects missing envelope", behavior: networkBehaviorEmpty, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "whitespace response rejects missing envelope", behavior: networkBehaviorWhitespace, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "malformed response rejects truncated envelope", behavior: networkBehaviorMalformed, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "duplicate response field rejects ambiguous envelope", behavior: networkBehaviorDuplicate, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "unknown response field rejects schema drift", behavior: networkBehaviorUnknown, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "wrong response content type rejects parser confusion", behavior: networkBehaviorWrongContentType, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeContentType, wantAttempts: 1},
		{name: "content encoding rejects undeclared transform", behavior: networkBehaviorContentEncoding, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeContentType, wantAttempts: 1},
		{name: "declared response one above limit rejects before read", behavior: networkBehaviorDeclaredOversize, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeBodyLimit, wantAttempts: 1},
		{name: "streamed response one above limit rejects bounded", behavior: networkBehaviorStreamedOversize, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeBodyLimit, wantAttempts: 1},
		{name: "truncated response retries then exhausts transport budget", behavior: networkBehaviorTruncated, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "failure envelope at success status rejects contradiction", behavior: networkBehaviorFailureAtSuccessStatus, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "success envelope at retry status rejects contradiction", behavior: networkBehaviorSuccessAtFailureStatus, status: core.HTTPStatusServiceUnavailable, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
		{name: "connection abort retries then exhausts transport budget", behavior: networkBehaviorAbort, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRetryExhausted, wantAttempts: 2},
		{name: "redirect rejection is terminal and typed", behavior: networkBehaviorRedirect307, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeRedirect, wantAttempts: 1},
		{name: "same origin 307 preserves method and succeeds", behavior: networkBehaviorRedirect307, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectSameOrigin, wantAttempts: 1, wantRequests: 2},
		{name: "same origin 302 method rewrite is rejected", behavior: networkBehaviorRedirect302, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectSameOrigin, wantErr: core.ErrExchangeRedirect, wantAttempts: 1},
		{name: "cross origin redirect is rejected", behavior: networkBehaviorCrossOriginRedirect, status: core.HTTPStatusOK, acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectSameOrigin, wantErr: core.ErrExchangeRedirect, wantAttempts: 1},
		{name: "unexpected 201 success rejects status contract", behavior: networkBehaviorTyped, status: mustHTTPStatus(t, http.StatusCreated), acceptedStatus: core.HTTPStatusOK, redirect: core.HTTPRedirectReject, wantErr: core.ErrExchangeResponse, wantAttempts: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var attempts atomic.Uint64
			var crossTarget string
			var crossServer *httptest.Server
			if tc.behavior == networkBehaviorCrossOriginRedirect {
				crossServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					writeRawSuccess(t, writer, tc.status)
				}))
				crossTarget = crossServer.URL + "/v1/final"
				defer crossServer.Close()
			}
			server := httptest.NewServer(networkHandler(t, tc.behavior, tc.status, &attempts, crossTarget))
			defer server.Close()

			request := idempotentClientRequest(t, server.URL)
			request.ExpectedStatus = tc.acceptedStatus
			policy := clientTestPolicy(2)
			policy.Redirect = tc.redirect
			client := mustTestClientRuntime(t, server.Client(), fixedClock{now: core.UnixNanoTimeFromInt64(1)}, fixedJitter{fraction: 1}, &recordingWaiter{})
			got, gotErr := SendJSON[receiveFixture, responseFixture](context.Background(), client, request, policy)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("SendJSON() error = %v, want %v", gotErr, tc.wantErr)
			}
			if got.Attempts != tc.wantAttempts {
				t.Fatalf("SendJSON() attempts = %d, want %d", got.Attempts, tc.wantAttempts)
			}
			wantRequests := tc.wantRequests
			if wantRequests == 0 {
				wantRequests = tc.wantAttempts
			}
			if gotRequests := attempts.Load(); gotRequests != wantRequests {
				t.Fatalf("server requests = %d, want %d", gotRequests, wantRequests)
			}
		})
	}
}

func networkHandler(t *testing.T, behavior networkBehavior, status core.HTTPStatusCode, attempts *atomic.Uint64, crossTarget string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts.Add(1)
		if request.URL.Path == "/v1/final" {
			writeRawSuccess(t, writer, status)
			return
		}
		switch behavior {
		case networkBehaviorRedirect307:
			http.Redirect(writer, request, "/v1/final", http.StatusTemporaryRedirect)
			return
		case networkBehaviorRedirect302:
			http.Redirect(writer, request, "/v1/final", http.StatusFound)
			return
		case networkBehaviorCrossOriginRedirect:
			http.Redirect(writer, request, crossTarget, http.StatusTemporaryRedirect)
			return
		case networkBehaviorAbort:
			abortConnection(t, writer)
			return
		case networkBehaviorTruncated:
			writeTruncatedResponse(t, writer)
			return
		}
		if _, err := ReceiveJSON[receiveFixture](request,
			core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent},
			ServerPolicy{RequestBodyLimit: core.NewByteCount(receiveFixtureBodyLimit)},
		); err != nil {
			t.Errorf("ReceiveJSON() error = %v, want nil", err)
			return
		}
		writeNetworkBehavior(t, writer, behavior, status)
	})
}

func writeNetworkBehavior(t *testing.T, writer http.ResponseWriter, behavior networkBehavior, status core.HTTPStatusCode) {
	t.Helper()
	switch behavior {
	case networkBehaviorTyped:
		writeTypedNetworkResponse(t, writer, status)
	case networkBehaviorEmpty:
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, "")
	case networkBehaviorWhitespace:
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, " \n\t")
	case networkBehaviorMalformed:
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, `{"request_id":`)
	case networkBehaviorDuplicate:
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, `{"request_id":"a","request_id":"b","data":{"value":"accepted"},"error":null}`)
	case networkBehaviorUnknown:
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, `{"request_id":"a","data":{"value":"accepted"},"error":null,"future":true}`)
	case networkBehaviorWrongContentType:
		writeRawResponse(writer, status, "text/plain", `{"request_id":"a","data":{"value":"accepted"},"error":null}`)
	case networkBehaviorContentEncoding:
		writer.Header().Set(core.HTTPHeaderContentEncoding, "gzip")
		writeRawResponse(writer, status, core.HTTPContentTypeJSON, `{"request_id":"a","data":{"value":"accepted"},"error":null}`)
	case networkBehaviorDeclaredOversize:
		writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writer.Header().Set(core.HTTPHeaderContentLength, fmt.Sprintf("%d", receiveFixtureBodyLimit+1))
		writer.WriteHeader(status.Int())
	case networkBehaviorStreamedOversize:
		writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writer.WriteHeader(status.Int())
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
		_, _ = writer.Write([]byte(strings.Repeat(" ", receiveFixtureBodyLimit+1)))
	case networkBehaviorFailureAtSuccessStatus:
		writeRawEnvelope(t, writer, status, failureResponseEnvelope(t))
	case networkBehaviorSuccessAtFailureStatus:
		writeRawEnvelope(t, writer, status, successResponseEnvelope(t))
	default:
		t.Errorf("unknown network behavior = %d", behavior)
	}
}

func writeTypedNetworkResponse(t *testing.T, writer http.ResponseWriter, status core.HTTPStatusCode) {
	t.Helper()
	if status >= 200 && status <= 299 {
		if err := WriteJSON(writer, ServerResponse[responseFixture]{Status: status, Envelope: successResponseEnvelope(t)}); err != nil {
			t.Errorf("WriteJSON(success) error = %v, want nil", err)
		}
		return
	}
	response := ServerResponse[responseFixture]{Status: status, Envelope: failureResponseEnvelope(t)}
	retryable, err := core.HTTPStatusIsRetryable(status)
	if err != nil {
		t.Errorf("HTTPStatusIsRetryable() error = %v, want nil", err)
		return
	}
	if retryable {
		response.RetryAfter = core.HTTPRetryDirective{Delay: core.NewNanosecondsDuration(time.Nanosecond)}
	}
	if err := WriteJSON(writer, response); err != nil {
		t.Errorf("WriteJSON(failure) error = %v, want nil", err)
	}
}

func writeRawSuccess(t *testing.T, writer http.ResponseWriter, status core.HTTPStatusCode) {
	t.Helper()
	writeRawEnvelope(t, writer, status, successResponseEnvelope(t))
}

func writeRawEnvelope(t *testing.T, writer http.ResponseWriter, status core.HTTPStatusCode, envelope core.APIEnvelope[responseFixture]) {
	t.Helper()
	body, err := core.EncodeValidatedJSON(envelope)
	if err != nil {
		t.Errorf("EncodeValidatedJSON() error = %v, want nil", err)
		return
	}
	writeRawResponse(writer, status, core.HTTPContentTypeJSON, string(body))
}

func writeRawResponse(writer http.ResponseWriter, status core.HTTPStatusCode, contentType, body string) {
	writer.Header().Set(core.HTTPHeaderContentType, contentType)
	writer.WriteHeader(status.Int())
	_, _ = writer.Write([]byte(body))
}

func abortConnection(t *testing.T, writer http.ResponseWriter) {
	t.Helper()
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		t.Errorf("response writer lacks hijacker")
		return
	}
	connection, _, err := hijacker.Hijack()
	if err != nil {
		t.Errorf("Hijack() error = %v, want nil", err)
		return
	}
	if err := connection.Close(); err != nil {
		t.Errorf("connection.Close() error = %v, want nil", err)
	}
}

func writeTruncatedResponse(t *testing.T, writer http.ResponseWriter) {
	t.Helper()
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		t.Errorf("response writer lacks hijacker")
		return
	}
	connection, buffered, err := hijacker.Hijack()
	if err != nil {
		t.Errorf("Hijack() error = %v, want nil", err)
		return
	}
	_, _ = buffered.WriteString("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"request_id\":")
	_ = buffered.Flush()
	if err := connection.Close(); err != nil {
		t.Errorf("connection.Close() error = %v, want nil", err)
	}
}

func mustHTTPStatus(t *testing.T, raw int) core.HTTPStatusCode {
	t.Helper()
	status, err := core.NewHTTPStatusCode(raw)
	if err != nil {
		t.Fatalf("NewHTTPStatusCode(%d) error = %v, want nil", raw, err)
	}
	return status
}
