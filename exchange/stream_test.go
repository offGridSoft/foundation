package exchange

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestReadBoundedHostileFailureTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantErr   error
		wantCause error
		setup     func() (context.Context, io.Reader)
		name      string
		want      string
		limit     int64
		phase     bodyReadPhase
	}{
		{name: "request empty stream is neutral bytes", setup: streamBytes(""), phase: bodyReadRequest, limit: 1},
		{name: "response empty stream is neutral bytes", setup: streamBytes(""), phase: bodyReadResponse, limit: 1},
		{name: "request one byte accepts exact limit", setup: streamBytes("a"), phase: bodyReadRequest, limit: 1, want: "a"},
		{name: "response one byte accepts exact limit", setup: streamBytes("a"), phase: bodyReadResponse, limit: 1, want: "a"},
		{name: "request one below limit remains exact", setup: streamBytes("abcd"), phase: bodyReadRequest, limit: 5, want: "abcd"},
		{name: "response one below limit remains exact", setup: streamBytes("abcd"), phase: bodyReadResponse, limit: 5, want: "abcd"},
		{name: "one byte chunks assemble request without truncation", setup: streamChunked("abcdef", 1), phase: bodyReadRequest, limit: 6, want: "abcdef"},
		{name: "two byte chunks assemble response without truncation", setup: streamChunked("abcdef", 2), phase: bodyReadResponse, limit: 6, want: "abcdef"},
		{name: "large request crosses internal buffer boundary", setup: streamBytes(strings.Repeat("r", streamReadBufferBytes+1)), phase: bodyReadRequest, limit: streamReadBufferBytes + 1, want: strings.Repeat("r", streamReadBufferBytes+1)},
		{name: "large response crosses internal buffer boundary", setup: streamBytes(strings.Repeat("s", streamReadBufferBytes+1)), phase: bodyReadResponse, limit: streamReadBufferBytes + 1, want: strings.Repeat("s", streamReadBufferBytes+1)},
		{name: "request read with bytes and eof preserves final bytes", setup: streamReader(bytesAndEOFReader{body: []byte("final")}), phase: bodyReadRequest, limit: 5, want: "final"},
		{name: "response read with bytes and eof preserves final bytes", setup: streamReader(bytesAndEOFReader{body: []byte("final")}), phase: bodyReadResponse, limit: 5, want: "final"},
		{name: "request one above limit rejects exact overflow", setup: streamBytes("ab"), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeBodyLimit},
		{name: "response one above limit rejects exact overflow", setup: streamBytes("ab"), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeBodyLimit},
		{name: "request far above limit rejects bounded", setup: streamBytes(strings.Repeat("a", 1024)), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeBodyLimit},
		{name: "response far above limit rejects bounded", setup: streamBytes(strings.Repeat("a", 1024)), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeBodyLimit},
		{name: "request nil context rejects before read", setup: streamNilContext("a"), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeRequest},
		{name: "response nil context rejects before read", setup: streamNilContext("a"), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "request nil reader rejects before allocation", setup: streamNilReader, phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeRequest},
		{name: "response nil reader rejects before allocation", setup: streamNilReader, phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "request zero limit rejects missing bound", setup: streamBytes("a"), phase: bodyReadRequest, limit: 0, wantErr: core.ErrExchangeRequest},
		{name: "response zero limit rejects missing bound", setup: streamBytes("a"), phase: bodyReadResponse, limit: 0, wantErr: core.ErrExchangeResponse},
		{name: "request negative limit rejects hostile bound", setup: streamBytes("a"), phase: bodyReadRequest, limit: -1, wantErr: core.ErrExchangeRequest},
		{name: "response negative limit rejects hostile bound", setup: streamBytes("a"), phase: bodyReadResponse, limit: -1, wantErr: core.ErrExchangeResponse},
		{name: "request limit one above global cap rejects before allocation", setup: streamBytes("a"), phase: bodyReadRequest, limit: core.StrictJSONMaxBytes + 1, wantErr: core.ErrExchangeRequest},
		{name: "response limit one above global cap rejects before allocation", setup: streamBytes("a"), phase: bodyReadResponse, limit: core.StrictJSONMaxBytes + 1, wantErr: core.ErrExchangeResponse},
		{name: "request cancelled before read preserves cancellation identity", setup: cancelledStream("a"), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeCancelled},
		{name: "response cancelled before read preserves cancellation identity", setup: cancelledStream("a"), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeCancelled},
		{name: "request cancellation after first chunk rejects partial result", setup: cancelDuringStream("ab"), phase: bodyReadRequest, limit: 2, wantErr: core.ErrExchangeCancelled},
		{name: "response cancellation after first chunk rejects partial result", setup: cancelDuringStream("ab"), phase: bodyReadResponse, limit: 2, wantErr: core.ErrExchangeCancelled},
		{name: "request repeated empty reads terminate with no progress", setup: streamReader(emptyReader{}), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeRequest, wantCause: io.ErrNoProgress},
		{name: "response repeated empty reads terminate with no progress", setup: streamReader(emptyReader{}), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeResponse, wantCause: io.ErrNoProgress},
		{name: "request negative count rejects broken reader contract", setup: streamReader(negativeCountReader{}), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeRequest},
		{name: "response negative count rejects broken reader contract", setup: streamReader(negativeCountReader{}), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "request count above buffer rejects broken reader contract", setup: streamReader(oversizedCountReader{}), phase: bodyReadRequest, limit: 1, wantErr: core.ErrExchangeRequest},
		{name: "response count above buffer rejects broken reader contract", setup: streamReader(oversizedCountReader{}), phase: bodyReadResponse, limit: 1, wantErr: core.ErrExchangeResponse},
		{name: "request injected read error rejects partial bytes", setup: streamReader(bytesAndErrorReader{}), phase: bodyReadRequest, limit: 2, wantErr: core.ErrExchangeRequest, wantCause: core.ErrFoundationContract},
		{name: "response injected read error rejects partial bytes", setup: streamReader(bytesAndErrorReader{}), phase: bodyReadResponse, limit: 2, wantErr: core.ErrExchangeResponse, wantCause: core.ErrFoundationContract},
		{name: "request eof after empty reads still completes", setup: streamReader(&emptyThenEOFReader{remaining: streamMaximumEmptyReadRuns - 1}), phase: bodyReadRequest, limit: 1},
		{name: "response eof after empty reads still completes", setup: streamReader(&emptyThenEOFReader{remaining: streamMaximumEmptyReadRuns - 1}), phase: bodyReadResponse, limit: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, reader := tc.setup()
			got, gotErr := readBounded(ctx, reader, tc.limit, tc.phase)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("readBounded() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantCause != nil && !errors.Is(gotErr, tc.wantCause) {
				t.Fatalf("readBounded() error = %v, want cause %v", gotErr, tc.wantCause)
			}
			if tc.wantErr == nil && string(got) != tc.want {
				t.Fatalf("readBounded() = %q, want %q", got, tc.want)
			}
			if tc.wantErr != nil && got != nil {
				t.Fatalf("readBounded() rejected bytes = %q, want nil", got)
			}
		})
	}
}

func streamBytes(body string) func() (context.Context, io.Reader) {
	return streamReader(strings.NewReader(body))
}

func streamChunked(body string, maximum int) func() (context.Context, io.Reader) {
	return func() (context.Context, io.Reader) {
		return context.Background(), &chunkReader{body: []byte(body), maximum: maximum}
	}
}

func streamReader(reader io.Reader) func() (context.Context, io.Reader) {
	return func() (context.Context, io.Reader) {
		return context.Background(), reader
	}
}

func streamNilContext(body string) func() (context.Context, io.Reader) {
	return func() (context.Context, io.Reader) {
		return nil, strings.NewReader(body)
	}
}

func streamNilReader() (context.Context, io.Reader) {
	return context.Background(), nil
}

func cancelledStream(body string) func() (context.Context, io.Reader) {
	return func() (context.Context, io.Reader) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, strings.NewReader(body)
	}
}

func cancelDuringStream(body string) func() (context.Context, io.Reader) {
	return func() (context.Context, io.Reader) {
		ctx, cancel := context.WithCancel(context.Background())
		return ctx, &cancellingReader{body: []byte(body), cancel: cancel}
	}
}

type chunkReader struct {
	body    []byte
	maximum int
}

func (r *chunkReader) Read(destination []byte) (int, error) {
	if len(r.body) == 0 {
		return 0, io.EOF
	}
	count := min(len(destination), r.maximum, len(r.body))
	copy(destination, r.body[:count])
	r.body = r.body[count:]
	return count, nil
}

type bytesAndEOFReader struct {
	body []byte
}

func (r bytesAndEOFReader) Read(destination []byte) (int, error) {
	count := copy(destination, r.body)
	return count, io.EOF
}

type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) {
	return 0, nil
}

type negativeCountReader struct{}

func (negativeCountReader) Read([]byte) (int, error) {
	return -1, nil
}

type oversizedCountReader struct{}

func (oversizedCountReader) Read(destination []byte) (int, error) {
	return len(destination) + 1, nil
}

type bytesAndErrorReader struct{}

func (bytesAndErrorReader) Read(destination []byte) (int, error) {
	destination[0] = 'x'
	return 1, core.ErrFoundationContract
}

type emptyThenEOFReader struct {
	remaining int
}

func (r *emptyThenEOFReader) Read([]byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	r.remaining--
	return 0, nil
}

type cancellingReader struct {
	cancel context.CancelFunc
	body   []byte
	done   bool
}

func (r *cancellingReader) Read(destination []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	count := copy(destination, r.body[:1])
	r.cancel()
	return count, nil
}
