package durability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

type ContextReader struct {
	Context context.Context
	Reader  io.Reader
}

func (r ContextReader) Validate() error {
	if err := contextcheck.Validate(r.Context); err != nil {
		return err
	}
	if r.Reader == nil {
		return core.ErrDurabilityContract
	}
	return nil
}

func (r ContextReader) Read(data []byte) (int, error) {
	if err := r.Validate(); err != nil {
		return 0, err
	}
	return r.Reader.Read(data)
}

type ReadRequest struct {
	Path         core.AbsoluteFilePath
	MaximumBytes core.ByteCount
}

func (r ReadRequest) Validate() error {
	if err := r.Path.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := r.MaximumBytes.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	maximum, err := r.MaximumBytes.Int64()
	if err != nil || maximum == math.MaxInt64 {
		return core.ErrDurabilityContract
	}
	return nil
}

func ReadBounded(ctx context.Context, request ReadRequest) ([]byte, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	maximum, _ := request.MaximumBytes.Int64()
	closed := false
	file, err := os.Open(request.Path.String())
	if err != nil {
		return nil, fmt.Errorf("open bounded durable file: %w", err)
	}
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	data, readErr := io.ReadAll(io.LimitReader(ContextReader{Context: ctx, Reader: file}, maximum+1))
	closeErr := file.Close()
	closed = true
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if int64(len(data)) > maximum {
		return nil, core.ErrDurableSizeLimit
	}
	return data, nil
}
