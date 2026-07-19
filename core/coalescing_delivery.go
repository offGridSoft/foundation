package core

import (
	"fmt"
	"math"
)

// DeliveryPhase is the compiler-owned durable-delivery state machine.
type DeliveryPhase uint8

const (
	DeliveryPhaseUnknown DeliveryPhase = iota
	DeliveryPhasePending
	DeliveryPhaseInFlight
	DeliveryPhaseAcknowledged
)

func (p DeliveryPhase) Valid() bool {
	return p >= DeliveryPhasePending && p <= DeliveryPhaseAcknowledged
}

func coalescingDeliveryError(cause error) error {
	return fmt.Errorf(ErrFmtCoalescingDelivery, fmt.Errorf("%w: %w", ErrDeliveryContract, cause))
}

// CoalescingDelivery retains exactly the newest valid value. Generation binds
// acknowledgements so an old response can never clear a newer replacement.
type CoalescingDelivery[T Validatable] struct {
	Value       T
	AvailableAt UnixNanoTime
	Generation  uint64
	Attempts    uint64
	Phase       DeliveryPhase
}

func NewCoalescingDelivery[T Validatable](value T, now UnixNanoTime) (CoalescingDelivery[T], error) {
	delivery := CoalescingDelivery[T]{Value: value, AvailableAt: now, Generation: 1, Phase: DeliveryPhasePending}
	return delivery, delivery.Validate()
}

func (d CoalescingDelivery[T]) Validate() error {
	if d.Generation == 0 || !d.Phase.Valid() {
		return coalescingDeliveryError(ErrFoundationContract)
	}
	if err := d.Value.Validate(); err != nil {
		return coalescingDeliveryError(err)
	}
	if err := d.AvailableAt.Validate(); err != nil {
		return coalescingDeliveryError(err)
	}
	return nil
}

func (d CoalescingDelivery[T]) Replace(value T, now UnixNanoTime) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if d.Generation == math.MaxUint64 {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	next := CoalescingDelivery[T]{Value: value, AvailableAt: now, Generation: d.Generation + 1, Phase: DeliveryPhasePending}
	return next, next.Validate()
}

func (d CoalescingDelivery[T]) Begin(now UnixNanoTime) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if err := now.Validate(); err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	if d.Phase != DeliveryPhasePending || now.Before(d.AvailableAt) {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	d.Phase = DeliveryPhaseInFlight
	return d, nil
}

func (d CoalescingDelivery[T]) Retry(generation uint64, now UnixNanoTime, policy BackoffPolicy, jitterFraction float64) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if d.Phase != DeliveryPhaseInFlight || generation != d.Generation || d.Attempts == math.MaxUint64 {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	if err := policy.Validate(); err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	attempt := policy.MaxAttempts - 1
	if d.Attempts < uint64(attempt) {
		attempt = int(d.Attempts)
	}
	delay, err := policy.Delay(attempt, jitterFraction)
	if err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	available, err := AddUnixNanoDuration(now, delay)
	if err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	d.Attempts++
	d.AvailableAt = available
	d.Phase = DeliveryPhasePending
	return d, d.Validate()
}

func (d CoalescingDelivery[T]) Acknowledge(generation uint64) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if d.Phase != DeliveryPhaseInFlight || generation != d.Generation {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	d.Phase = DeliveryPhaseAcknowledged
	return d, nil
}
