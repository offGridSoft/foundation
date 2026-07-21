package peachfuzz

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RunEvidenceArchiveSegment           = "effort"
	RunEvidenceContentType              = "application/vnd.offgridsoftware.peachfuzz-run-evidence+json"
	RunEvidenceDigestShardHexCharacters = 2
	RunEvidenceObjectDigestSeparator    = "@"
	RunEvidenceObjectExtension          = ".json"
	RunEvidenceUploadGrantMaxHeaders    = 8
)

type RunEvidenceObjectName struct {
	value string
}

func NewRunEvidenceObjectName(evidence RunEvidence, digest core.SHA256Hex) (RunEvidenceObjectName, error) {
	if err := evidence.Validate(); err != nil {
		return RunEvidenceObjectName{}, runEvidenceUploadError(err)
	}
	if err := digest.Validate(); err != nil {
		return RunEvidenceObjectName{}, runEvidenceUploadError(err)
	}
	value := path.Join(
		core.FoundationVersion2026,
		RunEvidenceArchiveSegment,
		digest.String()[:RunEvidenceDigestShardHexCharacters],
		evidence.Project.String(),
		evidence.Machine.String(),
		evidence.RunID.String()+RunEvidenceObjectDigestSeparator+digest.String()+RunEvidenceObjectExtension,
	)
	return ParseRunEvidenceObjectName(value)
}

func ParseRunEvidenceObjectName(value string) (RunEvidenceObjectName, error) {
	segments := strings.Split(value, "/")
	if len(segments) != 6 || segments[0] != core.FoundationVersion2026 || segments[1] != RunEvidenceArchiveSegment {
		return RunEvidenceObjectName{}, runEvidenceUploadError(ErrContract)
	}
	digest, runID, err := parseRunEvidenceObjectLeaf(segments[5])
	if err != nil || segments[2] != digest.String()[:RunEvidenceDigestShardHexCharacters] {
		return RunEvidenceObjectName{}, runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	_, projectErr := ParseProjectID(segments[3])
	_, machineErr := ParseMachineID(segments[4])
	if projectErr != nil || machineErr != nil || runID.Validate() != nil {
		return RunEvidenceObjectName{}, runEvidenceUploadError(errors.Join(projectErr, machineErr))
	}
	return RunEvidenceObjectName{value: value}, nil
}

func parseRunEvidenceObjectLeaf(leaf string) (core.SHA256Hex, RunID, error) {
	if !strings.HasSuffix(leaf, RunEvidenceObjectExtension) {
		return core.SHA256Hex{}, RunID{}, ErrContract
	}
	stem := strings.TrimSuffix(leaf, RunEvidenceObjectExtension)
	parts := strings.Split(stem, RunEvidenceObjectDigestSeparator)
	if len(parts) != 2 {
		return core.SHA256Hex{}, RunID{}, ErrContract
	}
	runID, runErr := ParseRunID(parts[0])
	digest, digestErr := core.ParseSHA256Hex(parts[1])
	return digest, runID, errors.Join(runErr, digestErr)
}

func (n RunEvidenceObjectName) String() string { return n.value }

func (n RunEvidenceObjectName) Validate() error {
	_, err := ParseRunEvidenceObjectName(n.value)
	return err
}

func (n RunEvidenceObjectName) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.value)
}

func (n *RunEvidenceObjectName) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return runEvidenceUploadError(err)
	}
	parsed, err := ParseRunEvidenceObjectName(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

type RunEvidenceDescriptor struct {
	Object RunEvidenceObjectName `json:"object"`
	Digest core.SHA256Hex        `json:"sha256"`
	Size   core.ByteCount        `json:"size_bytes"`
}

func DescriptorForSignedRunEvidence(signed SignedRunEvidence) (RunEvidenceDescriptor, error) {
	if err := signed.Verify(); err != nil {
		return RunEvidenceDescriptor{}, runEvidenceUploadError(err)
	}
	encoded, err := core.EncodeValidatedJSON(signed)
	if err != nil {
		return RunEvidenceDescriptor{}, runEvidenceUploadError(err)
	}
	digest := core.NewSHA256Hex(sha256.Sum256(encoded))
	object, err := NewRunEvidenceObjectName(signed.Body, digest)
	if err != nil {
		return RunEvidenceDescriptor{}, err
	}
	descriptor := RunEvidenceDescriptor{Object: object, Digest: digest, Size: core.NewByteCount(uint64(len(encoded)))}
	return descriptor, descriptor.Validate()
}

func (d RunEvidenceDescriptor) Validate() error {
	if err := d.Object.Validate(); err != nil {
		return runEvidenceUploadError(err)
	}
	if err := d.Digest.Validate(); err != nil {
		return runEvidenceUploadError(err)
	}
	if err := d.Size.Validate(); err != nil || d.Size.Uint64() > core.PeachfuzzRunEvidenceMaxBytes {
		return runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	return nil
}

type RunEvidenceUploadRequest struct {
	Evidence SignedRunEvidence `json:"evidence"`
	Schema   core.SchemaID     `json:"schema"`
}

func (RunEvidenceUploadRequest) APIBody() {}

func (r RunEvidenceUploadRequest) Validate() error {
	if r.Schema != core.SchemaPeachfuzzRunEvidenceUploadRequest {
		return runEvidenceUploadError(ErrContract)
	}
	if err := r.Evidence.Verify(); err != nil {
		return runEvidenceUploadError(err)
	}
	_, err := DescriptorForSignedRunEvidence(r.Evidence)
	return err
}

func (r RunEvidenceUploadRequest) Descriptor() (RunEvidenceDescriptor, error) {
	if err := r.Validate(); err != nil {
		return RunEvidenceDescriptor{}, err
	}
	return DescriptorForSignedRunEvidence(r.Evidence)
}

type RunEvidenceUploadGrant struct {
	Descriptor RunEvidenceDescriptor `json:"descriptor"`
	URL        core.SignedUploadURL  `json:"url"`
	Headers    []core.UploadHeader   `json:"headers"`
	ExpiresAt  core.UnixNanoTime     `json:"expires_at"`
	Provider   core.StorageProvider  `json:"provider"`
	Method     core.UploadMethod     `json:"method"`
	Schema     core.SchemaID         `json:"schema"`
}

func (RunEvidenceUploadGrant) APIBody() {}

func (g RunEvidenceUploadGrant) Validate() error {
	if g.Schema != core.SchemaPeachfuzzRunEvidenceUploadGrant || g.Provider != core.StorageProviderGCS || g.Method != core.UploadMethodSignedPUT {
		return runEvidenceUploadError(ErrContract)
	}
	if err := g.Descriptor.Validate(); err != nil {
		return err
	}
	if err := g.URL.Validate(); err != nil {
		return runEvidenceUploadError(err)
	}
	if err := core.ValidateUploadHeaders(g.Headers); err != nil || len(g.Headers) > RunEvidenceUploadGrantMaxHeaders {
		return runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	if err := core.ValidateRequiredUnixNanoTime(g.ExpiresAt); err != nil {
		return runEvidenceUploadError(err)
	}
	return nil
}

func (g RunEvidenceUploadGrant) ValidateRequest(request RunEvidenceUploadRequest) error {
	if err := g.Validate(); err != nil {
		return err
	}
	descriptor, err := request.Descriptor()
	if err != nil || descriptor != g.Descriptor {
		return runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	return nil
}

func runEvidenceUploadError(err error) error {
	return fmt.Errorf(ErrFmtRunEvidenceUpload, errors.Join(ErrContract, err))
}

var _ core.APIBody = RunEvidenceUploadRequest{}
var _ core.APIBody = RunEvidenceUploadGrant{}
