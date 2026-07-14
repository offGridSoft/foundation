package release

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.Validatable = DeployRequestID{}
	_ core.Validatable = DeployPrepareRequest{}
	_ core.Validatable = DeployPrepareResponse{}
	_ core.Validatable = DeployFinalizeRequest{}
	_ core.Validatable = DeployFinalizeResponse{}
)

const DeployTransportMaximumBytes = 512 << 10

type DeployRequestID struct {
	value string
}

func NewDeployRequestID(entropy [core.RandomIdentityEntropyBytes]byte) (DeployRequestID, error) {
	if entropy == [core.RandomIdentityEntropyBytes]byte{} {
		return DeployRequestID{}, fmt.Errorf(ErrFmtDeployRequestID, core.ErrReleaseContract)
	}
	return ParseDeployRequestID(hex.EncodeToString(entropy[:]))
}

func ParseDeployRequestID(value string) (DeployRequestID, error) {
	parsed, err := parseReleaseRandomHex(value)
	if err != nil {
		return DeployRequestID{}, fmt.Errorf(ErrFmtDeployRequestID, err)
	}
	return DeployRequestID{value: parsed}, nil
}

func (id DeployRequestID) String() string { return id.value }

func (id DeployRequestID) IsZero() bool { return id.value == "" }

func (id DeployRequestID) Validate() error {
	_, err := ParseDeployRequestID(id.value)
	return err
}

func (id DeployRequestID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *DeployRequestID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeployRequestID, core.ErrReleaseContract)
	}
	parsed, err := ParseDeployRequestID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type DeployPrepareRequest struct {
	Manifest  core.Signed[Manifest] `json:"manifest"`
	RequestID DeployRequestID       `json:"request_id"`
	Schema    core.SchemaID         `json:"schema"`
}

func (r DeployPrepareRequest) Validate() error {
	_, err := r.validatedJSON()
	return err
}

func (r DeployPrepareRequest) validateStructure() error {
	if r.Schema != core.SchemaReleaseDeployPrepareRequest {
		return fmt.Errorf(ErrFmtDeployPrepareRequest, core.ErrReleaseContract)
	}
	if err := r.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareRequest, err)
	}
	if err := r.Manifest.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareRequest, err)
	}
	return nil
}

func (r DeployPrepareRequest) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	encoded, err := appendDeployPrepareRequestJSON(nil, r)
	if err := validateDeployTransportSize(encoded, err); err != nil {
		return nil, wrapReleaseContract(ErrFmtDeployPrepareRequest, err)
	}
	return encoded, nil
}

func (r DeployPrepareRequest) Verify(releaseKeys core.SigningKeyring) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := r.Manifest.Verify(releaseKeys); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareRequest, err)
	}
	return nil
}

func (r DeployPrepareRequest) MarshalJSON() ([]byte, error) {
	return r.validatedJSON()
}

type DeployPrepareResponse struct {
	Plan      core.Signed[DeployPlan] `json:"plan"`
	RequestID DeployRequestID         `json:"request_id"`
	Schema    core.SchemaID           `json:"schema"`
}

func (r DeployPrepareResponse) Validate() error {
	_, err := r.validatedJSON()
	return err
}

func (r DeployPrepareResponse) validateStructure() error {
	if r.Schema != core.SchemaReleaseDeployPrepareResponse {
		return fmt.Errorf(ErrFmtDeployPrepareResponse, core.ErrReleaseContract)
	}
	if err := r.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareResponse, err)
	}
	if err := r.Plan.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareResponse, err)
	}
	if r.RequestID != r.Plan.Body.RequestID {
		return fmt.Errorf(ErrFmtDeployPrepareResponse, core.ErrReleaseContract)
	}
	return nil
}

func (r DeployPrepareResponse) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	encoded, err := appendDeployPrepareResponseJSON(nil, r)
	if err := validateDeployTransportSize(encoded, err); err != nil {
		return nil, wrapReleaseContract(ErrFmtDeployPrepareResponse, err)
	}
	return encoded, nil
}

func (r DeployPrepareResponse) Verify(
	request DeployPrepareRequest,
	releaseKeys core.SigningKeyring,
	serverKeys core.SigningKeyring,
) error {
	if err := request.Verify(releaseKeys); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := r.Plan.Verify(serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtDeployPrepareResponse, err)
	}
	if request.RequestID != r.RequestID || !sameManifest(request.Manifest.Body, r.Plan.Body.Manifest) {
		return fmt.Errorf(ErrFmtDeployPrepareResponse, core.ErrReleaseContract)
	}
	return nil
}

func (r DeployPrepareResponse) MarshalJSON() ([]byte, error) {
	return r.validatedJSON()
}

type DeployFinalizeRequest struct {
	Plan        core.Signed[DeployPlan] `json:"plan"`
	RequestID   DeployRequestID         `json:"request_id"`
	Objects     []UploadedArtifact      `json:"objects"`
	ObjectCount uint32                  `json:"object_count"`
	Schema      core.SchemaID           `json:"schema"`
}

func (r DeployFinalizeRequest) Validate() error {
	_, err := r.validatedJSON()
	return err
}

func (r DeployFinalizeRequest) validateStructure() error {
	if r.Schema != core.SchemaReleaseDeployFinalizeRequest {
		return fmt.Errorf(ErrFmtDeployFinalizeRequest, core.ErrReleaseContract)
	}
	if err := r.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	if err := r.Plan.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	if r.RequestID != r.Plan.Body.RequestID {
		return fmt.Errorf(ErrFmtDeployFinalizeRequest, core.ErrReleaseContract)
	}
	if err := validateDeployCompletions(r); err != nil {
		return err
	}
	return nil
}

func (r DeployFinalizeRequest) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	encoded, err := appendDeployFinalizeRequestJSON(nil, r)
	if err := validateDeployTransportSize(encoded, err); err != nil {
		return nil, wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	return encoded, nil
}

func (r DeployFinalizeRequest) Verify(serverKeys core.SigningKeyring) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := r.Plan.Verify(serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	return nil
}

func (r DeployFinalizeRequest) MarshalJSON() ([]byte, error) {
	return r.validatedJSON()
}

type DeployFinalizeResponse struct {
	Receipt   core.Signed[UploadReceipt] `json:"receipt"`
	Index     core.Signed[DownloadIndex] `json:"index"`
	RequestID DeployRequestID            `json:"request_id"`
	Schema    core.SchemaID              `json:"schema"`
}

func (r DeployFinalizeResponse) Validate() error {
	_, err := r.validatedJSON()
	return err
}

func (r DeployFinalizeResponse) validateStructure() error {
	if r.Schema != core.SchemaReleaseDeployFinalizeResponse {
		return fmt.Errorf(ErrFmtDeployFinalizeResponse, core.ErrReleaseContract)
	}
	if err := r.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	if err := r.Receipt.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	if err := r.Index.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	return nil
}

func (r DeployFinalizeResponse) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	encoded, err := appendDeployFinalizeResponseJSON(nil, r)
	if err := validateDeployTransportSize(encoded, err); err != nil {
		return nil, wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	return encoded, nil
}

func (r DeployFinalizeResponse) Verify(request DeployFinalizeRequest, serverKeys core.SigningKeyring) error {
	if err := request.Verify(serverKeys); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := r.Receipt.Verify(serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	if err := r.Index.Verify(serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	plan := request.Plan.Body
	if r.RequestID != request.RequestID || r.Receipt.Body.AttemptID != plan.AttemptID {
		return fmt.Errorf(ErrFmtDeployFinalizeResponse, core.ErrReleaseContract)
	}
	if err := r.Receipt.Body.VerifyManifest(plan.Manifest); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	if err := r.Index.Body.VerifyManifest(plan.Manifest); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeResponse, err)
	}
	if !slices.Equal(r.Receipt.Body.Objects, request.Objects) {
		return fmt.Errorf(ErrFmtDeployFinalizeResponse, core.ErrReleaseContract)
	}
	return nil
}

func (r DeployFinalizeResponse) MarshalJSON() ([]byte, error) {
	return r.validatedJSON()
}

func BuildDeployPrepareRequest(requestID DeployRequestID, manifest core.Signed[Manifest]) (DeployPrepareRequest, error) {
	request := DeployPrepareRequest{
		Schema: core.SchemaReleaseDeployPrepareRequest, RequestID: requestID, Manifest: cloneSignedManifest(manifest),
	}
	if err := request.Validate(); err != nil {
		return DeployPrepareRequest{}, err
	}
	return request, nil
}

func BuildDeployPrepareResponse(requestID DeployRequestID, plan core.Signed[DeployPlan]) (DeployPrepareResponse, error) {
	response := DeployPrepareResponse{
		Schema: core.SchemaReleaseDeployPrepareResponse, RequestID: requestID, Plan: cloneSignedDeployPlan(plan),
	}
	if err := response.Validate(); err != nil {
		return DeployPrepareResponse{}, err
	}
	return response, nil
}

func BuildDeployFinalizeRequest(
	requestID DeployRequestID,
	plan core.Signed[DeployPlan],
	objects []UploadedArtifact,
) (DeployFinalizeRequest, error) {
	set, err := core.BuildArtifactSet(objects)
	if err != nil {
		return DeployFinalizeRequest{}, wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	request := DeployFinalizeRequest{
		Schema: core.SchemaReleaseDeployFinalizeRequest, RequestID: requestID,
		Plan: cloneSignedDeployPlan(plan), Objects: set.Items, ObjectCount: set.Count,
	}
	if err := request.Validate(); err != nil {
		return DeployFinalizeRequest{}, err
	}
	return request, nil
}

func BuildDeployFinalizeResponse(
	requestID DeployRequestID,
	receipt core.Signed[UploadReceipt],
	index core.Signed[DownloadIndex],
) (DeployFinalizeResponse, error) {
	response := DeployFinalizeResponse{
		Schema: core.SchemaReleaseDeployFinalizeResponse, RequestID: requestID,
		Receipt: cloneSignedUploadReceipt(receipt), Index: cloneSignedDownloadIndex(index),
	}
	if err := response.Validate(); err != nil {
		return DeployFinalizeResponse{}, err
	}
	return response, nil
}

func ParseDeployPrepareRequest(data []byte) (DeployPrepareRequest, error) {
	return core.DecodeStrictJSON[DeployPrepareRequest](data)
}

func ParseDeployPrepareResponse(data []byte) (DeployPrepareResponse, error) {
	return core.DecodeStrictJSON[DeployPrepareResponse](data)
}

func ParseDeployFinalizeRequest(data []byte) (DeployFinalizeRequest, error) {
	return core.DecodeStrictJSON[DeployFinalizeRequest](data)
}

func ParseDeployFinalizeResponse(data []byte) (DeployFinalizeResponse, error) {
	return core.DecodeStrictJSON[DeployFinalizeResponse](data)
}

func validateDeployCompletions(request DeployFinalizeRequest) error {
	if err := (core.CollectionCardinality{
		Length: len(request.Objects), DeclaredCount: request.ObjectCount,
		Minimum: 1, Maximum: core.CollectionMaximumDefault, RequireDeclared: true,
	}).Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
	}
	if len(request.Objects) != len(request.Plan.Body.Targets) {
		return fmt.Errorf(ErrFmtDeployFinalizeRequest, core.ErrReleaseContract)
	}
	for index, object := range request.Objects {
		if err := object.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtDeployFinalizeRequest, err)
		}
		if index > 0 && request.Objects[index-1].ArtifactSetName() >= object.ArtifactSetName() {
			return fmt.Errorf(ErrFmtDeployFinalizeRequest, core.ErrReleaseContract)
		}
		if !uploadedArtifactMatchesTarget(object, request.Plan.Body.Targets[index], request.Plan.Body.Manifest) {
			return fmt.Errorf(ErrFmtDeployFinalizeRequest, core.ErrReleaseContract)
		}
	}
	return nil
}

func uploadedArtifactMatchesTarget(object UploadedArtifact, target UploadTarget, manifest Manifest) bool {
	if object.Artifact != target.Artifact || object.Provider != target.Provider || object.Bucket != target.Bucket ||
		object.Object != target.Object || object.AttemptID != target.AttemptID || object.Binding != target.Binding {
		return false
	}
	artifact, found := manifestArtifactByName(manifest, object.Artifact)
	return found && object.SHA256 == artifact.SHA256 && object.Size == artifact.Size
}

func sameManifest(left, right Manifest) bool {
	leftCanonical, leftErr := left.Canonical(nil)
	rightCanonical, rightErr := right.Canonical(nil)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftCanonical, rightCanonical)
}

func cloneSignedManifest(value core.Signed[Manifest]) core.Signed[Manifest] {
	value.Body.Artifacts = append([]Artifact(nil), value.Body.Artifacts...)
	return value
}

func cloneSignedDeployPlan(value core.Signed[DeployPlan]) core.Signed[DeployPlan] {
	value.Body.Targets = cloneAndSortUploadTargets(value.Body.Targets)
	value.Body.Manifest.Artifacts = append([]Artifact(nil), value.Body.Manifest.Artifacts...)
	return value
}

func cloneSignedUploadReceipt(value core.Signed[UploadReceipt]) core.Signed[UploadReceipt] {
	value.Body.Objects = append([]UploadedArtifact(nil), value.Body.Objects...)
	return value
}

func cloneSignedDownloadIndex(value core.Signed[DownloadIndex]) core.Signed[DownloadIndex] {
	value.Body.Downloads = append([]Download(nil), value.Body.Downloads...)
	return value
}

func appendDeployPrepareRequestJSON(dst []byte, value DeployPrepareRequest) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, value.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, value.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifest, value.Manifest)
	return closeDeployTransportJSON(dst, err)
}

func appendDeployPrepareResponseJSON(dst []byte, value DeployPrepareResponse) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, value.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, value.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlan, value.Plan)
	return closeDeployTransportJSON(dst, err)
}

func appendDeployFinalizeRequestJSON(dst []byte, value DeployFinalizeRequest) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, value.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, value.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlan, value.Plan)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldObjects, value.Objects)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldObjectCount, value.ObjectCount)
	return closeDeployTransportJSON(dst, err)
}

func appendDeployFinalizeResponseJSON(dst []byte, value DeployFinalizeResponse) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, value.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, value.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReceipt, value.Receipt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldIndex, value.Index)
	return closeDeployTransportJSON(dst, err)
}

func closeDeployTransportJSON(dst []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateDeployTransportSize(encoded []byte, encodeErr error) error {
	if encodeErr != nil {
		return encodeErr
	}
	if len(encoded) == 0 || len(encoded) > DeployTransportMaximumBytes {
		return core.ErrReleaseContract
	}
	return nil
}
