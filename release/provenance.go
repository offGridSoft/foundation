package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const MaxToolProvenanceItems uint32 = 64

type BuildToolchain struct {
	GarbleVersion ToolVersion `json:"garble_version"`
	GoVersion     ToolVersion `json:"go_version"`
}

func (t BuildToolchain) Validate() error {
	if err := t.GarbleVersion.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtToolchain, err)
	}
	if err := t.GoVersion.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtToolchain, err)
	}
	return nil
}

func (t BuildToolchain) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldGarbleVersion, t.GarbleVersion)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldGoVersion, t.GoVersion)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type VulnDBSnapshot struct {
	DBVersion  ToolVersion       `json:"db_version"`
	SnapshotAt core.UnixNanoTime `json:"snapshot_at_ns"`
}

func (s VulnDBSnapshot) Validate() error {
	if err := s.DBVersion.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtVulnDB, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(s.SnapshotAt); err != nil {
		return wrapReleaseContract(ErrFmtVulnDB, err)
	}
	return nil
}

func (s VulnDBSnapshot) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldDBVersion, s.DBVersion)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSnapshotAt, s.SnapshotAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type ReleaseGateEvidence struct {
	FastGateRef      EvidenceRef              `json:"fast_gate_ref"`
	FinalCertificate FinalCertificateEvidence `json:"final_certificate"`
}

func (e ReleaseGateEvidence) Validate() error {
	if err := e.FastGateRef.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtGateEvidence, err)
	}
	if err := e.FinalCertificate.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtGateEvidence, err)
	}
	return nil
}

func (e ReleaseGateEvidence) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldFastGateRef, e.FastGateRef)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFinalCertificate, e.FinalCertificate)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (e ReleaseGateEvidence) ValidateForCommit(commit core.BuildCommit) error {
	if err := e.Validate(); err != nil {
		return err
	}
	return e.FinalCertificate.ValidateForCommit(commit)
}

func (e ReleaseGateEvidence) RequireCertified(commit core.BuildCommit) error {
	if err := e.Validate(); err != nil {
		return err
	}
	return e.FinalCertificate.RequireCertified(commit)
}

type ToolProvenance struct {
	GoSum   GoSumHash   `json:"go_sum"`
	Module  ToolModule  `json:"module"`
	Version ToolVersion `json:"version"`
}

func (p ToolProvenance) Validate() error {
	if err := p.GoSum.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtToolProvenance, err)
	}
	if err := p.Module.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtToolProvenance, err)
	}
	if err := p.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtToolProvenance, err)
	}
	return nil
}

func (p ToolProvenance) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldGoSum, p.GoSum)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldModule, p.Module)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, p.Version)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type ToolProvenanceSet struct {
	Tools     []ToolProvenance `json:"tools"`
	ToolCount uint32           `json:"tool_count"`
}

func (s ToolProvenanceSet) Validate() error {
	if err := (core.CollectionCardinality{
		Length:          len(s.Tools),
		DeclaredCount:   s.ToolCount,
		Minimum:         1,
		Maximum:         MaxToolProvenanceItems,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtToolProvenance, core.ErrReleaseContract)
	}
	return validateToolProvenanceOrder(s.Tools)
}

func validateToolProvenanceOrder(tools []ToolProvenance) error {
	var previous string
	for idx, tool := range tools {
		if err := tool.Validate(); err != nil {
			return err
		}
		current := tool.Module.String()
		if idx > 0 && current <= previous {
			return fmt.Errorf(ErrFmtToolProvenance, core.ErrReleaseContract)
		}
		previous = current
	}
	return nil
}
