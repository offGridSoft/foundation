package release

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	UpdateDownloadHTTPBudget         = 2 * time.Minute
	UpdateSelfTestBudget             = 15 * time.Second
	UpdateSelfTestOutputMaximumBytes = 64 << 10
)

type UpdateAPIError struct {
	RequestID  core.APIRequestID
	Body       core.APIErrorBody
	StatusCode int
}

func (e UpdateAPIError) Error() string {
	return fmt.Sprintf(ErrFmtUpdateAPI, e.StatusCode, e.Body.Code, e.Body.Message)
}
func (e UpdateAPIError) Unwrap() error { return core.ErrReleaseContract }

type UpdateHTTPError struct {
	Cause      error
	StatusCode int
}

func (e UpdateHTTPError) Error() string { return fmt.Sprintf(ErrFmtUpdateHTTP, e.StatusCode, e.Cause) }
func (e UpdateHTTPError) Unwrap() error { return errors.Join(core.ErrReleaseContract, e.Cause) }

type UpdateClient struct {
	HTTP        *http.Client
	Endpoints   UpdateEndpoints
	ReleaseKeys core.SigningKeyring
	ServerKeys  core.SigningKeyring
}

func (c UpdateClient) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtUpdateClient, core.ErrReleaseContract)
	}
	if err := c.Endpoints.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateClient, err)
	}
	if err := c.ReleaseKeys.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateClient, err)
	}
	if err := c.ServerKeys.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUpdateClient, err)
	}
	return nil
}

func (c UpdateClient) Check(ctx context.Context, request UpdateCheckRequest) (UpdateCheckResponse, error) {
	if err := c.Validate(); err != nil {
		return UpdateCheckResponse{}, err
	}
	if err := request.Validate(); err != nil {
		return UpdateCheckResponse{}, err
	}
	if request.Product != c.Endpoints.Product {
		return UpdateCheckResponse{}, fmt.Errorf(ErrFmtUpdateClient, core.ErrReleaseContract)
	}
	response, err := deployPost[UpdateCheckRequest, UpdateCheckResponse](ctx, c.HTTP, c.Endpoints.Check, request)
	if err != nil {
		return UpdateCheckResponse{}, translateUpdateTransportError(err)
	}
	if err := response.Verify(request, c.ReleaseKeys, c.ServerKeys); err != nil {
		return UpdateCheckResponse{}, err
	}
	return response, nil
}

func (c UpdateClient) Report(ctx context.Context, diagnostic UpdateDiagnostic) (UpdateDiagnosticReceipt, error) {
	if err := c.Validate(); err != nil {
		return UpdateDiagnosticReceipt{}, err
	}
	if err := diagnostic.Validate(); err != nil {
		return UpdateDiagnosticReceipt{}, err
	}
	if diagnostic.Product != c.Endpoints.Product {
		return UpdateDiagnosticReceipt{}, fmt.Errorf(ErrFmtUpdateClient, core.ErrReleaseContract)
	}
	receipt, err := deployPost[UpdateDiagnostic, UpdateDiagnosticReceipt](ctx, c.HTTP, c.Endpoints.Diagnostic, diagnostic)
	if err != nil {
		return UpdateDiagnosticReceipt{}, translateUpdateTransportError(err)
	}
	if err := receipt.Verify(diagnostic, c.ServerKeys); err != nil {
		return UpdateDiagnosticReceipt{}, err
	}
	return receipt, nil
}

func translateUpdateTransportError(err error) error {
	if apiError, ok := errors.AsType[DeployAPIError](err); ok {
		return UpdateAPIError(apiError)
	}
	if httpError, ok := errors.AsType[DeployHTTPError](err); ok {
		return UpdateHTTPError(httpError)
	}
	return wrapReleaseContract(ErrFmtUpdateClient, err)
}
