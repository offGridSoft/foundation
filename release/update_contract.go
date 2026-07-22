package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/offGridSoft/foundation/v2026/core"
)

const UpdateTransportMaximumBytes = 768 << 10

type UpdateRequestID struct{ value string }

func NewUpdateRequestID(entropy [core.RandomIdentityEntropyBytes]byte) (UpdateRequestID, error) {
	if entropy == [core.RandomIdentityEntropyBytes]byte{} {
		return UpdateRequestID{}, fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	return ParseUpdateRequestID(hex.EncodeToString(entropy[:]))
}

func ParseUpdateRequestID(value string) (UpdateRequestID, error) {
	parsed, err := parseReleaseRandomHex(value)
	if err != nil {
		return UpdateRequestID{}, wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	return UpdateRequestID{value: parsed}, nil
}

func (id UpdateRequestID) String() string { return id.value }
func (id UpdateRequestID) Validate() error {
	_, err := ParseUpdateRequestID(id.value)
	return err
}
func (id UpdateRequestID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}
func (id *UpdateRequestID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	parsed, err := ParseUpdateRequestID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type UpdateCheckRequest struct {
	RequestedReleaseID *ReleaseID          `json:"requested_release_id,omitempty"`
	RequestID          UpdateRequestID     `json:"request_id"`
	InstalledVersion   core.ProductVersion `json:"installed_version"`
	InstalledReleaseID ReleaseID           `json:"installed_release_id"`
	InstalledCommit    core.BuildCommit    `json:"installed_commit"`
	InstalledSHA256    core.SHA256Hex      `json:"installed_sha256"`
	Schema             core.SchemaID       `json:"schema"`
	Product            core.Product        `json:"product"`
	Platform           core.Platform       `json:"platform"`
}

type UpdateCheckInput struct {
	RequestedReleaseID *ReleaseID
	RequestID          UpdateRequestID
	InstalledVersion   core.ProductVersion
	InstalledReleaseID ReleaseID
	InstalledCommit    core.BuildCommit
	InstalledSHA256    core.SHA256Hex
	Product            core.Product
	Platform           core.Platform
}

func BuildUpdateCheckRequest(input UpdateCheckInput) (UpdateCheckRequest, error) {
	request := UpdateCheckRequest{
		Schema: core.SchemaReleaseUpdateCheckRequest, RequestID: input.RequestID, Product: input.Product,
		InstalledVersion: input.InstalledVersion, InstalledReleaseID: input.InstalledReleaseID,
		InstalledCommit: input.InstalledCommit, InstalledSHA256: input.InstalledSHA256, Platform: input.Platform,
	}
	if input.RequestedReleaseID != nil {
		requested := *input.RequestedReleaseID
		request.RequestedReleaseID = &requested
	}
	if err := request.Validate(); err != nil {
		return UpdateCheckRequest{}, err
	}
	return request, nil
}

func (r UpdateCheckRequest) Validate() error {
	encoded, err := r.validatedJSON()
	if err != nil {
		return err
	}
	return validateUpdateBytes(encoded, ErrFmtUpdateCheck)
}

func (r UpdateCheckRequest) HTTPIdempotencyKey() (core.HTTPIdempotencyKey, error) {
	return core.ParseHTTPIdempotencyKey(r.RequestID.String())
}

func (r UpdateCheckRequest) validateStructure() error {
	if r.Schema != core.SchemaReleaseUpdateCheckRequest {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	for _, err := range []error{r.RequestID.Validate(), r.Product.Validate(), r.InstalledVersion.Validate(),
		r.InstalledReleaseID.Validate(), r.InstalledCommit.Validate(), r.InstalledSHA256.Validate(), r.Platform.Validate()} {
		if err != nil {
			return wrapReleaseContract(ErrFmtUpdateCheck, err)
		}
	}
	if !r.Platform.IsReleaseTarget() {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	if err := ValidateReleaseIDIdentity(r.InstalledReleaseID, r.Product, r.InstalledVersion, r.InstalledCommit); err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	if r.RequestedReleaseID != nil {
		if err := r.RequestedReleaseID.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtUpdateCheck, err)
		}
		if *r.RequestedReleaseID == r.InstalledReleaseID {
			return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
		}
	}
	return nil
}

func (UpdateCheckRequest) APIBody()                       {}
func (r UpdateCheckRequest) MarshalJSON() ([]byte, error) { return r.validatedJSON() }

func (r UpdateCheckRequest) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	dst, err := core.AppendJSONField(dst, core.JSONFieldSchema, r.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, r.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, r.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledVersion, r.InstalledVersion)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledReleaseID, r.InstalledReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledCommit, r.InstalledCommit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledSHA256, r.InstalledSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatform, r.Platform)
	if r.RequestedReleaseID != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestedReleaseID, r.RequestedReleaseID)
	}
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type UpdateCheckResponseBody struct {
	Publication *DeployFinalizeResponse `json:"publication,omitempty"`
	RequestID   UpdateRequestID         `json:"request_id"`
	Schema      core.SchemaID           `json:"schema"`
	Product     core.Product            `json:"product"`
	Decision    UpdateDecision          `json:"decision"`
}

func (b UpdateCheckResponseBody) Validate() error {
	if err := b.validateStructure(); err != nil {
		return err
	}
	encoded, err := b.appendJSON(nil)
	if err != nil {
		return err
	}
	return validateUpdateBytes(encoded, ErrFmtUpdateCheck)
}

func (b UpdateCheckResponseBody) validateStructure() error {
	if b.Schema != core.SchemaReleaseUpdateCheckResponse {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	if err := b.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	if err := b.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	if err := b.Decision.Validate(); err != nil {
		return err
	}
	if b.Decision != UpdateDecisionAvailable {
		if b.Publication != nil {
			return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
		}
		return nil
	}
	if b.Publication == nil {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	if err := b.Publication.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	if b.Publication.Manifest.Body.Product != b.Product {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	return nil
}

func (b UpdateCheckResponseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return b.appendJSON(dst)
}

func (b UpdateCheckResponseBody) appendJSON(dst []byte) ([]byte, error) {
	dst = append(dst, '{')
	dst, err := core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestID, b.RequestID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, b.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDecision, b.Decision)
	if b.Publication != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPublication, b.Publication)
	}
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b UpdateCheckResponseBody) SigningSchema() core.SchemaID { return b.Schema }
func (b UpdateCheckResponseBody) MarshalJSON() ([]byte, error) { return b.Canonical(nil) }

type UpdateCheckResponse struct {
	Authority core.Signed[UpdateCheckResponseBody] `json:"authority"`
}

func BuildUpdateCheckResponseBody(request UpdateCheckRequest, decision UpdateDecision, publication *DeployFinalizeResponse) (UpdateCheckResponseBody, error) {
	body := UpdateCheckResponseBody{Schema: core.SchemaReleaseUpdateCheckResponse, RequestID: request.RequestID, Product: request.Product, Decision: decision}
	if publication != nil {
		cloned := cloneDeployFinalizeResponse(*publication)
		body.Publication = &cloned
	}
	if err := request.Validate(); err != nil {
		return UpdateCheckResponseBody{}, err
	}
	if err := body.Validate(); err != nil {
		return UpdateCheckResponseBody{}, err
	}
	if publication != nil {
		if request.RequestedReleaseID != nil && publication.Manifest.Body.ReleaseID != *request.RequestedReleaseID {
			return UpdateCheckResponseBody{}, fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
		}
		if publication.Manifest.Body.ReleaseID == request.InstalledReleaseID {
			return UpdateCheckResponseBody{}, fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
		}
		if _, err := DownloadForPlatform(publication.Index.Body, request.Platform); err != nil {
			return UpdateCheckResponseBody{}, wrapReleaseContract(ErrFmtUpdateCheck, err)
		}
	}
	return body, nil
}

func (UpdateCheckResponse) APIBody()          {}
func (r UpdateCheckResponse) Validate() error { return r.Authority.Validate() }
func (r UpdateCheckResponse) Verify(request UpdateCheckRequest, releaseKeys, serverKeys core.SigningKeyring) error {
	body, err := r.verifyAuthority(request, serverKeys)
	if err != nil {
		return err
	}
	if body.Decision != UpdateDecisionAvailable {
		return nil
	}
	return verifyAvailableUpdate(request, *body.Publication, releaseKeys, serverKeys)
}

func (r UpdateCheckResponse) verifyAuthority(request UpdateCheckRequest, serverKeys core.SigningKeyring) (UpdateCheckResponseBody, error) {
	if err := request.Validate(); err != nil {
		return UpdateCheckResponseBody{}, err
	}
	if err := r.Authority.Verify(serverKeys); err != nil {
		return UpdateCheckResponseBody{}, wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	body := r.Authority.Body
	if body.RequestID != request.RequestID || body.Product != request.Product {
		return UpdateCheckResponseBody{}, fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	return body, nil
}

func verifyAvailableUpdate(request UpdateCheckRequest, publication DeployFinalizeResponse, releaseKeys, serverKeys core.SigningKeyring) error {
	if err := publication.VerifyPublication(releaseKeys, serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	manifest := publication.Manifest.Body
	if request.RequestedReleaseID != nil && manifest.ReleaseID != *request.RequestedReleaseID {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	if manifest.ReleaseID == request.InstalledReleaseID {
		return fmt.Errorf(ErrFmtUpdateCheck, core.ErrReleaseContract)
	}
	_, err := DownloadForPlatform(publication.Index.Body, request.Platform)
	if err != nil {
		return wrapReleaseContract(ErrFmtUpdateCheck, err)
	}
	return nil
}

type SelfTestFailure struct {
	Check   SelfTestCheck `json:"check"`
	Failure UpdateFailure `json:"failure"`
}

func (f SelfTestFailure) Validate() error {
	if err := f.Check.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSelfTestFailure, err)
	}
	if err := f.Failure.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSelfTestFailure, err)
	}
	return nil
}

type SelfTestResult struct {
	Failure      *SelfTestFailure    `json:"failure,omitempty"`
	Version      core.ProductVersion `json:"version"`
	Commit       core.BuildCommit    `json:"commit"`
	BinarySHA256 core.SHA256Hex      `json:"binary_sha256"`
	ReleaseKeyID core.SigningKeyID   `json:"release_key_id"`
	ServerKeyID  core.SigningKeyID   `json:"server_key_id"`
	Checks       []SelfTestCheck     `json:"checks"`
	CheckCount   uint32              `json:"check_count"`
	Schema       core.SchemaID       `json:"schema"`
	Product      core.Product        `json:"product"`
	Platform     core.Platform       `json:"platform"`
	Status       SelfTestStatus      `json:"status"`
}

type SelfTestInput struct {
	Failure      *SelfTestFailure
	Version      core.ProductVersion
	Commit       core.BuildCommit
	BinarySHA256 core.SHA256Hex
	ReleaseKeyID core.SigningKeyID
	ServerKeyID  core.SigningKeyID
	Checks       []SelfTestCheck
	Product      core.Product
	Platform     core.Platform
	Status       SelfTestStatus
}

func BuildSelfTestResult(input SelfTestInput) (SelfTestResult, error) {
	if len(input.Checks) == 0 || len(input.Checks) > SelfTestCheckCount {
		return SelfTestResult{}, fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
	}
	checkCount := uint32(input.Checks[len(input.Checks)-1])
	result := SelfTestResult{
		Schema: core.SchemaReleaseSelfTestResult, Product: input.Product, Version: input.Version, Commit: input.Commit,
		Platform: input.Platform, BinarySHA256: input.BinarySHA256, ReleaseKeyID: input.ReleaseKeyID,
		ServerKeyID: input.ServerKeyID, Status: input.Status, Checks: slices.Clone(input.Checks), CheckCount: checkCount,
	}
	if input.Failure != nil {
		failure := *input.Failure
		result.Failure = &failure
	}
	if err := result.Validate(); err != nil {
		return SelfTestResult{}, err
	}
	return result, nil
}

func (r SelfTestResult) Validate() error {
	encoded, err := r.validatedJSON()
	if err != nil {
		return err
	}
	return validateUpdateBytes(encoded, ErrFmtSelfTestResult)
}

func (r SelfTestResult) validateStructure() error {
	if r.Schema != core.SchemaReleaseSelfTestResult {
		return fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
	}
	if err := validateSelfTestIdentity(r); err != nil {
		return err
	}
	if err := validateSelfTestChecks(r); err != nil {
		return err
	}
	return validateSelfTestOutcome(r)
}

func validateSelfTestIdentity(r SelfTestResult) error {
	for _, err := range []error{r.Product.Validate(), r.Version.Validate(), r.Commit.Validate(), r.Platform.Validate(),
		r.BinarySHA256.Validate(), r.ReleaseKeyID.Validate(), r.ServerKeyID.Validate(), r.Status.Validate()} {
		if err != nil {
			return wrapReleaseContract(ErrFmtSelfTestResult, err)
		}
	}
	if !r.Platform.IsReleaseTarget() || len(r.Checks) == 0 || len(r.Checks) > SelfTestCheckCount || int(r.CheckCount) != len(r.Checks) {
		return fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
	}
	return nil
}

func validateSelfTestChecks(r SelfTestResult) error {
	for index, check := range r.Checks {
		if err := check.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtSelfTestResult, err)
		}
		if int(check) != index+1 {
			return fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
		}
	}
	return nil
}

func validateSelfTestOutcome(r SelfTestResult) error {
	if r.Status == SelfTestStatusPassed {
		if len(r.Checks) == SelfTestCheckCount && r.Failure == nil {
			return nil
		}
		return fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
	}
	if r.Failure != nil && r.Failure.Validate() == nil && r.Failure.Check == r.Checks[len(r.Checks)-1] {
		return nil
	}
	return fmt.Errorf(ErrFmtSelfTestResult, core.ErrReleaseContract)
}

func (SelfTestResult) APIBody()                       {}
func (r SelfTestResult) MarshalJSON() ([]byte, error) { return r.validatedJSON() }

func (r SelfTestResult) validatedJSON() ([]byte, error) {
	if err := r.validateStructure(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	dst, err := core.AppendJSONField(dst, core.JSONFieldSchema, r.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, r.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, r.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, r.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatform, r.Platform)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldBinarySHA256, r.BinarySHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseKeyID, r.ReleaseKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldServerKeyID, r.ServerKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStatus, r.Status)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldChecks, r.Checks)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckCount, r.CheckCount)
	if r.Failure != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFailure, r.Failure)
	}
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type UpdateTargetIdentity struct {
	Version   core.ProductVersion `json:"version"`
	ReleaseID ReleaseID           `json:"release_id"`
	Commit    core.BuildCommit    `json:"commit"`
	SHA256    core.SHA256Hex      `json:"sha256"`
}

func (i UpdateTargetIdentity) Validate() error {
	for _, err := range []error{i.Version.Validate(), i.ReleaseID.Validate(), i.Commit.Validate(), i.SHA256.Validate()} {
		if err != nil {
			return wrapReleaseContract(ErrFmtUpdateDiagnostic, err)
		}
	}
	return nil
}

type UpdateDiagnosticID struct{ digest core.SHA256Hex }

func ParseUpdateDiagnosticID(value string) (UpdateDiagnosticID, error) {
	digest, err := core.ParseSHA256Hex(value)
	if err != nil {
		return UpdateDiagnosticID{}, wrapReleaseContract(ErrFmtUpdateDiagnosticID, err)
	}
	return UpdateDiagnosticID{digest: digest}, nil
}
func (id UpdateDiagnosticID) String() string { return id.digest.String() }
func (id UpdateDiagnosticID) Validate() error {
	if err := id.digest.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnosticID, err)
	}
	return nil
}
func (id UpdateDiagnosticID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.String())
}
func (id *UpdateDiagnosticID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtUpdateDiagnosticID, core.ErrReleaseContract)
	}
	parsed, err := ParseUpdateDiagnosticID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type UpdateDiagnosticInput struct {
	Target             *UpdateTargetIdentity `json:"target,omitempty"`
	SelfTest           *SelfTestResult       `json:"self_test,omitempty"`
	InstalledVersion   core.ProductVersion   `json:"installed_version"`
	InstalledReleaseID ReleaseID             `json:"installed_release_id"`
	InstalledCommit    core.BuildCommit      `json:"installed_commit"`
	InstalledSHA256    core.SHA256Hex        `json:"installed_sha256"`
	OccurredAt         core.UnixNanoTime     `json:"occurred_at"`
	Product            core.Product          `json:"product"`
	Platform           core.Platform         `json:"platform"`
	Phase              UpdatePhase           `json:"phase"`
	Failure            UpdateFailure         `json:"failure"`
	Rollback           RollbackOutcome       `json:"rollback"`
}

type UpdateDiagnostic struct {
	Target             *UpdateTargetIdentity `json:"target,omitempty"`
	SelfTest           *SelfTestResult       `json:"self_test,omitempty"`
	InstalledSHA256    core.SHA256Hex        `json:"installed_sha256"`
	InstalledVersion   core.ProductVersion   `json:"installed_version"`
	InstalledReleaseID ReleaseID             `json:"installed_release_id"`
	InstalledCommit    core.BuildCommit      `json:"installed_commit"`
	DiagnosticID       UpdateDiagnosticID    `json:"diagnostic_id"`
	OccurredAt         core.UnixNanoTime     `json:"occurred_at"`
	Schema             core.SchemaID         `json:"schema"`
	Platform           core.Platform         `json:"platform"`
	Product            core.Product          `json:"product"`
	Phase              UpdatePhase           `json:"phase"`
	Failure            UpdateFailure         `json:"failure"`
	Rollback           RollbackOutcome       `json:"rollback"`
}

func (d UpdateDiagnostic) HTTPIdempotencyKey() (core.HTTPIdempotencyKey, error) {
	return core.ParseHTTPIdempotencyKey(d.DiagnosticID.String())
}

func BuildUpdateDiagnostic(identity UpdateDiagnosticInput) (UpdateDiagnostic, error) {
	diagnostic := UpdateDiagnostic{
		Schema: core.SchemaReleaseUpdateDiagnostic, Product: identity.Product, InstalledVersion: identity.InstalledVersion,
		InstalledReleaseID: identity.InstalledReleaseID, InstalledSHA256: identity.InstalledSHA256, Platform: identity.Platform,
		InstalledCommit: identity.InstalledCommit,
		Target:          cloneUpdateTarget(identity.Target), Phase: identity.Phase, Failure: identity.Failure,
		SelfTest: cloneSelfTestResult(identity.SelfTest), Rollback: identity.Rollback, OccurredAt: identity.OccurredAt,
	}
	id, err := calculateUpdateDiagnosticID(diagnostic.identity())
	if err != nil {
		return UpdateDiagnostic{}, err
	}
	diagnostic.DiagnosticID = id
	if err := diagnostic.Validate(); err != nil {
		return UpdateDiagnostic{}, err
	}
	return diagnostic, nil
}

func (d UpdateDiagnostic) identity() UpdateDiagnosticInput {
	return UpdateDiagnosticInput{
		Product: d.Product, InstalledVersion: d.InstalledVersion, InstalledReleaseID: d.InstalledReleaseID,
		InstalledCommit: d.InstalledCommit, InstalledSHA256: d.InstalledSHA256, Platform: d.Platform, Target: d.Target, Phase: d.Phase,
		Failure: d.Failure, SelfTest: d.SelfTest, Rollback: d.Rollback, OccurredAt: d.OccurredAt,
	}
}

func (d UpdateDiagnostic) Validate() error {
	encoded, err := d.validatedJSON()
	if err != nil {
		return err
	}
	return validateUpdateBytes(encoded, ErrFmtUpdateDiagnostic)
}

func (d UpdateDiagnostic) validateStructure() error {
	if d.Schema != core.SchemaReleaseUpdateDiagnostic {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	if err := validateUpdateDiagnosticIdentity(d); err != nil {
		return err
	}
	if err := validateUpdateDiagnosticTarget(d); err != nil {
		return err
	}
	if err := validateUpdateDiagnosticSelfTest(d); err != nil {
		return err
	}
	wantID, err := calculateUpdateDiagnosticID(d.identity())
	if err != nil || wantID != d.DiagnosticID {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	return nil
}

func validateUpdateDiagnosticIdentity(d UpdateDiagnostic) error {
	for _, err := range []error{d.DiagnosticID.Validate(), d.Product.Validate(), d.InstalledVersion.Validate(),
		d.InstalledReleaseID.Validate(), d.InstalledCommit.Validate(), d.InstalledSHA256.Validate(), d.Platform.Validate(), d.Phase.Validate(),
		d.Failure.Validate(), d.Rollback.Validate(), core.ValidateRequiredUnixNanoTime(d.OccurredAt)} {
		if err != nil {
			return wrapReleaseContract(ErrFmtUpdateDiagnostic, err)
		}
	}
	if !d.Platform.IsReleaseTarget() {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	if err := ValidateReleaseIDIdentity(d.InstalledReleaseID, d.Product, d.InstalledVersion, d.InstalledCommit); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnostic, err)
	}
	return nil
}

func validateUpdateDiagnosticTarget(d UpdateDiagnostic) error {
	if d.Target != nil && d.Target.Validate() != nil {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	if d.Target != nil {
		if err := ValidateReleaseIDIdentity(d.Target.ReleaseID, d.Product, d.Target.Version, d.Target.Commit); err != nil {
			return wrapReleaseContract(ErrFmtUpdateDiagnostic, err)
		}
	}
	if d.Phase != UpdatePhasePublication && d.Target == nil {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	return nil
}

func validateUpdateDiagnosticSelfTest(d UpdateDiagnostic) error {
	wantsSelfTest := d.Phase == UpdatePhaseCandidateSelfTest || d.Phase == UpdatePhaseInstalledSelfTest
	if !wantsSelfTest && d.SelfTest != nil {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	if !wantsSelfTest {
		return nil
	}
	if d.SelfTest == nil {
		return validateMissingUpdateSelfTest(d.Failure)
	}
	if err := d.SelfTest.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnostic, err)
	}
	if d.SelfTest.Product != d.Product || d.SelfTest.Platform != d.Platform {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	if d.SelfTest.Status == SelfTestStatusPassed {
		return validatePassedUpdateSelfTest(d.Failure)
	}
	return validateFailedUpdateSelfTest(*d.SelfTest, d.Failure)
}

func validatePassedUpdateSelfTest(failure UpdateFailure) error {
	if failure != UpdateFailureContract && failure != UpdateFailureIntegrity {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	return nil
}

func validateFailedUpdateSelfTest(result SelfTestResult, failure UpdateFailure) error {
	if result.Failure == nil || result.Failure.Failure != failure {
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
	return nil
}

func validateMissingUpdateSelfTest(failure UpdateFailure) error {
	switch failure {
	case UpdateFailureContract, UpdateFailureExecution, UpdateFailureFilesystem, UpdateFailureCancelled, UpdateFailureTimeout:
		return nil
	default:
		return fmt.Errorf(ErrFmtUpdateDiagnostic, core.ErrReleaseContract)
	}
}

func (UpdateDiagnostic) APIBody()                       {}
func (d UpdateDiagnostic) MarshalJSON() ([]byte, error) { return d.validatedJSON() }

func (d UpdateDiagnostic) validatedJSON() ([]byte, error) {
	if err := d.validateStructure(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	dst, err := core.AppendJSONField(dst, core.JSONFieldSchema, d.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDiagnosticID, d.DiagnosticID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, d.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledVersion, d.InstalledVersion)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledReleaseID, d.InstalledReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledCommit, d.InstalledCommit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldInstalledSHA256, d.InstalledSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatform, d.Platform)
	if d.Target != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTarget, d.Target)
	}
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPhase, d.Phase)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldFailure, d.Failure)
	if d.SelfTest != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSelfTest, d.SelfTest)
	}
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRollback, d.Rollback)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOccurredAt, d.OccurredAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type UpdateDiagnosticReceiptBody struct {
	DiagnosticID UpdateDiagnosticID    `json:"diagnostic_id"`
	RecordedAt   core.UnixNanoTime     `json:"recorded_at"`
	Schema       core.SchemaID         `json:"schema"`
	Disposition  DiagnosticDisposition `json:"disposition"`
}

func (b UpdateDiagnosticReceiptBody) Validate() error {
	if b.Schema != core.SchemaReleaseUpdateDiagnosticReceipt {
		return fmt.Errorf(ErrFmtUpdateDiagnosticReceipt, core.ErrReleaseContract)
	}
	if err := b.DiagnosticID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnosticReceipt, err)
	}
	if err := b.Disposition.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnosticReceipt, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.RecordedAt); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnosticReceipt, err)
	}
	return nil
}
func (b UpdateDiagnosticReceiptBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	dst, err := core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDiagnosticID, b.DiagnosticID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDisposition, b.Disposition)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRecordedAt, b.RecordedAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}
func (b UpdateDiagnosticReceiptBody) SigningSchema() core.SchemaID { return b.Schema }
func (b UpdateDiagnosticReceiptBody) MarshalJSON() ([]byte, error) { return b.Canonical(nil) }

type UpdateDiagnosticReceipt struct {
	Authority core.Signed[UpdateDiagnosticReceiptBody] `json:"authority"`
}

func (UpdateDiagnosticReceipt) APIBody()          {}
func (r UpdateDiagnosticReceipt) Validate() error { return r.Authority.Validate() }
func (r UpdateDiagnosticReceipt) Verify(diagnostic UpdateDiagnostic, serverKeys core.SigningKeyring) error {
	if err := diagnostic.Validate(); err != nil {
		return err
	}
	if err := r.Authority.Verify(serverKeys); err != nil {
		return wrapReleaseContract(ErrFmtUpdateDiagnosticReceipt, err)
	}
	if r.Authority.Body.DiagnosticID != diagnostic.DiagnosticID {
		return fmt.Errorf(ErrFmtUpdateDiagnosticReceipt, core.ErrReleaseContract)
	}
	return nil
}

func calculateUpdateDiagnosticID(identity UpdateDiagnosticInput) (UpdateDiagnosticID, error) {
	data, err := json.Marshal(identity)
	if err != nil {
		return UpdateDiagnosticID{}, wrapReleaseContract(ErrFmtUpdateDiagnosticID, err)
	}
	return UpdateDiagnosticID{digest: core.NewSHA256Hex(sha256.Sum256(data))}, nil
}

func cloneUpdateTarget(target *UpdateTargetIdentity) *UpdateTargetIdentity {
	if target == nil {
		return nil
	}
	clone := *target
	return &clone
}

func cloneSelfTestResult(result *SelfTestResult) *SelfTestResult {
	if result == nil {
		return nil
	}
	clone := *result
	clone.Checks = slices.Clone(result.Checks)
	if result.Failure != nil {
		failure := *result.Failure
		clone.Failure = &failure
	}
	return &clone
}

func cloneDeployFinalizeResponse(response DeployFinalizeResponse) DeployFinalizeResponse {
	response.Manifest = cloneSignedManifest(response.Manifest)
	response.Receipt = cloneSignedUploadReceipt(response.Receipt)
	response.Index = cloneSignedDownloadIndex(response.Index)
	return response
}

func validateUpdateBytes(data []byte, format string) error {
	if len(data) == 0 || len(data) > UpdateTransportMaximumBytes {
		return fmt.Errorf(format, core.ErrReleaseContract)
	}
	return nil
}
