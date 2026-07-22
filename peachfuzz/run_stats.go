package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	NanosPerCoreSecond = 1_000_000_000
	NanosPerCoreMinute = 60 * NanosPerCoreSecond
	NanosPerCoreHour   = 60 * NanosPerCoreMinute
	NanosPerCoreDay    = 24 * NanosPerCoreHour
	NanosPerCoreYear   = 365*NanosPerCoreDay + 6*NanosPerCoreHour
	CoreYearsFormat    = "%s.%02d core-years"
	CoreDaysFormat     = "%s.%01d core-days"
	CoreHoursFormat    = "%s.%01d core-hours"
	CoreMinutesFormat  = "%s.%01d core-minutes"
	CoreSecondsFormat  = "%s.%01d core-seconds"
	CoreYearsScale     = 100
	CoreSubYearScale   = 10
)

type effortDisplayUnit uint8

const (
	effortDisplayUnitUnknown effortDisplayUnit = iota
	effortDisplayUnitYears
	effortDisplayUnitDays
	effortDisplayUnitHours
	effortDisplayUnitMinutes
	effortDisplayUnitSeconds
)

type RunOutcome uint8

const (
	RunOutcomeUnknown RunOutcome = iota
	RunOutcomeCompleted
	RunOutcomeCandidateFound
	RunOutcomeSeedFailure
	RunOutcomeBuildFailed
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
	RunOutcomeTokenTimedOut            = "timed-out"
	RunOutcomeTokenInterrupted         = "interrupted"
	RunOutcomeTokenInfrastructureError = "infrastructure-error"
	RunOutcomeTokenUnsupported         = "unsupported"
)

func (o RunOutcome) String() string {
	if !o.IsValid() {
		return ""
	}
	return [...]string{"", RunOutcomeTokenCompleted, RunOutcomeTokenCandidateFound, RunOutcomeTokenSeedFailure, RunOutcomeTokenBuildFailed, RunOutcomeTokenTimedOut, RunOutcomeTokenInterrupted, RunOutcomeTokenInfrastructureError, RunOutcomeTokenUnsupported}[o]
}

func (o RunOutcome) IsValid() bool { return o > RunOutcomeUnknown && o <= RunOutcomeUnsupported }

// Failure reports the scheduler-policy subset. CandidateFound is success:
// the fuzzer produced exactly the evidence it was asked to produce.
func (o RunOutcome) Failure() bool {
	switch o {
	case RunOutcomeSeedFailure, RunOutcomeBuildFailed,
		RunOutcomeTimedOut, RunOutcomeInfrastructureError, RunOutcomeUnsupported:
		return true
	default:
		return false
	}
}

// RetainsDiagnostics reports whether a durable run record may retain the
// bounded child-output tails needed to explain the classified outcome.
func (o RunOutcome) RetainsDiagnostics() bool {
	return o.IsValid() && o != RunOutcomeCompleted
}

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

// ProjectSnapshot is the public, bounded projection consumed by product
// pages. Firestore is a query projection only; exact run records and GCS
// evidence remain the source facts behind every value.
type ProjectSnapshot struct {
	CoreYearsHumanized       string            `json:"coreYearsHumanized"`
	Project                  ProjectID         `json:"project"`
	RecordedSince            core.UnixNanoTime `json:"recordedSinceUnixNanos"`
	LastRunAt                core.UnixNanoTime `json:"lastRunAtUnixNanos"`
	CandidateSightings       uint64            `json:"candidateSightings"`
	CoreYears                float64           `json:"coreYears"`
	RunCount                 uint64            `json:"runCount"`
	ReportedExecutions       uint64            `json:"reportedExecutions"`
	Effort                   EffortNanoseconds `json:"cpuNanos"`
	UniqueCandidatesRetained uint64            `json:"uniqueCandidatesRetained"`
	PackagesExercised        uint64            `json:"packagesExercised"`
	TargetsExercised         uint64            `json:"targetsExercised"`
	LastOutcome              RunOutcome        `json:"lastOutcome"`
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
	if !s.matchesDerivedValues(wantCoreYears, wantHumanized) {
		return fmt.Errorf(ErrFmtProjectSnapshot, ErrContract)
	}
	return nil
}

func (s ProjectSnapshot) matchesDerivedValues(wantCoreYears float64, wantHumanized string) bool {
	validInterval := !s.LastRunAt.Time().Before(s.RecordedSince.Time())
	validCounters := s.RunCount > 0 && s.UniqueCandidatesRetained <= s.CandidateSightings
	validCoverage := s.PackagesExercised > 0 && s.TargetsExercised > 0 && s.PackagesExercised <= s.TargetsExercised
	return validInterval && validCounters && validCoverage && s.CoreYears == wantCoreYears && s.CoreYearsHumanized == wantHumanized
}

func EffortCoreYears(effort EffortNanoseconds) (float64, error) {
	if err := effort.Validate(); err != nil {
		return 0, fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	return effort.Float64() / float64(NanosPerCoreYear), nil
}

// HumanizeEffort owns the one wording ladder used by producers and every
// public projection. The API returns this alongside raw values so clients do
// not fork the marketing claim's rounding or unit selection.
func HumanizeEffort(effort EffortNanoseconds) (string, error) {
	if err := effort.Validate(); err != nil {
		return "", fmt.Errorf(ErrFmtProjectSnapshot, errors.Join(ErrContract, err))
	}
	switch {
	case effort.High != 0 || effort.Low >= NanosPerCoreYear:
		return formatTruncatedEffort(effort, effortDisplayUnitYears)
	case effort.Low >= NanosPerCoreDay:
		return formatTruncatedEffort(effort, effortDisplayUnitDays)
	case effort.Low >= NanosPerCoreHour:
		return formatTruncatedEffort(effort, effortDisplayUnitHours)
	case effort.Low >= NanosPerCoreMinute:
		return formatTruncatedEffort(effort, effortDisplayUnitMinutes)
	default:
		return formatTruncatedEffort(effort, effortDisplayUnitSeconds)
	}
}

func formatTruncatedEffort(effort EffortNanoseconds, unit effortDisplayUnit) (string, error) {
	divisor, scale, format, err := unit.contract()
	if err != nil {
		return "", err
	}
	whole, remainder, err := effort.quotientRemainder(divisor)
	if err != nil {
		return "", err
	}
	fraction := remainder * scale / divisor
	return fmt.Sprintf(format, whole.Decimal(), fraction), nil
}

func (u effortDisplayUnit) contract() (uint64, uint64, string, error) {
	switch u {
	case effortDisplayUnitYears:
		return NanosPerCoreYear, CoreYearsScale, CoreYearsFormat, nil
	case effortDisplayUnitDays:
		return NanosPerCoreDay, CoreSubYearScale, CoreDaysFormat, nil
	case effortDisplayUnitHours:
		return NanosPerCoreHour, CoreSubYearScale, CoreHoursFormat, nil
	case effortDisplayUnitMinutes:
		return NanosPerCoreMinute, CoreSubYearScale, CoreMinutesFormat, nil
	case effortDisplayUnitSeconds:
		return NanosPerCoreSecond, CoreSubYearScale, CoreSecondsFormat, nil
	default:
		return 0, 0, "", fmt.Errorf(ErrFmtProjectSnapshot, ErrContract)
	}
}
