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

const DeployHTTPBudget = 15 * time.Second

type DeployAPIError struct {
	RequestID  core.APIRequestID
	Body       core.APIErrorBody
	StatusCode int
}

func (e DeployAPIError) Error() string {
	return fmt.Sprintf(ErrFmtDeployAPI, e.StatusCode, e.Body.Code, e.Body.Message)
}

func (e DeployAPIError) Unwrap() error { return core.ErrReleaseContract }

type DeployHTTPError struct {
	Cause      error
	StatusCode int
}

func (e DeployHTTPError) Error() string {
	return fmt.Sprintf(ErrFmtDeployHTTP, e.StatusCode, e.Cause)
}

func (e DeployHTTPError) Unwrap() error {
	return errors.Join(core.ErrReleaseContract, e.Cause)
}

type DeployClient struct {
	HTTP        *http.Client
	Endpoints   DeployEndpoints
	ReleaseKeys core.SigningKeyring
	ServerKeys  core.SigningKeyring
}

func (c DeployClient) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtDeployClient, core.ErrReleaseContract)
	}
	if err := c.Endpoints.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDeployClient, err)
	}
	if err := c.ReleaseKeys.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDeployClient, err)
	}
	if err := c.ServerKeys.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDeployClient, err)
	}
	return nil
}

func (c DeployClient) Prepare(ctx context.Context, request DeployPrepareRequest) (DeployPrepareResponse, error) {
	if err := c.Validate(); err != nil {
		return DeployPrepareResponse{}, err
	}
	if err := request.Verify(c.ReleaseKeys); err != nil {
		return DeployPrepareResponse{}, err
	}
	response, err := deployPost[DeployPrepareRequest, DeployPrepareResponse](ctx, c.HTTP, c.Endpoints.Prepare, request)
	if err != nil {
		return DeployPrepareResponse{}, err
	}
	if err := response.Verify(request, c.ReleaseKeys, c.ServerKeys); err != nil {
		return DeployPrepareResponse{}, err
	}
	return response, nil
}

func (c DeployClient) Finalize(ctx context.Context, request DeployFinalizeRequest) (DeployFinalizeResponse, error) {
	if err := c.Validate(); err != nil {
		return DeployFinalizeResponse{}, err
	}
	if err := request.Verify(c.ReleaseKeys, c.ServerKeys); err != nil {
		return DeployFinalizeResponse{}, err
	}
	response, err := deployPost[DeployFinalizeRequest, DeployFinalizeResponse](ctx, c.HTTP, c.Endpoints.Finalize, request)
	if err != nil {
		return DeployFinalizeResponse{}, err
	}
	if err := response.Verify(request, c.ReleaseKeys, c.ServerKeys); err != nil {
		return DeployFinalizeResponse{}, err
	}
	return response, nil
}

// Publication returns the already-finalized publication for releaseID. A
// not-yet-published release returns DeployAPIError with HTTPStatusNotFound.
// Successful responses are accepted only after both server signatures bind to
// the caller's manifest.
func (c DeployClient) Publication(ctx context.Context, releaseID ReleaseID) (DeployFinalizeResponse, error) {
	if err := c.Validate(); err != nil {
		return DeployFinalizeResponse{}, err
	}
	if err := releaseID.Validate(); err != nil {
		return DeployFinalizeResponse{}, fmt.Errorf(ErrFmtDeployClient, err)
	}
	endpoint, err := c.Endpoints.Status(releaseID)
	if err != nil {
		return DeployFinalizeResponse{}, err
	}
	response, err := deployGet[DeployFinalizeResponse](ctx, c.HTTP, endpoint)
	if err != nil {
		return DeployFinalizeResponse{}, err
	}
	if err := response.VerifyPublication(c.ReleaseKeys, c.ServerKeys); err != nil {
		return DeployFinalizeResponse{}, err
	}
	if response.Manifest.Body.ReleaseID != releaseID {
		return DeployFinalizeResponse{}, fmt.Errorf(ErrFmtDeployClient, core.ErrReleaseContract)
	}
	return response, nil
}

// Latest returns the newest publication selected by the product-specific
// endpoint. The publication is accepted only after the offline release
// authority and online server authority both verify.
func (c DeployClient) Latest(ctx context.Context) (DeployFinalizeResponse, error) {
	if err := c.Validate(); err != nil {
		return DeployFinalizeResponse{}, err
	}
	response, err := deployGet[DeployFinalizeResponse](ctx, c.HTTP, c.Endpoints.Latest)
	if err != nil {
		return DeployFinalizeResponse{}, err
	}
	if err := response.VerifyPublication(c.ReleaseKeys, c.ServerKeys); err != nil {
		return DeployFinalizeResponse{}, err
	}
	if response.Manifest.Body.Product != c.Endpoints.Product {
		return DeployFinalizeResponse{}, fmt.Errorf(ErrFmtDeployClient, core.ErrReleaseContract)
	}
	return response, nil
}

func deployPost[Request core.Validatable, Response core.Validatable](
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request Request,
) (Response, error) {
	var zero Response
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtDeployClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	body, err := json.Marshal(request)
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, DeployHTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	httpRequest.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return zero, DeployHTTPError{Cause: core.ErrReleaseContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	contentType := httpResponse.Header.Get(core.HTTPHeaderContentType)
	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(responseBody) == 0 || len(responseBody) > core.StrictJSONMaxBytes {
		return zero, DeployHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrReleaseContract}
	}
	return decodeDeployHTTPResponse[Response](httpResponse.StatusCode, contentType, responseBody)
}

func deployGet[Response core.Validatable](
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
) (Response, error) {
	var zero Response
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtDeployClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	requestContext, cancel := context.WithTimeout(ctx, DeployHTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return zero, DeployHTTPError{Cause: core.ErrReleaseContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	contentType := httpResponse.Header.Get(core.HTTPHeaderContentType)
	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(responseBody) == 0 || len(responseBody) > core.StrictJSONMaxBytes {
		return zero, DeployHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrReleaseContract}
	}
	return decodeDeployHTTPResponse[Response](httpResponse.StatusCode, contentType, responseBody)
}

func decodeDeployHTTPResponse[Response core.Validatable](statusCode int, contentType string, responseBody []byte) (Response, error) {
	var zero Response
	if !strings.HasPrefix(contentType, core.HTTPContentTypeJSON) {
		return zero, DeployHTTPError{StatusCode: statusCode, Cause: core.ErrReleaseContract}
	}
	envelope, err := core.DecodeStrictJSON[core.APIEnvelope[Response]](responseBody)
	if err != nil {
		return zero, DeployHTTPError{StatusCode: statusCode, Cause: err}
	}
	if statusCode != core.HTTPStatusOK {
		if err := envelope.ValidateFailure(); err != nil {
			return zero, DeployHTTPError{StatusCode: statusCode, Cause: err}
		}
		return zero, DeployAPIError{StatusCode: statusCode, RequestID: envelope.RequestID, Body: *envelope.Error}
	}
	if err := envelope.ValidateSuccess(); err != nil {
		return zero, DeployHTTPError{StatusCode: statusCode, Cause: err}
	}
	return *envelope.Data, nil
}
