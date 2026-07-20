package peachfuzz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const HTTPClientTimeout = 15 * time.Second

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
	endpoint := c.Endpoint.String() + "?" + QueryProject + "=" + url.QueryEscape(project.String())
	snapshot, err := executePublic[ProjectSnapshot](ctx, c.HTTP, http.MethodGet, endpoint, nil)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snapshot.Project != project {
		return ProjectSnapshot{}, fmt.Errorf(ErrFmtSnapshotClient, ErrContract)
	}
	return snapshot, nil
}

func executePublic[T core.APIBody](ctx context.Context, client *http.Client, method, endpoint string, body []byte) (T, error) {
	var zero T
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtSnapshotClient, errors.Join(ErrContract, core.ErrNilContext))
	}
	request, err := newRequest(ctx, method, endpoint, body)
	if err != nil {
		return zero, HTTPError{Cause: err}
	}
	return executeRequest[T](client, request)
}

func executeRequest[T core.APIBody](client *http.Client, request *http.Request) (T, error) {
	var zero T
	httpClient := *client
	httpClient.CheckRedirect = refuseRedirect
	response, err := httpClient.Do(request)
	if err != nil {
		return zero, HTTPError{Cause: err}
	}
	return decodeResponse[T](response)
}

func newRequest(ctx context.Context, method, endpoint string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set(core.HTTPHeaderAccept, core.HTTPContentTypeJSON)
	if body != nil {
		request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	}
	return request, nil
}

func decodeResponse[T core.APIBody](response *http.Response) (T, error) {
	var zero T
	if response == nil || response.Body == nil {
		return zero, HTTPError{Cause: ErrContract}
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(body) == 0 || len(body) > core.StrictJSONMaxBytes {
		return zero, HTTPError{StatusCode: response.StatusCode, Cause: ErrContract}
	}
	envelope, err := core.DecodeStrictJSON[core.APIEnvelope[T]](body)
	if err != nil {
		return zero, HTTPError{StatusCode: response.StatusCode, Cause: errors.Join(ErrContract, err)}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return zero, failureFrom(response.StatusCode, envelope)
	}
	if err := envelope.ValidateSuccess(); err != nil {
		return zero, HTTPError{StatusCode: response.StatusCode, Cause: err}
	}
	return *envelope.Data, nil
}

func failureFrom[T core.APIBody](statusCode int, envelope core.APIEnvelope[T]) error {
	if err := envelope.ValidateFailure(); err != nil {
		return HTTPError{StatusCode: statusCode, Cause: err}
	}
	return HTTPError{StatusCode: statusCode, Code: envelope.Error.Code}
}

func refuseRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

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
