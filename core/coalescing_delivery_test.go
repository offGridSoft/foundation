package core

import (
	"errors"
	"math"
	"testing"
	"time"
)

type deliveryFixture struct{ Valid bool }

func (f deliveryFixture) Validate() error {
	if !f.Valid {
		return ErrFoundationContract
	}
	return nil
}

func TestCoalescingDeliveryTransitionTable(t *testing.T) {
	t.Parallel()

	now := NewUnixNanoTime(time.Unix(10, 0))
	valid, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
	if err != nil {
		t.Fatalf("NewCoalescingDelivery() error = %v, want nil", err)
	}
	inFlight, err := valid.Begin(now)
	if err != nil {
		t.Fatalf("Begin() error = %v, want nil", err)
	}
	cases := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{name: "pending begins when available", run: func() error { _, gotErr := valid.Begin(now); return gotErr }},
		{name: "pending cannot begin before available", run: func() error {
			future := valid
			future.AvailableAt = now.Add(time.Second)
			_, gotErr := future.Begin(now)
			return gotErr
		}, wantErr: true},
		{name: "pending cannot acknowledge", run: func() error { _, gotErr := valid.Acknowledge(valid.Generation); return gotErr }, wantErr: true},
		{name: "matching in-flight generation acknowledges", run: func() error { _, gotErr := inFlight.Acknowledge(inFlight.Generation); return gotErr }},
		{name: "old generation cannot acknowledge", run: func() error { _, gotErr := inFlight.Acknowledge(inFlight.Generation + 1); return gotErr }, wantErr: true},
		{name: "in-flight retry schedules", run: func() error {
			_, gotErr := inFlight.Retry(inFlight.Generation, now, BackoffPolicy{Base: time.Second, Max: time.Minute, MaxAttempts: 3}, 1)
			return gotErr
		}},
		{name: "old generation cannot schedule retry", run: func() error {
			_, gotErr := inFlight.Retry(inFlight.Generation+1, now, BackoffPolicy{Base: time.Second, Max: time.Minute, MaxAttempts: 3}, 1)
			return gotErr
		}, wantErr: true},
		{name: "pending cannot retry", run: func() error {
			_, gotErr := valid.Retry(valid.Generation, now, BackoffPolicy{Base: time.Second, Max: time.Minute, MaxAttempts: 3}, 1)
			return gotErr
		}, wantErr: true},
		{name: "zero clock cannot begin delivery", run: func() error { _, gotErr := valid.Begin(UnixNanoTime{}); return gotErr }, wantErr: true},
		{name: "invalid replacement rejected", run: func() error { _, gotErr := valid.Replace(deliveryFixture{}, now); return gotErr }, wantErr: true},
		{name: "generation exhaustion rejected", run: func() error {
			exhausted := valid
			exhausted.Generation = math.MaxUint64
			_, gotErr := exhausted.Replace(deliveryFixture{Valid: true}, now)
			return gotErr
		}, wantErr: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotErr := testCase.run()
			if testCase.wantErr != errors.Is(gotErr, ErrDeliveryContract) {
				t.Fatalf("transition error = %v, errors.Is delivery contract = %v, want %v", gotErr, errors.Is(gotErr, ErrDeliveryContract), testCase.wantErr)
			}
		})
	}
}

func TestCoalescingDeliveryReplacementInvalidatesOlderAcknowledgement(t *testing.T) {
	t.Parallel()

	now := NewUnixNanoTime(time.Unix(10, 0))
	first, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	oldRequest, err := first.Begin(now)
	if err != nil {
		t.Fatal(err)
	}
	newest, err := oldRequest.Replace(deliveryFixture{Valid: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newest.Acknowledge(oldRequest.Generation); !errors.Is(err, ErrDeliveryContract) {
		t.Fatalf("Acknowledge(old generation) error = %v, want delivery contract identity", err)
	}
	if newest.Generation != oldRequest.Generation+1 || newest.Phase != DeliveryPhasePending {
		t.Fatalf("replacement = %+v, want next generation pending", newest)
	}
}

func TestCoalescingDeliveryRetryBackoffTable(t *testing.T) {
	t.Parallel()

	now := NewUnixNanoTime(time.Unix(10, 0))
	policy := BackoffPolicy{Base: time.Second, Max: 4 * time.Second, MaxAttempts: 3}
	cases := []struct {
		name         string
		attempts     uint64
		jitter       float64
		wantDelay    time.Duration
		wantAttempts uint64
		wantErr      error
	}{
		{name: "first failure schedules full base delay", attempts: 0, jitter: 1, wantDelay: time.Second, wantAttempts: 1},
		{name: "second failure doubles full delay", attempts: 1, jitter: 1, wantDelay: 2 * time.Second, wantAttempts: 2},
		{name: "third failure reaches maximum delay", attempts: 2, jitter: 1, wantDelay: 4 * time.Second, wantAttempts: 3},
		{name: "later failure remains at maximum delay", attempts: 9, jitter: 1, wantDelay: 4 * time.Second, wantAttempts: 10},
		{name: "zero jitter permits immediate durable retry", attempts: 0, jitter: 0, wantDelay: 0, wantAttempts: 1},
		{name: "half jitter scales base delay", attempts: 0, jitter: 0.5, wantDelay: 500 * time.Millisecond, wantAttempts: 1},
		{name: "negative jitter is rejected", attempts: 0, jitter: -0.01, wantErr: ErrDeliveryContract},
		{name: "jitter above one is rejected", attempts: 0, jitter: 1.01, wantErr: ErrDeliveryContract},
		{name: "attempt counter exhaustion is rejected", attempts: math.MaxUint64, jitter: 1, wantErr: ErrDeliveryContract},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pending, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
			if err != nil {
				t.Fatalf("NewCoalescingDelivery() error = %v, want nil", err)
			}
			pending.Attempts = testCase.attempts
			inFlight, err := pending.Begin(now)
			if err != nil {
				t.Fatalf("Begin() error = %v, want nil", err)
			}
			got, gotErr := inFlight.Retry(inFlight.Generation, now, policy, testCase.jitter)
			if !errors.Is(gotErr, testCase.wantErr) {
				t.Fatalf("Retry() error = %v, want %v", gotErr, testCase.wantErr)
			}
			if testCase.wantErr != nil {
				return
			}
			if got.Phase != DeliveryPhasePending || got.Attempts != testCase.wantAttempts {
				t.Fatalf("Retry() phase/attempts = %v/%d, want %v/%d", got.Phase, got.Attempts, DeliveryPhasePending, testCase.wantAttempts)
			}
			if gotDelay := got.AvailableAt.Sub(now); gotDelay != testCase.wantDelay {
				t.Fatalf("Retry() delay = %v, want %v", gotDelay, testCase.wantDelay)
			}
		})
	}
}
