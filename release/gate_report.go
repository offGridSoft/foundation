package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const MaxGateChecks uint32 = 32

type GatePhase uint8

const (
	GatePhaseUnknown GatePhase = iota
	GatePhaseFast
	GatePhaseFinal
)

const (
	gatePhaseTokenFast  = "fast"
	gatePhaseTokenFinal = "final"
)

func gatePhaseNames() [GatePhaseFinal + 1]string {
	return [...]string{GatePhaseFast: gatePhaseTokenFast, GatePhaseFinal: gatePhaseTokenFinal}
}

func (p GatePhase) String() string {
	if p.IsValid() {
		return gatePhaseNames()[p]
	}
	return ""
}

func (p GatePhase) IsValid() bool {
	return p > GatePhaseUnknown && int(p) < len(gatePhaseNames()) && gatePhaseNames()[p] != ""
}

func (p GatePhase) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtGatePhase, core.ErrReleaseContract)
	}
	return nil
}

func ParseGatePhase(token string) (GatePhase, error) {
	for phase := GatePhaseFast; int(phase) < len(gatePhaseNames()); phase++ {
		if phase.String() == token {
			return phase, nil
		}
	}
	return GatePhaseUnknown, fmt.Errorf(ErrFmtGatePhase, core.ErrReleaseContract)
}

func (p GatePhase) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *GatePhase) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGatePhase, core.ErrReleaseContract)
	}
	parsed, err := ParseGatePhase(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type GateCheck uint8

const (
	GateCheckUnknown GateCheck = iota
	GateCheckGoFix
	GateCheckGoVet
	GateCheckFieldAlignment
	GateCheckGoCyclo
	GateCheckGoConst
	GateCheckNilAway
	GateCheckErrCheck
	GateCheckStaticCheck
	GateCheckDeadCode
	GateCheckDeadCodeTests
	GateCheckGoVulnCheck
	GateCheckGoSec
	GateCheckWitnessLint
	GateCheckGoTest
	GateCheckGoTestRaceShuffle
	GateCheckGitTreeClean
)

const (
	gateCheckTokenGoFix             = "go_fix"
	gateCheckTokenGoVet             = "go_vet"
	gateCheckTokenFieldAlignment    = "field_alignment"
	gateCheckTokenGoCyclo           = "go_cyclo"
	gateCheckTokenGoConst           = "go_const"
	gateCheckTokenNilAway           = "nil_away"
	gateCheckTokenErrCheck          = "err_check"
	gateCheckTokenStaticCheck       = "static_check"
	gateCheckTokenDeadCode          = "dead_code"
	gateCheckTokenDeadCodeTests     = "dead_code_tests"
	gateCheckTokenGoVulnCheck       = "go_vuln_check"
	gateCheckTokenGoSec             = "go_sec"
	gateCheckTokenWitnessLint       = "witness_lint"
	gateCheckTokenGoTest            = "go_test"
	gateCheckTokenGoTestRaceShuffle = "go_test_race_shuffle"
	gateCheckTokenGitTreeClean      = "git_tree_clean"
)

func gateCheckNames() [GateCheckGitTreeClean + 1]string {
	return [...]string{
		GateCheckGoFix:             gateCheckTokenGoFix,
		GateCheckGoVet:             gateCheckTokenGoVet,
		GateCheckFieldAlignment:    gateCheckTokenFieldAlignment,
		GateCheckGoCyclo:           gateCheckTokenGoCyclo,
		GateCheckGoConst:           gateCheckTokenGoConst,
		GateCheckNilAway:           gateCheckTokenNilAway,
		GateCheckErrCheck:          gateCheckTokenErrCheck,
		GateCheckStaticCheck:       gateCheckTokenStaticCheck,
		GateCheckDeadCode:          gateCheckTokenDeadCode,
		GateCheckDeadCodeTests:     gateCheckTokenDeadCodeTests,
		GateCheckGoVulnCheck:       gateCheckTokenGoVulnCheck,
		GateCheckGoSec:             gateCheckTokenGoSec,
		GateCheckWitnessLint:       gateCheckTokenWitnessLint,
		GateCheckGoTest:            gateCheckTokenGoTest,
		GateCheckGoTestRaceShuffle: gateCheckTokenGoTestRaceShuffle,
		GateCheckGitTreeClean:      gateCheckTokenGitTreeClean,
	}
}

func (c GateCheck) String() string {
	if c.IsValid() {
		return gateCheckNames()[c]
	}
	return ""
}

func (c GateCheck) IsValid() bool {
	return c > GateCheckUnknown && int(c) < len(gateCheckNames()) && gateCheckNames()[c] != ""
}

func (c GateCheck) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtGateCheck, core.ErrReleaseContract)
	}
	return nil
}

func ParseGateCheck(token string) (GateCheck, error) {
	for check := GateCheckGoFix; int(check) < len(gateCheckNames()); check++ {
		if check.String() == token {
			return check, nil
		}
	}
	return GateCheckUnknown, fmt.Errorf(ErrFmtGateCheck, core.ErrReleaseContract)
}

func (c GateCheck) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func (c *GateCheck) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateCheck, core.ErrReleaseContract)
	}
	parsed, err := ParseGateCheck(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func GateChecks(phase GatePhase) ([]GateCheck, error) {
	if err := phase.Validate(); err != nil {
		return nil, err
	}
	if phase == GatePhaseFinal {
		return []GateCheck{GateCheckGoTest, GateCheckGoTestRaceShuffle, GateCheckGitTreeClean}, nil
	}
	return []GateCheck{
		GateCheckGoFix, GateCheckGoVet, GateCheckFieldAlignment, GateCheckGoCyclo, GateCheckGoConst,
		GateCheckNilAway, GateCheckErrCheck, GateCheckStaticCheck, GateCheckDeadCode, GateCheckDeadCodeTests,
		GateCheckGoVulnCheck, GateCheckGoSec, GateCheckWitnessLint, GateCheckGitTreeClean,
	}, nil
}

type GateCheckResult struct {
	StdoutSHA256 core.SHA256Hex    `json:"stdout_sha256"`
	StderrSHA256 core.SHA256Hex    `json:"stderr_sha256"`
	StartedAt    core.UnixNanoTime `json:"started_at"`
	FinishedAt   core.UnixNanoTime `json:"finished_at"`
	Check        GateCheck         `json:"check"`
	Status       CommandStatus     `json:"status"`
}

func (r GateCheckResult) Validate() error {
	if err := r.Check.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtGateCheckResult, err)
	}
	if r.Status != CommandStatusSucceeded {
		return fmt.Errorf(ErrFmtGateCheckResult, core.ErrReleaseContract)
	}
	if err := r.StdoutSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtGateCheckResult, err)
	}
	if err := r.StderrSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtGateCheckResult, err)
	}
	return validateGateTimes(ErrFmtGateCheckResult, r.StartedAt, r.FinishedAt)
}

func (r GateCheckResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldCheck, r.Check)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStatus, r.Status)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStartedAt, r.StartedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFinishedAt, r.FinishedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStdoutSHA256, r.StdoutSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStderrSHA256, r.StderrSHA256)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type GateReport struct {
	Version    core.ProductVersion `json:"version"`
	ReleaseID  ReleaseID           `json:"release_id"`
	Commit     core.BuildCommit    `json:"commit"`
	Checks     []GateCheckResult   `json:"checks"`
	StartedAt  core.UnixNanoTime   `json:"started_at"`
	FinishedAt core.UnixNanoTime   `json:"finished_at"`
	CheckCount uint32              `json:"check_count"`
	Schema     core.SchemaID       `json:"schema"`
	Phase      GatePhase           `json:"phase"`
	Product    core.Product        `json:"product"`
}

func (r GateReport) Validate() error {
	if r.Schema != core.SchemaReleaseGateReport {
		return fmt.Errorf(ErrFmtGateReport, core.ErrReleaseContract)
	}
	if err := ValidateReleaseIDIdentity(r.ReleaseID, r.Product, r.Version, r.Commit); err != nil {
		return wrapReleaseContract(ErrFmtGateReport, err)
	}
	if err := validateGateTimes(ErrFmtGateReport, r.StartedAt, r.FinishedAt); err != nil {
		return err
	}
	return validateGateCheckResults(r)
}

func validateGateCheckResults(report GateReport) error {
	if err := (core.CollectionCardinality{
		Length: len(report.Checks), DeclaredCount: report.CheckCount,
		Minimum: 1, Maximum: MaxGateChecks, RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtGateReport, core.ErrReleaseContract)
	}
	expected, err := GateChecks(report.Phase)
	if err != nil || len(expected) != len(report.Checks) {
		return fmt.Errorf(ErrFmtGateReport, core.ErrReleaseContract)
	}
	for index, result := range report.Checks {
		if err := result.Validate(); err != nil || result.Check != expected[index] || result.StartedAt.Before(report.StartedAt) || report.FinishedAt.Before(result.FinishedAt) {
			return fmt.Errorf(ErrFmtGateReport, core.ErrReleaseContract)
		}
	}
	return nil
}

func validateGateTimes(format string, started, finished core.UnixNanoTime) error {
	if err := core.ValidateRequiredUnixNanoTime(started); err != nil {
		return wrapReleaseContract(format, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(finished); err != nil || finished.Before(started) {
		return fmt.Errorf(format, core.ErrReleaseContract)
	}
	return nil
}

func (r GateReport) Canonical(dst []byte) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendGateReportJSON(dst, r)
}

func (r GateReport) SigningSchema() core.SchemaID {
	return r.Schema
}

func (r GateReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendGateReportJSON(nil, r)
}

func appendGateReportJSON(dst []byte, r GateReport) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, r.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPhase, r.Phase)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, r.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, r.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, r.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, r.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStartedAt, r.StartedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFinishedAt, r.FinishedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldChecks, r.Checks)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckCount, r.CheckCount)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

var _ core.Validatable = GateCheckResult{}
var _ core.CanonicalBody = GateReport{}
