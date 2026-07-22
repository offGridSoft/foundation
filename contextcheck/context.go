package contextcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// ErrorState is the closed context identity carried by an execution error.
// Cancellation wins when a joined error contains both standard identities;
// shutdown ownership must outrank an elapsed child deadline.
type ErrorState uint8

const (
	ErrorStateUnknown ErrorState = iota
	ErrorStateNone
	ErrorStateCancelled
	ErrorStateDeadlineExceeded
)

const (
	ErrorStateNameUnknown          = "unknown"
	ErrorStateNameNone             = "none"
	ErrorStateNameCancelled        = "cancelled"
	ErrorStateNameDeadlineExceeded = "deadline-exceeded"
	ErrFmtErrorState               = "contextcheck.ErrorState: %w"
)

func (s ErrorState) IsValid() bool {
	return s >= ErrorStateNone && s <= ErrorStateDeadlineExceeded
}

func (s ErrorState) Validate() error {
	if !s.IsValid() {
		return core.ErrContextContract
	}
	return nil
}

func (s ErrorState) String() string {
	switch s {
	case ErrorStateNone:
		return ErrorStateNameNone
	case ErrorStateCancelled:
		return ErrorStateNameCancelled
	case ErrorStateDeadlineExceeded:
		return ErrorStateNameDeadlineExceeded
	default:
		return ErrorStateNameUnknown
	}
}

func (s ErrorState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf(ErrFmtErrorState, err)
	}
	return json.Marshal(s.String())
}

func (s *ErrorState) UnmarshalJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf(ErrFmtErrorState, core.ErrContextContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtErrorState, errors.Join(core.ErrContextContract, err))
	}
	parsed, err := ParseErrorState(token)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func ParseErrorState(token string) (ErrorState, error) {
	switch token {
	case ErrorStateNameNone:
		return ErrorStateNone, nil
	case ErrorStateNameCancelled:
		return ErrorStateCancelled, nil
	case ErrorStateNameDeadlineExceeded:
		return ErrorStateDeadlineExceeded, nil
	default:
		return ErrorStateUnknown, fmt.Errorf(ErrFmtErrorState, core.ErrContextContract)
	}
}

// StateOfError projects standard context identities into one closed state.
func StateOfError(err error) ErrorState {
	if errors.Is(err, context.Canceled) {
		return ErrorStateCancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorStateDeadlineExceeded
	}
	return ErrorStateNone
}

// Validate rejects nil, cancelled, and expired contexts at execution and I/O
// ingress while preserving the strongest standard error identity.
func Validate(ctx context.Context) error {
	if ctx == nil {
		return core.ErrNilContext
	}
	return ctx.Err()
}
