package shutdown

import (
	"context"
	"errors"
	"sync"

	"github.com/offGridSoft/foundation/v2026/core"
)

type StepAction func(context.Context) error

type Step struct {
	Run StepAction
	ID  StepID
}

func (s Step) Validate() error {
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if s.Run == nil {
		return core.ErrShutdownContract
	}
	return nil
}

type PlanPolicy struct {
	StepBudget  core.NanosecondsDuration
	TotalBudget core.NanosecondsDuration
}

func (p PlanPolicy) Validate() error {
	if err := p.StepBudget.Validate(); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	if err := p.TotalBudget.Validate(); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	if p.StepBudget.IsZero() || p.TotalBudget.IsZero() || p.TotalBudget.Nanoseconds() < p.StepBudget.Nanoseconds() {
		return core.ErrShutdownContract
	}
	return nil
}

type Plan struct {
	steps  []Step
	policy PlanPolicy
	mu     sync.Mutex
	state  planState
}

type planState uint8

const (
	planStateUnknown planState = iota
	planStateOpen
	planStateRunning
	planStateComplete
)

func NewPlan(policy PlanPolicy) (*Plan, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Plan{
		steps:  make([]Step, 0, core.ShutdownMaximumSteps),
		policy: policy,
		state:  planStateOpen,
	}, nil
}

func (p *Plan) Register(step Step) error {
	if p == nil {
		return core.ErrShutdownContract
	}
	if err := step.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != planStateOpen || len(p.steps) >= core.ShutdownMaximumSteps {
		return core.ErrShutdownContract
	}
	for _, registered := range p.steps {
		if registered.ID == step.ID {
			return core.ErrShutdownContract
		}
	}
	p.steps = append(p.steps, step)
	return nil
}

func (p *Plan) Validate() error {
	if p == nil {
		return core.ErrShutdownContract
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.validateLocked()
}

func (p *Plan) validateLocked() error {
	if err := p.policy.Validate(); err != nil {
		return err
	}
	if p.state == planStateUnknown || p.state > planStateComplete {
		return core.ErrShutdownContract
	}
	if len(p.steps) == 0 || len(p.steps) > core.ShutdownMaximumSteps {
		return core.ErrShutdownContract
	}
	for index, step := range p.steps {
		if err := step.Validate(); err != nil {
			return err
		}
		for prior := range index {
			if p.steps[prior].ID == step.ID {
				return core.ErrShutdownContract
			}
		}
	}
	return nil
}

type StepOutcome uint8

const (
	StepOutcomeUnknown StepOutcome = iota
	StepOutcomeCompleted
	StepOutcomeFailed
	StepOutcomeTimedOut
	StepOutcomePanicked
	StepOutcomeTotalBudgetExceeded
)

func (o StepOutcome) Validate() error {
	if !o.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}

type StepResult struct {
	Err     error
	ID      StepID
	Outcome StepOutcome
}

func (r StepResult) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.Outcome.Validate(); err != nil {
		return err
	}
	return validateStepResultError(r)
}

func validateStepResultError(result StepResult) error {
	if result.Outcome == StepOutcomeCompleted {
		if result.Err != nil {
			return core.ErrShutdownContract
		}
		return nil
	}
	if result.Err == nil {
		return core.ErrShutdownContract
	}
	identity := stepOutcomeIdentity(result.Outcome)
	if identity == nil || !errors.Is(result.Err, identity) {
		return core.ErrShutdownContract
	}
	return nil
}

func stepOutcomeIdentity(outcome StepOutcome) error {
	switch outcome {
	case StepOutcomeFailed:
		return core.ErrShutdownStepFailure
	case StepOutcomeTimedOut:
		return core.ErrShutdownStepTimeout
	case StepOutcomePanicked:
		return core.ErrShutdownStepPanic
	case StepOutcomeTotalBudgetExceeded:
		return core.ErrShutdownTotalTimeout
	default:
		return nil
	}
}

type Report struct {
	Results []StepResult
}

func (r Report) Validate() error {
	if len(r.Results) == 0 || len(r.Results) > core.ShutdownMaximumSteps {
		return core.ErrShutdownContract
	}
	for index, result := range r.Results {
		if err := result.Validate(); err != nil {
			return err
		}
		for prior := range index {
			if r.Results[prior].ID == result.ID {
				return core.ErrShutdownContract
			}
		}
	}
	return nil
}

func (r Report) Error() error {
	if err := r.Validate(); err != nil {
		return err
	}
	var resultErr error
	for _, result := range r.Results {
		resultErr = errors.Join(resultErr, result.Err)
	}
	return resultErr
}

func (p *Plan) Run(parent context.Context) (Report, error) {
	if parent == nil {
		return Report{}, core.ErrNilContext
	}
	steps, policy, err := p.beginRun()
	if err != nil {
		return Report{}, err
	}
	defer p.completeRun()
	root, cancel := context.WithTimeout(context.WithoutCancel(parent), policy.TotalBudget.Duration())
	defer cancel()
	report := Report{Results: make([]StepResult, 0, len(steps))}
	for index := len(steps) - 1; index >= 0; index-- {
		report.Results = append(report.Results, runStep(root, policy.StepBudget, steps[index]))
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, report.Error()
}

func (p *Plan) beginRun() ([]Step, PlanPolicy, error) {
	if p == nil {
		return nil, PlanPolicy{}, core.ErrShutdownContract
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != planStateOpen {
		return nil, PlanPolicy{}, core.ErrShutdownContract
	}
	if err := p.validateLocked(); err != nil {
		return nil, PlanPolicy{}, err
	}
	p.state = planStateRunning
	steps := make([]Step, len(p.steps))
	copy(steps, p.steps)
	return steps, p.policy, nil
}

func (p *Plan) completeRun() {
	p.mu.Lock()
	p.state = planStateComplete
	p.mu.Unlock()
}

func runStep(root context.Context, budget core.NanosecondsDuration, step Step) StepResult {
	if root.Err() != nil {
		return newStepResult(step.ID, StepOutcomeTotalBudgetExceeded, core.ErrShutdownTotalTimeout)
	}
	ctx, cancel := context.WithTimeout(root, budget.Duration())
	defer cancel()
	result := make(chan error, 1)
	go callStep(ctx, step, result)
	select {
	case err := <-result:
		if ctx.Err() != nil {
			return timedOutStepResult(root, step.ID)
		}
		if err != nil {
			panicErr, isPanic := errors.AsType[StepPanicError](err)
			if isPanic && panicErr.Validate() == nil {
				return newStepResult(step.ID, StepOutcomePanicked, err)
			}
			return newStepResult(step.ID, StepOutcomeFailed, errors.Join(core.ErrShutdownStepFailure, err))
		}
		return newStepResult(step.ID, StepOutcomeCompleted, nil)
	case <-ctx.Done():
		return timedOutStepResult(root, step.ID)
	}
}

func callStep(ctx context.Context, step Step, result chan<- error) {
	runCapturingPanic(
		func() error { return step.Run(ctx) },
		result,
		func(value PanicValue) error { return newStepPanicError(value) },
	)
}

func timedOutStepResult(root context.Context, id StepID) StepResult {
	if root.Err() != nil {
		return newStepResult(id, StepOutcomeTotalBudgetExceeded, errors.Join(core.ErrShutdownTotalTimeout, context.DeadlineExceeded))
	}
	return newStepResult(id, StepOutcomeTimedOut, errors.Join(core.ErrShutdownStepTimeout, context.DeadlineExceeded))
}

func newStepResult(id StepID, outcome StepOutcome, err error) StepResult {
	return StepResult{ID: id, Outcome: outcome, Err: err}
}
