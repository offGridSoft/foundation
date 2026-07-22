package durability

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestReadBoundedExactLimitOverflowAndIngressTable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path, _ := core.ParseAbsoluteFilePath(filepath.Join(dir, "state"))
	if err := os.WriteFile(path.String(), []byte("12345678"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cases := []struct {
		name      string
		ctx       context.Context
		request   ReadRequest
		want      string
		wantError error
	}{
		{name: "p01_exact_limit", ctx: context.Background(), request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(8)}, want: "12345678"},
		{name: "p02_above_content", ctx: context.Background(), request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(9)}, want: "12345678"},
		{name: "n01_one_below", ctx: context.Background(), request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(7)}, wantError: core.ErrDurableSizeLimit},
		{name: "n02_zero_request", ctx: context.Background(), request: ReadRequest{}, wantError: core.ErrDurabilityContract},
		{name: "n03_relative_path", ctx: context.Background(), request: ReadRequest{Path: "relative", MaximumBytes: core.NewByteCount(8)}, wantError: core.ErrDurabilityContract},
		{name: "n04_max_int_overflow_guard", ctx: context.Background(), request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(math.MaxInt64)}, wantError: core.ErrDurabilityContract},
		{name: "n05_nil_context", request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(8)}, wantError: core.ErrNilContext},
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cases = append(cases, struct {
		name      string
		ctx       context.Context
		request   ReadRequest
		want      string
		wantError error
	}{name: "n06_cancelled", ctx: cancelled, request: ReadRequest{Path: path, MaximumBytes: core.NewByteCount(8)}, wantError: context.Canceled})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadBounded(tc.ctx, tc.request)
			if tc.wantError == nil && (err != nil || string(got) != tc.want) {
				t.Fatalf("ReadBounded() = (%q,%v), want (%q,nil)", got, err, tc.want)
			}
			if tc.wantError != nil && !errors.Is(err, tc.wantError) {
				t.Fatalf("ReadBounded() error = %v, want errors.Is %v", err, tc.wantError)
			}
		})
	}
}

func TestContextReaderCancellationAndUnderlyingIdentity(t *testing.T) {
	t.Parallel()

	if _, err := (ContextReader{}).Read(make([]byte, 1)); !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("ContextReader(zero).Read() error = %v, want ErrNilContext", err)
	}
	if _, err := (ContextReader{Context: context.Background()}).Read(make([]byte, 1)); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("ContextReader(nil reader).Read() error = %v, want ErrDurabilityContract", err)
	}
	sentinel := errors.New("reader failure")
	reader := ContextReader{Context: context.Background(), Reader: hostileReader{err: sentinel}}
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, sentinel) {
		t.Fatalf("ContextReader(underlying).Read() error = %v, want sentinel", err)
	}
	reader = ContextReader{Context: context.Background(), Reader: bytes.NewBufferString("abc")}
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "abc" {
		t.Fatalf("io.ReadAll(ContextReader) = (%q,%v), want abc/nil", data, err)
	}
}

type hostileReader struct{ err error }

func (r hostileReader) Read([]byte) (int, error) { return 0, r.err }

func TestContextReaderStopsStreamWhenContextDiesMidRead(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := &cancellingReader{cancel: cancel}
	data, err := io.ReadAll(ContextReader{Context: ctx, Reader: source})
	if !errors.Is(err, context.Canceled) || string(data) != "a" {
		t.Fatalf("io.ReadAll(mid-stream cancel) = (%q,%v), want first chunk then context.Canceled", data, err)
	}
	if source.reads != 1 {
		t.Fatalf("underlying reads = %d, want exactly 1 before cancellation stopped the stream", source.reads)
	}
}

type cancellingReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancellingReader) Read(data []byte) (int, error) {
	r.reads++
	if len(data) == 0 {
		return 0, nil
	}
	data[0] = 'a'
	r.cancel()
	return 1, nil
}
