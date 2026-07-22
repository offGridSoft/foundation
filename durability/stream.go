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

const errFmtBoundedFileNotRegular = "bounded durable file is not a regular file: %w"

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
	if err := requireRegularFile(request.Path.String()); err != nil {
		return nil, err
	}
	closed := false
	file, err := openBoundedFile(request.Path.String())
	if err != nil {
		return nil, fmt.Errorf("open bounded durable file: %w", err)
	}
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if err := requireOpenRegularFile(file); err != nil {
		closeErr := file.Close()
		closed = true
		return nil, errors.Join(err, closeErr)
	}
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

func requireRegularFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat bounded durable file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf(errFmtBoundedFileNotRegular, core.ErrDurabilityContract)
	}
	return nil
}

func requireOpenRegularFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat open bounded durable file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf(errFmtBoundedFileNotRegular, core.ErrDurabilityContract)
	}
	return nil
}
