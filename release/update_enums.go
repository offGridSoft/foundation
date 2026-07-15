package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type UpdateDecision uint8

const (
	UpdateDecisionUnknown UpdateDecision = iota
	UpdateDecisionCurrent
	UpdateDecisionAvailable
	UpdateDecisionUnavailable
)

const (
	updateDecisionTokenCurrent     = "current"
	updateDecisionTokenAvailable   = "available"
	updateDecisionTokenUnavailable = "unavailable"
)

func updateDecisionNames() [UpdateDecisionUnavailable + 1]string {
	return [...]string{
		UpdateDecisionCurrent:     updateDecisionTokenCurrent,
		UpdateDecisionAvailable:   updateDecisionTokenAvailable,
		UpdateDecisionUnavailable: updateDecisionTokenUnavailable,
	}
}

func (d UpdateDecision) String() string {
	if d.IsValid() {
		return updateDecisionNames()[d]
	}
	return ""
}
func (d UpdateDecision) IsValid() bool {
	return d > UpdateDecisionUnknown && int(d) < len(updateDecisionNames()) && updateDecisionNames()[d] != ""
}
func (d UpdateDecision) Validate() error {
	if !d.IsValid() {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	return nil
}
func (d UpdateDecision) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}
func (d *UpdateDecision) UnmarshalJSON(data []byte) error {
	names := updateDecisionNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], UpdateDecisionCurrent, ErrFmtUpdateCheck)
	if err == nil {
		*d = parsed
	}
	return err
}

type SelfTestCheck uint8

const (
	SelfTestCheckUnknown SelfTestCheck = iota
	SelfTestCheckReleaseStamp
	SelfTestCheckProduct
	SelfTestCheckVersion
	SelfTestCheckCommit
	SelfTestCheckPlatform
	SelfTestCheckBinaryDigest
	SelfTestCheckReleaseAuthority
	SelfTestCheckServerAuthority
)

const SelfTestCheckCount = int(SelfTestCheckServerAuthority)

func selfTestCheckNames() [SelfTestCheckServerAuthority + 1]string {
	return [...]string{
		SelfTestCheckReleaseStamp:     "release_stamp",
		SelfTestCheckProduct:          "product",
		SelfTestCheckVersion:          "version",
		SelfTestCheckCommit:           "commit",
		SelfTestCheckPlatform:         "platform",
		SelfTestCheckBinaryDigest:     "binary_digest",
		SelfTestCheckReleaseAuthority: "release_authority",
		SelfTestCheckServerAuthority:  "server_authority",
	}
}

func (c SelfTestCheck) String() string {
	if c.IsValid() {
		return selfTestCheckNames()[c]
	}
	return ""
}
func (c SelfTestCheck) IsValid() bool {
	return c > SelfTestCheckUnknown && int(c) < len(selfTestCheckNames()) && selfTestCheckNames()[c] != ""
}
func (c SelfTestCheck) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtSelfTestCheck, core.ErrReleaseContract)
	}
	return nil
}
func (c SelfTestCheck) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}
func (c *SelfTestCheck) UnmarshalJSON(data []byte) error {
	names := selfTestCheckNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], SelfTestCheckReleaseStamp, ErrFmtSelfTestCheck)
	if err == nil {
		*c = parsed
	}
	return err
}

type SelfTestStatus uint8

const (
	SelfTestStatusUnknown SelfTestStatus = iota
	SelfTestStatusPassed
	SelfTestStatusFailed
)

func selfTestStatusNames() [SelfTestStatusFailed + 1]string {
	return [...]string{SelfTestStatusPassed: "passed", SelfTestStatusFailed: "failed"}
}

func (s SelfTestStatus) String() string {
	if s.IsValid() {
		return selfTestStatusNames()[s]
	}
	return ""
}
func (s SelfTestStatus) IsValid() bool {
	return s > SelfTestStatusUnknown && int(s) < len(selfTestStatusNames()) && selfTestStatusNames()[s] != ""
}
func (s SelfTestStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtSelfTestStatus, core.ErrReleaseContract)
	}
	return nil
}
func (s SelfTestStatus) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}
func (s *SelfTestStatus) UnmarshalJSON(data []byte) error {
	names := selfTestStatusNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], SelfTestStatusPassed, ErrFmtSelfTestStatus)
	if err == nil {
		*s = parsed
	}
	return err
}

type UpdatePhase uint8

const (
	UpdatePhaseUnknown UpdatePhase = iota
	UpdatePhasePublication
	UpdatePhaseDownload
	UpdatePhaseArtifactVerification
	UpdatePhaseCandidateSelfTest
	UpdatePhaseReplacement
	UpdatePhaseInstalledSelfTest
	UpdatePhaseRollback
)

func updatePhaseNames() [UpdatePhaseRollback + 1]string {
	return [...]string{
		UpdatePhasePublication:          "publication",
		UpdatePhaseDownload:             "download",
		UpdatePhaseArtifactVerification: "artifact_verification",
		UpdatePhaseCandidateSelfTest:    "candidate_self_test",
		UpdatePhaseReplacement:          "replacement",
		UpdatePhaseInstalledSelfTest:    "installed_self_test",
		UpdatePhaseRollback:             "rollback",
	}
}

func (p UpdatePhase) String() string {
	if p.IsValid() {
		return updatePhaseNames()[p]
	}
	return ""
}
func (p UpdatePhase) IsValid() bool {
	return p > UpdatePhaseUnknown && int(p) < len(updatePhaseNames()) && updatePhaseNames()[p] != ""
}
func (p UpdatePhase) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtUpdatePhase, core.ErrReleaseContract)
	}
	return nil
}
func (p UpdatePhase) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}
func (p *UpdatePhase) UnmarshalJSON(data []byte) error {
	names := updatePhaseNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], UpdatePhasePublication, ErrFmtUpdatePhase)
	if err == nil {
		*p = parsed
	}
	return err
}

type UpdateFailure uint8

const (
	UpdateFailureUnknown UpdateFailure = iota
	UpdateFailureContract
	UpdateFailureNetwork
	UpdateFailureIntegrity
	UpdateFailureExecution
	UpdateFailureFilesystem
	UpdateFailureCancelled
	UpdateFailureTimeout
)

func updateFailureNames() [UpdateFailureTimeout + 1]string {
	return [...]string{
		UpdateFailureContract:   "contract",
		UpdateFailureNetwork:    "network",
		UpdateFailureIntegrity:  "integrity",
		UpdateFailureExecution:  "execution",
		UpdateFailureFilesystem: "filesystem",
		UpdateFailureCancelled:  "cancelled",
		UpdateFailureTimeout:    "timeout",
	}
}

func (f UpdateFailure) String() string {
	if f.IsValid() {
		return updateFailureNames()[f]
	}
	return ""
}
func (f UpdateFailure) IsValid() bool {
	return f > UpdateFailureUnknown && int(f) < len(updateFailureNames()) && updateFailureNames()[f] != ""
}
func (f UpdateFailure) Validate() error {
	if !f.IsValid() {
		return fmt.Errorf(ErrFmtUpdateFailure, core.ErrReleaseContract)
	}
	return nil
}
func (f UpdateFailure) MarshalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(f.String())
}
func (f *UpdateFailure) UnmarshalJSON(data []byte) error {
	names := updateFailureNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], UpdateFailureContract, ErrFmtUpdateFailure)
	if err == nil {
		*f = parsed
	}
	return err
}

type RollbackOutcome uint8

const (
	RollbackOutcomeUnknown RollbackOutcome = iota
	RollbackOutcomeNotRequired
	RollbackOutcomeSucceeded
	RollbackOutcomeFailed
)

func rollbackOutcomeNames() [RollbackOutcomeFailed + 1]string {
	return [...]string{
		RollbackOutcomeNotRequired: "not_required",
		RollbackOutcomeSucceeded:   "succeeded",
		RollbackOutcomeFailed:      "failed",
	}
}

func (o RollbackOutcome) String() string {
	if o.IsValid() {
		return rollbackOutcomeNames()[o]
	}
	return ""
}
func (o RollbackOutcome) IsValid() bool {
	return o > RollbackOutcomeUnknown && int(o) < len(rollbackOutcomeNames()) && rollbackOutcomeNames()[o] != ""
}
func (o RollbackOutcome) Validate() error {
	if !o.IsValid() {
		return fmt.Errorf(ErrFmtRollbackOutcome, core.ErrReleaseContract)
	}
	return nil
}
func (o RollbackOutcome) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}
func (o *RollbackOutcome) UnmarshalJSON(data []byte) error {
	names := rollbackOutcomeNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], RollbackOutcomeNotRequired, ErrFmtRollbackOutcome)
	if err == nil {
		*o = parsed
	}
	return err
}

type DiagnosticDisposition uint8

const (
	DiagnosticDispositionUnknown DiagnosticDisposition = iota
	DiagnosticDispositionRecorded
	DiagnosticDispositionDuplicate
)

func diagnosticDispositionNames() [DiagnosticDispositionDuplicate + 1]string {
	return [...]string{
		DiagnosticDispositionRecorded:  "recorded",
		DiagnosticDispositionDuplicate: "duplicate",
	}
}

func (d DiagnosticDisposition) String() string {
	if d.IsValid() {
		return diagnosticDispositionNames()[d]
	}
	return ""
}
func (d DiagnosticDisposition) IsValid() bool {
	return d > DiagnosticDispositionUnknown && int(d) < len(diagnosticDispositionNames()) && diagnosticDispositionNames()[d] != ""
}
func (d DiagnosticDisposition) Validate() error {
	if !d.IsValid() {
		return fmt.Errorf(ErrFmtDiagnosticDisposition, core.ErrReleaseContract)
	}
	return nil
}
func (d DiagnosticDisposition) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}
func (d *DiagnosticDisposition) UnmarshalJSON(data []byte) error {
	names := diagnosticDispositionNames()
	parsed, err := parseUpdateEnumJSON(data, names[:], DiagnosticDispositionRecorded, ErrFmtDiagnosticDisposition)
	if err == nil {
		*d = parsed
	}
	return err
}

func parseUpdateEnumJSON[E ~uint8](data []byte, names []string, first E, format string) (E, error) {
	var zero E
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return zero, fmt.Errorf(format, core.ErrReleaseContract)
	}
	for value := first; int(value) < len(names); value++ {
		if names[value] == token {
			return value, nil
		}
	}
	return zero, fmt.Errorf(format, core.ErrReleaseContract)
}
