package peachfuzz

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
	foundationkeygen "github.com/offGridSoft/foundation/v2026/keygen"
)

func TestRunOutcomeOwnsExecutionPolicyOGSBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		outcome            RunOutcome
		failure            bool
		preflightCacheable bool
		retainsDiagnostics bool
	}{
		{name: "completed", outcome: RunOutcomeCompleted, preflightCacheable: true},
		{name: "candidate", outcome: RunOutcomeCandidateFound, retainsDiagnostics: true},
		{name: "seed failure", outcome: RunOutcomeSeedFailure, failure: true, retainsDiagnostics: true},
		{name: "build failure", outcome: RunOutcomeBuildFailed, failure: true, preflightCacheable: true, retainsDiagnostics: true},
		{name: "ordinary test failure", outcome: RunOutcomeOrdinaryTestFailed, failure: true, preflightCacheable: true, retainsDiagnostics: true},
		{name: "timeout", outcome: RunOutcomeTimedOut, failure: true, retainsDiagnostics: true},
		{name: "interrupted", outcome: RunOutcomeInterrupted, retainsDiagnostics: true},
		{name: "infrastructure", outcome: RunOutcomeInfrastructureError, failure: true, retainsDiagnostics: true},
		{name: "unsupported", outcome: RunOutcomeUnsupported, failure: true, retainsDiagnostics: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.outcome.Failure(); got != test.failure {
				t.Fatalf("RunOutcome.Failure() = %t, want %t", got, test.failure)
			}
			if got := test.outcome.PreflightCacheable(); got != test.preflightCacheable {
				t.Fatalf("RunOutcome.PreflightCacheable() = %t, want %t", got, test.preflightCacheable)
			}
			if got := test.outcome.RetainsDiagnostics(); got != test.retainsDiagnostics {
				t.Fatalf("RunOutcome.RetainsDiagnostics() = %t, want %t", got, test.retainsDiagnostics)
			}
		})
	}
}

func TestRunOutcomeOwnsOneClosedTokenDomainOGSBoundary(t *testing.T) {
	t.Parallel()

	outcomes := [...]RunOutcome{
		RunOutcomeCompleted,
		RunOutcomeCandidateFound,
		RunOutcomeSeedFailure,
		RunOutcomeBuildFailed,
		RunOutcomeOrdinaryTestFailed,
		RunOutcomeTimedOut,
		RunOutcomeInterrupted,
		RunOutcomeInfrastructureError,
		RunOutcomeUnsupported,
	}
	for index, outcome := range outcomes {
		if err := outcome.Validate(); err != nil {
			t.Fatalf("RunOutcome(%d).Validate() error = %v, want nil", outcome, err)
		}
		parsed, err := ParseRunOutcome(outcome.String())
		if err != nil || parsed != outcome {
			t.Fatalf("ParseRunOutcome(%q) = (%v, %v), want (%v, nil)", outcome.String(), parsed, err, outcome)
		}
		for _, earlier := range outcomes[:index] {
			if earlier.String() == outcome.String() {
				t.Fatalf("RunOutcome tokens collide: %v and %v", earlier, outcome)
			}
		}
	}
}

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

func TestSignedRunEvidenceBindsMachineToSignerOGSBoundaryTable(t *testing.T) {
	t.Parallel()

	key, machine := signedRunEvidenceKey(t)
	body := validRunEvidence(t)
	body.Machine = machine
	generic, err := foundationcore.SignCanonical(key, body)
	if err != nil {
		t.Fatalf("SignCanonical() error = %v, want nil", err)
	}
	signed, err := NewSignedRunEvidence(generic)
	if err != nil {
		t.Fatalf("NewSignedRunEvidence() error = %v, want nil", err)
	}
	if err := signed.Verify(); err != nil {
		t.Fatalf("SignedRunEvidence.Verify() error = %v, want nil", err)
	}
	encoded, err := foundationcore.EncodeValidatedJSON(signed)
	if err != nil {
		t.Fatalf("EncodeValidatedJSON() error = %v, want nil", err)
	}
	decoded, err := foundationcore.DecodeStrictJSON[SignedRunEvidence](encoded)
	if err != nil {
		t.Fatalf("DecodeStrictJSON() error = %v, want nil", err)
	}
	if decoded != signed {
		t.Fatalf("DecodeStrictJSON() = %+v, want %+v", decoded, signed)
	}

	tests := []struct {
		mutate func(*SignedRunEvidence)
		name   string
	}{
		{name: "different valid machine", mutate: func(candidate *SignedRunEvidence) {
			candidate.Body.Machine, _ = ParseMachineID(strings.Repeat("d", MachineIDTextBytes))
		}},
		{name: "body changed after signing", mutate: func(candidate *SignedRunEvidence) {
			candidate.Body.CPU = foundationcore.NanosecondsDurationFromInt64(candidate.Body.CPU.Nanoseconds() + 1)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := signed
			tc.mutate(&candidate)
			if err := candidate.Verify(); !errors.Is(err, ErrContract) {
				t.Fatalf("SignedRunEvidence.Verify() error = %v, want errors.Is(..., %v)", err, ErrContract)
			}
		})
	}
}

func signedRunEvidenceKey(tb testing.TB) (foundationcore.Ed25519SigningKey, MachineID) {
	tb.Helper()
	generated, err := foundationkeygen.GenerateEd25519SigningKey()
	if err != nil {
		tb.Fatalf("GenerateEd25519SigningKey() error = %v, want nil", err)
	}
	key, err := foundationcore.ParseEd25519SigningKeyBase64(generated.PrivateKeyBase64)
	if err != nil {
		tb.Fatalf("ParseEd25519SigningKeyBase64() error = %v, want nil", err)
	}
	machine, err := MachineIDFromSigningPublicKey(generated.PublicKeyHex)
	if err != nil {
		tb.Fatalf("MachineIDFromSigningPublicKey() error = %v, want nil", err)
	}
	return key, machine
}

func TestMachineEvidenceIdentityIsOneAtomicContractOGSBoundaryTable(t *testing.T) {
	t.Parallel()

	generated, err := foundationkeygen.GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() error = %v, want nil", err)
	}
	identity, err := NewMachineEvidenceIdentity(generated)
	if err != nil {
		t.Fatalf("NewMachineEvidenceIdentity() error = %v, want nil", err)
	}
	privateKey, err := identity.PrivateSigningKey()
	if err != nil {
		t.Fatalf("PrivateSigningKey() error = %v, want nil", err)
	}
	publicKey, err := privateKey.PublicKey()
	if err != nil || publicKey != identity.SigningKey.PublicKeyHex {
		t.Fatalf("PrivateSigningKey().PublicKey() = (%v, %v), want (%v, nil)", publicKey, err, identity.SigningKey.PublicKeyHex)
	}
	encoded, err := foundationcore.EncodeValidatedJSON(identity)
	if err != nil {
		t.Fatalf("EncodeValidatedJSON() error = %v, want nil", err)
	}
	decoded, err := foundationcore.DecodeStrictJSON[MachineEvidenceIdentity](encoded)
	if err != nil || decoded != identity {
		t.Fatalf("DecodeStrictJSON() = (%+v, %v), want (%+v, nil)", decoded, err, identity)
	}

	otherGenerated, err := foundationkeygen.GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey(other) error = %v, want nil", err)
	}
	tests := []struct {
		mutate func(*MachineEvidenceIdentity)
		name   string
	}{
		{name: "machine does not match key", mutate: func(candidate *MachineEvidenceIdentity) {
			candidate.Machine, _ = ParseMachineID(strings.Repeat("d", MachineIDTextBytes))
		}},
		{name: "public key does not match private key", mutate: func(candidate *MachineEvidenceIdentity) {
			candidate.SigningKey.PublicKeyHex = otherGenerated.PublicKeyHex
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := identity
			tc.mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrContract) {
				t.Fatalf("MachineEvidenceIdentity.Validate() error = %v, want errors.Is(..., %v)", err, ErrContract)
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
