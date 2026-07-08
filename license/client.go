package license

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

const (
	CheckInResponseByteCap = 1 << 16
	CheckInBudget          = 8 * time.Second
	CheckInMinInterval     = 24 * time.Hour
	WarnWindow             = 7 * 24 * time.Hour
	ClockSkewAllowance     = 60 * time.Minute
	CheckInMaxRetryAfter   = 24 * time.Hour
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

var CheckInBackoff = core.BackoffPolicy{
	Base:        100 * time.Millisecond,
	Max:         2 * time.Second,
	MaxAttempts: 4,
}

type CheckInPayload interface {
	Validate() error
}

type CheckInResponse[B Body] struct {
	Lease       *core.Signed[B] `json:"lease,omitempty"`
	Remediation string          `json:"remediation"`
	Refusal     Refusal         `json:"refusal"`
	Granted     bool            `json:"granted"`
}

func (r CheckInResponse[B]) Validate() error {
	if r.Granted {
		if r.Refusal != RefusalNone || r.Remediation != "" || r.Lease == nil {
			return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
		}
		return r.Lease.Validate()
	}
	if r.Lease != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	if err := r.Refusal.Validate(); err != nil || r.Refusal == RefusalNone {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	if strings.TrimSpace(r.Remediation) == "" {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	return nil
}

type Client[P CheckInPayload, B Body] struct {
	HTTP     *http.Client
	Jitter   func() float64
	Endpoint CheckInEndpoint
	APIKey   APICallKey
}

func (c Client[P, B]) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtCheckInClient, core.ErrLicenseContract)
	}
	if err := c.Endpoint.Validate(); err != nil {
		return err
	}
	if !c.APIKey.IsZero() {
		return c.APIKey.Validate()
	}
	return nil
}

func (c Client[P, B]) Do(ctx context.Context, in P) (CheckInResponse[B], error) {
	if err := c.Validate(); err != nil {
		return CheckInResponse[B]{}, err
	}
	if err := in.Validate(); err != nil {
		return CheckInResponse[B]{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, CheckInBudget)
	defer cancel()
	body, err := json.Marshal(in)
	if err != nil {
		return CheckInResponse[B]{}, fmt.Errorf(ErrFmtCheckInTransport, err)
	}
	return c.roundTrip(ctx, body)
}

func (c Client[P, B]) roundTrip(ctx context.Context, body []byte) (CheckInResponse[B], error) {
	policy := CheckInBackoff
	var lastErr error
	var retryAfter time.Duration
	var retryAfterHint time.Duration
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			wait := maxDuration(policy.Delay(attempt-1, c.jitterFraction()), retryAfter)
			if err := sleepContext(ctx, wait); err != nil {
				return CheckInResponse[B]{}, transportError(err)
			}
		}
		result := c.attempt(ctx, body)
		retryAfter = result.RetryAfter
		retryAfterHint = maxDuration(retryAfterHint, result.RetryAfterHint)
		switch result.Outcome {
		case core.HTTPOutcomeSuccess:
			if verr := result.Response.Validate(); verr != nil {
				return CheckInResponse[B]{}, verr
			}
			return result.Response, nil
		case core.HTTPOutcomeRetryable:
			lastErr = result.Err
		default:
			return CheckInResponse[B]{}, result.Err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract)
	}
	return CheckInResponse[B]{}, CheckInRetryExhaustedError{Cause: lastErr, RetryAfter: retryAfterHint}
}

type attemptResult[B Body] struct {
	Err            error
	Response       CheckInResponse[B]
	RetryAfter     time.Duration
	RetryAfterHint time.Duration
	Outcome        core.HTTPOutcome
}

func (c Client[P, B]) attempt(ctx context.Context, body []byte) attemptResult[B] {
	request, err := c.buildRequest(ctx, body)
	if err != nil {
		return attemptResult[B]{Outcome: core.HTTPOutcomeTerminal, Err: err}
	}
	reply, err := c.HTTP.Do(request)
	if err != nil {
		return attemptResult[B]{Outcome: core.HTTPOutcomeRetryable, Err: transportError(err)}
	}
	if reply == nil {
		return attemptResult[B]{
			Outcome: core.HTTPOutcomeTerminal,
			Err:     fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract),
		}
	}
	defer func() { _ = reply.Body.Close() }()
	outcome := core.ClassifyHTTPStatus(reply.StatusCode)
	if outcome != core.HTTPOutcomeSuccess {
		retryAfter := parseRetryAfter(reply.Header.Get(core.HTTPHeaderRetryAfter), time.Now().UTC())
		err = readFailureResponse[B](reply, reply.StatusCode, retryAfter.Hint)
		return attemptResult[B]{
			Outcome:        outcome,
			RetryAfter:     retryAfter.Wait,
			RetryAfterHint: retryAfter.Hint,
			Err:            err,
		}
	}
	decoded, err := readResponse[B](reply)
	if err != nil {
		return attemptResult[B]{Outcome: core.HTTPOutcomeTerminal, Err: err}
	}
	return attemptResult[B]{Response: decoded, Outcome: core.HTTPOutcomeSuccess}
}

func (c Client[P, B]) buildRequest(ctx context.Context, body []byte) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf(ErrFmtCheckInTransport, err)
	}
	request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	if !c.APIKey.IsZero() {
		request.Header.Set(OffgridAPICallKeyHeader, c.APIKey.String())
	}
	return request, nil
}

func readResponse[B Body](reply *http.Response) (CheckInResponse[B], error) {
	data, err := readCappedResponseBody(reply.Body)
	if err != nil {
		return CheckInResponse[B]{}, err
	}
	decoded, err := core.DecodeStrictJSON[core.APIEnvelope[CheckInResponse[B]]](data)
	if err != nil {
		return CheckInResponse[B]{}, fmt.Errorf(ErrFmtCheckInResponse, err)
	}
	if err := decoded.ValidateSuccess(); err != nil {
		return CheckInResponse[B]{}, fmt.Errorf(ErrFmtCheckInResponse, err)
	}
	if err := decoded.Data.Validate(); err != nil {
		return CheckInResponse[B]{}, err
	}
	return *decoded.Data, nil
}

func readFailureResponse[B Body](reply *http.Response, statusCode int, retryAfter time.Duration) error {
	data, err := readCappedResponseBody(reply.Body)
	if err != nil {
		return err
	}
	decoded, err := core.DecodeStrictJSON[core.APIEnvelope[CheckInResponse[B]]](data)
	if err != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, err)
	}
	if err := decoded.ValidateFailure(); err != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, err)
	}
	return CheckInAPIError{
		StatusCode: statusCode,
		RequestID:  decoded.RequestID,
		Body:       *decoded.Error,
		RetryAfter: retryAfter,
	}
}

func readCappedResponseBody(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, CheckInResponseByteCap+1))
	if err != nil {
		return nil, fmt.Errorf(ErrFmtCheckInTransport, err)
	}
	if len(data) > CheckInResponseByteCap {
		return nil, fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	return data, nil
}

func (c Client[P, B]) jitterFraction() float64 {
	if c.Jitter != nil {
		return conservativeJitterFraction(c.Jitter())
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 1
	}
	return conservativeJitterFraction(float64(binary.BigEndian.Uint64(raw[:])>>11) / float64(uint64(1)<<53))
}

func conservativeJitterFraction(value float64) float64 {
	if !(value > 0) {
		return 1
	}
	if value > 1 {
		return 1
	}
	return value
}

func transportError(err error) error {
	return fmt.Errorf(ErrFmtCheckInTransport, err)
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type retryAfterDecision struct {
	Wait time.Duration
	Hint time.Duration
}

func parseRetryAfter(header string, now time.Time) retryAfterDecision {
	header = strings.TrimSpace(header)
	if header == "" {
		return retryAfterDecision{}
	}
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return retryAfterSeconds(seconds)
	}
	if _, err := strconv.ParseUint(header, 10, 64); err == nil {
		return retryAfterDecision{Wait: CheckInBackoff.Max, Hint: CheckInMaxRetryAfter}
	}
	when, err := http.ParseTime(header)
	if err != nil || !when.After(now) {
		return retryAfterDecision{}
	}
	return clampRetryAfter(when.Sub(now))
}

func clampRetryAfter(d time.Duration) retryAfterDecision {
	if d > CheckInMaxRetryAfter {
		d = CheckInMaxRetryAfter
	}
	return retryAfterDecision{Wait: minDuration(d, CheckInBackoff.Max), Hint: d}
}

func retryAfterSeconds(seconds int64) retryAfterDecision {
	if seconds <= 0 {
		return retryAfterDecision{}
	}
	if seconds > int64(CheckInMaxRetryAfter/time.Second) {
		return retryAfterDecision{Wait: CheckInBackoff.Max, Hint: CheckInMaxRetryAfter}
	}
	d := time.Duration(seconds) * time.Second
	return retryAfterDecision{Wait: minDuration(d, CheckInBackoff.Max), Hint: d}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
