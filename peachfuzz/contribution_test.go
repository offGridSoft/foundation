package peachfuzz

import (
	"errors"
	"math"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
)

func TestProjectContributionSetUpsertAggregatesMachineEvidenceOGS(t *testing.T) {
	t.Parallel()
	first := validMachineContribution(t, 'a')
	first.Snapshot.PackagesExercised = 4
	first.Snapshot.TargetsExercised = 13
	first.Snapshot.ReportedExecutions = 100
	second := validMachineContribution(t, 'b')
	second.Snapshot.RecordedSince = foundationcore.UnixNanoTimeFromInt64(2)
	second.Snapshot.LastRunAt = foundationcore.UnixNanoTimeFromInt64(3)
	second.Snapshot.PackagesExercised = 2
	second.Snapshot.TargetsExercised = 8
	second.Snapshot.ReportedExecutions = 200

	set, _, err := (ProjectContributionSet{}).Upsert(first)
	if err != nil {
		t.Fatalf("Upsert(first) error = %v, want nil", err)
	}
	set, receipt, err := set.Upsert(second)
	if err != nil {
		t.Fatalf("Upsert(second) error = %v, want nil", err)
	}
	got := receipt.Aggregate
	if got.RunCount != 2 || got.ReportedExecutions != 300 || got.Effort.Nanoseconds() != 2*NanosPerCoreYear {
		t.Fatalf("aggregate counters = runs:%d executions:%d effort:%d, want 2/300/%d", got.RunCount, got.ReportedExecutions, got.Effort.Nanoseconds(), 2*NanosPerCoreYear)
	}
	if got.PackagesExercised != 4 || got.TargetsExercised != 13 {
		t.Fatalf("aggregate coverage = %d packages/%d targets, want max 4/13", got.PackagesExercised, got.TargetsExercised)
	}
	if got.RecordedSince != first.Snapshot.RecordedSince || got.LastRunAt != second.Snapshot.LastRunAt {
		t.Fatalf("aggregate interval = %v..%v, want %v..%v", got.RecordedSince, got.LastRunAt, first.Snapshot.RecordedSince, second.Snapshot.LastRunAt)
	}
	if len(set.Contributions) != 2 || set.Contributions[0].Machine != first.Machine || set.Contributions[1].Machine != second.Machine {
		t.Fatalf("contributions = %+v, want deterministic machine order", set.Contributions)
	}
}

func TestMachineContributionRejectsEvidenceRegressionOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	previous := validMachineContribution(t, 'a')
	previous.Snapshot.RunCount = 2
	previous.Snapshot.ReportedExecutions = 10
	previous.Snapshot.CandidateSightings = 2
	previous.Snapshot.UniqueCandidatesRetained = 1
	previous.Snapshot.PackagesExercised = 2
	previous.Snapshot.TargetsExercised = 3
	for _, testCase := range []struct {
		mutate func(*MachineContribution)
		name   string
	}{
		{name: "machine", mutate: func(c *MachineContribution) { c.Machine = validMachineID(t, 'b') }},
		{name: "project", mutate: func(c *MachineContribution) { c.Snapshot.Project, _ = ParseProjectID("kernel") }},
		{name: "recorded since", mutate: func(c *MachineContribution) { c.Snapshot.RecordedSince = foundationcore.UnixNanoTimeFromInt64(2) }},
		{name: "last run", mutate: func(c *MachineContribution) { c.Snapshot.LastRunAt = foundationcore.UnixNanoTimeFromInt64(1) }},
		{name: "runs", mutate: func(c *MachineContribution) { c.Snapshot.RunCount-- }},
		{name: "executions", mutate: func(c *MachineContribution) { c.Snapshot.ReportedExecutions-- }},
		{name: "sightings", mutate: func(c *MachineContribution) { c.Snapshot.CandidateSightings--; c.Snapshot.UniqueCandidatesRetained = 0 }},
		{name: "retained", mutate: func(c *MachineContribution) { c.Snapshot.UniqueCandidatesRetained-- }},
		{name: "packages", mutate: func(c *MachineContribution) { c.Snapshot.PackagesExercised-- }},
		{name: "targets", mutate: func(c *MachineContribution) { c.Snapshot.TargetsExercised-- }},
		{name: "effort", mutate: func(c *MachineContribution) {
			c.Snapshot.Effort = foundationcore.NanosecondsDurationFromInt64(NanosPerCoreYear - 1)
			c.Snapshot, _ = deriveEffortProjection(c.Snapshot)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			next := previous
			testCase.mutate(&next)
			if err := next.ValidateSuccessor(previous); !errors.Is(err, foundationcore.ErrPeachfuzzContributionRegression) {
				t.Fatalf("ValidateSuccessor() error = %v, want %v", err, foundationcore.ErrPeachfuzzContributionRegression)
			}
		})
	}
}

func TestProjectContributionSetRejectsAggregateOverflowOGS(t *testing.T) {
	t.Parallel()
	first := validMachineContribution(t, 'a')
	first.Snapshot.ReportedExecutions = math.MaxUint64
	second := validMachineContribution(t, 'b')
	second.Snapshot.ReportedExecutions = 1
	set := ProjectContributionSet{Contributions: []MachineContribution{first, second}}
	if _, err := set.Aggregate(); !errors.Is(err, ErrContract) {
		t.Fatalf("Aggregate() error = %v, want %v", err, ErrContract)
	}
}

func TestProjectContributionSetRejectsHostileShapeOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	first := validMachineContribution(t, 'a')
	second := validMachineContribution(t, 'b')
	for _, testCase := range []struct {
		set  ProjectContributionSet
		name string
	}{
		{name: "empty"},
		{name: "duplicate machine", set: ProjectContributionSet{Contributions: []MachineContribution{first, first}}},
		{name: "reverse order", set: ProjectContributionSet{Contributions: []MachineContribution{second, first}}},
		{name: "cross project", set: ProjectContributionSet{Contributions: []MachineContribution{first, contributionForProject(t, second, "kernel")}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if err := testCase.set.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrContract)
			}
		})
	}
}

func validMachineContribution(t *testing.T, digit byte) MachineContribution {
	t.Helper()
	return MachineContribution{Machine: validMachineID(t, digit), Snapshot: validProjectSnapshot(t)}
}

func validMachineID(t *testing.T, digit byte) MachineID {
	t.Helper()
	machine, err := ParseMachineID(strings.Repeat(string(digit), MachineIDTextBytes))
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func contributionForProject(t *testing.T, contribution MachineContribution, project string) MachineContribution {
	t.Helper()
	parsed, err := ParseProjectID(project)
	if err != nil {
		t.Fatal(err)
	}
	contribution.Snapshot.Project = parsed
	return contribution
}
