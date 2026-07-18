package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	NanosPerCoreSecond = int64(time.Second)
	NanosPerCoreMinute = 60 * NanosPerCoreSecond
	NanosPerCoreHour   = 60 * NanosPerCoreMinute
	NanosPerCoreDay    = 24 * NanosPerCoreHour
	NanosPerCoreYear   = 365*NanosPerCoreDay + 6*NanosPerCoreHour
	CoreYearsFormat    = "%.2f core-years"
	CoreDaysFormat     = "%.1f core-days"
	CoreHoursFormat    = "%.1f core-hours"
	CoreMinutesFormat  = "%.1f core-minutes"
	CoreSecondsFormat  = "%.1f core-seconds"
)

type RunStatsSchema uint8

const (
	RunStatsSchemaUnknown RunStatsSchema = iota
	RunStatsSchema2026
)

func (s RunStatsSchema) String() string {
	if s == RunStatsSchema2026 {
		return core.FoundationVersion2026
	}
	return ""
}

func (s RunStatsSchema) IsValid() bool { return s == RunStatsSchema2026 }

func (s RunStatsSchema) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtRunStats, ErrContract)
	}
	return nil
}

func (s RunStatsSchema) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *RunStatsSchema) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil || token != core.FoundationVersion2026 {
		return fmt.Errorf(ErrFmtRunStats, ErrContract)
	}
	*s = RunStatsSchema2026
	return nil
}

type RunOutcome uint8

const (
	RunOutcomeUnknown RunOutcome = iota
	RunOutcomeCompleted
	RunOutcomeCandidateFound
	RunOutcomeSeedFailure
	RunOutcomeBuildFailed
	RunOutcomeOrdinaryTestFailed
	RunOutcomeTimedOut
	RunOutcomeInterrupted
	RunOutcomeInfrastructureError
	RunOutcomeUnsupported
)

const (
	RunOutcomeTokenCompleted           = "completed"
	RunOutcomeTokenCandidateFound      = "candidate-found"
	RunOutcomeTokenSeedFailure         = "seed-failure"
	RunOutcomeTokenBuildFailed         = "build-failed"
	RunOutcomeTokenOrdinaryTestFailed  = "ordinary-test-failed"
	RunOutcomeTokenTimedOut            = "timed-out"
	RunOutcomeTokenInterrupted         = "interrupted"
	RunOutcomeTokenInfrastructureError = "infrastructure-error"
	RunOutcomeTokenUnsupported         = "unsupported"
)

func (o RunOutcome) String() string {
	if !o.IsValid() {
		return ""
	}
	return [...]string{"", RunOutcomeTokenCompleted, RunOutcomeTokenCandidateFound, RunOutcomeTokenSeedFailure, RunOutcomeTokenBuildFailed, RunOutcomeTokenOrdinaryTestFailed, RunOutcomeTokenTimedOut, RunOutcomeTokenInterrupted, RunOutcomeTokenInfrastructureError, RunOutcomeTokenUnsupported}[o]
}

func (o RunOutcome) IsValid() bool { return o > RunOutcomeUnknown && o <= RunOutcomeUnsupported }
func (o RunOutcome) Validate() error {
	if !o.IsValid() {
		return fmt.Errorf(ErrFmtOutcome, ErrContract)
	}
	return nil
}

func ParseRunOutcome(token string) (RunOutcome, error) {
	for outcome := RunOutcomeCompleted; outcome <= RunOutcomeUnsupported; outcome++ {
		if outcome.String() == token {
			return outcome, nil
		}
	}
	return RunOutcomeUnknown, fmt.Errorf(ErrFmtOutcome, ErrContract)
}

func (o RunOutcome) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}
func (o *RunOutcome) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtOutcome, ErrContract)
	}
	parsed, err := ParseRunOutcome(token)
	if err == nil {
		*o = parsed
	}
	return err
}

type RunStats struct {
	Project                  ProjectID                `json:"project"`
	PackagePath              PackageImportPath        `json:"packagePath"`
	FuzzTarget               FuzzTargetName           `json:"fuzzTarget"`
	Commit                   CommitSHA                `json:"commit"`
	Machine                  MachineID                `json:"machine"`
	RunID                    RunID                    `json:"runId"`
	End                      core.UnixNanoTime        `json:"endUnixNanos"`
	Start                    core.UnixNanoTime        `json:"startUnixNanos"`
	CPU                      core.NanosecondsDuration `json:"cpuNanos"`
	ReportedExecutions       uint64                   `json:"reportedExecutions"`
	CandidateSightings       uint32                   `json:"candidateSightings"`
	UniqueCandidatesRetained uint32                   `json:"uniqueCandidatesRetained"`
	Outcome                  RunOutcome               `json:"outcome"`
	Schema                   RunStatsSchema           `json:"schema"`
}

func (s RunStats) Validate() error {
	checks := []error{s.RunID.Validate(), s.Project.Validate(), s.PackagePath.Validate(), s.FuzzTarget.Validate(), s.Commit.Validate(), s.Machine.Validate(), s.Outcome.Validate(), s.Start.Validate(), s.End.Validate(), s.CPU.Validate(), s.Schema.Validate()}
	for _, err := range checks {
		if err != nil {
			return fmt.Errorf(ErrFmtRunStats, errors.Join(ErrContract, err))
		}
	}
	if s.End.Time().Before(s.Start.Time()) || s.UniqueCandidatesRetained > s.CandidateSightings {
		return fmt.Errorf(ErrFmtRunStats, ErrContract)
	}
	return nil
}

// ProjectSnapshot is the public, bounded projection consumed by product
// pages. Firestore is a query projection only; exact run records and GCS
// evidence remain the source facts behind every value.
type ProjectSnapshot struct {
	CoreYearsHumanized       string                   `json:"coreYearsHumanized"`
	Project                  ProjectID                `json:"project"`
	RecordedSince            core.UnixNanoTime        `json:"recordedSinceUnixNanos"`
	LastRunAt                core.UnixNanoTime        `json:"lastRunAtUnixNanos"`
	CandidateSightings       uint64                   `json:"candidateSightings"`
	CoreYears                float64                  `json:"coreYears"`
	RunCount                 uint64                   `json:"runCount"`
	ReportedExecutions       uint64                   `json:"reportedExecutions"`
	Effort                   core.NanosecondsDuration `json:"cpuNanos"`
	UniqueCandidatesRetained uint64                   `json:"uniqueCandidatesRetained"`
	PackagesExercised        uint64                   `json:"packagesExercised"`
	TargetsExercised         uint64                   `json:"targetsExercised"`
	LastOutcome              RunOutcome               `json:"lastOutcome"`
}

func (ProjectSnapshot) APIBody() {}

func (s ProjectSnapshot) Validate() error {
	checks := []error{s.Project.Validate(), s.RecordedSince.Validate(), s.LastRunAt.Validate(), s.Effort.Validate(), s.LastOutcome.Validate()}
	for _, err := range checks {
		if err != nil {
			return fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
		}
	}
	wantCoreYears, err := EffortCoreYears(s.Effort)
	if err != nil {
		return fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	wantHumanized, err := HumanizeEffort(s.Effort)
	if err != nil {
		return fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	if s.LastRunAt.Time().Before(s.RecordedSince.Time()) || s.RunCount == 0 || s.UniqueCandidatesRetained > s.CandidateSightings ||
		s.CoreYears != wantCoreYears || s.CoreYearsHumanized != wantHumanized {
		return fmt.Errorf(ErrFmtProjectSnapshot, ErrContract)
	}
	return nil
}

func EffortCoreYears(effort core.NanosecondsDuration) (float64, error) {
	if err := effort.Validate(); err != nil {
		return 0, fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	return float64(effort.Nanoseconds()) / float64(NanosPerCoreYear), nil
}

// HumanizeEffort owns the one wording ladder used by producers and every
// public projection. The API returns this alongside raw values so clients do
// not fork the marketing claim's rounding or unit selection.
func HumanizeEffort(effort core.NanosecondsDuration) (string, error) {
	if err := effort.Validate(); err != nil {
		return "", fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	nanos := effort.Nanoseconds()
	switch {
	case nanos >= NanosPerCoreYear:
		return fmt.Sprintf(CoreYearsFormat, float64(nanos)/float64(NanosPerCoreYear)), nil
	case nanos >= NanosPerCoreDay:
		return fmt.Sprintf(CoreDaysFormat, float64(nanos)/float64(NanosPerCoreDay)), nil
	case nanos >= NanosPerCoreHour:
		return fmt.Sprintf(CoreHoursFormat, float64(nanos)/float64(NanosPerCoreHour)), nil
	case nanos >= NanosPerCoreMinute:
		return fmt.Sprintf(CoreMinutesFormat, float64(nanos)/float64(NanosPerCoreMinute)), nil
	default:
		return fmt.Sprintf(CoreSecondsFormat, float64(nanos)/float64(NanosPerCoreSecond)), nil
	}
}

type RecordDisposition uint8

const (
	RecordDispositionUnknown RecordDisposition = iota
	RecordDispositionRecorded
	RecordDispositionDuplicate
)

const (
	RecordDispositionTokenRecorded  = "recorded"
	RecordDispositionTokenDuplicate = "duplicate"
)

func (d RecordDisposition) String() string {
	if d == RecordDispositionRecorded {
		return RecordDispositionTokenRecorded
	}
	if d == RecordDispositionDuplicate {
		return RecordDispositionTokenDuplicate
	}
	return ""
}

func (d RecordDisposition) IsValid() bool {
	return d > RecordDispositionUnknown && d <= RecordDispositionDuplicate
}

func (d RecordDisposition) Validate() error {
	if !d.IsValid() {
		return fmt.Errorf(ErrFmtDisposition, ErrContract)
	}
	return nil
}
func (d RecordDisposition) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}
func (d *RecordDisposition) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtDisposition, ErrContract)
	}
	switch token {
	case RecordDispositionTokenRecorded:
		*d = RecordDispositionRecorded
	case RecordDispositionTokenDuplicate:
		*d = RecordDispositionDuplicate
	default:
		return fmt.Errorf(ErrFmtDisposition, ErrContract)
	}
	return nil
}

type RunStatsReceipt struct {
	RunID       RunID             `json:"runId"`
	Disposition RecordDisposition `json:"disposition"`
}

func (RunStatsReceipt) APIBody() {}

func (r RunStatsReceipt) Validate() error {
	if err := r.RunID.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := r.Disposition.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}
