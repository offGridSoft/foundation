package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	GarbleToolVersionToken = "v0.16.0"
)

func GarbleToolVersion() (ToolVersion, error) {
	return ParseToolVersion(GarbleToolVersionToken)
}

type ReleaseDataRequest struct {
	Version   core.ProductVersion `json:"version"`
	ReleaseID ReleaseID           `json:"release_id"`
	RequestID DeployRequestID     `json:"request_id"`
	Commit    core.BuildCommit    `json:"commit"`
	Schema    core.SchemaID       `json:"schema"`
	Product   core.Product        `json:"product"`
}

func BuildReleaseDataRequest(product core.Product, version core.ProductVersion, commit core.BuildCommit, requestID DeployRequestID) (ReleaseDataRequest, error) {
	releaseID, err := BuildReleaseID(product, version, commit)
	if err != nil {
		return ReleaseDataRequest{}, err
	}
	request := ReleaseDataRequest{
		Version: version, ReleaseID: releaseID, RequestID: requestID, Commit: commit,
		Schema: core.SchemaReleaseDataRequest, Product: product,
	}
	if err := request.Validate(); err != nil {
		return ReleaseDataRequest{}, err
	}
	return request, nil
}

func (r ReleaseDataRequest) Validate() error {
	if r.Schema != core.SchemaReleaseDataRequest {
		return fmt.Errorf(ErrFmtReleaseDataRequest, core.ErrReleaseContract)
	}
	if err := r.RequestID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseDataRequest, err)
	}
	if err := ValidateReleaseIDIdentity(r.ReleaseID, r.Product, r.Version, r.Commit); err != nil {
		return wrapReleaseContract(ErrFmtReleaseDataRequest, err)
	}
	return nil
}

func (r ReleaseDataRequest) HTTPIdempotencyKey() (core.HTTPIdempotencyKey, error) {
	return core.ParseHTTPIdempotencyKey(r.RequestID.String())
}

func (r ReleaseDataRequest) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	type wire ReleaseDataRequest
	return json.Marshal(wire(r))
}

type ReleaseDataResponse struct {
	ReleaseSigningKey core.GeneratedSigningKey `json:"release_signing_key"`
	ServerPublicKey   core.Ed25519PublicKeyHex `json:"server_public_key"`
	GarbleVersion     ToolVersion              `json:"garble_version"`
	Request           ReleaseDataRequest       `json:"request"`
	Schema            core.SchemaID            `json:"schema"`
	GarbleCustodySeed core.GarbleCustodySeed   `json:"garble_custody_seed"`
}

func (ReleaseDataResponse) APIBody() {}

func (r ReleaseDataResponse) Validate() error {
	if r.Schema != core.SchemaReleaseDataResponse {
		return fmt.Errorf(ErrFmtReleaseDataResponse, core.ErrReleaseContract)
	}
	checks := []error{
		r.Request.Validate(), r.ReleaseSigningKey.Validate(), r.GarbleCustodySeed.Validate(),
		r.ServerPublicKey.Validate(), r.GarbleVersion.Validate(),
	}
	for _, err := range checks {
		if err != nil {
			return wrapReleaseContract(ErrFmtReleaseDataResponse, err)
		}
	}
	expectedGarbleVersion, err := GarbleToolVersion()
	if err != nil || r.GarbleVersion != expectedGarbleVersion {
		return fmt.Errorf(ErrFmtReleaseDataResponse, core.ErrReleaseContract)
	}
	return nil
}

func (r ReleaseDataResponse) SigningKey() (core.Ed25519SigningKey, error) {
	if err := r.Validate(); err != nil {
		return core.Ed25519SigningKey{}, err
	}
	key, err := core.ParseEd25519SigningKeyBase64(r.ReleaseSigningKey.PrivateKeyBase64)
	if err != nil {
		return core.Ed25519SigningKey{}, wrapReleaseContract(ErrFmtReleaseDataResponse, err)
	}
	return key, nil
}

func (r ReleaseDataResponse) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	type wire ReleaseDataResponse
	return json.Marshal(wire(r))
}

var (
	_ core.Validatable = ReleaseDataRequest{}
	_ core.APIBody     = ReleaseDataResponse{}
)
