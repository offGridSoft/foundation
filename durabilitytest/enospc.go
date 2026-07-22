package durabilitytest

import (
	"io"
	"sync"
	"syscall"

	"github.com/offGridSoft/foundation/v2026/core"
)

type ENOSPCConfig struct {
	Writer        io.Writer
	CapacityBytes uint64
}

func (c ENOSPCConfig) Validate() error {
	if c.Writer == nil {
		return core.ErrDurabilityContract
	}
	return nil
}

type ENOSPCWriter struct {
	underlying io.Writer
	capacity   uint64
	written    uint64
	mu         sync.Mutex
}

func NewENOSPCWriter(config ENOSPCConfig) (*ENOSPCWriter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &ENOSPCWriter{underlying: config.Writer, capacity: config.CapacityBytes}, nil
}

func (w *ENOSPCWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.written >= w.capacity {
		return 0, syscall.ENOSPC
	}
	remaining := w.capacity - w.written
	allowed := min(uint64(len(data)), remaining)
	n, err := w.underlying.Write(data[:allowed])
	if n < 0 || uint64(n) > allowed {
		return 0, core.ErrDurableShortWrite
	}
	w.written += uint64(n)
	if err != nil {
		return n, err
	}
	if uint64(n) != allowed {
		return n, core.ErrDurableShortWrite
	}
	if allowed != uint64(len(data)) {
		return n, syscall.ENOSPC
	}
	return n, nil
}

func (w *ENOSPCWriter) BytesWritten() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.written
}

func (w *ENOSPCWriter) CapacityBytes() uint64 {
	return w.capacity
}
