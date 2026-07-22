package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type SignalPolicy struct {
	SecondSignal SecondSignalAction
	GraceExpiry  GraceExpiryAction
	GracePeriod  core.NanosecondsDuration
	ForceBudget  core.NanosecondsDuration
}

func (p SignalPolicy) Validate() error {
	if err := p.SecondSignal.Validate(); err != nil {
		return err
	}
	if err := p.GraceExpiry.Validate(); err != nil {
		return err
	}
	if err := p.GracePeriod.Validate(); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	if err := p.ForceBudget.Validate(); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	forceEnabled := p.SecondSignal == SecondSignalForce || p.GraceExpiry == GraceExpiryForce
	if forceEnabled != !p.ForceBudget.IsZero() {
		return core.ErrShutdownContract
	}
	if (p.GraceExpiry == GraceExpiryForce) != !p.GracePeriod.IsZero() {
		return core.ErrShutdownContract
	}
	return nil
}

type ForceRequest struct {
	Reason        ForceReason
	FirstSignal   SignalKind
	TriggerSignal SignalKind
}

func (r ForceRequest) Validate() error {
	if err := r.Reason.Validate(); err != nil {
		return err
	}
	if err := r.FirstSignal.Validate(); err != nil {
		return err
	}
	if r.Reason == ForceReasonSecondSignal {
		return r.TriggerSignal.Validate()
	}
	if r.TriggerSignal != SignalKindUnknown {
		return core.ErrShutdownContract
	}
	return nil
}

type ForceAction func(context.Context, ForceRequest) error

type ForceResult struct {
	Err     error
	Request ForceRequest
	Outcome ForceOutcome
}

func (r ForceResult) Validate() error {
	if err := r.Request.Validate(); err != nil {
		return err
	}
	if err := r.Outcome.Validate(); err != nil {
		return err
	}
	return validateForceResultError(r)
}

func validateForceResultError(result ForceResult) error {
	if result.Outcome == ForceOutcomeCompleted {
		if result.Err != nil {
			return core.ErrShutdownContract
		}
		return nil
	}
	if result.Err == nil {
		return core.ErrShutdownContract
	}
	identity := forceOutcomeIdentity(result.Outcome)
	if identity == nil || !errors.Is(result.Err, identity) {
		return core.ErrShutdownContract
	}
	return nil
}

func forceOutcomeIdentity(outcome ForceOutcome) error {
	switch outcome {
	case ForceOutcomeFailed:
		return core.ErrShutdownForceFailure
	case ForceOutcomeTimedOut:
		return core.ErrShutdownForceTimeout
	case ForceOutcomePanicked:
		return core.ErrShutdownForcePanic
	default:
		return nil
	}
}

type SignalCause struct {
	Kind SignalKind
}

func (e SignalCause) Validate() error { return e.Kind.Validate() }
func (e SignalCause) Error() string   { return fmt.Sprintf("shutdown signal: %s", e.Kind) }
func (e SignalCause) Unwrap() error   { return core.ErrShutdownSignalReceived }

type WatchRequest struct {
	Parent  context.Context
	Signals <-chan os.Signal
	Force   ForceAction
	Policy  SignalPolicy
}

func (r WatchRequest) Validate() error {
	if r.Parent == nil || r.Signals == nil {
		return core.ErrShutdownContract
	}
	if err := r.Policy.Validate(); err != nil {
		return err
	}
	forceEnabled := r.Policy.SecondSignal == SecondSignalForce || r.Policy.GraceExpiry == GraceExpiryForce
	if forceEnabled != (r.Force != nil) {
		return core.ErrShutdownContract
	}
	return nil
}

type NotifyRequest struct {
	Parent context.Context
	Force  ForceAction
	Policy SignalPolicy
	Set    SignalSet
}

func (r NotifyRequest) Validate() error {
	if r.Parent == nil {
		return core.ErrShutdownContract
	}
	if err := r.Set.Validate(); err != nil {
		return err
	}
	if err := r.Policy.Validate(); err != nil {
		return err
	}
	forceEnabled := r.Policy.SecondSignal == SecondSignalForce || r.Policy.GraceExpiry == GraceExpiryForce
	if forceEnabled != (r.Force != nil) {
		return core.ErrShutdownContract
	}
	return nil
}

type Controller struct {
	ctx            context.Context
	cancel         context.CancelCauseFunc
	stop           chan struct{}
	done           chan struct{}
	forced         chan ForceResult
	releaseSignals func()
	stopOnce       sync.Once
	releaseOnce    sync.Once
}

func Notify(request NotifyRequest) (*Controller, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	signals := make(chan os.Signal, core.ShutdownSignalBuffer)
	registered := operatingSystemSignals(request.Set)
	if len(registered) == 0 {
		return nil, core.ErrShutdownContract
	}
	signal.Notify(signals, registered...)
	release := func() { signal.Stop(signals) }
	controller, err := newController(WatchRequest{Parent: request.Parent, Signals: signals, Policy: request.Policy, Force: request.Force}, release)
	if err != nil {
		release()
		return nil, err
	}
	return controller, nil
}

func Watch(request WatchRequest) (*Controller, error) {
	return newController(request, func() {})
}

func newController(request WatchRequest, release func()) (*Controller, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancelCause(request.Parent)
	controller := &Controller{
		ctx:            ctx,
		cancel:         cancel,
		stop:           make(chan struct{}, 1),
		done:           make(chan struct{}, 1),
		forced:         make(chan ForceResult, 1),
		releaseSignals: release,
	}
	go controller.run(request)
	return controller, nil
}

func (c *Controller) Context() context.Context {
	return c.ctx
}

func (c *Controller) Done() <-chan struct{} {
	return c.done
}

func (c *Controller) Forced() <-chan ForceResult {
	return c.forced
}

func (c *Controller) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
		c.release()
		c.cancel(context.Canceled)
	})
}

func (c *Controller) release() {
	c.releaseOnce.Do(c.releaseSignals)
}

func (c *Controller) run(request WatchRequest) {
	defer close(c.done)
	defer close(c.forced)
	defer c.release()
	first, ok := c.waitForSignal(request.Parent, request.Signals)
	if !ok {
		return
	}
	c.cancel(SignalCause{Kind: first})
	if request.Policy.SecondSignal == SecondSignalOperatingSystemDefault {
		c.release()
	}
	if request.Policy.SecondSignal == SecondSignalOperatingSystemDefault && request.Policy.GraceExpiry == GraceExpiryDisabled {
		return
	}
	c.waitForEscalation(request, first)
}

func (c *Controller) waitForSignal(parent context.Context, signals <-chan os.Signal) (SignalKind, bool) {
	for {
		select {
		case <-c.stop:
			return SignalKindUnknown, false
		case <-parent.Done():
			c.cancel(context.Cause(parent))
			return SignalKindUnknown, false
		case observed, open := <-signals:
			if !open {
				c.cancel(core.ErrShutdownSignalSourceClosed)
				return SignalKindUnknown, false
			}
			kind := classifyOperatingSystemSignal(observed)
			if kind.IsValid() {
				return kind, true
			}
		}
	}
}

func (c *Controller) waitForEscalation(request WatchRequest, first SignalKind) {
	grace, stopGrace := graceDeadline(request.Policy)
	defer stopGrace()
	signals := request.Signals
	if request.Policy.SecondSignal == SecondSignalOperatingSystemDefault {
		signals = nil
	}
	for {
		select {
		case <-c.stop:
			return
		case <-request.Parent.Done():
			return
		case <-grace:
			c.force(request, ForceRequest{Reason: ForceReasonGraceExpired, FirstSignal: first})
			return
		case observed, open := <-signals:
			if !open {
				return
			}
			kind := classifyOperatingSystemSignal(observed)
			if !kind.IsValid() {
				continue
			}
			if request.Policy.SecondSignal == SecondSignalForce {
				c.force(request, ForceRequest{Reason: ForceReasonSecondSignal, FirstSignal: first, TriggerSignal: kind})
			}
			return
		}
	}
}

func graceDeadline(policy SignalPolicy) (<-chan time.Time, func()) {
	if policy.GraceExpiry == GraceExpiryDisabled {
		return nil, func() {}
	}
	timer := time.NewTimer(policy.GracePeriod.Duration())
	return timer.C, func() { timer.Stop() }
}

func (c *Controller) force(request WatchRequest, forceRequest ForceRequest) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(request.Parent), request.Policy.ForceBudget.Duration())
	defer cancel()
	result := make(chan error, 1)
	go callForceAction(ctx, request.Force, forceRequest, result)
	var outcome ForceResult
	select {
	case err := <-result:
		outcome = forceResultFromError(ctx, forceRequest, err)
	case <-ctx.Done():
		outcome = ForceResult{Request: forceRequest, Outcome: ForceOutcomeTimedOut, Err: errors.Join(core.ErrShutdownForceTimeout, context.DeadlineExceeded)}
	}
	c.forced <- outcome
}

func callForceAction(ctx context.Context, action ForceAction, request ForceRequest, result chan<- error) {
	runCapturingPanic(
		func() error { return action(ctx, request) },
		result,
		func(value PanicValue) error { return newForcePanicError(value) },
	)
}

func forceResultFromError(ctx context.Context, request ForceRequest, err error) ForceResult {
	if ctx.Err() != nil {
		return ForceResult{Request: request, Outcome: ForceOutcomeTimedOut, Err: errors.Join(core.ErrShutdownForceTimeout, context.DeadlineExceeded)}
	}
	if err == nil {
		return ForceResult{Request: request, Outcome: ForceOutcomeCompleted}
	}
	panicErr, isPanic := errors.AsType[ForcePanicError](err)
	if isPanic && panicErr.Validate() == nil {
		return ForceResult{Request: request, Outcome: ForceOutcomePanicked, Err: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ForceResult{Request: request, Outcome: ForceOutcomeTimedOut, Err: errors.Join(core.ErrShutdownForceTimeout, context.DeadlineExceeded)}
	}
	return ForceResult{Request: request, Outcome: ForceOutcomeFailed, Err: errors.Join(core.ErrShutdownForceFailure, err)}
}
