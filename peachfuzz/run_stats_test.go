package peachfuzz

import (
	"errors"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
)

func TestRunStatsValidateOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	valid := validRunStats(t)
	tests := []struct {
		mutate func(*RunStats)
		name   string
	}{
		{name: "invalid run id", mutate: func(s *RunStats) { s.RunID = RunID{} }},
		{name: "invalid project", mutate: func(s *RunStats) { s.Project = ProjectID{} }},
		{name: "invalid package", mutate: func(s *RunStats) { s.PackagePath = PackageImportPath{} }},
		{name: "invalid target", mutate: func(s *RunStats) { s.FuzzTarget = FuzzTargetName{} }},
		{name: "invalid commit", mutate: func(s *RunStats) { s.Commit = CommitSHA{} }},
		{name: "invalid machine", mutate: func(s *RunStats) { s.Machine = MachineID{} }},
		{name: "invalid outcome", mutate: func(s *RunStats) { s.Outcome = RunOutcomeUnknown }},
		{name: "missing start", mutate: func(s *RunStats) { s.Start = foundationcore.UnixNanoTime{} }},
		{name: "missing end", mutate: func(s *RunStats) { s.End = foundationcore.UnixNanoTime{} }},
		{name: "reversed interval", mutate: func(s *RunStats) { s.End = foundationcore.UnixNanoTimeFromInt64(0) }},
		{name: "negative cpu", mutate: func(s *RunStats) { s.CPU = foundationcore.NanosecondsDurationFromInt64(-1) }},
		{name: "retained exceeds sightings", mutate: func(s *RunStats) { s.CandidateSightings = 1; s.UniqueCandidatesRetained = 2 }},
		{name: "wrong schema", mutate: func(s *RunStats) { s.Schema = RunStatsSchemaUnknown }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			test.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("RunStats.Validate() error = %v, want %v", err, ErrContract)
			}
		})
	}
}

func TestProjectSnapshotOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	valid := validProjectSnapshot(t)
	for _, tc := range []struct {
		mutate func(*ProjectSnapshot)
		name   string
	}{
		{name: "invalid project", mutate: func(s *ProjectSnapshot) { s.Project = ProjectID{} }},
		{name: "missing recorded since", mutate: func(s *ProjectSnapshot) { s.RecordedSince = foundationcore.UnixNanoTime{} }},
		{name: "missing last run", mutate: func(s *ProjectSnapshot) { s.LastRunAt = foundationcore.UnixNanoTime{} }},
		{name: "negative effort", mutate: func(s *ProjectSnapshot) { s.Effort = foundationcore.NanosecondsDurationFromInt64(-1) }},
		{name: "drifted core years", mutate: func(s *ProjectSnapshot) { s.CoreYears = 2 }},
		{name: "drifted humanized effort", mutate: func(s *ProjectSnapshot) { s.CoreYearsHumanized = "2.00 core-years" }},
		{name: "invalid outcome", mutate: func(s *ProjectSnapshot) { s.LastOutcome = RunOutcomeUnknown }},
		{name: "reversed interval", mutate: func(s *ProjectSnapshot) { s.LastRunAt = foundationcore.UnixNanoTimeFromInt64(0) }},
		{name: "zero runs", mutate: func(s *ProjectSnapshot) { s.RunCount = 0 }},
		{name: "retained exceeds sightings", mutate: func(s *ProjectSnapshot) { s.CandidateSightings = 1; s.UniqueCandidatesRetained = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			tc.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("ProjectSnapshot.Validate() error = %v, want %v", err, ErrContract)
			}
		})
	}
	if valid.CoreYears != 1 || valid.CoreYearsHumanized != "1.00 core-years" {
		t.Fatalf("ProjectSnapshot effort projection = %v %q", valid.CoreYears, valid.CoreYearsHumanized)
	}
}

func TestRunStatsReceiptOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	stats := validRunStats(t)
	valid := RunStatsReceipt{RunID: stats.RunID, Disposition: RecordDispositionRecorded}
	if err := valid.Validate(); err != nil {
		t.Fatalf("RunStatsReceipt.Validate() error = %v", err)
	}
	invalidID := valid
	invalidID.RunID = RunID{}
	if err := invalidID.Validate(); !errors.Is(err, ErrContract) {
		t.Fatalf("invalid RunID error = %v, want %v", err, ErrContract)
	}
	invalidDisposition := valid
	invalidDisposition.Disposition = RecordDispositionUnknown
	if err := invalidDisposition.Validate(); !errors.Is(err, ErrContract) {
		t.Fatalf("invalid disposition error = %v, want %v", err, ErrContract)
	}
}

func TestHumanizeEffortOGSUnitLadder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		want  string
		nanos int64
	}{
		{name: "seconds", nanos: 30 * NanosPerCoreSecond, want: "30.0 core-seconds"},
		{name: "minutes", nanos: 30 * NanosPerCoreMinute, want: "30.0 core-minutes"},
		{name: "hours", nanos: 12 * NanosPerCoreHour, want: "12.0 core-hours"},
		{name: "days", nanos: 10 * NanosPerCoreDay, want: "10.0 core-days"},
		{name: "years", nanos: 2 * NanosPerCoreYear, want: "2.00 core-years"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := HumanizeEffort(foundationcore.NanosecondsDurationFromInt64(tc.nanos))
			if err != nil {
				t.Fatalf("HumanizeEffort() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("HumanizeEffort() = %q, want %q", got, tc.want)
			}
		})
	}
	if _, err := HumanizeEffort(foundationcore.NanosecondsDurationFromInt64(-1)); !errors.Is(err, ErrContract) {
		t.Fatalf("negative HumanizeEffort() error = %v, want %v", err, ErrContract)
	}
}

func validProjectSnapshot(t *testing.T) ProjectSnapshot {
	t.Helper()
	project, err := ParseProjectID("peachfuzz")
	if err != nil {
		t.Fatal(err)
	}
	effort := foundationcore.NanosecondsDurationFromInt64(NanosPerCoreYear)
	coreYears, err := EffortCoreYears(effort)
	if err != nil {
		t.Fatal(err)
	}
	humanized, err := HumanizeEffort(effort)
	if err != nil {
		t.Fatal(err)
	}
	return ProjectSnapshot{
		Project: project, RecordedSince: foundationcore.UnixNanoTimeFromInt64(1), LastRunAt: foundationcore.UnixNanoTimeFromInt64(2),
		Effort: effort, CoreYears: coreYears, CoreYearsHumanized: humanized, LastOutcome: RunOutcomeCompleted, RunCount: 1,
	}
}

func validRunStats(t *testing.T) RunStats {
	t.Helper()
	project, err := ParseProjectID("witness")
	if err != nil {
		t.Fatal(err)
	}
	packagePath, err := ParsePackageImportPath("example.com/witness/internal/run")
	if err != nil {
		t.Fatal(err)
	}
	fuzz, err := ParseFuzzTargetName("FuzzDecode")
	if err != nil {
		t.Fatal(err)
	}
	runID, err := ParseRunID(strings.Repeat("a", RunIDTextBytes))
	if err != nil {
		t.Fatal(err)
	}
	machine, err := ParseMachineID(strings.Repeat("b", MachineIDTextBytes))
	if err != nil {
		t.Fatal(err)
	}
	commit, err := ParseCommitSHA(strings.Repeat("c", CommitSHATextBytes))
	if err != nil {
		t.Fatal(err)
	}
	return RunStats{RunID: runID, Project: project, PackagePath: packagePath, FuzzTarget: fuzz, Commit: commit, Machine: machine, Outcome: RunOutcomeCompleted, Start: foundationcore.UnixNanoTimeFromInt64(1), End: foundationcore.UnixNanoTimeFromInt64(2), CPU: foundationcore.NanosecondsDurationFromInt64(1), ReportedExecutions: 3, Schema: RunStatsSchema2026}
}
