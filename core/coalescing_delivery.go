package core

import (
	"encoding/json"
	"errors"
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

const (
	deliveryPhaseTokenPending      = "pending"
	deliveryPhaseTokenInFlight     = "in_flight"
	deliveryPhaseTokenAcknowledged = "acknowledged"
)

func deliveryPhaseNames() [DeliveryPhaseAcknowledged + 1]string {
	return [...]string{
		DeliveryPhasePending:      deliveryPhaseTokenPending,
		DeliveryPhaseInFlight:     deliveryPhaseTokenInFlight,
		DeliveryPhaseAcknowledged: deliveryPhaseTokenAcknowledged,
	}
}

func (p DeliveryPhase) IsValid() bool {
	return p > DeliveryPhaseUnknown && int(p) < len(deliveryPhaseNames()) && deliveryPhaseNames()[p] != ""
}

func (p DeliveryPhase) String() string {
	if !p.IsValid() {
		return ""
	}
	return deliveryPhaseNames()[p]
}

func (p DeliveryPhase) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtDeliveryPhase, ErrDeliveryContract)
	}
	return nil
}

func ParseDeliveryPhase(value string) (DeliveryPhase, error) {
	for phase := DeliveryPhasePending; int(phase) < len(deliveryPhaseNames()); phase++ {
		if deliveryPhaseNames()[phase] == value {
			return phase, nil
		}
	}
	return DeliveryPhaseUnknown, fmt.Errorf(ErrFmtDeliveryPhase, ErrDeliveryContract)
}

func (p DeliveryPhase) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *DeliveryPhase) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeliveryPhase, ErrDeliveryContract)
	}
	parsed, err := ParseDeliveryPhase(value)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

func coalescingDeliveryError(cause error) error {
	return fmt.Errorf(ErrFmtCoalescingDelivery, errors.Join(ErrDeliveryContract, cause))
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
	if d.Generation == 0 || d.Attempts == math.MaxUint64 {
		return coalescingDeliveryError(ErrFoundationContract)
	}
	if err := d.Phase.Validate(); err != nil {
		return coalescingDeliveryError(err)
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

func (d CoalescingDelivery[T]) Begin(now UnixNanoTime, policy BackoffPolicy) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if err := policy.Validate(); err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	if err := now.Validate(); err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	if d.Phase != DeliveryPhasePending || now.Before(d.AvailableAt) || d.Attempts >= policy.MaxAttempts {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	d.Attempts++
	d.Phase = DeliveryPhaseInFlight
	return d, nil
}

func (d CoalescingDelivery[T]) Retry(generation uint64, now UnixNanoTime, policy BackoffPolicy, jitterFraction float64) (CoalescingDelivery[T], error) {
	if err := d.Validate(); err != nil {
		return CoalescingDelivery[T]{}, err
	}
	if err := policy.Validate(); err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	if d.Phase != DeliveryPhaseInFlight || generation != d.Generation || d.Attempts == 0 || d.Attempts >= policy.MaxAttempts {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(ErrFoundationContract)
	}
	delay, err := policy.Delay(d.Attempts-1, jitterFraction)
	if err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
	available, err := AddUnixNanoDuration(now, delay.Duration())
	if err != nil {
		return CoalescingDelivery[T]{}, coalescingDeliveryError(err)
	}
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
