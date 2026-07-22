package durabilitytest

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"syscall"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestENOSPCWriterEveryThresholdBoundaryTable(t *testing.T) {
	t.Parallel()

	payload := []byte("0123456789")
	for capacity := uint64(0); capacity <= uint64(len(payload))+2; capacity++ {
		capacity := capacity
		t.Run(string(rune('a'+capacity)), func(t *testing.T) {
			t.Parallel()
			var sink bytes.Buffer
			writer, err := NewENOSPCWriter(ENOSPCConfig{Writer: &sink, CapacityBytes: capacity})
			if err != nil {
				t.Fatalf("NewENOSPCWriter(capacity=%d) error = %v, want nil", capacity, err)
			}
			n, writeErr := writer.Write(payload)
			wantWritten := min(capacity, uint64(len(payload)))
			if uint64(n) != wantWritten || writer.BytesWritten() != wantWritten || uint64(sink.Len()) != wantWritten {
				t.Fatalf("Write(capacity=%d) counts = n:%d observed:%d sink:%d, want %d", capacity, n, writer.BytesWritten(), sink.Len(), wantWritten)
			}
			wantENOSPC := capacity < uint64(len(payload))
			if errors.Is(writeErr, syscall.ENOSPC) != wantENOSPC {
				t.Fatalf("Write(capacity=%d) error = %v, errors.Is(ENOSPC) want %t", capacity, writeErr, wantENOSPC)
			}
			if got := sink.Bytes(); !bytes.Equal(got, payload[:wantWritten]) {
				t.Fatalf("Write(capacity=%d) bytes = %q, want %q", capacity, got, payload[:wantWritten])
			}
			if writer.CapacityBytes() != capacity {
				t.Fatalf("CapacityBytes() = %d, want %d", writer.CapacityBytes(), capacity)
			}
			nextN, nextErr := writer.Write([]byte("x"))
			if capacity <= uint64(len(payload)) && (nextN != 0 || !errors.Is(nextErr, syscall.ENOSPC)) {
				t.Fatalf("Write(after exhausted capacity=%d) = (%d,%v), want (0,ENOSPC)", capacity, nextN, nextErr)
			}
		})
	}
}

func TestENOSPCWriterUnderlyingFailuresAndInvalidCounts(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("underlying EIO")
	cases := []struct {
		writer    io.Writer
		wantError error
		name      string
		wantN     int
	}{
		{name: "n01_underlying_error_wins", writer: hostileWriter{n: 0, err: sentinel}, wantError: sentinel},
		{name: "n02_short_nil_is_typed", writer: hostileWriter{n: 1}, wantError: core.ErrDurableShortWrite, wantN: 1},
		{name: "n03_negative_count_is_typed", writer: hostileWriter{n: -1}, wantError: core.ErrDurableShortWrite},
		{name: "n04_oversized_count_is_typed", writer: hostileWriter{n: 99}, wantError: core.ErrDurableShortWrite},
		{name: "p01_full_write", writer: hostileWriter{n: 3}, wantN: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer, err := NewENOSPCWriter(ENOSPCConfig{Writer: tc.writer, CapacityBytes: 10})
			if err != nil {
				t.Fatalf("NewENOSPCWriter() error = %v", err)
			}
			n, err := writer.Write([]byte("abc"))
			if n != tc.wantN || (tc.wantError == nil && err != nil) || (tc.wantError != nil && !errors.Is(err, tc.wantError)) {
				t.Fatalf("Write() = (%d,%v), want (%d, errors.Is %v)", n, err, tc.wantN, tc.wantError)
			}
			if errors.Is(err, syscall.ENOSPC) {
				t.Fatalf("Write() error = %v, underlying failure must not be overlaid with ENOSPC", err)
			}
		})
	}

	if _, err := NewENOSPCWriter(ENOSPCConfig{}); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("NewENOSPCWriter(nil) error = %v, want ErrDurabilityContract", err)
	}
}

func TestENOSPCWriterConcurrentWritesNeverExceedCapacity(t *testing.T) {
	t.Parallel()

	var sink lockedBuffer
	writer, err := NewENOSPCWriter(ENOSPCConfig{Writer: &sink, CapacityBytes: 257})
	if err != nil {
		t.Fatalf("NewENOSPCWriter() error = %v", err)
	}
	var group sync.WaitGroup
	for range 32 {
		group.Go(func() {
			for range 32 {
				_, _ = writer.Write([]byte("abcdefghij"))
			}
		})
	}
	group.Wait()
	if writer.BytesWritten() != 257 || sink.Len() != 257 {
		t.Fatalf("concurrent bytes = observed:%d sink:%d, want exact capacity 257", writer.BytesWritten(), sink.Len())
	}
}

type hostileWriter struct {
	err error
	n   int
}

func (w hostileWriter) Write([]byte) (int, error) { return w.n, w.err }

type lockedBuffer struct {
	bytes.Buffer
	mu sync.Mutex
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(data)
}

func (b *lockedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Len()
}
