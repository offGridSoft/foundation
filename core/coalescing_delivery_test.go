package core

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func TestDeliveryPhaseJSONBoundaryTable(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name    string
		encoded string
		want    DeliveryPhase
		wantErr bool
	}{
		{name: "pending", encoded: `"pending"`, want: DeliveryPhasePending},
		{name: "in flight", encoded: `"in_flight"`, want: DeliveryPhaseInFlight},
		{name: "acknowledged", encoded: `"acknowledged"`, want: DeliveryPhaseAcknowledged},
		{name: "unknown token", encoded: `"other"`, wantErr: true},
		{name: "numeric wire value", encoded: `1`, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got DeliveryPhase
			gotErr := json.Unmarshal([]byte(testCase.encoded), &got)
			if testCase.wantErr {
				if !errors.Is(gotErr, ErrDeliveryContract) {
					t.Fatalf("UnmarshalJSON() error = %v, want delivery contract identity", gotErr)
				}
				return
			}
			if gotErr != nil {
				t.Fatalf("UnmarshalJSON() error = %v, want nil", gotErr)
			}
			if got != testCase.want || got.String() == "" || !got.IsValid() {
				t.Fatalf("phase = %v, want %v", got, testCase.want)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v, want nil", err)
			}
			if string(encoded) != testCase.encoded {
				t.Fatalf("MarshalJSON() = %s, want %s", encoded, testCase.encoded)
			}
		})
	}
}

type deliveryFixture struct{ Valid bool }

func (f deliveryFixture) Validate() error {
	if !f.Valid {
		return ErrFoundationContract
	}
	return nil
}

func deliveryBackoffPolicy(base, maximum time.Duration, maxAttempts uint64) BackoffPolicy {
	return BackoffPolicy{
		Base:        NewNanosecondsDuration(base),
		Max:         NewNanosecondsDuration(maximum),
		MaxAttempts: maxAttempts,
	}
}

func TestCoalescingDeliveryTransitionTable(t *testing.T) {
	t.Parallel()

	now := NewUnixNanoTime(time.Unix(10, 0))
	policy := deliveryBackoffPolicy(time.Second, time.Minute, 3)
	valid, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
	if err != nil {
		t.Fatalf("NewCoalescingDelivery() error = %v, want nil", err)
	}
	inFlight, err := valid.Begin(now, policy)
	if err != nil {
		t.Fatalf("Begin() error = %v, want nil", err)
	}
	cases := []struct {
		run     func() error
		name    string
		wantErr bool
	}{
		{name: "pending begins when available", run: func() error { _, gotErr := valid.Begin(now, policy); return gotErr }},
		{name: "pending cannot begin before available", run: func() error {
			future := valid
			future.AvailableAt = now.Add(time.Second)
			_, gotErr := future.Begin(now, policy)
			return gotErr
		}, wantErr: true},
		{name: "pending cannot acknowledge", run: func() error { _, gotErr := valid.Acknowledge(valid.Generation); return gotErr }, wantErr: true},
		{name: "matching in-flight generation acknowledges", run: func() error { _, gotErr := inFlight.Acknowledge(inFlight.Generation); return gotErr }},
		{name: "old generation cannot acknowledge", run: func() error { _, gotErr := inFlight.Acknowledge(inFlight.Generation + 1); return gotErr }, wantErr: true},
		{name: "in-flight retry schedules", run: func() error {
			_, gotErr := inFlight.Retry(inFlight.Generation, now, policy, 1)
			return gotErr
		}},
		{name: "old generation cannot schedule retry", run: func() error {
			_, gotErr := inFlight.Retry(inFlight.Generation+1, now, policy, 1)
			return gotErr
		}, wantErr: true},
		{name: "pending cannot retry", run: func() error {
			_, gotErr := valid.Retry(valid.Generation, now, policy, 1)
			return gotErr
		}, wantErr: true},
		{name: "zero clock cannot begin delivery", run: func() error { _, gotErr := valid.Begin(UnixNanoTime{}, policy); return gotErr }, wantErr: true},
		{name: "exact attempt limit cannot begin delivery", run: func() error {
			exhausted := valid
			exhausted.Attempts = policy.MaxAttempts
			_, gotErr := exhausted.Begin(now, policy)
			return gotErr
		}, wantErr: true},
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
	oldRequest, err := first.Begin(now, deliveryBackoffPolicy(time.Second, time.Minute, 3))
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
	cases := []struct {
		wantErr       error
		name          string
		policy        BackoffPolicy
		priorAttempts uint64
		jitter        float64
		wantDelay     time.Duration
		wantAttempts  uint64
	}{
		{name: "first failure schedules second attempt", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: 1, wantDelay: time.Second, wantAttempts: 1},
		{name: "second failure schedules final attempt", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 1, jitter: 1, wantDelay: 2 * time.Second, wantAttempts: 2},
		{name: "final failure cannot schedule fourth attempt", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 2, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "exact attempt limit cannot begin", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 3, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "one above attempt limit cannot begin", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 4, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "far above attempt limit cannot begin", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: math.MaxUint64 - 1, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "attempt counter exhaustion is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, math.MaxUint64), priorAttempts: math.MaxUint64, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "single attempt policy never retries", policy: deliveryBackoffPolicy(time.Second, time.Second, 1), priorAttempts: 0, jitter: 1, wantErr: ErrDeliveryContract},
		{name: "zero jitter permits immediate durable retry", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: 0, wantDelay: 0, wantAttempts: 1},
		{name: "smallest positive jitter remains bounded", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: math.SmallestNonzeroFloat64, wantDelay: 0, wantAttempts: 1},
		{name: "half jitter scales base delay", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: 0.5, wantDelay: 500 * time.Millisecond, wantAttempts: 1},
		{name: "one jitter includes full ceiling", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 1, jitter: 1, wantDelay: 2 * time.Second, wantAttempts: 2},
		{name: "smallest negative jitter is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: -math.SmallestNonzeroFloat64, wantErr: ErrDeliveryContract},
		{name: "jitter above one is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: math.Nextafter(1, 2), wantErr: ErrDeliveryContract},
		{name: "negative infinity jitter is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: math.Inf(-1), wantErr: ErrDeliveryContract},
		{name: "positive infinity jitter is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: math.Inf(1), wantErr: ErrDeliveryContract},
		{name: "not a number jitter is rejected", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), priorAttempts: 0, jitter: math.NaN(), wantErr: ErrDeliveryContract},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pending, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
			if err != nil {
				t.Fatalf("NewCoalescingDelivery() error = %v, want nil", err)
			}
			pending.Attempts = testCase.priorAttempts
			inFlight, err := pending.Begin(now, testCase.policy)
			if err != nil {
				if errors.Is(err, testCase.wantErr) {
					return
				}
				t.Fatalf("Begin() error = %v, want nil", err)
			}
			got, gotErr := inFlight.Retry(inFlight.Generation, now, testCase.policy, testCase.jitter)
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

func TestCoalescingDeliveryEnforcesExactAttemptBudgetSequence(t *testing.T) {
	t.Parallel()

	now := NewUnixNanoTime(time.Unix(10, 0))
	policy := deliveryBackoffPolicy(time.Nanosecond, time.Nanosecond, 3)
	delivery, err := NewCoalescingDelivery(deliveryFixture{Valid: true}, now)
	if err != nil {
		t.Fatalf("NewCoalescingDelivery() error = %v, want nil", err)
	}
	for wantAttempt := uint64(1); wantAttempt <= policy.MaxAttempts; wantAttempt++ {
		inFlight, beginErr := delivery.Begin(now, policy)
		if beginErr != nil || inFlight.Attempts != wantAttempt || inFlight.Phase != DeliveryPhaseInFlight {
			t.Fatalf("Begin() attempt %d = phase %v attempts %d error %v, want %v/%d/nil", wantAttempt, inFlight.Phase, inFlight.Attempts, beginErr, DeliveryPhaseInFlight, wantAttempt)
		}
		if wantAttempt == policy.MaxAttempts {
			if _, retryErr := inFlight.Retry(inFlight.Generation, now, policy, 1); !errors.Is(retryErr, ErrDeliveryContract) {
				t.Fatalf("Retry() after final attempt error = %v, want %v", retryErr, ErrDeliveryContract)
			}
			exhausted := inFlight
			exhausted.Phase = DeliveryPhasePending
			if _, beginErr := exhausted.Begin(now, policy); !errors.Is(beginErr, ErrDeliveryContract) {
				t.Fatalf("Begin() from exhausted pending state error = %v, want %v", beginErr, ErrDeliveryContract)
			}
			return
		}
		delivery, err = inFlight.Retry(inFlight.Generation, now, policy, 0)
		if err != nil {
			t.Fatalf("Retry() after attempt %d error = %v, want nil", wantAttempt, err)
		}
	}
	t.Fatalf("attempt sequence completed attempts = %d, want terminal proof at %d", delivery.Attempts, policy.MaxAttempts)
}
