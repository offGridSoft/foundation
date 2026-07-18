package release

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/offGridSoft/foundation/v2026/core"
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
	response, err := releaseDataPost(ctx, c, request)
	if err != nil {
		return ReleaseDataResponse{}, err
	}
	if response.Request != request {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataClient, core.ErrReleaseContract)
	}
	return response, nil
}

func releaseDataPost(ctx context.Context, client ReleaseDataClient, request ReleaseDataRequest) (ReleaseDataResponse, error) {
	if ctx == nil {
		return ReleaseDataResponse{}, fmt.Errorf(ErrFmtReleaseDataClient, errors.Join(core.ErrReleaseContract, core.ErrNilContext))
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, ReleaseAPIHTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, client.Endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	bearer, err := client.Token.BearerValue()
	if err != nil {
		return ReleaseDataResponse{}, err
	}
	httpRequest.Header.Set(core.HTTPHeaderAuthorization, bearer)
	httpRequest.Header.Set(core.HTTPHeaderAccept, core.HTTPContentTypeJSON)
	httpRequest.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	return executeReleaseDataRequest(client.HTTP, httpRequest)
}

var (
	_ core.Validatable = ReleaseDataSource{}
	_ core.Validatable = ReleaseDataClient{}
)

func executeReleaseDataRequest(client *http.Client, request *http.Request) (ReleaseDataResponse, error) {
	boundedClient := *client
	boundedClient.CheckRedirect = refuseReleaseDataRedirect
	httpResponse, err := boundedClient.Do(request)
	if err != nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: err}
	}
	return readReleaseDataResponse(httpResponse)
}

func refuseReleaseDataRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func readReleaseDataResponse(httpResponse *http.Response) (ReleaseDataResponse, error) {
	if httpResponse == nil || httpResponse.Body == nil {
		return ReleaseDataResponse{}, DeployHTTPError{Cause: core.ErrReleaseContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(responseBody) == 0 || len(responseBody) > core.StrictJSONMaxBytes {
		return ReleaseDataResponse{}, DeployHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrReleaseContract}
	}
	return decodeDeployHTTPResponse[ReleaseDataResponse](
		httpResponse.StatusCode,
		httpResponse.Header.Get(core.HTTPHeaderContentType),
		responseBody,
	)
}
