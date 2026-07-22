package shutdown

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

var errForcedCleanup = errors.New("forced cleanup failure")

type unknownOperatingSystemSignal struct{}

func (unknownOperatingSystemSignal) Signal()        {}
func (unknownOperatingSystemSignal) String() string { return "hostile-unknown-signal" }

func TestSignalPolicyRejectsEveryIncoherentBoundary(t *testing.T) {
	t.Parallel()

	forceBudget := core.NewNanosecondsDuration(time.Second)
	gracePeriod := core.NewNanosecondsDuration(time.Minute)
	valid := []SignalPolicy{
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryDisabled},
		{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryDisabled, ForceBudget: forceBudget},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryForce, GracePeriod: gracePeriod, ForceBudget: forceBudget},
		{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryForce, GracePeriod: gracePeriod, ForceBudget: forceBudget},
	}
	for index, policy := range valid {
		if err := policy.Validate(); err != nil {
			t.Fatalf("valid SignalPolicy %d Validate() error = %v", index, err)
		}
	}

	invalid := []SignalPolicy{
		{},
		{SecondSignal: SecondSignalUnknown, GraceExpiry: GraceExpiryDisabled},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryUnknown},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryDisabled, GracePeriod: gracePeriod},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryDisabled, ForceBudget: forceBudget},
		{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryDisabled},
		{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryDisabled, GracePeriod: gracePeriod, ForceBudget: forceBudget},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryForce, ForceBudget: forceBudget},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryForce, GracePeriod: gracePeriod},
		{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryForce, GracePeriod: core.NanosecondsDurationFromInt64(-1), ForceBudget: forceBudget},
		{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryDisabled, ForceBudget: core.NanosecondsDurationFromInt64(-1)},
	}
	for index, policy := range invalid {
		if err := policy.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid SignalPolicy %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
	}
}

func TestWatchFiltersUnknownSignalAndCancelsWithTypedCause(t *testing.T) {
	t.Parallel()

	unknownSignals := make(chan os.Signal, 1)
	unknownController, err := Watch(WatchRequest{Parent: t.Context(), Signals: unknownSignals, Policy: defaultSignalPolicy()})
	if err != nil {
		t.Fatalf("Watch(unknown filter) error = %v", err)
	}
	unknownSignals <- unknownOperatingSystemSignal{}
	close(unknownSignals)
	<-unknownController.Done()
	if !errors.Is(context.Cause(unknownController.Context()), core.ErrShutdownSignalSourceClosed) {
		t.Fatalf("unknown then closed cause = %v, want ErrShutdownSignalSourceClosed proving unknown was filtered", context.Cause(unknownController.Context()))
	}

	signals := make(chan os.Signal, 1)
	released := make(chan struct{}, 1)
	request := WatchRequest{
		Parent:  t.Context(),
		Signals: signals,
		Policy:  defaultSignalPolicy(),
	}
	controller, err := newController(request, func() { released <- struct{}{} })
	if err != nil {
		t.Fatalf("newController() error = %v", err)
	}
	signals <- firstSupportedOperatingSystemSignal()
	<-controller.Context().Done()
	var cause SignalCause
	if !errors.As(context.Cause(controller.Context()), &cause) || !errors.Is(context.Cause(controller.Context()), core.ErrShutdownSignalReceived) {
		t.Fatalf("controller cause = %v, want SignalCause and ErrShutdownSignalReceived", context.Cause(controller.Context()))
	}
	if cause.Kind != SignalKindInterrupt || controller.Context().Err() != context.Canceled {
		t.Fatalf("controller signal = %s context error = %v, want interrupt/context.Canceled", cause.Kind, controller.Context().Err())
	}
	<-controller.Done()
	if len(released) != 1 {
		t.Fatalf("signal release count = %d, want exactly one", len(released))
	}
	if _, open := <-controller.Forced(); open {
		t.Fatal("Forced channel remains open after controller completion")
	}
}

func TestWatchClosedSourceParentCancellationAndStopHostileTable(t *testing.T) {
	t.Parallel()

	closed := make(chan os.Signal)
	close(closed)
	closedController, err := Watch(WatchRequest{Parent: t.Context(), Signals: closed, Policy: defaultSignalPolicy()})
	if err != nil {
		t.Fatalf("Watch(closed source) error = %v", err)
	}
	<-closedController.Done()
	if !errors.Is(context.Cause(closedController.Context()), core.ErrShutdownSignalSourceClosed) {
		t.Fatalf("closed source cause = %v, want ErrShutdownSignalSourceClosed", context.Cause(closedController.Context()))
	}

	parent, cancelParent := context.WithCancelCause(t.Context())
	parentCause := errors.New("parent shutdown")
	parentController, err := Watch(WatchRequest{Parent: parent, Signals: make(chan os.Signal), Policy: defaultSignalPolicy()})
	if err != nil {
		t.Fatalf("Watch(parent cancellation) error = %v", err)
	}
	cancelParent(parentCause)
	<-parentController.Done()
	if !errors.Is(context.Cause(parentController.Context()), parentCause) {
		t.Fatalf("parent cancellation cause = %v, want %v", context.Cause(parentController.Context()), parentCause)
	}

	stopController, err := Watch(WatchRequest{Parent: t.Context(), Signals: make(chan os.Signal), Policy: defaultSignalPolicy()})
	if err != nil {
		t.Fatalf("Watch(stop) error = %v", err)
	}
	stopController.Stop()
	stopController.Stop()
	<-stopController.Done()
	if !errors.Is(context.Cause(stopController.Context()), context.Canceled) {
		t.Fatalf("Stop() cause = %v, want context.Canceled", context.Cause(stopController.Context()))
	}
}

func TestWatchSecondSignalRunsBoundedTypedForceAction(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 2)
	requests := make(chan ForceRequest, 1)
	controller, err := Watch(WatchRequest{
		Parent:  t.Context(),
		Signals: signals,
		Policy:  forceSignalPolicy(time.Second),
		Force: func(ctx context.Context, request ForceRequest) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			requests <- request
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	signals <- firstSupportedOperatingSystemSignal()
	<-controller.Context().Done()
	signals <- secondSupportedOperatingSystemSignal()
	result := <-controller.Forced()
	request := <-requests
	if result.Validate() != nil || result.Outcome != ForceOutcomeCompleted || result.Request != request {
		t.Fatalf("force result = %+v validate=%v request=%+v, want matching completed state", result, result.Validate(), request)
	}
	if request.Reason != ForceReasonSecondSignal || request.FirstSignal != SignalKindInterrupt || !request.TriggerSignal.IsValid() {
		t.Fatalf("force request = %+v, want typed second-signal escalation", request)
	}
	<-controller.Done()
}

func TestWatchGraceExpiryRunsForceActionWithoutTriggerSignal(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 1)
	requests := make(chan ForceRequest, 1)
	controller, err := Watch(WatchRequest{
		Parent:  t.Context(),
		Signals: signals,
		Policy: SignalPolicy{
			SecondSignal: SecondSignalOperatingSystemDefault,
			GraceExpiry:  GraceExpiryForce,
			GracePeriod:  core.NewNanosecondsDuration(10 * time.Millisecond),
			ForceBudget:  core.NewNanosecondsDuration(time.Second),
		},
		Force: func(_ context.Context, request ForceRequest) error {
			requests <- request
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	signals <- firstSupportedOperatingSystemSignal()
	result := <-controller.Forced()
	request := <-requests
	if result.Outcome != ForceOutcomeCompleted || request.Reason != ForceReasonGraceExpired || request.FirstSignal != SignalKindInterrupt || request.TriggerSignal != SignalKindUnknown {
		t.Fatalf("grace force result/request = %+v/%+v, want completed grace-expired state", result, request)
	}
	<-controller.Done()
}

func TestWatchForceFailurePanicAndTimeoutPreserveTypedIdentity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		budget     time.Duration
		action     ForceAction
		outcome    ForceOutcome
		identity   error
		underlying error
	}{
		{name: "failure", budget: time.Second, action: func(context.Context, ForceRequest) error { return errForcedCleanup }, outcome: ForceOutcomeFailed, identity: core.ErrShutdownForceFailure, underlying: errForcedCleanup},
		{name: "panic", budget: time.Second, action: func(context.Context, ForceRequest) error { panic("hostile force panic") }, outcome: ForceOutcomePanicked, identity: core.ErrShutdownForcePanic},
		{name: "deadline returned", budget: time.Second, action: func(context.Context, ForceRequest) error { return context.DeadlineExceeded }, outcome: ForceOutcomeTimedOut, identity: core.ErrShutdownForceTimeout, underlying: context.DeadlineExceeded},
		{name: "blocked", budget: 10 * time.Millisecond, action: func(ctx context.Context, _ ForceRequest) error { <-ctx.Done(); return context.Cause(ctx) }, outcome: ForceOutcomeTimedOut, identity: core.ErrShutdownForceTimeout, underlying: context.DeadlineExceeded},
		{name: "late success cannot escape budget", budget: 10 * time.Millisecond, action: func(ctx context.Context, _ ForceRequest) error { <-ctx.Done(); return nil }, outcome: ForceOutcomeTimedOut, identity: core.ErrShutdownForceTimeout, underlying: context.DeadlineExceeded},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			signals := make(chan os.Signal, 2)
			controller, err := Watch(WatchRequest{Parent: t.Context(), Signals: signals, Policy: forceSignalPolicy(testCase.budget), Force: testCase.action})
			if err != nil {
				t.Fatalf("Watch() error = %v", err)
			}
			signals <- firstSupportedOperatingSystemSignal()
			signals <- firstSupportedOperatingSystemSignal()
			result := <-controller.Forced()
			if result.Validate() != nil || result.Outcome != testCase.outcome || !errors.Is(result.Err, testCase.identity) {
				t.Fatalf("force result = %+v validate=%v, want outcome %s and identity %v", result, result.Validate(), testCase.outcome, testCase.identity)
			}
			if testCase.underlying != nil && !errors.Is(result.Err, testCase.underlying) {
				t.Fatalf("force result error = %v, want underlying %v", result.Err, testCase.underlying)
			}
			<-controller.Done()
		})
	}
}

func TestSignalRequestsAndResultsRejectImpossibleStates(t *testing.T) {
	t.Parallel()

	validSecond := ForceRequest{Reason: ForceReasonSecondSignal, FirstSignal: SignalKindInterrupt, TriggerSignal: SignalKindTerminate}
	validGrace := ForceRequest{Reason: ForceReasonGraceExpired, FirstSignal: SignalKindInterrupt}
	invalidRequests := []ForceRequest{
		{},
		{Reason: ForceReasonUnknown, FirstSignal: SignalKindInterrupt},
		{Reason: ForceReasonSecondSignal, FirstSignal: SignalKindUnknown, TriggerSignal: SignalKindInterrupt},
		{Reason: ForceReasonSecondSignal, FirstSignal: SignalKindInterrupt, TriggerSignal: SignalKindUnknown},
		{Reason: ForceReasonGraceExpired, FirstSignal: SignalKindInterrupt, TriggerSignal: SignalKindTerminate},
	}
	for index, request := range invalidRequests {
		if err := request.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid ForceRequest %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
	}
	if validSecond.Validate() != nil || validGrace.Validate() != nil {
		t.Fatalf("valid ForceRequest errors = %v/%v", validSecond.Validate(), validGrace.Validate())
	}

	invalidResults := []ForceResult{
		{},
		{Request: validSecond, Outcome: ForceOutcomeUnknown},
		{Request: validSecond, Outcome: ForceOutcomeCompleted, Err: errForcedCleanup},
		{Request: validSecond, Outcome: ForceOutcomeFailed},
		{Request: validSecond, Outcome: ForceOutcomeFailed, Err: core.ErrShutdownForceTimeout},
		{Request: validSecond, Outcome: ForceOutcomeTimedOut, Err: core.ErrShutdownForceFailure},
		{Request: validSecond, Outcome: ForceOutcomePanicked, Err: core.ErrShutdownForceFailure},
	}
	for index, result := range invalidResults {
		if err := result.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid ForceResult %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
	}
	validResults := []ForceResult{
		{Request: validSecond, Outcome: ForceOutcomeCompleted},
		{Request: validSecond, Outcome: ForceOutcomeFailed, Err: core.ErrShutdownForceFailure},
		{Request: validSecond, Outcome: ForceOutcomeTimedOut, Err: core.ErrShutdownForceTimeout},
		{Request: validSecond, Outcome: ForceOutcomePanicked, Err: core.ErrShutdownForcePanic},
	}
	for index, result := range validResults {
		if err := result.Validate(); err != nil {
			t.Fatalf("valid ForceResult %d Validate() error = %v", index, err)
		}
	}
}

func TestSignalRequestValidationRejectsNilAndForceMismatch(t *testing.T) {
	t.Parallel()

	forcePolicy := forceSignalPolicy(time.Second)
	force := func(context.Context, ForceRequest) error { return nil }
	watchCases := []WatchRequest{
		{},
		{Parent: t.Context(), Policy: defaultSignalPolicy()},
		{Signals: make(chan os.Signal), Policy: defaultSignalPolicy()},
		{Parent: t.Context(), Signals: make(chan os.Signal)},
		{Parent: t.Context(), Signals: make(chan os.Signal), Policy: forcePolicy},
		{Parent: t.Context(), Signals: make(chan os.Signal), Policy: defaultSignalPolicy(), Force: force},
	}
	for index, request := range watchCases {
		if err := request.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid WatchRequest %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
		if _, err := Watch(request); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("Watch(invalid %d) error = %v, want ErrShutdownContract", index, err)
		}
	}
	notifyCases := []NotifyRequest{
		{},
		{Parent: t.Context(), Set: SignalSetUnknown, Policy: defaultSignalPolicy()},
		{Parent: t.Context(), Set: SignalSetStandard},
		{Parent: t.Context(), Set: SignalSetStandard, Policy: forcePolicy},
		{Parent: t.Context(), Set: SignalSetStandard, Policy: defaultSignalPolicy(), Force: force},
	}
	for index, request := range notifyCases {
		if err := request.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid NotifyRequest %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
		if _, err := Notify(request); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("Notify(invalid %d) error = %v, want ErrShutdownContract", index, err)
		}
	}
}

func defaultSignalPolicy() SignalPolicy {
	return SignalPolicy{SecondSignal: SecondSignalOperatingSystemDefault, GraceExpiry: GraceExpiryDisabled}
}

func forceSignalPolicy(budget time.Duration) SignalPolicy {
	return SignalPolicy{SecondSignal: SecondSignalForce, GraceExpiry: GraceExpiryDisabled, ForceBudget: core.NewNanosecondsDuration(budget)}
}

func firstSupportedOperatingSystemSignal() os.Signal {
	return os.Interrupt
}

func secondSupportedOperatingSystemSignal() os.Signal {
	signals := operatingSystemSignals(SignalSetStandard)
	if len(signals) == 0 {
		return os.Interrupt
	}
	return signals[len(signals)-1]
}
