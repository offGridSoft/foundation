package exchange

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const redirectMaximumHops = 10

type Clock interface {
	Now() core.UnixNanoTime
}

type JitterSource interface {
	Fraction() float64
}

type Waiter interface {
	Wait(context.Context, core.NanosecondsDuration) error
}

type Client struct {
	HTTP   *http.Client
	Clock  Clock
	Jitter JitterSource
	Waiter Waiter
}

func (c Client) Validate() error {
	if c.HTTP == nil {
		return responseError(core.ErrExchangeContract)
	}
	return nil
}

type ClientPolicy struct {
	AttemptTimeout    core.NanosecondsDuration
	RequestBodyLimit  core.ByteCount
	ResponseBodyLimit core.ByteCount
	Retry             core.HTTPRetryPolicy
	Redirect          core.HTTPRedirectPolicy
}

func (p ClientPolicy) Validate() error {
	if err := validatePositiveDuration(p.AttemptTimeout); err != nil {
		return responseError(err)
	}
	if err := validateBodyLimit(p.RequestBodyLimit); err != nil {
		return responseError(err)
	}
	if err := validateBodyLimit(p.ResponseBodyLimit); err != nil {
		return responseError(err)
	}
	if err := p.Retry.Validate(); err != nil {
		return responseError(err)
	}
	if err := p.Redirect.Validate(); err != nil {
		return responseError(err)
	}
	return nil
}

func validatePositiveDuration(duration core.NanosecondsDuration) error {
	if err := duration.Validate(); err != nil {
		return err
	}
	if duration.IsZero() || duration.Duration() > core.BackoffMaxDuration {
		return core.ErrExchangeContract
	}
	return nil
}

func validateBodyLimit(limit core.ByteCount) error {
	if err := limit.Validate(); err != nil {
		return err
	}
	if limit.Uint64() > core.StrictJSONMaxBytes {
		return core.ErrExchangeBodyLimit
	}
	return nil
}

type Request[T core.Validatable] struct {
	Body           *T
	Semantics      core.HTTPRequestSemantics
	Endpoint       core.APIEndpoint
	Headers        core.HTTPHeaders
	Query          core.HTTPQuery
	ExpectedStatus core.HTTPStatusCode
}

func (r Request[T]) Validate() error {
	if err := r.validateMetadata(); err != nil {
		return err
	}
	return r.validateBody()
}

func (r Request[T]) validateMetadata() error {
	if err := r.Endpoint.Validate(); err != nil {
		return requestError(err)
	}
	if err := r.Semantics.Validate(); err != nil {
		return requestError(err)
	}
	if !validExpectedStatus(r.ExpectedStatus) {
		return requestError(core.ErrExchangeContract)
	}
	if err := r.Headers.Validate(); err != nil {
		return requestError(err)
	}
	if err := r.Query.Validate(); err != nil {
		return requestError(err)
	}
	if jsonManagedHeaderPresent(r.Headers) {
		return requestError(core.ErrExchangeContract)
	}
	return nil
}

func jsonManagedHeaderPresent(headers core.HTTPHeaders) bool {
	return managedRequestHeaderPresent(headers) ||
		headers.Contains(core.HTTPHeaderContentType) ||
		headers.Contains(core.HTTPHeaderIdempotencyKey)
}

func (r Request[T]) validateBody() error {
	requiresBody := r.Semantics.Method == core.HTTPMethodPost || r.Semantics.Method == core.HTTPMethodPut || r.Semantics.Method == core.HTTPMethodPatch
	if requiresBody != (r.Body != nil) {
		return requestError(core.ErrExchangeContract)
	}
	if r.Body != nil {
		if err := (*r.Body).Validate(); err != nil {
			return requestError(err)
		}
	}
	return nil
}

type Response[T core.Validatable] struct {
	Envelope core.APIEnvelope[T]
	Status   core.HTTPStatusCode
	Attempts uint64
}

func (r Response[T]) Validate() error {
	if r.Attempts < 1 {
		return responseError(core.ErrExchangeContract)
	}
	if err := r.Status.Validate(); err != nil {
		return responseError(err)
	}
	if err := r.Envelope.Validate(); err != nil {
		return responseError(err)
	}
	return nil
}

type ResponseError struct {
	RequestID  core.APIRequestID
	Body       core.APIErrorBody
	RetryAfter core.NanosecondsDuration
	Status     core.HTTPStatusCode
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("exchange response rejected: status=%d code=%s", e.Status.Int(), e.Body.Code.String())
}

func (e ResponseError) Unwrap() error {
	return core.ErrExchangeResponse
}

type RetryExhaustedError struct {
	Cause      error
	Attempts   uint64
	RetryAfter core.NanosecondsDuration
}

func (e RetryExhaustedError) Error() string {
	return fmt.Sprintf("exchange retry budget exhausted after %d attempts: %v", e.Attempts, e.Cause)
}

func (e RetryExhaustedError) Unwrap() error {
	return errors.Join(core.ErrExchangeRetryExhausted, e.Cause)
}

type attemptResult[T core.Validatable] struct {
	Err            error
	Response       Response[T]
	RetryAfter     core.NanosecondsDuration
	RetryAfterHint core.NanosecondsDuration
	Retryable      bool
}

func SendJSON[RequestBody, ResponseBody core.Validatable](ctx context.Context, client Client, request Request[RequestBody], policy ClientPolicy) (Response[ResponseBody], error) {
	var zero Response[ResponseBody]
	if err := validateClientContext(ctx); err != nil {
		return zero, err
	}
	if err := client.Validate(); err != nil {
		return zero, err
	}
	if err := policy.Validate(); err != nil {
		return zero, err
	}
	if err := request.Validate(); err != nil {
		return zero, err
	}
	body, err := encodeRequestBody(request.Body, policy.RequestBodyLimit)
	if err != nil {
		return zero, err
	}
	return sendAttempts[RequestBody, ResponseBody](ctx, client, request, policy, body)
}

func sendAttempts[RequestBody, ResponseBody core.Validatable](ctx context.Context, client Client, request Request[RequestBody], policy ClientPolicy, body []byte) (Response[ResponseBody], error) {
	var last attemptResult[ResponseBody]
	var retryAfterHint core.NanosecondsDuration
	for attempt := uint64(0); attempt < policy.Retry.Backoff.MaxAttempts; attempt++ {
		if attempt > 0 {
			if err := waitBeforeRetry(ctx, client, policy, attempt-1, last.RetryAfter); err != nil {
				return last.Response, err
			}
		}
		input := sendAttemptInput[RequestBody]{
			Parent: ctx, Client: client, Request: request, Policy: policy,
			Body: body, Attempt: attempt + 1,
		}
		last = sendAttempt[RequestBody, ResponseBody](input)
		if last.RetryAfterHint.Duration() > retryAfterHint.Duration() {
			retryAfterHint = last.RetryAfterHint
		}
		replayable, replayErr := request.Semantics.Replay.AllowsAutomaticReplay()
		if replayErr != nil {
			return last.Response, requestError(replayErr)
		}
		if last.Err == nil || !last.Retryable || !replayable {
			return last.Response, last.Err
		}
	}
	return last.Response, RetryExhaustedError{
		Cause: last.Err, Attempts: policy.Retry.Backoff.MaxAttempts, RetryAfter: retryAfterHint,
	}
}

func encodeRequestBody[T core.Validatable](body *T, limit core.ByteCount) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	encoded, err := core.EncodeValidatedJSON(*body)
	if err != nil {
		return nil, requestError(err)
	}
	if uint64(len(encoded)) > limit.Uint64() {
		return nil, errors.Join(core.ErrExchangeRequest, core.ErrExchangeBodyLimit)
	}
	return encoded, nil
}

type sendAttemptInput[RequestBody core.Validatable] struct {
	Client  Client
	Parent  context.Context
	Body    []byte
	Request Request[RequestBody]
	Policy  ClientPolicy
	Attempt uint64
}

func sendAttempt[RequestBody, ResponseBody core.Validatable](input sendAttemptInput[RequestBody]) attemptResult[ResponseBody] {
	attemptContext, cancel := context.WithTimeout(input.Parent, input.Policy.AttemptTimeout.Duration())
	defer cancel()
	httpRequest, err := buildHTTPRequest(attemptContext, input.Request, input.Body)
	if err != nil {
		return attemptResult[ResponseBody]{Err: requestError(err)}
	}
	httpClient := configuredHTTPClient(input.Client.HTTP, input.Policy.Redirect)
	httpResponse, err := httpClient.Do(httpRequest) // #nosec G704 -- endpoint and redirect destinations are validated typed contracts.
	if err != nil {
		return classifyHTTPDoError[ResponseBody](input.Parent, httpResponse, err, input.Attempt)
	}
	readInput := responseReadInput{
		Context: attemptContext, Client: input.Client, Response: httpResponse,
		Expected: input.Request.ExpectedStatus, Policy: input.Policy, Attempt: input.Attempt,
	}
	return readHTTPResponse[ResponseBody](readInput)
}

func buildHTTPRequest[T core.Validatable](ctx context.Context, request Request[T], body []byte) (*http.Request, error) {
	var reader io.Reader
	if request.Body != nil {
		reader = bytes.NewReader(body)
	}
	target, err := requestTarget(request.Endpoint, request.Query)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, request.Semantics.Method.String(), target, reader)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set(core.HTTPHeaderAccept, core.HTTPContentTypeJSON)
	httpRequest.Header.Set(core.HTTPHeaderAcceptEncoding, core.HTTPContentEncodingIdentity)
	if request.Body != nil {
		httpRequest.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	}
	if !request.Semantics.IdempotencyKey.IsZero() {
		httpRequest.Header.Set(core.HTTPHeaderIdempotencyKey, request.Semantics.IdempotencyKey.String())
	}
	for _, header := range request.Headers.Values {
		httpRequest.Header.Set(header.Name, header.Value)
	}
	return httpRequest, nil
}

func requestTarget(endpoint core.APIEndpoint, query core.HTTPQuery) (string, error) {
	encoded, err := query.Encode()
	if err != nil {
		return "", err
	}
	if encoded == "" {
		return endpoint.String(), nil
	}
	return endpoint.String() + "?" + encoded, nil
}

func configuredHTTPClient(source *http.Client, policy core.HTTPRedirectPolicy) *http.Client {
	client := *source
	client.CheckRedirect = func(next *http.Request, prior []*http.Request) error {
		if policy == core.HTTPRedirectReject || len(prior) == 0 || len(prior) >= redirectMaximumHops {
			return core.ErrExchangeRedirect
		}
		origin := prior[0]
		if !sameOrigin(origin.URL, next.URL) || origin.Method != next.Method {
			return core.ErrExchangeRedirect
		}
		return nil
	}
	return &client
}

func sameOrigin(left, right *url.URL) bool {
	return left != nil && right != nil && left.Scheme == right.Scheme && left.Host == right.Host
}

func classifyHTTPDoError[T core.Validatable](parent context.Context, response *http.Response, err error, attempt uint64) attemptResult[T] {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if parent.Err() != nil {
		return attemptResult[T]{Response: Response[T]{Attempts: attempt}, Err: errors.Join(core.ErrExchangeCancelled, err)}
	}
	if errors.Is(err, core.ErrExchangeRedirect) {
		return attemptResult[T]{Response: Response[T]{Attempts: attempt}, Err: errors.Join(core.ErrExchangeResponse, core.ErrExchangeRedirect, err)}
	}
	return attemptResult[T]{Response: Response[T]{Attempts: attempt}, Err: errors.Join(core.ErrExchangeTransport, err), Retryable: true}
}

type responseReadInput struct {
	Context  context.Context
	Client   Client
	Response *http.Response
	Expected core.HTTPStatusCode
	Policy   ClientPolicy
	Attempt  uint64
}

func readHTTPResponse[T core.Validatable](input responseReadInput) attemptResult[T] {
	result := attemptResult[T]{Response: Response[T]{Attempts: input.Attempt}}
	if input.Response == nil || input.Response.Body == nil {
		result.Err = responseError(core.ErrExchangeContract)
		return result
	}
	result = readHTTPResponseBody[T](input, result)
	return closeAttemptResponse(input.Response.Body, result)
}

func readHTTPResponseBody[T core.Validatable](input responseReadInput, result attemptResult[T]) attemptResult[T] {
	status, err := core.NewHTTPStatusCode(input.Response.StatusCode)
	if err != nil {
		result.Err = responseError(err)
		return result
	}
	result.Response.Status = status
	if err := validateResponseMetadata(input.Response); err != nil {
		result.Err = err
		result.Retryable = retryableStatus(status)
		return result
	}
	limit, err := input.Policy.ResponseBodyLimit.Int64()
	if err != nil {
		result.Err = responseError(err)
		return result
	}
	if input.Response.ContentLength > limit {
		result.Err = errors.Join(core.ErrExchangeResponse, core.ErrExchangeBodyLimit)
		return result
	}
	body, err := readBounded(input.Context, input.Response.Body, limit, bodyReadResponse)
	if err != nil {
		result.Err = err
		result.Retryable = !errors.Is(err, core.ErrExchangeBodyLimit)
		return result
	}
	envelope, err := core.DecodeStrictJSON[core.APIEnvelope[T]](body)
	if err != nil {
		result.Err = responseError(err)
		result.Retryable = retryableStatus(status)
		return result
	}
	result.Response.Envelope = envelope
	outcome, err := core.ClassifyHTTPStatus(status, input.Expected)
	if err != nil {
		result.Err = responseError(err)
		return result
	}
	decoded := decodedResponseInput[T]{
		Client: input.Client, Response: input.Response, Policy: input.Policy,
		Result: result, Outcome: outcome,
	}
	return classifyDecodedResponse(decoded)
}

func closeAttemptResponse[T core.Validatable](body io.Closer, result attemptResult[T]) attemptResult[T] {
	closeErr := body.Close()
	if closeErr != nil && result.Err == nil {
		result.Err = errors.Join(core.ErrExchangeResponse, core.ErrExchangeTransport, closeErr)
		result.Retryable = true
	}
	return result
}

// retryableStatus widens retry eligibility for responses that fail protocol
// validation: a retryable status carrying a non-protocol body is an
// intermediary (load balancer, proxy) speaking for an unhealthy origin, and
// giving up on it defeats the retry policy. Conformant envelopes that
// contradict their status remain terminal.
func retryableStatus(status core.HTTPStatusCode) bool {
	retryable, err := core.HTTPStatusIsRetryable(status)
	return err == nil && retryable
}

func validateResponseMetadata(response *http.Response) error {
	contentTypes := response.Header.Values(core.HTTPHeaderContentType)
	if len(contentTypes) != 1 || contentTypes[0] != core.HTTPContentTypeJSON {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeContentType)
	}
	if len(response.Header.Values(core.HTTPHeaderContentEncoding)) != 0 {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeContentType)
	}
	return nil
}

type decodedResponseInput[T core.Validatable] struct {
	Client   Client
	Response *http.Response
	Result   attemptResult[T]
	Policy   ClientPolicy
	Outcome  core.HTTPOutcome
}

func classifyDecodedResponse[T core.Validatable](input decodedResponseInput[T]) attemptResult[T] {
	result := input.Result
	switch input.Outcome {
	case core.HTTPOutcomeSuccess:
		if err := result.Response.Envelope.ValidateSuccess(); err != nil {
			result.Err = responseError(err)
			return result
		}
		if err := result.Response.Validate(); err != nil {
			result.Err = err
		}
		return result
	case core.HTTPOutcomeRetryable:
		if err := result.Response.Envelope.ValidateFailure(); err != nil {
			result.Err = responseError(err)
			return result
		}
		retryAfterHint, err := parseRetryAfter(input.Response.Header.Values(core.HTTPHeaderRetryAfter), clientClock(input.Client).Now(), input.Policy.Retry.MaximumRetryAfter)
		if err != nil {
			result.Err = err
			return result
		}
		result.RetryAfterHint = retryAfterHint
		result.RetryAfter = minimumDuration(retryAfterHint, input.Policy.Retry.RetryWaitLimit)
		result.Retryable = true
		result.Err = responseAPIError(result.Response, retryAfterHint)
		return result
	default:
		if err := result.Response.Envelope.ValidateFailure(); err != nil {
			result.Err = responseError(err)
			return result
		}
		result.Err = responseAPIError(result.Response, core.NanosecondsDuration{})
		return result
	}
}

func responseAPIError[T core.Validatable](response Response[T], retryAfter core.NanosecondsDuration) error {
	if response.Envelope.Error == nil {
		return responseError(core.ErrExchangeContract)
	}
	return ResponseError{
		Status: response.Status, RequestID: response.Envelope.RequestID,
		Body: *response.Envelope.Error, RetryAfter: retryAfter,
	}
}

func minimumDuration(left, right core.NanosecondsDuration) core.NanosecondsDuration {
	if left.Duration() < right.Duration() {
		return left
	}
	return right
}

func waitBeforeRetry(ctx context.Context, client Client, policy ClientPolicy, failedAttempt uint64, retryAfter core.NanosecondsDuration) error {
	delay, err := policy.Retry.Backoff.Delay(failedAttempt, conservativeJitterFraction(clientJitter(client).Fraction()))
	if err != nil {
		return responseError(err)
	}
	if retryAfter.Duration() > delay.Duration() {
		delay = retryAfter
	}
	if err := clientWaiter(client).Wait(ctx, delay); err != nil {
		return errors.Join(core.ErrExchangeCancelled, err)
	}
	return nil
}

func conservativeJitterFraction(value float64) float64 {
	if !(value > 0) || value > 1 {
		return 1
	}
	return value
}

func parseRetryAfter(values []string, now core.UnixNanoTime, maximum core.NanosecondsDuration) (core.NanosecondsDuration, error) {
	if err := validatePositiveDuration(maximum); err != nil {
		return core.NanosecondsDuration{}, responseError(err)
	}
	if len(values) == 0 {
		return core.NanosecondsDuration{}, nil
	}
	if len(values) != 1 || values[0] == "" {
		return core.NanosecondsDuration{}, responseError(core.ErrExchangeContract)
	}
	if seconds, err := parseRetryAfterSeconds(values[0]); err == nil {
		return clampRetryAfterSeconds(seconds, maximum), nil
	}
	parsed, err := time.Parse(http.TimeFormat, values[0])
	if err != nil {
		return core.NanosecondsDuration{}, responseError(err)
	}
	if err := now.Validate(); err != nil {
		return core.NanosecondsDuration{}, responseError(err)
	}
	delay := parsed.Sub(now.Time())
	if delay <= 0 {
		return core.NanosecondsDuration{}, nil
	}
	if delay > maximum.Duration() {
		delay = maximum.Duration()
	}
	return core.NewNanosecondsDuration(delay), nil
}

func parseRetryAfterSeconds(value string) (int64, error) {
	if value == "" {
		return 0, core.ErrExchangeContract
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, core.ErrExchangeContract
		}
	}
	return strconv.ParseInt(value, 10, 64)
}

func clampRetryAfterSeconds(seconds int64, maximum core.NanosecondsDuration) core.NanosecondsDuration {
	maximumUnits := int64(maximum.Duration() / time.Second)
	if maximum.Duration()%time.Second != 0 {
		maximumUnits++
	}
	if seconds > maximumUnits {
		return maximum
	}
	delay := time.Duration(seconds) * time.Second
	if delay > maximum.Duration() {
		return maximum
	}
	return core.NewNanosecondsDuration(delay)
}

func validateClientContext(ctx context.Context) error {
	if ctx == nil {
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeCancelled, core.ErrNilContext)
	}
	select {
	case <-ctx.Done():
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeCancelled, context.Cause(ctx))
	default:
		return nil
	}
}

type systemClock struct{}

func (systemClock) Now() core.UnixNanoTime {
	return core.NewUnixNanoTime(time.Now().UTC())
}

type cryptoJitter struct{}

func (cryptoJitter) Fraction() float64 {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 1
	}
	return float64(binary.BigEndian.Uint64(raw[:])>>11) / float64(uint64(1)<<53)
}

type timerWaiter struct{}

func (timerWaiter) Wait(ctx context.Context, delay core.NanosecondsDuration) error {
	timer := time.NewTimer(delay.Duration())
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return nil
	}
}

func clientClock(client Client) Clock {
	if client.Clock != nil {
		return client.Clock
	}
	return systemClock{}
}

func clientJitter(client Client) JitterSource {
	if client.Jitter != nil {
		return client.Jitter
	}
	return cryptoJitter{}
}

func clientWaiter(client Client) Waiter {
	if client.Waiter != nil {
		return client.Waiter
	}
	return timerWaiter{}
}
