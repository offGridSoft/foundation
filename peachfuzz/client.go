package peachfuzz

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
	HTTPClientTimeout = 15 * time.Second
)

type SnapshotClient struct {
	HTTP     *http.Client
	Endpoint core.APIEndpoint
}

func (c SnapshotClient) Validate() error {
	if c.HTTP == nil || c.HTTP.Timeout != HTTPClientTimeout {
		return fmt.Errorf(ErrFmtSnapshotClient, ErrContract)
	}
	if err := c.Endpoint.Validate(); err != nil {
		return fmt.Errorf(ErrFmtSnapshotClient, errors.Join(ErrContract, err))
	}
	return nil
}

func (c SnapshotClient) Snapshot(ctx context.Context, project ProjectID) (ProjectSnapshot, error) {
	if err := c.Validate(); err != nil {
		return ProjectSnapshot{}, err
	}
	if err := project.Validate(); err != nil {
		return ProjectSnapshot{}, err
	}
	request := exchange.Request[core.HTTPNoBody]{
		Endpoint: c.Endpoint,
		Query: core.HTTPQuery{Parameters: []core.HTTPQueryParameter{{
			Name:  QueryProject,
			Value: project.String(),
		}}},
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySafe},
		ExpectedStatus: core.HTTPStatusOK,
	}
	response, err := exchange.SendJSON[core.HTTPNoBody, ProjectSnapshot](ctx, exchange.Client{HTTP: c.HTTP}, request, snapshotClientPolicy())
	if err != nil {
		return ProjectSnapshot{}, snapshotExchangeError(response, err)
	}
	snapshot := *response.Envelope.Data
	if snapshot.Project != project {
		return ProjectSnapshot{}, fmt.Errorf(ErrFmtSnapshotClient, ErrContract)
	}
	return snapshot, nil
}

func snapshotClientPolicy() exchange.ClientPolicy {
	return exchange.ClientPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(HTTPClientTimeout),
		RequestBodyLimit:  core.NewByteCount(1),
		ResponseBodyLimit: core.NewByteCount(core.StrictJSONMaxBytes),
		Retry:             core.DefaultHTTPRetryPolicy(),
		Redirect:          core.HTTPRedirectReject,
	}
}

func snapshotExchangeError(response exchange.Response[ProjectSnapshot], cause error) error {
	statusCode := response.Status.Int()
	if apiError, ok := errors.AsType[exchange.ResponseError](cause); ok {
		return HTTPError{StatusCode: apiError.Status.Int(), Code: apiError.Body.Code, Cause: errors.Join(ErrContract, cause)}
	}
	return HTTPError{StatusCode: statusCode, Cause: errors.Join(ErrContract, cause)}
}

type HTTPError struct {
	Cause      error
	StatusCode int
	Code       core.APICode
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("peachfuzz api status %d: %v", e.StatusCode, e.Unwrap())
}

func (e HTTPError) Unwrap() error {
	if e.Cause != nil {
		return errors.Join(ErrUnavailable, e.Cause)
	}
	return ErrUnavailable
}

var (
	_ core.Validatable = SnapshotClient{}
)
