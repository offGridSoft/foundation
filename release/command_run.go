package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type CommandKind uint8

const (
	CommandKindUnknown CommandKind = iota
	CommandKindRelease
	CommandKindDeploy
)

const (
	CommandKindTokenRelease = "release"
	CommandKindTokenDeploy  = "deploy"
)

func commandKindNames() [CommandKindDeploy + 1]string {
	return [...]string{
		CommandKindRelease: CommandKindTokenRelease,
		CommandKindDeploy:  CommandKindTokenDeploy,
	}
}

func (k CommandKind) String() string {
	if k.IsValid() {
		return commandKindNames()[k]
	}
	return ""
}

func (k CommandKind) IsValid() bool {
	return k > CommandKindUnknown && int(k) < len(commandKindNames()) && commandKindNames()[k] != ""
}

func (k CommandKind) Validate() error {
	if !k.IsValid() {
		return fmt.Errorf(ErrFmtCommandKind, core.ErrReleaseContract)
	}
	return nil
}

func ParseCommandKind(token string) (CommandKind, error) {
	for kind := CommandKindRelease; int(kind) < len(commandKindNames()); kind++ {
		if commandKindNames()[kind] == token {
			return kind, nil
		}
	}
	return CommandKindUnknown, fmt.Errorf(ErrFmtCommandKind, core.ErrReleaseContract)
}

func (k CommandKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.String())
}

func (k *CommandKind) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCommandKind, core.ErrReleaseContract)
	}
	parsed, err := ParseCommandKind(token)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

type CommandStatus uint8

const (
	CommandStatusUnknown CommandStatus = iota
	CommandStatusSucceeded
	CommandStatusFailed
)

const (
	CommandStatusTokenSucceeded = "succeeded"
	CommandStatusTokenFailed    = "failed"
)

func commandStatusNames() [CommandStatusFailed + 1]string {
	return [...]string{
		CommandStatusSucceeded: CommandStatusTokenSucceeded,
		CommandStatusFailed:    CommandStatusTokenFailed,
	}
}

func (s CommandStatus) String() string {
	if s.IsValid() {
		return commandStatusNames()[s]
	}
	return ""
}

func (s CommandStatus) IsValid() bool {
	return s > CommandStatusUnknown && int(s) < len(commandStatusNames()) && commandStatusNames()[s] != ""
}

func (s CommandStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtCommandStatus, core.ErrReleaseContract)
	}
	return nil
}

func ParseCommandStatus(token string) (CommandStatus, error) {
	for status := CommandStatusSucceeded; int(status) < len(commandStatusNames()); status++ {
		if commandStatusNames()[status] == token {
			return status, nil
		}
	}
	return CommandStatusUnknown, fmt.Errorf(ErrFmtCommandStatus, core.ErrReleaseContract)
}

func (s CommandStatus) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *CommandStatus) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCommandStatus, core.ErrReleaseContract)
	}
	parsed, err := ParseCommandStatus(token)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type TreeState uint8

const (
	TreeStateUnknown TreeState = iota
	TreeStateClean
	TreeStateDirty
)

const (
	TreeStateTokenClean = "clean"
	TreeStateTokenDirty = "dirty"
)

func treeStateNames() [TreeStateDirty + 1]string {
	return [...]string{
		TreeStateClean: TreeStateTokenClean,
		TreeStateDirty: TreeStateTokenDirty,
	}
}

func (s TreeState) String() string {
	if s.IsValid() {
		return treeStateNames()[s]
	}
	return ""
}

func (s TreeState) IsValid() bool {
	return s > TreeStateUnknown && int(s) < len(treeStateNames()) && treeStateNames()[s] != ""
}

func (s TreeState) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtTreeState, core.ErrReleaseContract)
	}
	return nil
}

func ParseTreeState(token string) (TreeState, error) {
	for state := TreeStateClean; int(state) < len(treeStateNames()); state++ {
		if treeStateNames()[state] == token {
			return state, nil
		}
	}
	return TreeStateUnknown, fmt.Errorf(ErrFmtTreeState, core.ErrReleaseContract)
}

func (s TreeState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *TreeState) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtTreeState, core.ErrReleaseContract)
	}
	parsed, err := ParseTreeState(token)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type MachineIdentity struct {
	HostnameSHA256 core.SHA256Hex `json:"hostname_sha256"`
	UserSHA256     core.SHA256Hex `json:"user_sha256"`
	Platform       core.Platform  `json:"platform"`
}

func (m MachineIdentity) Validate() error {
	if err := m.Platform.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtMachine, err)
	}
	if err := m.HostnameSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtMachine, err)
	}
	if err := m.UserSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtMachine, err)
	}
	return nil
}

func (m MachineIdentity) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldPlatform, m.Platform)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldHostnameSHA256, m.HostnameSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUserSHA256, m.UserSHA256)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type CommandRun struct {
	Version        core.ProductVersion `json:"version"`
	GitCommit      core.BuildCommit    `json:"git_commit"`
	ReleaseID      ReleaseID           `json:"release_id"`
	EvidenceRef    ObjectKey           `json:"evidence_ref"`
	OperatorSHA256 core.SHA256Hex      `json:"operator_sha256"`
	Machine        MachineIdentity     `json:"machine"`
	StartedAt      core.UnixNanoTime   `json:"started_at"`
	FinishedAt     core.UnixNanoTime   `json:"finished_at"`
	Kind           CommandKind         `json:"kind"`
	Status         CommandStatus       `json:"status"`
	TreeState      TreeState           `json:"tree_state"`
	Product        core.Product        `json:"product"`
}

func (r CommandRun) Validate() error {
	if err := validateCommandRunIdentity(r); err != nil {
		return err
	}
	if err := validateCommandRunTimes(r); err != nil {
		return err
	}
	if err := r.EvidenceRef.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	return nil
}

func (r CommandRun) Canonical(dst []byte) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendCommandRunJSON(dst, r)
}

func (r CommandRun) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendCommandRunJSON(nil, r)
}

func appendCommandRunJSON(dst []byte, r CommandRun) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldKind, r.Kind)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStatus, r.Status)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTreeState, r.TreeState)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, r.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, r.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, r.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldGitCommit, r.GitCommit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldMachine, r.Machine)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOperatorSHA256, r.OperatorSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStartedAt, r.StartedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFinishedAt, r.FinishedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldEvidenceRef, r.EvidenceRef)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateCommandRunIdentity(r CommandRun) error {
	if err := r.Kind.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.Status.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.TreeState.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.GitCommit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.Machine.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.OperatorSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	return nil
}

func validateCommandRunTimes(r CommandRun) error {
	if r.StartedAt.IsZero() || r.FinishedAt.IsZero() {
		return fmt.Errorf(ErrFmtCommandRun, core.ErrReleaseContract)
	}
	if err := r.StartedAt.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if err := r.FinishedAt.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtCommandRun, err)
	}
	if r.FinishedAt.Before(r.StartedAt) {
		return fmt.Errorf(ErrFmtCommandRun, core.ErrReleaseContract)
	}
	return nil
}
