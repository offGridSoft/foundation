package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestPlanStepPanicPreservesPanicValueDiagnostics(t *testing.T) {
	t.Parallel()

	id, err := ParseStepID("wedged")
	if err != nil {
		t.Fatalf("ParseStepID() error = %v", err)
	}
	plan := mustTestPlan(t, testPlanPolicy(time.Second, time.Second), Step{ID: id, Run: func(context.Context) error { panic("disk handle wedged") }})
	report, runErr := plan.Run(t.Context())
	if !errors.Is(runErr, core.ErrShutdownStepPanic) {
		t.Fatalf("Plan.Run() error = %v, want ErrShutdownStepPanic identity", runErr)
	}
	result := report.Results[0]
	if result.Outcome != StepOutcomePanicked || result.Validate() != nil {
		t.Fatalf("panicked result = %+v validate=%v, want valid panicked outcome", result, result.Validate())
	}
	var panicErr StepPanicError
	if !errors.As(result.Err, &panicErr) || panicErr.Validate() != nil || panicErr.Value.Type != PanicTypeName("string") || panicErr.Value.Diagnostic != PanicDiagnostic(strconv.QuoteToASCII("disk handle wedged")) {
		t.Fatalf("panicked result detail = %+v as=%t validate=%v, want typed string diagnostic", panicErr, errors.As(result.Err, &panicErr), panicErr.Validate())
	}
}

func TestWatchForcePanicPreservesPanicValueDiagnostics(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 2)
	controller, err := Watch(WatchRequest{
		Parent:  t.Context(),
		Signals: signals,
		Policy:  forceSignalPolicy(time.Second),
		Force:   func(context.Context, ForceRequest) error { panic("force ripcord snapped") },
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	signals <- firstSupportedOperatingSystemSignal()
	signals <- firstSupportedOperatingSystemSignal()
	result := <-controller.Forced()
	if result.Outcome != ForceOutcomePanicked || !errors.Is(result.Err, core.ErrShutdownForcePanic) || result.Validate() != nil {
		t.Fatalf("force result = %+v validate=%v, want valid panicked outcome", result, result.Validate())
	}
	var panicErr ForcePanicError
	if !errors.As(result.Err, &panicErr) || panicErr.Validate() != nil || panicErr.Value.Type != PanicTypeName("string") || panicErr.Value.Diagnostic != PanicDiagnostic(strconv.QuoteToASCII("force ripcord snapped")) {
		t.Fatalf("force panic detail = %+v as=%t validate=%v, want typed string diagnostic", panicErr, errors.As(result.Err, &panicErr), panicErr.Validate())
	}
	<-controller.Done()
}

func TestPlanStepSpoofedPanicIdentityCannotClaimRecoveredState(t *testing.T) {
	t.Parallel()

	id, err := ParseStepID("spoofed")
	if err != nil {
		t.Fatalf("ParseStepID() error = %v", err)
	}
	plan := mustTestPlan(t, testPlanPolicy(time.Second, time.Second), Step{ID: id, Run: func(context.Context) error {
		return fmt.Errorf("cleanup wrapper: %w", core.ErrShutdownStepPanic)
	}})
	report, runErr := plan.Run(t.Context())
	if !errors.Is(runErr, core.ErrShutdownStepPanic) || !errors.Is(runErr, core.ErrShutdownStepFailure) {
		t.Fatalf("Plan.Run(spoofed identity) error = %v, want original identity under ErrShutdownStepFailure", runErr)
	}
	if report.Results[0].Outcome != StepOutcomeFailed || report.Validate() != nil {
		t.Fatalf("spoofed result = %+v validate=%v, want failed rather than compiler-owned panicked classification", report.Results[0], report.Validate())
	}
}

func TestForceActionSpoofedPanicIdentityCannotClaimRecoveredState(t *testing.T) {
	t.Parallel()

	signals := make(chan os.Signal, 2)
	controller, err := Watch(WatchRequest{
		Parent:  t.Context(),
		Signals: signals,
		Policy:  forceSignalPolicy(time.Second),
		Force: func(context.Context, ForceRequest) error {
			return fmt.Errorf("spoofed force panic: %w", core.ErrShutdownForcePanic)
		},
	})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	signals <- firstSupportedOperatingSystemSignal()
	signals <- firstSupportedOperatingSystemSignal()
	result := <-controller.Forced()
	if result.Outcome != ForceOutcomeFailed || result.Validate() != nil || !errors.Is(result.Err, core.ErrShutdownForceFailure) || !errors.Is(result.Err, core.ErrShutdownForcePanic) {
		t.Fatalf("spoofed force result = %+v validate=%v, want failed with failure and preserved spoof identity", result, result.Validate())
	}
	<-controller.Done()
}

func TestPanicCaptureSurvivesPanickingFormatterAndBoundsHostileText(t *testing.T) {
	t.Parallel()

	id, _ := ParseStepID("hostile-panic-value")
	plan := mustTestPlan(t, testPlanPolicy(time.Second, time.Second), Step{ID: id, Run: func(context.Context) error { panic(panickingPanicStringer{}) }})
	report, err := plan.Run(t.Context())
	var panicErr StepPanicError
	if !errors.Is(err, core.ErrShutdownStepPanic) || !errors.As(report.Results[0].Err, &panicErr) || panicErr.Validate() != nil {
		t.Fatalf("Plan.Run(panicking formatter) = (%+v,%v) detail=%+v, want valid typed panic", report, err, panicErr)
	}
	if panicErr.Value.Type != PanicTypeName("shutdown.panickingPanicStringer") || panicErr.Value.Diagnostic != PanicDiagnostic(strconv.QuoteToASCII("panic diagnostic unavailable")) {
		t.Fatalf("hostile panic detail = %+v, want exact type and safe fallback", panicErr.Value)
	}

	oversized := newStepPanicError(newPanicValue("string", string(make([]byte, core.ShutdownPanicSourceMaxRunes+100))))
	if oversized.Validate() != nil || len(oversized.Value.Diagnostic.String()) > core.ShutdownPanicDiagnosticMaxRunes {
		t.Fatalf("oversized panic detail = %+v validate=%v, want bounded valid diagnostic", oversized.Value, oversized.Validate())
	}
	if !errors.Is((StepPanicError{}).Validate(), core.ErrShutdownContract) || !errors.Is((ForcePanicError{}).Validate(), core.ErrShutdownContract) {
		t.Fatalf("zero panic validation = step %v force %v, want %v", (StepPanicError{}).Validate(), (ForcePanicError{}).Validate(), core.ErrShutdownContract)
	}
}

type panickingPanicStringer struct{}

func (panickingPanicStringer) String() string { panic("formatter panic") }

func TestReportRejectsEmptyOversizedAndDuplicateResults(t *testing.T) {
	t.Parallel()

	completed := func(name string) StepResult {
		id, err := ParseStepID(name)
		if err != nil {
			t.Fatalf("ParseStepID(%q) error = %v", name, err)
		}
		return StepResult{ID: id, Outcome: StepOutcomeCompleted}
	}
	oversized := make([]StepResult, core.ShutdownMaximumSteps+1)
	for index := range oversized {
		oversized[index] = completed(fmt.Sprintf("step-%02d", index))
	}
	cases := []struct {
		name   string
		report Report
	}{
		{name: "empty results are rejected", report: Report{}},
		{name: "one above maximum results is rejected", report: Report{Results: oversized}},
		{name: "duplicate result identifiers are rejected", report: Report{Results: []StepResult{completed("dup"), completed("dup")}}},
		{name: "invalid inner result is rejected", report: Report{Results: []StepResult{{ID: completed("x").ID, Outcome: StepOutcomeFailed}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.report.Validate(); !errors.Is(err, core.ErrShutdownContract) {
				t.Fatalf("Report.Validate() error = %v, want ErrShutdownContract", err)
			}
			if err := tc.report.Error(); !errors.Is(err, core.ErrShutdownContract) {
				t.Fatalf("Report.Error() error = %v, want validation failure surfaced", err)
			}
		})
	}
	exact := make([]StepResult, core.ShutdownMaximumSteps)
	for index := range exact {
		exact[index] = completed(fmt.Sprintf("max-%02d", index))
	}
	full := Report{Results: exact}
	if err := full.Validate(); err != nil {
		t.Fatalf("Report.Validate(exact maximum) error = %v, want nil", err)
	}
	if err := full.Error(); err != nil {
		t.Fatalf("Report.Error(all completed) error = %v, want nil", err)
	}
}

func TestWatchCombinedEscalationPolicyDeliversExactlyOneForce(t *testing.T) {
	t.Parallel()

	combined := func(grace time.Duration) SignalPolicy {
		return SignalPolicy{
			SecondSignal: SecondSignalForce,
			GraceExpiry:  GraceExpiryForce,
			GracePeriod:  core.NewNanosecondsDuration(grace),
			ForceBudget:  core.NewNanosecondsDuration(time.Second),
		}
	}

	t.Run("second signal wins before a long grace deadline", func(t *testing.T) {
		t.Parallel()
		signals := make(chan os.Signal, 2)
		controller, err := Watch(WatchRequest{
			Parent:  t.Context(),
			Signals: signals,
			Policy:  combined(time.Hour),
			Force:   func(context.Context, ForceRequest) error { return nil },
		})
		if err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
		signals <- firstSupportedOperatingSystemSignal()
		<-controller.Context().Done()
		signals <- secondSupportedOperatingSystemSignal()
		result := <-controller.Forced()
		if result.Outcome != ForceOutcomeCompleted || result.Request.Reason != ForceReasonSecondSignal {
			t.Fatalf("combined force result = %+v, want completed second-signal reason", result)
		}
		<-controller.Done()
		if _, open := <-controller.Forced(); open {
			t.Fatalf("Forced() open after one result = %v, want false", open)
		}
	})

	t.Run("grace expiry wins when no second signal arrives", func(t *testing.T) {
		t.Parallel()
		signals := make(chan os.Signal, 1)
		controller, err := Watch(WatchRequest{
			Parent:  t.Context(),
			Signals: signals,
			Policy:  combined(10 * time.Millisecond),
			Force:   func(context.Context, ForceRequest) error { return nil },
		})
		if err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
		signals <- firstSupportedOperatingSystemSignal()
		result := <-controller.Forced()
		if result.Outcome != ForceOutcomeCompleted || result.Request.Reason != ForceReasonGraceExpired || result.Request.TriggerSignal != SignalKindUnknown {
			t.Fatalf("grace force result = %+v, want completed grace-expired reason without trigger", result)
		}
		<-controller.Done()
	})
}

func TestWatchEscalationWindowIgnoresUnknownSignalsAndClosedSourceEndsQuietly(t *testing.T) {
	t.Parallel()

	t.Run("unknown signal during escalation is ignored then valid signal forces", func(t *testing.T) {
		t.Parallel()
		signals := make(chan os.Signal, 3)
		controller, err := Watch(WatchRequest{
			Parent:  t.Context(),
			Signals: signals,
			Policy:  forceSignalPolicy(time.Second),
			Force:   func(context.Context, ForceRequest) error { return nil },
		})
		if err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
		signals <- firstSupportedOperatingSystemSignal()
		<-controller.Context().Done()
		signals <- unknownOperatingSystemSignal{}
		signals <- secondSupportedOperatingSystemSignal()
		result := <-controller.Forced()
		if result.Outcome != ForceOutcomeCompleted || result.Request.Reason != ForceReasonSecondSignal || !result.Request.TriggerSignal.IsValid() {
			t.Fatalf("force after unknown noise = %+v, want completed second-signal escalation", result)
		}
		<-controller.Done()
	})

	t.Run("source closed during escalation ends without a force", func(t *testing.T) {
		t.Parallel()
		signals := make(chan os.Signal, 1)
		controller, err := Watch(WatchRequest{
			Parent:  t.Context(),
			Signals: signals,
			Policy:  forceSignalPolicy(time.Second),
			Force:   func(context.Context, ForceRequest) error { return nil },
		})
		if err != nil {
			t.Fatalf("Watch() error = %v", err)
		}
		signals <- firstSupportedOperatingSystemSignal()
		<-controller.Context().Done()
		close(signals)
		<-controller.Done()
		if result, open := <-controller.Forced(); open {
			t.Fatalf("Forced() = (%+v,%t), want closed channel with no force after source closed", result, open)
		}
	})
}
