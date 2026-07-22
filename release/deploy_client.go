package release

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/exchange"
)

const (
	ReleaseAPIHTTPBudget = 15 * time.Second
)

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

func deployPost[Request core.HTTPIdempotentBody, Response core.Validatable](
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request Request,
) (Response, error) {
	var zero Response
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtDeployClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	key, err := request.HTTPIdempotencyKey()
	if err != nil {
		return zero, DeployHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, ReleaseAPIHTTPBudget)
	defer cancel()
	exchangeRequest := exchange.Request[Request]{
		Body: &request, Endpoint: endpoint,
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent, IdempotencyKey: key},
		ExpectedStatus: core.HTTPStatusOK,
	}
	response, err := exchange.SendJSON[Request, Response](requestContext, exchange.Client{HTTP: httpClient}, exchangeRequest, releaseAPIClientPolicy())
	return deployExchangeResult(response, err)
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
	requestContext, cancel := context.WithTimeout(ctx, ReleaseAPIHTTPBudget)
	defer cancel()
	exchangeRequest := exchange.Request[core.HTTPNoBody]{
		Endpoint:       endpoint,
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySafe},
		ExpectedStatus: core.HTTPStatusOK,
	}
	response, err := exchange.SendJSON[core.HTTPNoBody, Response](requestContext, exchange.Client{HTTP: httpClient}, exchangeRequest, releaseAPIClientPolicy())
	return deployExchangeResult(response, err)
}

func deployExchangeResult[Response core.Validatable](response exchange.Response[Response], cause error) (Response, error) {
	var zero Response
	if cause == nil {
		return *response.Envelope.Data, nil
	}
	if apiError, ok := errors.AsType[exchange.ResponseError](cause); ok {
		return zero, DeployAPIError{StatusCode: apiError.Status.Int(), RequestID: apiError.RequestID, Body: apiError.Body}
	}
	return zero, DeployHTTPError{StatusCode: response.Status.Int(), Cause: cause}
}

func releaseAPIClientPolicy() exchange.ClientPolicy {
	return exchange.ClientPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(ReleaseAPIHTTPBudget),
		RequestBodyLimit:  core.NewByteCount(core.StrictJSONMaxBytes),
		ResponseBodyLimit: core.NewByteCount(core.StrictJSONMaxBytes),
		Retry:             core.DefaultHTTPRetryPolicy(),
		Redirect:          core.HTTPRedirectReject,
	}
}
