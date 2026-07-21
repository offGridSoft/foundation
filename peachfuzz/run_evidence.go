package peachfuzz

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	jsonFieldCandidateSightings       = "candidate_sightings"
	jsonFieldCommit                   = "commit"
	jsonFieldCPUNanos                 = "cpu_nanos"
	jsonFieldEndUnixNanos             = "end_unix_nanos"
	jsonFieldFuzzTarget               = "fuzz_target"
	jsonFieldMachine                  = "machine"
	jsonFieldOutcome                  = "outcome"
	jsonFieldPackagePath              = "package_path"
	jsonFieldProject                  = "project"
	jsonFieldReportedExecutions       = "reported_executions"
	jsonFieldRunID                    = "run_id"
	jsonFieldStartUnixNanos           = "start_unix_nanos"
	jsonFieldUniqueCandidatesRetained = "unique_candidates_retained"
)

// RunEvidence is one immutable fuzz-slice delta. It is the fleet-independent
// accounting atom: one machine seals one value under one globally unique
// RunID, and no machine reports or mutates an aggregate.
type RunEvidence struct {
	Project                  ProjectID                `json:"project"`
	PackagePath              PackageImportPath        `json:"package_path"`
	FuzzTarget               FuzzTargetName           `json:"fuzz_target"`
	Commit                   CommitSHA                `json:"commit"`
	Machine                  MachineID                `json:"machine"`
	RunID                    RunID                    `json:"run_id"`
	End                      core.UnixNanoTime        `json:"end_unix_nanos"`
	Start                    core.UnixNanoTime        `json:"start_unix_nanos"`
	CPU                      core.NanosecondsDuration `json:"cpu_nanos"`
	ReportedExecutions       uint64                   `json:"reported_executions"`
	CandidateSightings       uint32                   `json:"candidate_sightings"`
	UniqueCandidatesRetained uint32                   `json:"unique_candidates_retained"`
	Outcome                  RunOutcome               `json:"outcome"`
	Schema                   core.SchemaID            `json:"schema"`
}

func (e RunEvidence) Validate() error {
	checks := []error{e.RunID.Validate(), e.Project.Validate(), e.PackagePath.Validate(), e.FuzzTarget.Validate(), e.Commit.Validate(), e.Machine.Validate(), e.Outcome.Validate(), e.Start.Validate(), e.End.Validate(), e.CPU.Validate(), e.Schema.Validate()}
	for _, err := range checks {
		if err != nil {
			return fmt.Errorf(ErrFmtRunEvidence, errors.Join(ErrContract, err))
		}
	}
	if e.Schema != core.SchemaPeachfuzzRunEvidence || e.End.Time().Before(e.Start.Time()) || e.UniqueCandidatesRetained > e.CandidateSightings {
		return fmt.Errorf(ErrFmtRunEvidence, ErrContract)
	}
	return nil
}

func (e RunEvidence) SigningSchema() core.SchemaID { return e.Schema }

func (e RunEvidence) Canonical(dst []byte) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return appendRunEvidenceJSON(dst, e)
}

func (e RunEvidence) MarshalJSON() ([]byte, error) { return e.Canonical(nil) }

func appendRunEvidenceJSON(dst []byte, e RunEvidence) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, e.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldMachine, e.Machine)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRunID, e.RunID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProject, e.Project)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPackagePath, e.PackagePath)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFuzzTarget, e.FuzzTarget)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, e.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOutcome, e.Outcome)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStartUnixNanos, e.Start)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldEndUnixNanos, e.End)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCPUNanos, e.CPU)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReportedExecutions, e.ReportedExecutions)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCandidateSightings, e.CandidateSightings)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUniqueCandidatesRetained, e.UniqueCandidatesRetained)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}
