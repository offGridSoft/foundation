package exchange

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type copyFixture struct {
	ctx         context.Context
	destination io.Writer
	source      io.Reader
	output      *bytes.Buffer
}

func TestCopyBoundedHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantErr     error
		wantCause   error
		setup       func() copyFixture
		name        string
		wantOutput  string
		limit       uint64
		wantWritten int64
	}{
		{name: "empty source writes zero bytes", setup: copyBytes(""), limit: 1},
		{name: "one byte accepts exact limit", setup: copyBytes("a"), limit: 1, wantWritten: 1, wantOutput: "a"},
		{name: "one below limit accepts eof", setup: copyBytes("abcd"), limit: 5, wantWritten: 4, wantOutput: "abcd"},
		{name: "exact internal buffer accepts boundary", setup: copyBytes(strings.Repeat("a", streamReadBufferBytes)), limit: streamReadBufferBytes, wantWritten: streamReadBufferBytes, wantOutput: strings.Repeat("a", streamReadBufferBytes)},
		{name: "one above internal buffer accepts declared size", setup: copyBytes(strings.Repeat("b", streamReadBufferBytes+1)), limit: streamReadBufferBytes + 1, wantWritten: streamReadBufferBytes + 1, wantOutput: strings.Repeat("b", streamReadBufferBytes+1)},
		{name: "one above limit is detected without writing overflow", setup: copyBytes("ab"), limit: 1, wantWritten: 1, wantOutput: "a", wantErr: core.ErrExchangeBodyLimit},
		{name: "far above limit is detected after bounded write", setup: copyBytes(strings.Repeat("z", 4096)), limit: 3, wantWritten: 3, wantOutput: "zzz", wantErr: core.ErrExchangeBodyLimit},
		{name: "single byte chunks preserve exact stream", setup: copyReader(&chunkReader{body: []byte("abcdef"), maximum: 1}), limit: 6, wantWritten: 6, wantOutput: "abcdef"},
		{name: "bytes and eof in same read preserve bytes", setup: copyReader(bytesAndEOFReader{body: []byte("final")}), limit: 5, wantWritten: 5, wantOutput: "final"},
		{name: "ninety nine empty reads then eof remains neutral", setup: copyReader(&emptyThenEOFReader{remaining: streamMaximumEmptyReadRuns - 1}), limit: 1},
		{name: "exact limit then ninety nine empty probes and eof accepts", setup: copyReader(&exactThenEmptyProbeReader{remaining: streamMaximumEmptyReadRuns - 1}), limit: 1, wantWritten: 1, wantOutput: "a"},
		{name: "exact limit then ninety nine empty probes cannot hide overflow", setup: copyReader(&exactThenEmptyProbeReader{remaining: streamMaximumEmptyReadRuns - 1, overflow: true}), limit: 1, wantWritten: 1, wantOutput: "a", wantErr: core.ErrExchangeBodyLimit},
		{name: "exact limit then one hundred empty probes terminate before hidden overflow", setup: copyReader(&exactThenEmptyProbeReader{remaining: streamMaximumEmptyReadRuns, overflow: true}), limit: 1, wantWritten: 1, wantOutput: "a", wantErr: core.ErrExchangeTransport, wantCause: io.ErrNoProgress},
		{name: "one hundred empty reads terminate", setup: copyReader(emptyReader{}), limit: 1, wantErr: core.ErrExchangeTransport, wantCause: io.ErrNoProgress},
		{name: "reader error after bytes preserves written prefix and identity", setup: copyReader(bytesAndErrorReader{}), limit: 2, wantWritten: 1, wantOutput: "x", wantErr: core.ErrExchangeTransport, wantCause: core.ErrFoundationContract},
		{name: "negative reader count is rejected before slice", setup: copyReader(negativeCountReader{}), limit: 1, wantErr: core.ErrExchangeTransport},
		{name: "reader count beyond supplied buffer is rejected", setup: copyReader(oversizedCountReader{}), limit: 1, wantErr: core.ErrExchangeTransport},
		{name: "nil context rejects before read", setup: copyNilContext("a"), limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "cancelled context rejects before read", setup: copyCancelled("a"), limit: 1, wantErr: core.ErrExchangeCancelled},
		{name: "midstream cancellation rejects after exact first byte", setup: copyCancelDuring("ab"), limit: 2, wantWritten: 1, wantOutput: "a", wantErr: core.ErrExchangeCancelled},
		{name: "nil source rejects before allocation", setup: copyNilSource, limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "nil destination rejects before read", setup: copyNilDestination("a"), limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "zero byte limit rejects missing bound", setup: copyBytes("a"), limit: 0, wantErr: core.ErrExchangeResponse},
		{name: "writer error preserves exact partial count", setup: copyWithWriter("ab", errorAfterOneWriter{}), limit: 2, wantWritten: 1, wantErr: core.ErrExchangeWrite, wantCause: core.ErrFoundationContract},
		{name: "short writer preserves exact partial count", setup: copyWithWriter("ab", shortWriter{}), limit: 2, wantWritten: 1, wantErr: core.ErrExchangeWrite, wantCause: io.ErrShortWrite},
		{name: "negative writer count rejects broken contract", setup: copyWithWriter("a", negativeWriter{}), limit: 1, wantErr: core.ErrExchangeWrite},
		{name: "oversized writer count rejects broken contract", setup: copyWithWriter("a", oversizedWriter{}), limit: 1, wantErr: core.ErrExchangeWrite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fixture := tc.setup()
			written, gotErr := copyBounded(fixture.ctx, fixture.destination, fixture.source, core.NewByteCount(tc.limit))
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("copyBounded() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantCause != nil && !errors.Is(gotErr, tc.wantCause) {
				t.Fatalf("copyBounded() error = %v, want cause %v", gotErr, tc.wantCause)
			}
			if written != tc.wantWritten {
				t.Fatalf("copyBounded() written = %d, want %d", written, tc.wantWritten)
			}
			if fixture.output != nil && fixture.output.String() != tc.wantOutput {
				t.Fatalf("copyBounded() output bytes = %q, want %q", fixture.output.String(), tc.wantOutput)
			}
		})
	}
}

type exactThenEmptyProbeReader struct {
	remaining int
	started   bool
	overflow  bool
}

func (r *exactThenEmptyProbeReader) Read(buffer []byte) (int, error) {
	if !r.started {
		r.started = true
		buffer[0] = 'a'
		return 1, nil
	}
	if r.remaining > 0 {
		r.remaining--
		return 0, nil
	}
	if r.overflow {
		buffer[0] = 'b'
		return 1, io.EOF
	}
	return 0, io.EOF
}

func TestBoundedContractsHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	endpoint := mustBoundedEndpoint(t, "http://127.0.0.1:1")
	baselineRequest := BoundedRequest[core.APIEndpoint]{
		Target: endpoint, Body: []byte("x"),
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt},
		ExpectedStatus: core.HTTPStatusOK, RequestContentType: core.HTTPMediaTypeOctetStream,
		ExpectedResponseContentType: core.HTTPMediaTypeOctetStream,
	}
	baselinePolicy := boundedTestPolicy(1)
	cases := []struct {
		run     func() error
		wantErr error
		name    string
	}{
		{name: "ordinary bounded post accepts exact contract", run: baselineRequest.Validate},
		{name: "response media may be explicitly unconstrained", run: func() error {
			value := baselineRequest
			value.ExpectedResponseContentType = core.HTTPMediaTypeUnknown
			return value.Validate()
		}},
		{name: "get without body accepts explicit single attempt", run: func() error {
			value := baselineRequest
			value.Body = nil
			value.RequestContentType = core.HTTPMediaTypeUnknown
			value.Semantics.Method = core.HTTPMethodGet
			return value.Validate()
		}},
		{name: "zero target rejects before network", run: func() error { value := baselineRequest; value.Target = core.APIEndpoint{}; return value.Validate() }, wantErr: core.ErrExchangeRequest},
		{name: "post safe replay rejects semantic lie", run: func() error {
			value := baselineRequest
			value.Semantics.Replay = core.HTTPReplaySafe
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "post idempotent without key rejects hidden replay", run: func() error {
			value := baselineRequest
			value.Semantics.Replay = core.HTTPReplayIdempotent
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "bounded operation refuses idempotent retry mode even with key", run: func() error {
			value := baselineRequest
			value.Semantics.Replay = core.HTTPReplayIdempotent
			value.Semantics.IdempotencyKey, _ = core.ParseHTTPIdempotencyKey("bounded-key")
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "post empty body rejects missing bytes", run: func() error { value := baselineRequest; value.Body = nil; return value.Validate() }, wantErr: core.ErrExchangeRequest},
		{name: "get body rejects parser confusion", run: func() error {
			value := baselineRequest
			value.Semantics.Method = core.HTTPMethodGet
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "body without media type rejects ambiguity", run: func() error {
			value := baselineRequest
			value.RequestContentType = core.HTTPMediaTypeUnknown
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "body with future media enum rejects", run: func() error {
			value := baselineRequest
			value.RequestContentType = core.HTTPMediaType(255)
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "future response media enum rejects", run: func() error {
			value := baselineRequest
			value.ExpectedResponseContentType = core.HTTPMediaType(255)
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "non success expected status rejects", run: func() error {
			value := baselineRequest
			value.ExpectedStatus = core.HTTPStatusInternalServerError
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "zero expected status rejects", run: func() error {
			value := baselineRequest
			value.ExpectedStatus = core.HTTPStatusCodeUnknown
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "newline header name rejects injection", run: func() error {
			value := baselineRequest
			value.Headers.Values = []core.HTTPHeader{{Name: "X-Test\n", Value: "x"}}
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "newline header value rejects injection", run: func() error {
			value := baselineRequest
			value.Headers.Values = []core.HTTPHeader{{Name: "X-Test", Value: "x\n"}}
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "reserved accept header rejects split ownership", run: func() error {
			value := baselineRequest
			value.Headers.Values = []core.HTTPHeader{{Name: core.HTTPHeaderAccept, Value: core.HTTPContentTypeJSON}}
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "reserved content length rejects framing override", run: func() error {
			value := baselineRequest
			value.Headers.Values = []core.HTTPHeader{{Name: core.HTTPHeaderContentLength, Value: "1"}}
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "duplicate query name rejects ambiguous target", run: func() error {
			value := baselineRequest
			value.Query.Parameters = []core.HTTPQueryParameter{{Name: "mode", Value: "a"}, {Name: "mode", Value: "b"}}
			return value.Validate()
		}, wantErr: core.ErrExchangeRequest},
		{name: "duplicate captured header rejects ambiguous output", run: func() error {
			value := baselineRequest
			value.CaptureHeaders.Names = []string{"X-One", "x-one"}
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "invalid captured header rejects injection", run: func() error {
			value := baselineRequest
			value.CaptureHeaders.Names = []string{"X-One\n"}
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "ordinary bounded policy accepts exact contract", run: baselinePolicy.Validate},
		{name: "one nanosecond attempt floor accepts", run: func() error {
			value := baselinePolicy
			value.AttemptTimeout = core.NewNanosecondsDuration(time.Nanosecond)
			return value.Validate()
		}},
		{name: "zero attempt timeout rejects missing deadline", run: func() error {
			value := baselinePolicy
			value.AttemptTimeout = core.NanosecondsDuration{}
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "negative attempt timeout rejects hostile duration", run: func() error {
			value := baselinePolicy
			value.AttemptTimeout = core.NanosecondsDurationFromInt64(-1)
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "zero request limit rejects unbounded body", run: func() error {
			value := baselinePolicy
			value.RequestBodyLimit = core.ByteCount{}
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "request one above global bound rejects", run: func() error {
			value := baselinePolicy
			value.RequestBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1)
			return value.Validate()
		}, wantErr: core.ErrExchangeBodyLimit},
		{name: "zero response limit rejects unbounded body", run: func() error {
			value := baselinePolicy
			value.ResponseBodyLimit = core.ByteCount{}
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
		{name: "response one above global bound rejects", run: func() error {
			value := baselinePolicy
			value.ResponseBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1)
			return value.Validate()
		}, wantErr: core.ErrExchangeBodyLimit},
		{name: "unknown redirect rejects zero enum", run: func() error {
			value := baselinePolicy
			value.Redirect = core.HTTPRedirectUnknown
			return value.Validate()
		}, wantErr: core.ErrExchangeResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotErr := tc.run()
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("contract error = %v, want %v", gotErr, tc.wantErr)
			}
		})
	}
}

func TestBoundedAndStreamRealNetworkLayerTable(t *testing.T) {
	t.Parallel()

	t.Run("bounded bytes send typed media query headers and exact response", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Method != http.MethodPost || request.URL.Query().Get("mode") != "full" || request.Header.Get("X-Test-Mode") != "hostile" {
				t.Errorf("request = %s %s headers=%v, want typed request", request.Method, request.URL.String(), request.Header)
			}
			if request.Header.Get(core.HTTPHeaderContentType) != core.HTTPContentTypeTimestampQuery || request.Header.Get(core.HTTPHeaderAccept) != core.HTTPContentTypeTimestampReply {
				t.Errorf("media headers = %v, want timestamp query/reply", request.Header)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil || string(body) != "query" {
				t.Errorf("request body = %q, %v, want query", body, err)
			}
			writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeTimestampReply)
			writer.Header().Set("X-Receipt", "receipt-1")
			_, _ = writer.Write([]byte("reply"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		request := BoundedRequest[core.APIEndpoint]{
			Target: endpoint, Body: []byte("query"),
			Headers:        core.HTTPHeaders{Values: []core.HTTPHeader{{Name: "X-Test-Mode", Value: "hostile"}}},
			Query:          core.HTTPQuery{Parameters: []core.HTTPQueryParameter{{Name: "mode", Value: "full"}}},
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK, RequestContentType: core.HTTPMediaTypeTimestampQuery,
			ExpectedResponseContentType: core.HTTPMediaTypeTimestampReply,
			CaptureHeaders:              HeaderSelection{Names: []string{"X-Receipt"}},
		}
		response, err := SendBounded(t.Context(), Client{HTTP: server.Client()}, request, boundedTestPolicy(5))
		if err != nil || string(response.Body) != "reply" || response.Status != core.HTTPStatusOK {
			t.Fatalf("SendBounded() = (%+v, %v), want typed reply", response, err)
		}
		if receipt, ok := response.Headers.Get("X-Receipt"); !ok || receipt != "receipt-1" {
			t.Fatalf("captured receipt = %q/%v, want receipt-1/true", receipt, ok)
		}
	})

	t.Run("stream upload exact boundary captures selected response header", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, err := io.ReadAll(request.Body)
			if err != nil || string(body) != "artifact" {
				t.Errorf("upload body = %q, %v, want artifact", body, err)
			}
			writer.Header().Set("X-Generation", "17")
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		response, err := SendStream(t.Context(), Client{HTTP: server.Client()}, StreamUploadRequest[core.APIEndpoint]{
			Target: endpoint, Body: strings.NewReader("artifact"), ContentLength: core.NewByteLength(8),
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK, CaptureHeaders: HeaderSelection{Names: []string{"X-Generation"}},
		}, streamTestPolicy())
		if err != nil || response.BytesWritten.Uint64() != 8 {
			t.Fatalf("SendStream() = (%+v, %v), want eight bytes", response, err)
		}
	})

	t.Run("stream upload accepts exact empty body", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			body, err := io.ReadAll(request.Body)
			if err != nil || len(body) != 0 || request.ContentLength != 0 {
				t.Errorf("empty upload = bytes:%d length:%d error:%v, want 0/0/nil", len(body), request.ContentLength, err)
			}
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		response, err := SendStream(t.Context(), Client{HTTP: server.Client()}, StreamUploadRequest[core.APIEndpoint]{
			Target: endpoint, Body: strings.NewReader(""), ContentLength: core.NewByteLength(0),
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK,
		}, streamTestPolicy())
		if err != nil || response.BytesWritten.Uint64() != 0 {
			t.Fatalf("SendStream(empty) = (%+v, %v), want zero bytes and nil", response, err)
		}
	})

	t.Run("stream upload one above declared length is refused", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = io.Copy(io.Discard, request.Body)
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		_, err := SendStream(t.Context(), Client{HTTP: server.Client()}, StreamUploadRequest[core.APIEndpoint]{
			Target: endpoint, Body: strings.NewReader("ab"), ContentLength: core.NewByteLength(1),
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK,
		}, streamTestPolicy())
		if !errors.Is(err, core.ErrExchangeTransport) {
			t.Fatalf("SendStream(oversize) error = %v, want %v", err, core.ErrExchangeTransport)
		}
	})

	t.Run("stream download one above limit never writes overflow byte", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			_, _ = writer.Write([]byte("abcd"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		var destination bytes.Buffer
		_, err := ReceiveStream(t.Context(), Client{HTTP: server.Client()}, StreamDownloadRequest[core.APIEndpoint]{
			Target: endpoint, Destination: &destination, ResponseBodyLimit: core.NewByteCount(3),
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK,
		}, streamTestPolicy())
		if !errors.Is(err, core.ErrExchangeBodyLimit) || destination.String() != "abc" {
			t.Fatalf("ReceiveStream(oversize) = output %q error %v, want abc/%v", destination.String(), err, core.ErrExchangeBodyLimit)
		}
	})

	t.Run("bounded rejection preserves status through media failure", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeTextPlain)
			writer.WriteHeader(core.HTTPStatusServiceUnavailable.Int())
			_, _ = writer.Write([]byte("busy"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		_, err := SendBounded(t.Context(), Client{HTTP: server.Client()}, BoundedRequest[core.APIEndpoint]{
			Target: endpoint, Semantics: core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK, ExpectedResponseContentType: core.HTTPMediaTypeTimestampReply,
		}, boundedTestPolicy(8))
		statusErr, hasStatus := errors.AsType[StatusError](err)
		if !errors.Is(err, core.ErrExchangeContentType) || !hasStatus || statusErr.Status != core.HTTPStatusServiceUnavailable {
			t.Fatalf("SendBounded(media rejection) error = %v status=%+v/%v, want %v and status %v", err, statusErr, hasStatus, core.ErrExchangeContentType, core.HTTPStatusServiceUnavailable)
		}
	})

	t.Run("bounded rejection preserves status through body overflow", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeTextPlain)
			writer.WriteHeader(core.HTTPStatusServiceUnavailable.Int())
			_, _ = writer.Write([]byte("xx"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		_, err := SendBounded(t.Context(), Client{HTTP: server.Client()}, BoundedRequest[core.APIEndpoint]{
			Target: endpoint, Semantics: core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK, ExpectedResponseContentType: core.HTTPMediaTypeTextPlain,
		}, boundedTestPolicy(1))
		statusErr, hasStatus := errors.AsType[StatusError](err)
		if !errors.Is(err, core.ErrExchangeBodyLimit) || !hasStatus || statusErr.Status != core.HTTPStatusServiceUnavailable {
			t.Fatalf("SendBounded(overflow rejection) error = %v status=%+v/%v, want %v and status %v", err, statusErr, hasStatus, core.ErrExchangeBodyLimit, core.HTTPStatusServiceUnavailable)
		}
	})

	t.Run("stream upload rejection preserves status through bounded drain failure", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = io.Copy(io.Discard, request.Body)
			writer.WriteHeader(core.HTTPStatusServiceUnavailable.Int())
			_, _ = writer.Write([]byte("xx"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		policy := streamTestPolicy()
		policy.ErrorBodyLimit = core.NewByteCount(1)
		_, err := SendStream(t.Context(), Client{HTTP: server.Client()}, StreamUploadRequest[core.APIEndpoint]{
			Target: endpoint, Body: strings.NewReader("a"), ContentLength: core.NewByteLength(1),
			Semantics: core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt}, ExpectedStatus: core.HTTPStatusOK,
		}, policy)
		statusErr, hasStatus := errors.AsType[StatusError](err)
		if !errors.Is(err, core.ErrExchangeBodyLimit) || !hasStatus || statusErr.Status != core.HTTPStatusServiceUnavailable {
			t.Fatalf("SendStream(drain rejection) error = %v status=%+v/%v, want %v and status %v", err, statusErr, hasStatus, core.ErrExchangeBodyLimit, core.HTTPStatusServiceUnavailable)
		}
	})

	t.Run("stream download rejection drains boundedly without touching destination", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(core.HTTPStatusTooManyRequests.Int())
			_, _ = writer.Write([]byte("xx"))
		}))
		defer server.Close()
		endpoint := mustBoundedEndpoint(t, server.URL)
		policy := streamTestPolicy()
		policy.ErrorBodyLimit = core.NewByteCount(1)
		var destination bytes.Buffer
		_, err := ReceiveStream(t.Context(), Client{HTTP: server.Client()}, StreamDownloadRequest[core.APIEndpoint]{
			Target: endpoint, Destination: &destination, ResponseBodyLimit: core.NewByteCount(8),
			Semantics: core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySingleAttempt}, ExpectedStatus: core.HTTPStatusOK,
		}, policy)
		statusErr, hasStatus := errors.AsType[StatusError](err)
		if !errors.Is(err, core.ErrExchangeBodyLimit) || !hasStatus || statusErr.Status != core.HTTPStatusTooManyRequests || destination.Len() != 0 {
			t.Fatalf("ReceiveStream(rejection) = output %q error %v status=%+v/%v, want empty/%v/status %v", destination.String(), err, statusErr, hasStatus, core.ErrExchangeBodyLimit, core.HTTPStatusTooManyRequests)
		}
	})
}

func boundedTestPolicy(limit uint64) BoundedPolicy {
	return BoundedPolicy{
		AttemptTimeout:   core.NewNanosecondsDuration(time.Second),
		RequestBodyLimit: core.NewByteCount(limit), ResponseBodyLimit: core.NewByteCount(limit),
		Redirect: core.HTTPRedirectReject,
	}
}

func streamTestPolicy() StreamPolicy {
	return StreamPolicy{
		AttemptTimeout: core.NewNanosecondsDuration(time.Second),
		ErrorBodyLimit: core.NewByteCount(1024), Redirect: core.HTTPRedirectReject,
	}
}

func mustBoundedEndpoint(t *testing.T, baseURL string) core.APIEndpoint {
	t.Helper()
	endpoint, err := core.ParseAPIEndpoint(baseURL + "/v1/bounded")
	if err != nil {
		t.Fatalf("ParseAPIEndpoint() error = %v, want nil", err)
	}
	return endpoint
}

func copyBytes(value string) func() copyFixture {
	return func() copyFixture {
		output := &bytes.Buffer{}
		return copyFixture{ctx: context.Background(), destination: output, source: strings.NewReader(value), output: output}
	}
}

func copyReader(reader io.Reader) func() copyFixture {
	return func() copyFixture {
		output := &bytes.Buffer{}
		return copyFixture{ctx: context.Background(), destination: output, source: reader, output: output}
	}
}

func copyNilContext(value string) func() copyFixture {
	return func() copyFixture {
		output := &bytes.Buffer{}
		return copyFixture{destination: output, source: strings.NewReader(value), output: output}
	}
}

func copyCancelled(value string) func() copyFixture {
	return func() copyFixture {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		output := &bytes.Buffer{}
		return copyFixture{ctx: ctx, destination: output, source: strings.NewReader(value), output: output}
	}
}

func copyCancelDuring(value string) func() copyFixture {
	return func() copyFixture {
		ctx, cancel := context.WithCancel(context.Background())
		output := &bytes.Buffer{}
		return copyFixture{ctx: ctx, destination: output, source: &cancellingReader{body: []byte(value), cancel: cancel}, output: output}
	}
}

func copyNilSource() copyFixture {
	output := &bytes.Buffer{}
	return copyFixture{ctx: context.Background(), destination: output, output: output}
}

func copyNilDestination(value string) func() copyFixture {
	return func() copyFixture {
		return copyFixture{ctx: context.Background(), source: strings.NewReader(value)}
	}
}

func copyWithWriter(value string, writer io.Writer) func() copyFixture {
	return func() copyFixture {
		return copyFixture{ctx: context.Background(), destination: writer, source: strings.NewReader(value)}
	}
}

type errorAfterOneWriter struct{}

func (errorAfterOneWriter) Write([]byte) (int, error) { return 1, core.ErrFoundationContract }

type shortWriter struct{}

func (shortWriter) Write([]byte) (int, error) { return 1, nil }

type negativeWriter struct{}

func (negativeWriter) Write([]byte) (int, error) { return -1, nil }

type oversizedWriter struct{}

func (oversizedWriter) Write(body []byte) (int, error) { return len(body) + 1, nil }
