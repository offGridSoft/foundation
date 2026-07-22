package peachfuzz

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RunEvidenceArchiveSegment                 = "effort"
	RunEvidenceContentType                    = "application/vnd.offgridsoftware.peachfuzz-run-evidence+json"
	RunEvidenceDigestShardHexCharacters       = 2
	RunEvidenceObjectDigestSeparator          = "@"
	RunEvidenceObjectExtension                = ".json"
	RunEvidenceObjectPathSeparator            = "/"
	RunEvidenceUploadGrantMaxHeaders          = 8
	RunEvidenceUploadDispositionTokenRequired = "upload-required"
	RunEvidenceUploadDispositionTokenPresent  = "already-present"
)

// RunEvidenceDigestShard is the closed 256-way partition key derived from
// the first byte of a signed evidence digest. Every value of the underlying
// uint8 domain is valid, so callers cannot construct an out-of-range shard.
type RunEvidenceDigestShard uint8

func (s RunEvidenceDigestShard) IsValid() bool { return true }

func (s RunEvidenceDigestShard) Validate() error { return nil }

func (s RunEvidenceDigestShard) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *RunEvidenceDigestShard) UnmarshalJSON(data []byte) error {
	if s == nil {
		return runEvidenceUploadError(ErrContract)
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return runEvidenceUploadError(err)
	}
	parsed, err := ParseRunEvidenceDigestShard(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s RunEvidenceDigestShard) String() string {
	const radix = 16
	encoded := strconv.FormatUint(uint64(s), radix)
	if len(encoded) == RunEvidenceDigestShardHexCharacters {
		return encoded
	}
	return "0" + encoded
}

func ParseRunEvidenceDigestShard(value string) (RunEvidenceDigestShard, error) {
	const bitSize = 8
	parsed, err := strconv.ParseUint(value, 16, bitSize)
	shard := RunEvidenceDigestShard(parsed)
	if err != nil || len(value) != RunEvidenceDigestShardHexCharacters || shard.String() != value {
		return 0, runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	return shard, nil
}

func RunEvidenceDigestShardPrefix(shard RunEvidenceDigestShard) string {
	return path.Join(core.FoundationVersion2026, RunEvidenceArchiveSegment, shard.String()) + RunEvidenceObjectPathSeparator
}

type RunEvidenceUploadDisposition uint8

const (
	RunEvidenceUploadDispositionUnknown RunEvidenceUploadDisposition = iota
	RunEvidenceUploadDispositionRequired
	RunEvidenceUploadDispositionPresent
)

func (d RunEvidenceUploadDisposition) String() string {
	switch d {
	case RunEvidenceUploadDispositionRequired:
		return RunEvidenceUploadDispositionTokenRequired
	case RunEvidenceUploadDispositionPresent:
		return RunEvidenceUploadDispositionTokenPresent
	default:
		return ""
	}
}

func (d RunEvidenceUploadDisposition) Validate() error {
	if !d.IsValid() {
		return runEvidenceUploadError(ErrContract)
	}
	return nil
}

func (d RunEvidenceUploadDisposition) IsValid() bool {
	return d > RunEvidenceUploadDispositionUnknown && d <= RunEvidenceUploadDispositionPresent
}

func (d RunEvidenceUploadDisposition) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}

func (d *RunEvidenceUploadDisposition) UnmarshalJSON(data []byte) error {
	if d == nil {
		return runEvidenceUploadError(ErrContract)
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return runEvidenceUploadError(err)
	}
	for candidate := RunEvidenceUploadDispositionRequired; candidate <= RunEvidenceUploadDispositionPresent; candidate++ {
		if candidate.String() == value {
			*d = candidate
			return nil
		}
	}
	return runEvidenceUploadError(ErrContract)
}

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
	shard, shardErr := ParseRunEvidenceDigestShard(segments[2])
	if err != nil || shardErr != nil || shard.String() != digest.String()[:RunEvidenceDigestShardHexCharacters] {
		return RunEvidenceObjectName{}, runEvidenceUploadError(errors.Join(ErrContract, err, shardErr))
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

// RunEvidenceUploadResponse makes immutable retries explicit. A confirmed
// existing object carries no upload capability; an absent object carries one
// exact short-lived grant. The two states cannot be confused by callers.
type RunEvidenceUploadResponse struct {
	Grant       *RunEvidenceUploadGrant      `json:"grant,omitempty"`
	Descriptor  RunEvidenceDescriptor        `json:"descriptor"`
	Disposition RunEvidenceUploadDisposition `json:"disposition"`
	Schema      core.SchemaID                `json:"schema"`
}

func (RunEvidenceUploadResponse) APIBody() {}

func (r RunEvidenceUploadResponse) Validate() error {
	if r.Schema != core.SchemaPeachfuzzRunEvidenceUploadResponse {
		return runEvidenceUploadError(ErrContract)
	}
	if err := r.Descriptor.Validate(); err != nil {
		return err
	}
	if err := r.Disposition.Validate(); err != nil {
		return err
	}
	switch r.Disposition {
	case RunEvidenceUploadDispositionRequired:
		if r.Grant == nil || r.Grant.Descriptor != r.Descriptor {
			return runEvidenceUploadError(ErrContract)
		}
		return r.Grant.Validate()
	case RunEvidenceUploadDispositionPresent:
		if r.Grant != nil {
			return runEvidenceUploadError(ErrContract)
		}
		return nil
	default:
		return runEvidenceUploadError(ErrContract)
	}
}

func (r RunEvidenceUploadResponse) ValidateRequest(request RunEvidenceUploadRequest) error {
	if err := r.Validate(); err != nil {
		return err
	}
	descriptor, err := request.Descriptor()
	if err != nil || descriptor != r.Descriptor {
		return runEvidenceUploadError(errors.Join(ErrContract, err))
	}
	if r.Grant != nil {
		return r.Grant.ValidateRequest(request)
	}
	return nil
}

func runEvidenceUploadError(err error) error {
	return fmt.Errorf(ErrFmtRunEvidenceUpload, errors.Join(ErrContract, err))
}

var _ core.APIBody = RunEvidenceUploadRequest{}
var _ core.APIBody = RunEvidenceUploadGrant{}
var _ core.APIBody = RunEvidenceUploadResponse{}
