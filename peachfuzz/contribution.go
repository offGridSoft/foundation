package peachfuzz

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/offGridSoft/foundation/v2026/core"
)

const ProjectContributionMaxMachines = 64

// MachineContribution is one machine's complete cumulative project evidence.
// The control plane replaces only the matching Machine identity and derives the
// public aggregate from the full typed contribution set.
type MachineContribution struct {
	Snapshot ProjectSnapshot `json:"snapshot"`
	Machine  MachineID       `json:"machine"`
}

func (MachineContribution) APIBody() {}

func (c MachineContribution) Validate() error {
	if err := c.Machine.Validate(); err != nil {
		return machineContributionError(err)
	}
	if err := c.Snapshot.Validate(); err != nil {
		return machineContributionError(err)
	}
	return nil
}

// ValidateSuccessor rejects any write that would erase evidence already
// accepted for the same machine. Exact retries are valid and idempotent.
func (c MachineContribution) ValidateSuccessor(previous MachineContribution) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := previous.Validate(); err != nil {
		return machineContributionError(err)
	}
	if c.Machine != previous.Machine || c.Snapshot.Project != previous.Snapshot.Project ||
		c.Snapshot.RecordedSince != previous.Snapshot.RecordedSince || contributionRegressed(c.Snapshot, previous.Snapshot) {
		return machineContributionError(core.ErrPeachfuzzContributionRegression)
	}
	return nil
}

func contributionRegressed(next, previous ProjectSnapshot) bool {
	return next.LastRunAt.Time().Before(previous.LastRunAt.Time()) ||
		next.RunCount < previous.RunCount || next.ReportedExecutions < previous.ReportedExecutions ||
		next.CandidateSightings < previous.CandidateSightings ||
		next.UniqueCandidatesRetained < previous.UniqueCandidatesRetained ||
		next.PackagesExercised < previous.PackagesExercised || next.TargetsExercised < previous.TargetsExercised ||
		next.Effort.Nanoseconds() < previous.Effort.Nanoseconds()
}

// ProjectContributionSet is the bounded, deterministic persistence contract
// for one public project document. Contributions are strictly Machine-sorted.
type ProjectContributionSet struct {
	Contributions []MachineContribution `json:"contributions"`
}

func (s ProjectContributionSet) Validate() error {
	if len(s.Contributions) == 0 || len(s.Contributions) > ProjectContributionMaxMachines {
		return projectContributionSetError(ErrContract)
	}
	project := s.Contributions[0].Snapshot.Project
	for index, contribution := range s.Contributions {
		if err := contribution.Validate(); err != nil || contribution.Snapshot.Project != project {
			return projectContributionSetError(errors.Join(ErrContract, err))
		}
		if index > 0 && s.Contributions[index-1].Machine.String() >= contribution.Machine.String() {
			return projectContributionSetError(ErrContract)
		}
	}
	return nil
}

// Upsert validates a monotonic machine update and returns a cloned set plus
// the exact aggregate receipt. No caller can mutate the input set through it.
func (s ProjectContributionSet) Upsert(next MachineContribution) (ProjectContributionSet, ContributionReceipt, error) {
	if err := next.Validate(); err != nil {
		return ProjectContributionSet{}, ContributionReceipt{}, err
	}
	contributions := append([]MachineContribution(nil), s.Contributions...)
	index := contributionIndex(contributions, next.Machine)
	if index >= 0 {
		if err := next.ValidateSuccessor(contributions[index]); err != nil {
			return ProjectContributionSet{}, ContributionReceipt{}, err
		}
		contributions[index] = next
	} else {
		if len(contributions) >= ProjectContributionMaxMachines {
			return ProjectContributionSet{}, ContributionReceipt{}, projectContributionSetError(ErrContract)
		}
		contributions = append(contributions, next)
	}
	slices.SortFunc(contributions, compareMachineContribution)
	updated := ProjectContributionSet{Contributions: contributions}
	aggregate, err := updated.Aggregate()
	if err != nil {
		return ProjectContributionSet{}, ContributionReceipt{}, err
	}
	receipt := ContributionReceipt{Contribution: next, Aggregate: aggregate}
	return updated, receipt, receipt.Validate()
}

func contributionIndex(contributions []MachineContribution, machine MachineID) int {
	for index, contribution := range contributions {
		if contribution.Machine == machine {
			return index
		}
	}
	return -1
}

func compareMachineContribution(left, right MachineContribution) int {
	if left.Machine.String() < right.Machine.String() {
		return -1
	}
	if left.Machine.String() > right.Machine.String() {
		return 1
	}
	return 0
}

func (s ProjectContributionSet) Aggregate() (ProjectSnapshot, error) {
	if err := s.Validate(); err != nil {
		return ProjectSnapshot{}, err
	}
	aggregate := s.Contributions[0].Snapshot
	for _, contribution := range s.Contributions[1:] {
		var err error
		aggregate, err = addContribution(aggregate, contribution.Snapshot)
		if err != nil {
			return ProjectSnapshot{}, err
		}
	}
	return deriveEffortProjection(aggregate)
}

func addContribution(aggregate, contribution ProjectSnapshot) (ProjectSnapshot, error) {
	var err error
	if aggregate.RunCount, err = checkedAddUint64(aggregate.RunCount, contribution.RunCount); err != nil {
		return ProjectSnapshot{}, err
	}
	if aggregate.ReportedExecutions, err = checkedAddUint64(aggregate.ReportedExecutions, contribution.ReportedExecutions); err != nil {
		return ProjectSnapshot{}, err
	}
	if aggregate.CandidateSightings, err = checkedAddUint64(aggregate.CandidateSightings, contribution.CandidateSightings); err != nil {
		return ProjectSnapshot{}, err
	}
	if aggregate.UniqueCandidatesRetained, err = checkedAddUint64(aggregate.UniqueCandidatesRetained, contribution.UniqueCandidatesRetained); err != nil {
		return ProjectSnapshot{}, err
	}
	effort, err := checkedAddInt64(aggregate.Effort.Nanoseconds(), contribution.Effort.Nanoseconds())
	if err != nil {
		return ProjectSnapshot{}, err
	}
	aggregate.Effort = core.NanosecondsDurationFromInt64(effort)
	aggregate.PackagesExercised = max(aggregate.PackagesExercised, contribution.PackagesExercised)
	aggregate.TargetsExercised = max(aggregate.TargetsExercised, contribution.TargetsExercised)
	if contribution.RecordedSince.Time().Before(aggregate.RecordedSince.Time()) {
		aggregate.RecordedSince = contribution.RecordedSince
	}
	if contribution.LastRunAt.Time().After(aggregate.LastRunAt.Time()) {
		aggregate.LastRunAt = contribution.LastRunAt
		aggregate.LastOutcome = contribution.LastOutcome
	}
	return aggregate, nil
}

func deriveEffortProjection(snapshot ProjectSnapshot) (ProjectSnapshot, error) {
	coreYears, err := EffortCoreYears(snapshot.Effort)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	humanized, err := HumanizeEffort(snapshot.Effort)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snapshot.CoreYears = coreYears
	snapshot.CoreYearsHumanized = humanized
	return snapshot, snapshot.Validate()
}

func checkedAddUint64(left, right uint64) (uint64, error) {
	if right > math.MaxUint64-left {
		return 0, projectContributionSetError(ErrContract)
	}
	return left + right, nil
}

func checkedAddInt64(left, right int64) (int64, error) {
	if right > math.MaxInt64-left {
		return 0, projectContributionSetError(ErrContract)
	}
	return left + right, nil
}

// ContributionReceipt binds the accepted machine state to the transaction's
// resulting public aggregate.
type ContributionReceipt struct {
	Contribution MachineContribution `json:"contribution"`
	Aggregate    ProjectSnapshot     `json:"aggregate"`
}

func (ContributionReceipt) APIBody() {}

func (r ContributionReceipt) Validate() error {
	if err := r.Contribution.Validate(); err != nil {
		return contributionReceiptError(err)
	}
	if err := r.Aggregate.Validate(); err != nil || r.Aggregate.Project != r.Contribution.Snapshot.Project {
		return contributionReceiptError(errors.Join(ErrContract, err))
	}
	return nil
}

func machineContributionError(err error) error {
	return fmt.Errorf(ErrFmtMachineContribution, errors.Join(ErrContract, err))
}

func projectContributionSetError(err error) error {
	return fmt.Errorf(ErrFmtProjectContributionSet, errors.Join(ErrContract, err))
}

func contributionReceiptError(err error) error {
	return fmt.Errorf(ErrFmtContributionReceipt, errors.Join(ErrContract, err))
}
