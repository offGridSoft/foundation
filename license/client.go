package license

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
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

const (
	checkInBackoffBase        = 100 * time.Millisecond
	checkInBackoffMax         = 2 * time.Second
	checkInBackoffMaxAttempts = 4
)

func DefaultCheckInBackoff() core.BackoffPolicy {
	return core.BackoffPolicy{
		Base:        checkInBackoffBase,
		Max:         checkInBackoffMax,
		MaxAttempts: checkInBackoffMaxAttempts,
	}
}

type CheckInPayload interface {
	core.Validatable
}

type CheckInResponse[G CheckInGrant] struct {
	Grant       *G          `json:"grant,omitempty"`
	Remediation Remediation `json:"remediation"`
	Refusal     Refusal     `json:"refusal"`
	Granted     bool        `json:"granted"`
}

// APIBody lets Foundation response contracts satisfy lfw/api.Body
// structurally without reversing the dependency into the transport package.
func (CheckInResponse[G]) APIBody() {}

func (r CheckInResponse[G]) Validate() error {
	if r.Granted {
		return r.validateGranted()
	}
	return r.validateRefused()
}

func (r CheckInResponse[G]) validateGranted() error {
	if r.Refusal != RefusalNone || r.Remediation != RemediationNone || r.Grant == nil {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	if err := (*r.Grant).Validate(); err != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
	}
	return nil
}

func (r CheckInResponse[G]) validateRefused() error {
	if r.Grant != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	if err := r.Refusal.Validate(); err != nil || r.Refusal == RefusalNone {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	want, err := RemediationForRefusal(r.Refusal)
	if err != nil || r.Remediation != want {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	return nil
}

type Client[P CheckInPayload, G CheckInGrant] struct {
	HTTP     *http.Client
	Jitter   func() float64
	Endpoint CheckInEndpoint
	APIKey   APICallKey
	Keyring  core.SigningKeyring
	Backoff  core.BackoffPolicy
}

func (c Client[P, G]) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtCheckInClient, core.ErrLicenseContract)
	}
	if err := c.Endpoint.Validate(); err != nil {
		return err
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

func (c Client[P, G]) Do(ctx context.Context, in P) (CheckInResponse[G], error) {
	if ctx == nil {
		return CheckInResponse[G]{}, fmt.Errorf(ErrFmtCheckInClient, errors.Join(core.ErrLicenseContract, core.ErrNilContext))
	}
	if err := c.Validate(); err != nil {
		return CheckInResponse[G]{}, err
	}
	if err := in.Validate(); err != nil {
		return CheckInResponse[G]{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, CheckInBudget)
	defer cancel()
	body, err := json.Marshal(in)
	if err != nil {
		return CheckInResponse[G]{}, transportError(err)
	}
	return c.roundTrip(ctx, body)
}

func (c Client[P, G]) roundTrip(ctx context.Context, body []byte) (CheckInResponse[G], error) {
	policy := c.backoffPolicy()
	var lastErr error
	var retryAfter time.Duration
	var retryAfterHint time.Duration
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay, err := policy.Delay(attempt-1, c.jitterFraction())
			if err != nil {
				return CheckInResponse[G]{}, transportError(err)
			}
			wait := maxDuration(delay, retryAfter)
			if err := sleepContext(ctx, wait); err != nil {
				return CheckInResponse[G]{}, transportError(err)
			}
		}
		result := c.attempt(ctx, body, policy.Max)
		retryAfter = result.RetryAfter
		retryAfterHint = maxDuration(retryAfterHint, result.RetryAfterHint)
		switch result.Outcome {
		case core.HTTPOutcomeSuccess:
			if verr := result.Response.Validate(); verr != nil {
				return CheckInResponse[G]{}, verr
			}
			return result.Response, nil
		case core.HTTPOutcomeRetryable:
			lastErr = result.Err
		default:
			return CheckInResponse[G]{}, result.Err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract)
	}
	return CheckInResponse[G]{}, CheckInRetryExhaustedError{Cause: lastErr, RetryAfter: retryAfterHint}
}

type attemptResult[G CheckInGrant] struct {
	Err            error
	Response       CheckInResponse[G]
	RetryAfter     time.Duration
	RetryAfterHint time.Duration
	Outcome        core.HTTPOutcome
}

func (c Client[P, G]) attempt(ctx context.Context, body []byte, maxRetryAfterWait time.Duration) attemptResult[G] {
	request, err := c.buildRequest(ctx, body)
	if err != nil {
		return attemptResult[G]{Outcome: core.HTTPOutcomeTerminal, Err: err}
	}
	reply, err := c.do(request)
	if err != nil {
		return attemptResult[G]{Outcome: core.HTTPOutcomeRetryable, Err: transportError(err)}
	}
	if reply == nil || reply.Body == nil {
		return attemptResult[G]{
			Outcome: core.HTTPOutcomeTerminal,
			Err:     fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract),
		}
	}
	defer func() {
		_ = reply.Body.Close() // Safe: the bounded body read owns the meaningful transport result.
	}()
	outcome := core.ClassifyHTTPStatus(reply.StatusCode)
	if outcome != core.HTTPOutcomeSuccess {
		retryAfter := parseRetryAfter(reply.Header.Get(core.HTTPHeaderRetryAfter), time.Now().UTC(), maxRetryAfterWait)
		err = readFailureResponse[G](reply, reply.StatusCode, retryAfter.Hint)
		return attemptResult[G]{
			Outcome:        outcome,
			RetryAfter:     retryAfter.Wait,
			RetryAfterHint: retryAfter.Hint,
			Err:            err,
		}
	}
	decoded, err := readResponse[G](reply, c.Keyring)
	if err != nil {
		return attemptResult[G]{Outcome: core.HTTPOutcomeTerminal, Err: err}
	}
	return attemptResult[G]{Response: decoded, Outcome: core.HTTPOutcomeSuccess}
}

func (c Client[P, G]) do(request *http.Request) (*http.Response, error) {
	client := *c.HTTP
	client.CheckRedirect = rejectCheckInRedirect
	return client.Do(request) // #nosec G704 -- request URL comes from typed CheckInEndpoint validation and redirects are rejected.
}

func rejectCheckInRedirect(*http.Request, []*http.Request) error {
	return fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract)
}

func (c Client[P, G]) buildRequest(ctx context.Context, body []byte) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, transportError(err)
	}
	request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	if !c.APIKey.IsZero() {
		request.Header.Set(OffgridAPICallKeyHeader, c.APIKey.String())
	}
	return request, nil
}

func readResponse[G CheckInGrant](reply *http.Response, keyring core.SigningKeyring) (CheckInResponse[G], error) {
	data, err := readCappedResponseBody(reply.Body)
	if err != nil {
		return CheckInResponse[G]{}, err
	}
	decoded, err := core.DecodeStrictJSON[core.APIEnvelope[CheckInResponse[G]]](data)
	if err != nil {
		return CheckInResponse[G]{}, fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
	}
	if err := decoded.ValidateSuccess(); err != nil {
		return CheckInResponse[G]{}, fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
	}
	if err := verifyGrantedGrant(*decoded.Data, keyring); err != nil {
		return CheckInResponse[G]{}, err
	}
	return *decoded.Data, nil
}

func verifyGrantedGrant[G CheckInGrant](response CheckInResponse[G], keyring core.SigningKeyring) error {
	if !response.Granted {
		return nil
	}
	if response.Grant == nil {
		return fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	if err := (*response.Grant).Verify(keyring); err != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
	}
	return nil
}

func readFailureResponse[G CheckInGrant](reply *http.Response, statusCode int, retryAfter time.Duration) error {
	data, err := readCappedResponseBody(reply.Body)
	if err != nil {
		return err
	}
	decoded, err := core.DecodeStrictJSON[core.APIEnvelope[CheckInResponse[G]]](data)
	if err != nil {
		return CheckInHTTPError{
			StatusCode: statusCode,
			Cause:      fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err)),
		}
	}
	if err := decoded.ValidateFailure(); err != nil {
		return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
	}
	return CheckInAPIError{
		StatusCode: statusCode,
		RequestID:  decoded.RequestID,
		Body:       *decoded.Error,
		RetryAfter: retryAfter,
	}
}

func readCappedResponseBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	data, err := io.ReadAll(io.LimitReader(body, CheckInResponseByteCap+1))
	if err != nil {
		return nil, transportError(err)
	}
	if len(data) > CheckInResponseByteCap {
		return nil, fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	return data, nil
}

func (c Client[P, G]) jitterFraction() float64 {
	if c.Jitter != nil {
		return conservativeJitterFraction(c.Jitter())
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 1
	}
	return conservativeJitterFraction(float64(binary.BigEndian.Uint64(raw[:])>>11) / float64(uint64(1)<<53))
}

func (c Client[P, B]) backoffPolicy() core.BackoffPolicy {
	if c.Backoff == (core.BackoffPolicy{}) {
		return DefaultCheckInBackoff()
	}
	return c.Backoff
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
	return fmt.Errorf(ErrFmtCheckInTransport, errors.Join(core.ErrLicenseContract, err))
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

func parseRetryAfter(header string, now time.Time, maxWait time.Duration) retryAfterDecision {
	header = strings.TrimSpace(header)
	if header == "" {
		return retryAfterDecision{}
	}
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return retryAfterSeconds(seconds, maxWait)
	}
	if _, err := strconv.ParseUint(header, 10, 64); err == nil {
		return retryAfterDecision{Wait: maxWait, Hint: CheckInMaxRetryAfter}
	}
	if allDecimalDigits(header) {
		return retryAfterDecision{Wait: maxWait, Hint: CheckInMaxRetryAfter}
	}
	when, err := http.ParseTime(header)
	if err != nil || !when.After(now) {
		return retryAfterDecision{}
	}
	return clampRetryAfter(when.Sub(now), maxWait)
}

func allDecimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func clampRetryAfter(d time.Duration, maxWait time.Duration) retryAfterDecision {
	if d > CheckInMaxRetryAfter {
		d = CheckInMaxRetryAfter
	}
	return retryAfterDecision{Wait: minDuration(d, maxWait), Hint: d}
}

func retryAfterSeconds(seconds int64, maxWait time.Duration) retryAfterDecision {
	if seconds <= 0 {
		return retryAfterDecision{}
	}
	if seconds > int64(CheckInMaxRetryAfter/time.Second) {
		return retryAfterDecision{Wait: maxWait, Hint: CheckInMaxRetryAfter}
	}
	d := time.Duration(seconds) * time.Second
	return retryAfterDecision{Wait: minDuration(d, maxWait), Hint: d}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
