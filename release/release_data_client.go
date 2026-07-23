package release

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/exchange"
	"github.com/offGridSoft/foundation/v2026/workloadidentity"
)

// ReleaseDataSource composes workload identity with the product-bound release
// endpoint. Bug and Witness consume this one boundary instead of rebuilding
// the identity-token-to-release-client protocol independently.
type ReleaseDataSource struct {
	HTTP      *http.Client
	Identity  workloadidentity.TokenSource
	Endpoints DeployEndpoints
}

func (s ReleaseDataSource) Validate() error {
	if s.HTTP == nil || s.Identity == nil {
		return fmt.Errorf(ErrFmtReleaseDataSource, core.ErrReleaseContract)
	}
	checks := []error{s.Identity.Validate(), s.Endpoints.Validate()}
	for _, err := range checks {
		if err != nil {
			return wrapReleaseContract(ErrFmtReleaseDataSource, err)
		}
	}
	return nil
}

func (s ReleaseDataSource) Fetch(ctx context.Context, request ReleaseDataRequest) (ReleaseDataResponse, error) {
	if err := s.Validate(); err != nil {
		return ReleaseDataResponse{}, err
	}
	if ctx == nil {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataSource, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	if err := request.Validate(); err != nil || request.Product != s.Endpoints.Product {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataSource, core.ErrReleaseContract)
	}
	token, err := s.Identity.Token(ctx)
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	client := ReleaseDataClient{HTTP: s.HTTP, Endpoint: s.Endpoints.Data, Token: token, Product: s.Endpoints.Product}
	return client.Fetch(ctx, request)
}

type ReleaseDataClient struct {
	HTTP     *http.Client
	Endpoint core.APIEndpoint
	Token    workloadidentity.Token
	Product  core.Product
}

func (c ReleaseDataClient) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtReleaseDataClient, core.ErrReleaseContract)
	}
	checks := []error{c.Endpoint.Validate(), c.Token.Validate(), c.Product.Validate()}
	for _, err := range checks {
		if err != nil {
			return wrapReleaseContract(ErrFmtReleaseDataClient, err)
		}
	}
	return nil
}

func (c ReleaseDataClient) Fetch(ctx context.Context, request ReleaseDataRequest) (ReleaseDataResponse, error) {
	if err := c.Validate(); err != nil {
		return ReleaseDataResponse{}, err
	}
	if err := request.Validate(); err != nil || request.Product != c.Product {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataClient, core.ErrReleaseContract)
	}
	if ctx == nil {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	idempotencyKey, err := request.HTTPIdempotencyKey()
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	bearer, err := c.Token.BearerValue()
	if err != nil {
		return ReleaseDataResponse{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, ReleaseAPIHTTPBudget)
	defer cancel()
	exchangeRequest := exchange.Request[ReleaseDataRequest]{
		Body:     &request,
		Endpoint: c.Endpoint,
		Headers: core.HTTPHeaders{Values: []core.HTTPHeader{{
			Name: core.HTTPHeaderAuthorization, Value: bearer,
		}}},
		Semantics: core.HTTPRequestSemantics{
			Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent, IdempotencyKey: idempotencyKey,
		},
		ExpectedStatus: core.HTTPStatusOK,
	}
	exchangeClient, err := exchange.NewClient(c.HTTP)
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	exchangeResponse, err := exchange.SendJSON[ReleaseDataRequest, ReleaseDataResponse](
		requestContext, exchangeClient, exchangeRequest, releaseAPIClientPolicy(),
	)
	response, err := deployExchangeResult(exchangeResponse, err)
	if err != nil {
		return ReleaseDataResponse{}, err
	}
	if response.Request != request {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataClient, core.ErrReleaseContract)
	}
	return response, nil
}

var (
	_ core.Validatable = ReleaseDataSource{}
	_ core.Validatable = ReleaseDataClient{}
)
