package shutdown

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	errFirstCleanup  = errors.New("first cleanup failure")
	errSecondCleanup = errors.New("second cleanup failure")
)

func TestPlanRunsLIFOContinuesAndPreservesEveryFailure(t *testing.T) {
	t.Parallel()

	order := make(chan StepID, 3)
	first, _ := ParseStepID("first")
	second, _ := ParseStepID("second")
	third, _ := ParseStepID("third")
	plan := Plan{
		Policy: testPlanPolicy(time.Second, 3*time.Second),
		Steps: []Step{
			{ID: first, Run: func(context.Context) error { order <- first; return errFirstCleanup }},
			{ID: second, Run: func(context.Context) error { order <- second; return nil }},
			{ID: third, Run: func(context.Context) error { order <- third; return errSecondCleanup }},
		},
	}
	report, err := plan.Run(t.Context())
	if !errors.Is(err, errFirstCleanup) || !errors.Is(err, errSecondCleanup) || !errors.Is(err, core.ErrShutdownStepFailure) {
		t.Fatalf("Plan.Run() error = %v, want both causes and ErrShutdownStepFailure", err)
	}
	if report.Validate() != nil || len(report.Results) != 3 {
		t.Fatalf("Plan.Run() report = %+v validate=%v, want three valid results", report, report.Validate())
	}
	if gotFirst, gotSecond, gotThird := <-order, <-order, <-order; gotFirst != third || gotSecond != second || gotThird != first {
		t.Fatalf("cleanup order = [%s %s %s], want [%s %s %s]", gotFirst, gotSecond, gotThird, third, second, first)
	}
	if report.Results[0].ID != third || report.Results[0].Outcome != StepOutcomeFailed || report.Results[1].Outcome != StepOutcomeCompleted || report.Results[2].ID != first || report.Results[2].Outcome != StepOutcomeFailed {
		t.Fatalf("result order/outcomes = %+v, want third-failed second-completed first-failed", report.Results)
	}
}

func TestPlanDetachesCancelledParentForCleanup(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(context.WithValue(t.Context(), shutdownContextKey(1), "preserved"))
	cancel()
	observed := make(chan struct{}, 1)
	id, _ := ParseStepID("detached")
	plan := Plan{Policy: testPlanPolicy(time.Second, time.Second), Steps: []Step{{ID: id, Run: func(ctx context.Context) error {
		if ctx.Err() != nil || ctx.Value(shutdownContextKey(1)) != "preserved" {
			return core.ErrShutdownContract
		}
		observed <- struct{}{}
		return nil
	}}}}
	report, err := plan.Run(parent)
	if err != nil || report.Results[0].Outcome != StepOutcomeCompleted || len(observed) != 1 {
		t.Fatalf("Plan.Run(cancelled parent) = (%+v,%v), observed=%d; want completed detached cleanup", report, err, len(observed))
	}
}

func TestPlanBoundsTimeoutPanicAndContinuesHostileTable(t *testing.T) {
	t.Parallel()

	timedOut, _ := ParseStepID("timed-out")
	panicked, _ := ParseStepID("panicked")
	continued, _ := ParseStepID("continued")
	didContinue := make(chan struct{}, 1)
	plan := Plan{
		Policy: testPlanPolicy(20*time.Millisecond, 200*time.Millisecond),
		Steps: []Step{
			{ID: continued, Run: func(context.Context) error { didContinue <- struct{}{}; return nil }},
			{ID: timedOut, Run: func(ctx context.Context) error { <-ctx.Done(); return context.Cause(ctx) }},
			{ID: panicked, Run: func(context.Context) error { panic("hostile cleanup panic") }},
		},
	}
	report, err := plan.Run(t.Context())
	if !errors.Is(err, core.ErrShutdownStepPanic) || !errors.Is(err, core.ErrShutdownStepTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Plan.Run() error = %v, want panic, timeout, and deadline identities", err)
	}
	if len(didContinue) != 1 || len(report.Results) != 3 {
		t.Fatalf("continued=%d results=%d, want 1/3 after panic and timeout", len(didContinue), len(report.Results))
	}
	if report.Results[0].Outcome != StepOutcomePanicked || report.Results[1].Outcome != StepOutcomeTimedOut || report.Results[2].Outcome != StepOutcomeCompleted {
		t.Fatalf("outcomes = [%s %s %s], want panicked/timed-out/completed", report.Results[0].Outcome, report.Results[1].Outcome, report.Results[2].Outcome)
	}
}

func TestPlanTotalBudgetBoundsRemainingSteps(t *testing.T) {
	t.Parallel()

	first, _ := ParseStepID("first")
	second, _ := ParseStepID("second")
	plan := Plan{
		Policy: testPlanPolicy(20*time.Millisecond, 20*time.Millisecond),
		Steps: []Step{
			{ID: first, Run: func(context.Context) error { return nil }},
			{ID: second, Run: func(ctx context.Context) error { <-ctx.Done(); return context.Cause(ctx) }},
		},
	}
	report, err := plan.Run(t.Context())
	if !errors.Is(err, core.ErrShutdownTotalTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Plan.Run(total exhausted) error = %v, want total timeout and deadline", err)
	}
	if report.Results[0].Outcome != StepOutcomeTotalBudgetExceeded || report.Results[1].Outcome != StepOutcomeTotalBudgetExceeded {
		t.Fatalf("total-exhausted outcomes = [%s %s], want both total-budget-exceeded", report.Results[0].Outcome, report.Results[1].Outcome)
	}
}

func TestPlanValidationRejectsEveryInvalidBoundary(t *testing.T) {
	t.Parallel()

	validID, _ := ParseStepID("valid")
	validStep := Step{ID: validID, Run: func(context.Context) error { return nil }}
	validPolicy := testPlanPolicy(time.Second, time.Second)
	tooMany := make([]Step, core.ShutdownMaximumSteps+1)
	for index := range tooMany {
		id, err := ParseStepID("step-" + string(rune('a'+index%26)) + string(rune('A'+index/26)))
		if err != nil {
			t.Fatalf("ParseStepID(tooMany[%d]) error = %v", index, err)
		}
		tooMany[index] = Step{ID: id, Run: validStep.Run}
	}
	cases := []Plan{
		{},
		{Policy: validPolicy},
		{Policy: PlanPolicy{StepBudget: core.NewNanosecondsDuration(time.Second), TotalBudget: core.NewNanosecondsDuration(time.Millisecond)}, Steps: []Step{validStep}},
		{Policy: PlanPolicy{TotalBudget: core.NewNanosecondsDuration(time.Second)}, Steps: []Step{validStep}},
		{Policy: PlanPolicy{StepBudget: core.NewNanosecondsDuration(time.Second)}, Steps: []Step{validStep}},
		{Policy: PlanPolicy{StepBudget: core.NanosecondsDurationFromInt64(-1), TotalBudget: core.NewNanosecondsDuration(time.Second)}, Steps: []Step{validStep}},
		{Policy: validPolicy, Steps: []Step{{ID: validID}}},
		{Policy: validPolicy, Steps: []Step{{ID: "", Run: validStep.Run}}},
		{Policy: validPolicy, Steps: []Step{validStep, validStep}},
		{Policy: validPolicy, Steps: tooMany},
	}
	for index, plan := range cases {
		if err := plan.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("invalid plan %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
	}
	var nilContext context.Context
	if _, err := (Plan{Policy: validPolicy, Steps: []Step{validStep}}).Run(nilContext); !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("Plan.Run(nil) error = %v, want ErrNilContext", err)
	}
}

func TestStepResultRejectsImpossibleStateErrorCombinations(t *testing.T) {
	t.Parallel()

	id, _ := ParseStepID("step")
	cases := []StepResult{
		{},
		{ID: id, Outcome: StepOutcomeUnknown},
		{ID: id, Outcome: StepOutcomeCompleted, Err: errFirstCleanup},
		{ID: id, Outcome: StepOutcomeFailed},
		{ID: id, Outcome: StepOutcomeFailed, Err: errFirstCleanup},
		{ID: id, Outcome: StepOutcomeTimedOut, Err: core.ErrShutdownStepFailure},
		{ID: id, Outcome: StepOutcomePanicked, Err: core.ErrShutdownStepTimeout},
		{ID: id, Outcome: StepOutcomeTotalBudgetExceeded, Err: core.ErrShutdownStepPanic},
	}
	for index, result := range cases {
		if err := result.Validate(); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("impossible result %d Validate() error = %v, want ErrShutdownContract", index, err)
		}
	}
}

type shutdownContextKey uint8

func testPlanPolicy(step, total time.Duration) PlanPolicy {
	return PlanPolicy{StepBudget: core.NewNanosecondsDuration(step), TotalBudget: core.NewNanosecondsDuration(total)}
}
