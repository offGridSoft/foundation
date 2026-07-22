package shutdown

import (
	"context"
	"errors"

	"github.com/offGridSoft/foundation/v2026/core"
)

type StepAction func(context.Context) error

type Step struct {
	ID  StepID
	Run StepAction
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
	Policy PlanPolicy
	Steps  []Step
}

func (p Plan) Validate() error {
	if err := p.Policy.Validate(); err != nil {
		return err
	}
	if len(p.Steps) == 0 || len(p.Steps) > core.ShutdownMaximumSteps {
		return core.ErrShutdownContract
	}
	for index, step := range p.Steps {
		if err := step.Validate(); err != nil {
			return err
		}
		for prior := range index {
			if p.Steps[prior].ID == step.ID {
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
	ID      StepID
	Outcome StepOutcome
	Err     error
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

func (p Plan) Run(parent context.Context) (Report, error) {
	if parent == nil {
		return Report{}, core.ErrNilContext
	}
	if err := p.Validate(); err != nil {
		return Report{}, err
	}
	root, cancel := context.WithTimeout(context.WithoutCancel(parent), p.Policy.TotalBudget.Duration())
	defer cancel()
	report := Report{Results: make([]StepResult, 0, len(p.Steps))}
	for index := len(p.Steps) - 1; index >= 0; index-- {
		report.Results = append(report.Results, runStep(root, p.Policy.StepBudget, p.Steps[index]))
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, report.Error()
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
			if errors.Is(err, core.ErrShutdownStepPanic) {
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
	defer func() {
		if recover() != nil {
			result <- core.ErrShutdownStepPanic
		}
	}()
	result <- step.Run(ctx)
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
