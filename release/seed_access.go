package release

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	SeedGrantAccessLifetime  = 5 * time.Minute
	SeedGrantAccessClockSkew = 30 * time.Second
)

// SeedGrantAccessBody is the machine-auth request for a server-signed garble
// seed grant. The release pipeline signs it with the product's release key;
// Offgridsoftware verifies against the pinned release public key, so seed
// issuance requires no interactive session. Replay is harmless by design: the
// server idempotently returns the same grant for one ReleaseID.
type SeedGrantAccessBody struct {
	Request   SeedRequest       `json:"request"`
	IssuedAt  core.UnixNanoTime `json:"issued_at"`
	ExpiresAt core.UnixNanoTime `json:"expires_at"`
	Schema    core.SchemaID     `json:"schema"`
}

func BuildSeedGrantAccessBody(request SeedRequest, issuedAt core.UnixNanoTime) (SeedGrantAccessBody, error) {
	body := SeedGrantAccessBody{
		Request: request, IssuedAt: issuedAt, ExpiresAt: issuedAt.Add(SeedGrantAccessLifetime),
		Schema: core.SchemaReleaseSeedGrantAccess,
	}
	if err := body.Validate(); err != nil {
		return SeedGrantAccessBody{}, err
	}
	return body, nil
}

func (b SeedGrantAccessBody) Validate() error {
	if b.Schema != core.SchemaReleaseSeedGrantAccess {
		return fmt.Errorf(ErrFmtSeedGrantAccess, core.ErrReleaseContract)
	}
	if err := b.Request.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrantAccess, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.IssuedAt); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrantAccess, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.ExpiresAt); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrantAccess, err)
	}
	if !b.ExpiresAt.Equal(b.IssuedAt.Add(SeedGrantAccessLifetime)) {
		return fmt.Errorf(ErrFmtSeedGrantAccess, core.ErrReleaseContract)
	}
	return nil
}

func (b SeedGrantAccessBody) ValidateAt(now core.UnixNanoTime) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := core.ValidateRequiredUnixNanoTime(now); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrantAccess, err)
	}
	if now.Before(b.IssuedAt.Add(-SeedGrantAccessClockSkew)) || now.After(b.ExpiresAt) {
		return fmt.Errorf(ErrFmtSeedGrantAccess, core.ErrReleaseContract)
	}
	return nil
}

func (b SeedGrantAccessBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequest, b.Request)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldExpiresAt, b.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b SeedGrantAccessBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

func (b SeedGrantAccessBody) SigningSchema() core.SchemaID {
	return b.Schema
}

var _ core.CanonicalBody = SeedGrantAccessBody{}

// SeedGrant requests a server-signed garble seed grant over the machine-auth
// seed endpoint. The signed request must verify against the client's release
// keyring, and the returned grant must verify against the server keyring and
// bind to the exact requested release identity.
func (c DeployClient) SeedGrant(ctx context.Context, request core.Signed[SeedGrantAccessBody]) (core.Signed[SeedGrantBody], error) {
	if err := c.Validate(); err != nil {
		return core.Signed[SeedGrantBody]{}, err
	}
	if err := request.Verify(c.ReleaseKeys); err != nil {
		return core.Signed[SeedGrantBody]{}, err
	}
	if request.Body.Request.Product != c.Endpoints.Product {
		return core.Signed[SeedGrantBody]{}, fmt.Errorf(ErrFmtDeployClient, core.ErrReleaseContract)
	}
	grant, err := seedGrantPost(ctx, c.HTTP, c.Endpoints.Seed, request)
	if err != nil {
		return core.Signed[SeedGrantBody]{}, err
	}
	if err := VerifySeedGrant(grant, request.Body.Request, c.ServerKeys); err != nil {
		return core.Signed[SeedGrantBody]{}, err
	}
	return grant, nil
}

func seedGrantPost(
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request core.Signed[SeedGrantAccessBody],
) (core.Signed[SeedGrantBody], error) {
	statusCode, contentType, responseBody, err := seedGrantExchange(ctx, httpClient, endpoint, request)
	if err != nil {
		return core.Signed[SeedGrantBody]{}, err
	}
	return decodeSeedGrantHTTPResponse(statusCode, contentType, responseBody)
}

func seedGrantExchange(
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request core.Signed[SeedGrantAccessBody],
) (int, string, []byte, error) {
	if ctx == nil {
		return 0, "", nil, fmt.Errorf(ErrFmtDeployClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	body, err := json.Marshal(request)
	if err != nil {
		return 0, "", nil, DeployHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, ReleaseAPIHTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return 0, "", nil, DeployHTTPError{Cause: err}
	}
	httpRequest.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return 0, "", nil, DeployHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return 0, "", nil, DeployHTTPError{Cause: core.ErrReleaseContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(responseBody) == 0 || len(responseBody) > core.StrictJSONMaxBytes {
		return 0, "", nil, DeployHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrReleaseContract}
	}
	return httpResponse.StatusCode, httpResponse.Header.Get(core.HTTPHeaderContentType), responseBody, nil
}

// decodeSeedGrantHTTPResponse mirrors the seed handler's wire contract: a
// success writes the bare signed grant; a fault writes the API failure
// envelope.
func decodeSeedGrantHTTPResponse(statusCode int, contentType string, responseBody []byte) (core.Signed[SeedGrantBody], error) {
	if !strings.HasPrefix(contentType, core.HTTPContentTypeJSON) {
		return core.Signed[SeedGrantBody]{}, DeployHTTPError{StatusCode: statusCode, Cause: core.ErrReleaseContract}
	}
	if statusCode != core.HTTPStatusOK {
		envelope, err := core.DecodeStrictJSONStructure[core.APIEnvelope[core.Signed[SeedGrantBody]]](responseBody)
		if err != nil {
			return core.Signed[SeedGrantBody]{}, DeployHTTPError{StatusCode: statusCode, Cause: err}
		}
		if err := envelope.ValidateFailure(); err != nil {
			return core.Signed[SeedGrantBody]{}, DeployHTTPError{StatusCode: statusCode, Cause: err}
		}
		return core.Signed[SeedGrantBody]{}, DeployAPIError{StatusCode: statusCode, RequestID: envelope.RequestID, Body: *envelope.Error}
	}
	grant, err := core.DecodeStrictJSON[core.Signed[SeedGrantBody]](responseBody)
	if err != nil {
		return core.Signed[SeedGrantBody]{}, DeployHTTPError{StatusCode: statusCode, Cause: err}
	}
	return grant, nil
}
