package release

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestGateEnumWireContractsHostileTable(t *testing.T) {
	t.Parallel()

	for phase := GatePhaseFast; phase <= GatePhaseFinal; phase++ {
		requireGatePhaseRoundTrip(t, phase)
	}
	for check := GateCheckGoFix; check <= GateCheckGitTreeClean; check++ {
		requireGateCheckRoundTrip(t, check)
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`""`), []byte(`"unknown"`), []byte(`1`)} {
		var phase GatePhase
		if err := json.Unmarshal(raw, &phase); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("json.Unmarshal GatePhase(%q) error = %v, want ErrReleaseContract", raw, err)
		}
		var check GateCheck
		if err := json.Unmarshal(raw, &check); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("json.Unmarshal GateCheck(%q) error = %v, want ErrReleaseContract", raw, err)
		}
	}
}

func TestGateReportHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate  func(*GateReport)
		name    string
		wantErr bool
	}{
		{name: "valid fast report", mutate: func(*GateReport) {}},
		{name: "valid final report", mutate: makeFinalGateReport},
		{name: "zero schema rejected", wantErr: true, mutate: func(r *GateReport) { r.Schema = core.SchemaUnknown }},
		{name: "cross schema rejected", wantErr: true, mutate: func(r *GateReport) { r.Schema = core.SchemaReleaseManifest }},
		{name: "unknown phase rejected", wantErr: true, mutate: func(r *GateReport) { r.Phase = GatePhaseUnknown }},
		{name: "wrong product rejected", wantErr: true, mutate: func(r *GateReport) { r.Product = core.ProductBug }},
		{name: "zero commit rejected", wantErr: true, mutate: func(r *GateReport) { r.Commit = core.BuildCommit{} }},
		{name: "foreign release ID rejected", wantErr: true, mutate: func(r *GateReport) { r.ReleaseID = mustOtherReleaseID(t) }},
		{name: "zero start rejected", wantErr: true, mutate: func(r *GateReport) { r.StartedAt = core.UnixNanoTime{} }},
		{name: "finish before start rejected", wantErr: true, mutate: func(r *GateReport) { r.FinishedAt = core.UnixNanoTimeFromInt64(r.StartedAt.UnixNano() - 1) }},
		{name: "count mismatch rejected", wantErr: true, mutate: func(r *GateReport) { r.CheckCount++ }},
		{name: "missing check rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks = r.Checks[:len(r.Checks)-1]; r.CheckCount-- }},
		{name: "reordered check rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks[0], r.Checks[1] = r.Checks[1], r.Checks[0] }},
		{name: "failed check rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks[0].Status = CommandStatusFailed }},
		{name: "zero stdout digest rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks[0].StdoutSHA256 = core.SHA256Hex{} }},
		{name: "zero stderr digest rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks[0].StderrSHA256 = core.SHA256Hex{} }},
		{name: "check before report rejected", wantErr: true, mutate: func(r *GateReport) { r.Checks[0].StartedAt = core.UnixNanoTimeFromInt64(r.StartedAt.UnixNano() - 1) }},
		{name: "check after report rejected", wantErr: true, mutate: func(r *GateReport) {
			r.Checks[len(r.Checks)-1].FinishedAt = core.UnixNanoTimeFromInt64(r.FinishedAt.UnixNano() + 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			report := validGateReport(t, GatePhaseFast)
			tc.mutate(&report)
			err := report.Validate()
			if !tc.wantErr && err != nil {
				t.Fatalf("GateReport.Validate() error = %v, want nil", err)
			}
			if tc.wantErr && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GateReport.Validate() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestGateReportCanonicalStrictRoundTrip(t *testing.T) {
	t.Parallel()

	report := validGateReport(t, GatePhaseFast)
	canonical, err := report.Canonical(nil)
	if err != nil {
		t.Fatalf("GateReport.Canonical() error = %v", err)
	}
	decoded, err := core.DecodeStrictJSON[GateReport](canonical)
	if err != nil {
		t.Fatalf("DecodeStrictJSON[GateReport]() error = %v", err)
	}
	roundTrip, err := decoded.Canonical(nil)
	if err != nil {
		t.Fatalf("decoded GateReport.Canonical() error = %v", err)
	}
	if string(roundTrip) != string(canonical) {
		t.Fatalf("GateReport canonical round trip = %s, want %s", roundTrip, canonical)
	}
}

func requireGatePhaseRoundTrip(t *testing.T, phase GatePhase) {
	t.Helper()
	encoded, err := json.Marshal(phase)
	if err != nil {
		t.Fatalf("json.Marshal(GatePhase(%d)) error = %v", phase, err)
	}
	var decoded GatePhase
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(GatePhase(%d)) error = %v", phase, err)
	}
	if decoded != phase {
		t.Fatalf("GatePhase JSON round trip = %d, want %d", decoded, phase)
	}
}

func requireGateCheckRoundTrip(t *testing.T, check GateCheck) {
	t.Helper()
	encoded, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("json.Marshal(GateCheck(%d)) error = %v", check, err)
	}
	var decoded GateCheck
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(GateCheck(%d)) error = %v", check, err)
	}
	if decoded != check {
		t.Fatalf("GateCheck JSON round trip = %d, want %d", decoded, check)
	}
}

func makeFinalGateReport(report *GateReport) {
	final := gateReportForPhase(report, GatePhaseFinal)
	*report = final
}

func validGateReport(t *testing.T, phase GatePhase) GateReport {
	t.Helper()
	checks, err := GateChecks(phase)
	if err != nil {
		t.Fatal(err)
	}
	started := core.UnixNanoTimeFromInt64(1_800_000_000_000_000_000)
	results := make([]GateCheckResult, 0, len(checks))
	for index, check := range checks {
		checkStart := core.UnixNanoTimeFromInt64(started.UnixNano() + int64(index*2+1))
		results = append(results, GateCheckResult{
			Check: check, Status: CommandStatusSucceeded, StartedAt: checkStart,
			FinishedAt:   core.UnixNanoTimeFromInt64(checkStart.UnixNano() + 1),
			StdoutSHA256: mustSHA256(t, "a"), StderrSHA256: mustSHA256(t, "b"),
		})
	}
	return GateReport{
		Schema: core.SchemaReleaseGateReport, Phase: phase, Product: core.ProductWitness,
		Version: mustVersion(t), ReleaseID: mustReleaseID(t), Commit: mustCommit(t),
		StartedAt: started, FinishedAt: core.UnixNanoTimeFromInt64(started.UnixNano() + int64(len(checks)*2+2)),
		Checks: results, CheckCount: uint32(len(results)),
	}
}

func gateReportForPhase(report *GateReport, phase GatePhase) GateReport {
	checks, _ := GateChecks(phase)
	results := make([]GateCheckResult, 0, len(checks))
	for index, check := range checks {
		result := report.Checks[index]
		result.Check = check
		results = append(results, result)
	}
	report.Phase = phase
	report.Checks = results
	report.CheckCount = uint32(len(results))
	return *report
}
