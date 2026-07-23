package license

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
	CheckInRequestByteCap  = 128 << 10
	CheckInResponseByteCap = 128 << 10
	CheckInBudget          = 8 * time.Second
	CheckInMinInterval     = 24 * time.Hour
	WarnWindow             = 7 * 24 * time.Hour
	ClockSkewAllowance     = 60 * time.Minute
)

type CheckInAPIError struct {
	RequestID  core.APIRequestID
	Body       core.APIErrorBody
	RetryAfter time.Duration
	StatusCode int
}

func (e CheckInAPIError) Error() string {
	return fmt.Sprintf("license check-in rejected: status=%d code=%s message=%s", e.StatusCode, e.Body.Code, e.Body.Message)
}

func (e CheckInAPIError) Unwrap() error {
	return core.ErrLicenseContract
}

type CheckInHTTPError struct {
	Cause      error
	StatusCode int
}

func (e CheckInHTTPError) Error() string {
	return fmt.Sprintf("license check-in HTTP failure: status=%d: %v", e.StatusCode, e.Cause)
}

func (e CheckInHTTPError) Unwrap() error {
	return errors.Join(core.ErrLicenseContract, e.Cause)
}

type CheckInRetryExhaustedError struct {
	Cause      error
	RetryAfter time.Duration
}

func (e CheckInRetryExhaustedError) Error() string {
	return fmt.Sprintf("license check-in retry exhausted: retry_after=%s: %v", e.RetryAfter, e.Cause)
}

func (e CheckInRetryExhaustedError) Unwrap() error {
	return e.Cause
}

type CheckInPayload interface {
	core.HTTPIdempotentBody
	CheckInRequestNonce() CheckInNonce
}

type Client[P CheckInPayload, R CheckInResponseBody] struct {
	HTTP     *http.Client
	Endpoint core.APIEndpoint
	APIKey   APICallKey
	Keyring  core.SigningKeyring
	Backoff  core.BackoffPolicy
}

func (c Client[P, R]) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtCheckInClient, core.ErrLicenseContract)
	}
	if err := c.Endpoint.Validate(); err != nil {
		return fmt.Errorf(ErrFmtCheckInClient, errors.Join(core.ErrLicenseContract, err))
	}
	if !c.APIKey.IsZero() {
		if err := c.APIKey.Validate(); err != nil {
			return err
		}
	}
	if c.Backoff != (core.BackoffPolicy{}) {
		if err := c.Backoff.Validate(); err != nil {
			return err
		}
	}
	if err := c.Keyring.Validate(); err != nil {
		return fmt.Errorf(ErrFmtCheckInClient, errors.Join(core.ErrLicenseContract, err))
	}
	return nil
}

func (c Client[P, R]) Do(ctx context.Context, input P) (R, error) {
	var zero R
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtCheckInClient, errors.Join(core.ErrLicenseContract, core.ErrNilContext))
	}
	if err := c.Validate(); err != nil {
		return zero, err
	}
	if err := input.Validate(); err != nil {
		return zero, err
	}
	idempotencyKey, err := input.HTTPIdempotencyKey()
	if err != nil {
		return zero, transportError(err)
	}
	requestContext, cancel := context.WithTimeout(ctx, CheckInBudget)
	defer cancel()
	exchangeClient, err := exchange.NewClient(c.HTTP)
	if err != nil {
		return zero, transportError(err)
	}
	response, err := exchange.SendJSON[P, R](
		requestContext,
		exchangeClient,
		c.exchangeRequest(input, idempotencyKey),
		c.exchangePolicy(),
	)
	if err != nil {
		return zero, checkInExchangeError(response, err)
	}
	decoded := *response.Envelope.Data
	if err := decoded.Verify(c.Keyring); err != nil {
		return zero, err
	}
	if err := decoded.VerifyRequestNonce(input.CheckInRequestNonce()); err != nil {
		return zero, err
	}
	return decoded, nil
}

func (c Client[P, R]) exchangeRequest(input P, key core.HTTPIdempotencyKey) exchange.Request[P] {
	headers := core.HTTPHeaders{}
	if !c.APIKey.IsZero() {
		headers.Values = []core.HTTPHeader{{Name: OffgridAPICallKeyHeader, Value: c.APIKey.String()}}
	}
	return exchange.Request[P]{
		Body: &input, Endpoint: c.Endpoint, Headers: headers,
		Semantics: core.HTTPRequestSemantics{
			Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent, IdempotencyKey: key,
		},
		ExpectedStatus: core.HTTPStatusOK,
	}
}

func (c Client[P, R]) exchangePolicy() exchange.ClientPolicy {
	backoff := c.backoffPolicy()
	retry := core.DefaultHTTPRetryPolicy()
	retry.Backoff = backoff
	retry.RetryWaitLimit = backoff.Max
	return exchange.ClientPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(CheckInBudget),
		RequestBodyLimit:  core.NewByteCount(CheckInRequestByteCap),
		ResponseBodyLimit: core.NewByteCount(CheckInResponseByteCap),
		Retry:             retry,
		Redirect:          core.HTTPRedirectReject,
	}
}

func (c Client[P, R]) backoffPolicy() core.BackoffPolicy {
	if c.Backoff == (core.BackoffPolicy{}) {
		return core.DefaultHTTPBackoffPolicy()
	}
	return c.Backoff
}

func checkInExchangeError[R core.Validatable](response exchange.Response[R], cause error) error {
	if exhausted, ok := errors.AsType[exchange.RetryExhaustedError](cause); ok {
		return CheckInRetryExhaustedError{
			Cause:      checkInExchangeCause(response.Status, exhausted.Cause),
			RetryAfter: exhausted.RetryAfter.Duration(),
		}
	}
	return checkInExchangeCause(response.Status, cause)
}

func checkInExchangeCause(status core.HTTPStatusCode, cause error) error {
	if apiError, ok := errors.AsType[exchange.ResponseError](cause); ok {
		return CheckInAPIError{
			StatusCode: apiError.Status.Int(), RequestID: apiError.RequestID,
			Body: apiError.Body, RetryAfter: apiError.RetryAfter.Duration(),
		}
	}
	return CheckInHTTPError{StatusCode: status.Int(), Cause: cause}
}

func transportError(err error) error {
	return fmt.Errorf(ErrFmtCheckInTransport, errors.Join(core.ErrLicenseContract, err))
}
