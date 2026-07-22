package contextcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type contextTestKey uint8

const contextTestValueKey contextTestKey = iota

func TestValidateHostileContextStateTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		makeContext func() (context.Context, context.CancelFunc)
		wantErr     error
	}{
		{name: "nil interface is rejected by foundation identity", makeContext: func() (context.Context, context.CancelFunc) { return nil, func() {} }, wantErr: core.ErrNilContext},
		{name: "background context is active", makeContext: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} }},
		{name: "todo context is active", makeContext: func() (context.Context, context.CancelFunc) { return context.TODO(), func() {} }},
		{name: "value context is active", makeContext: func() (context.Context, context.CancelFunc) {
			return context.WithValue(context.Background(), contextTestValueKey, "value"), func() {}
		}},
		{name: "future deadline remains active", makeContext: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Unix(253402300799, 0))
		}},
		{name: "explicit cancellation preserves standard identity", makeContext: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		}, wantErr: context.Canceled},
		{name: "cancellation cause preserves standard identity", makeContext: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(errors.New("operator stop"))
			return ctx, func() {}
		}, wantErr: context.Canceled},
		{name: "past deadline preserves standard identity", makeContext: func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Unix(1, 0))
		}, wantErr: context.DeadlineExceeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := tc.makeContext()
			defer cancel()
			got := Validate(ctx)
			if !errors.Is(got, tc.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestStateOfErrorHostileIdentityPrecedenceTable(t *testing.T) {
	t.Parallel()

	local := errors.New("local failure")
	cases := []struct {
		name string
		err  error
		want ErrorState
	}{
		{name: "nil has no context identity", want: ErrorStateNone},
		{name: "unrelated error has no context identity", err: local, want: ErrorStateNone},
		{name: "cancellation is closed cancelled state", err: context.Canceled, want: ErrorStateCancelled},
		{name: "wrapped cancellation remains cancelled", err: fmt.Errorf("worker: %w", context.Canceled), want: ErrorStateCancelled},
		{name: "joined cancellation remains cancelled", err: errors.Join(local, context.Canceled), want: ErrorStateCancelled},
		{name: "deadline is closed deadline state", err: context.DeadlineExceeded, want: ErrorStateDeadlineExceeded},
		{name: "wrapped deadline remains deadline", err: fmt.Errorf("worker: %w", context.DeadlineExceeded), want: ErrorStateDeadlineExceeded},
		{name: "joined deadline remains deadline", err: errors.Join(local, context.DeadlineExceeded), want: ErrorStateDeadlineExceeded},
		{name: "cancellation outranks joined deadline", err: errors.Join(context.DeadlineExceeded, context.Canceled), want: ErrorStateCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := StateOfError(tc.err)
			if got != tc.want {
				t.Fatalf("StateOfError() = %v, want %v", got, tc.want)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("StateOfError().Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestErrorStateExhaustsClosedEnumAndRejectsFutureValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		state   ErrorState
		want    string
		wantErr error
	}{
		{name: "zero value is rejected", state: ErrorStateUnknown, want: ErrorStateNameUnknown, wantErr: core.ErrContextContract},
		{name: "no identity is valid", state: ErrorStateNone, want: ErrorStateNameNone},
		{name: "cancelled identity is valid", state: ErrorStateCancelled, want: ErrorStateNameCancelled},
		{name: "deadline identity is valid", state: ErrorStateDeadlineExceeded, want: ErrorStateNameDeadlineExceeded},
		{name: "one above domain is rejected", state: ErrorStateDeadlineExceeded + 1, want: ErrorStateNameUnknown, wantErr: core.ErrContextContract},
		{name: "maximum future value is rejected", state: ErrorState(^uint8(0)), want: ErrorStateNameUnknown, wantErr: core.ErrContextContract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.state.String()
			if got != tc.want {
				t.Fatalf("ErrorState.String() = %q, want %q", got, tc.want)
			}
			if err := tc.state.Validate(); !errors.Is(err, tc.wantErr) {
				t.Fatalf("ErrorState.Validate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestErrorStateHostileJSONBoundaryTable(t *testing.T) {
	t.Parallel()

	noneJSON, err := json.Marshal(ErrorStateNameNone)
	if err != nil {
		t.Fatalf("json.Marshal(none token) error = %v, want nil", err)
	}
	cancelledJSON, err := json.Marshal(ErrorStateNameCancelled)
	if err != nil {
		t.Fatalf("json.Marshal(cancelled token) error = %v, want nil", err)
	}
	deadlineJSON, err := json.Marshal(ErrorStateNameDeadlineExceeded)
	if err != nil {
		t.Fatalf("json.Marshal(deadline token) error = %v, want nil", err)
	}
	cases := []struct {
		name    string
		raw     []byte
		want    ErrorState
		wantErr error
	}{
		{name: "none token round trips", raw: noneJSON, want: ErrorStateNone},
		{name: "cancelled token round trips", raw: cancelledJSON, want: ErrorStateCancelled},
		{name: "deadline token round trips", raw: deadlineJSON, want: ErrorStateDeadlineExceeded},
		{name: "zero bytes are rejected", raw: []byte{}, wantErr: core.ErrContextContract},
		{name: "truncated string is rejected", raw: []byte(`"cancelled`), wantErr: core.ErrContextContract},
		{name: "empty token is rejected", raw: []byte(`""`), wantErr: core.ErrContextContract},
		{name: "unknown token is rejected", raw: []byte(`"future"`), wantErr: core.ErrContextContract},
		{name: "case mutation is rejected", raw: []byte(`"Cancelled"`), wantErr: core.ErrContextContract},
		{name: "number is rejected", raw: []byte(`1`), wantErr: core.ErrContextContract},
		{name: "boolean is rejected", raw: []byte(`true`), wantErr: core.ErrContextContract},
		{name: "null is rejected", raw: []byte(`null`), wantErr: core.ErrContextContract},
		{name: "array is rejected", raw: []byte(`[]`), wantErr: core.ErrContextContract},
		{name: "object is rejected", raw: []byte(`{}`), wantErr: core.ErrContextContract},
		{name: "trailing value is rejected", raw: []byte(`"none" true`), wantErr: core.ErrContextContract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ErrorStateCancelled
			err := got.UnmarshalJSON(tc.raw)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("json.Unmarshal(%q) error = %v, want %v", tc.raw, err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if got != ErrorStateCancelled {
					t.Fatalf("json.Unmarshal(%q) mutated receiver to %v, want %v", tc.raw, got, ErrorStateCancelled)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("json.Unmarshal(%q) = %v, want %v", tc.raw, got, tc.want)
			}
			encoded, marshalErr := json.Marshal(got)
			if marshalErr != nil || string(encoded) != string(tc.raw) {
				t.Fatalf("json.Marshal(%v) = (%q, %v), want (%q, nil)", got, encoded, marshalErr, tc.raw)
			}
		})
	}

	var nilState *ErrorState
	if err := nilState.UnmarshalJSON(noneJSON); !errors.Is(err, core.ErrContextContract) {
		t.Fatalf("nil ErrorState.UnmarshalJSON() error = %v, want %v", err, core.ErrContextContract)
	}
	invalidStates := []ErrorState{ErrorStateUnknown, ErrorStateDeadlineExceeded + 1, ErrorState(^uint8(0))}
	for _, state := range invalidStates {
		if _, err := json.Marshal(state); !errors.Is(err, core.ErrContextContract) {
			t.Fatalf("json.Marshal(%v) error = %v, want %v", state, err, core.ErrContextContract)
		}
	}
}
