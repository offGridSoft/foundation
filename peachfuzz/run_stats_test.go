package peachfuzz

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
)

func TestRunEvidenceCanonicalRoundTripPinsSigningDomain(t *testing.T) {
	t.Parallel()
	evidence := validRunEvidence(t)
	canonical, err := evidence.Canonical(nil)
	if err != nil {
		t.Fatalf("RunEvidence.Canonical() error = %v, want nil", err)
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("json.Marshal(RunEvidence) error = %v, want nil", err)
	}
	if !bytes.Equal(encoded, canonical) {
		t.Fatalf("json.Marshal(RunEvidence) = %q, want canonical %q", encoded, canonical)
	}
	decoded, err := foundationcore.DecodeStrictJSONStructure[RunEvidence](canonical)
	if err != nil {
		t.Fatalf("DecodeStrictJSONStructure[RunEvidence]() error = %v, want nil", err)
	}
	if decoded != evidence {
		t.Fatalf("decoded RunEvidence = %#v, want %#v", decoded, evidence)
	}
	if evidence.SigningSchema().ResolveSigningDomain() != foundationcore.SigningDomainPeachfuzzRunEvidence {
		t.Fatalf("RunEvidence signing domain = %v, want %v", evidence.SigningSchema().ResolveSigningDomain(), foundationcore.SigningDomainPeachfuzzRunEvidence)
	}
}

func TestRunEvidenceValidateOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	valid := validRunEvidence(t)
	tests := []struct {
		mutate func(*RunEvidence)
		name   string
	}{
		{name: "invalid run id", mutate: func(s *RunEvidence) { s.RunID = RunID{} }},
		{name: "invalid project", mutate: func(s *RunEvidence) { s.Project = ProjectID{} }},
		{name: "invalid package", mutate: func(s *RunEvidence) { s.PackagePath = PackageImportPath{} }},
		{name: "invalid target", mutate: func(s *RunEvidence) { s.FuzzTarget = FuzzTargetName{} }},
		{name: "invalid commit", mutate: func(s *RunEvidence) { s.Commit = CommitSHA{} }},
		{name: "invalid machine", mutate: func(s *RunEvidence) { s.Machine = MachineID{} }},
		{name: "invalid outcome", mutate: func(s *RunEvidence) { s.Outcome = RunOutcomeUnknown }},
		{name: "missing start", mutate: func(s *RunEvidence) { s.Start = foundationcore.UnixNanoTime{} }},
		{name: "missing end", mutate: func(s *RunEvidence) { s.End = foundationcore.UnixNanoTime{} }},
		{name: "reversed interval", mutate: func(s *RunEvidence) { s.End = foundationcore.UnixNanoTimeFromInt64(0) }},
		{name: "negative cpu", mutate: func(s *RunEvidence) { s.CPU = foundationcore.NanosecondsDurationFromInt64(-1) }},
		{name: "retained exceeds sightings", mutate: func(s *RunEvidence) { s.CandidateSightings = 1; s.UniqueCandidatesRetained = 2 }},
		{name: "wrong schema", mutate: func(s *RunEvidence) { s.Schema = foundationcore.SchemaUnknown }},
		{name: "unknown executions carry no count", mutate: func(s *RunEvidence) { s.Executions = ExecutionObservation{Count: 1} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			test.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("RunEvidence.Validate() error = %v, want %v", err, ErrContract)
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
		{name: "zero packages", mutate: func(s *ProjectSnapshot) { s.PackagesExercised = 0 }},
		{name: "zero targets", mutate: func(s *ProjectSnapshot) { s.TargetsExercised = 0 }},
		{name: "packages exceed targets", mutate: func(s *ProjectSnapshot) { s.PackagesExercised = 3; s.TargetsExercised = 2 }},
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

func TestHumanizeEffortTruncatesEveryPublicTrustValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		want  string
		nanos int64
	}{
		{name: "years never round to next hundredth", nanos: 2*NanosPerCoreYear - 1, want: "1.99 core-years"},
		{name: "days never round to next tenth", nanos: 11*NanosPerCoreDay - 1, want: "10.9 core-days"},
		{name: "hours never round to next tenth", nanos: 13*NanosPerCoreHour - 1, want: "12.9 core-hours"},
		{name: "minutes never round to next tenth", nanos: 31*NanosPerCoreMinute - 1, want: "30.9 core-minutes"},
		{name: "seconds never round to next tenth", nanos: 31*NanosPerCoreSecond - 1, want: "30.9 core-seconds"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := HumanizeEffort(foundationcore.NanosecondsDurationFromInt64(tc.nanos))
			if err != nil {
				t.Fatalf("HumanizeEffort() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("HumanizeEffort() = %q, want truncation %q", got, tc.want)
			}
		})
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
		PackagesExercised: 1, TargetsExercised: 1,
	}
}

func validRunEvidence(t *testing.T) RunEvidence {
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
	return RunEvidence{RunID: runID, Project: project, PackagePath: packagePath, FuzzTarget: fuzz, Commit: commit, Machine: machine, Outcome: RunOutcomeCompleted, Start: foundationcore.UnixNanoTimeFromInt64(1), End: foundationcore.UnixNanoTimeFromInt64(2), CPU: foundationcore.NanosecondsDurationFromInt64(1), Executions: ExecutionObservation{Count: 3, Known: true}, Schema: foundationcore.SchemaPeachfuzzRunEvidence}
}
