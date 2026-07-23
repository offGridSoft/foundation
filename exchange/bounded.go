package exchange

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const ErrFmtStatusError = "exchange status rejected: status=%d expected=%d"

type RequestTarget interface {
	core.Validatable
	String() string
}

type BoundedPolicy struct {
	AttemptTimeout    core.NanosecondsDuration
	RequestBodyLimit  core.ByteCount
	ResponseBodyLimit core.ByteCount
	Redirect          core.HTTPRedirectPolicy
}

func (p BoundedPolicy) Validate() error {
	if err := validatePositiveDuration(p.AttemptTimeout); err != nil {
		return responseError(err)
	}
	if err := validateBodyLimit(p.RequestBodyLimit); err != nil {
		return responseError(err)
	}
	if err := validateBodyLimit(p.ResponseBodyLimit); err != nil {
		return responseError(err)
	}
	if err := p.Redirect.Validate(); err != nil {
		return responseError(err)
	}
	return nil
}

type HeaderSelection struct {
	Names []string
}

func (s HeaderSelection) Validate() error {
	if len(s.Names) > core.HTTPHeaderMaximumCount {
		return responseError(core.ErrExchangeContract)
	}
	for index, name := range s.Names {
		if err := core.ValidateHTTPHeaderName(name); err != nil {
			return responseError(err)
		}
		for prior := range index {
			if strings.EqualFold(s.Names[prior], name) {
				return responseError(core.ErrExchangeContract)
			}
		}
	}
	return nil
}

type BoundedRequest[Target RequestTarget] struct {
	Target                      Target
	Semantics                   core.HTTPRequestSemantics
	Body                        []byte
	Headers                     core.HTTPHeaders
	Query                       core.HTTPQuery
	CaptureHeaders              HeaderSelection
	ExpectedStatus              core.HTTPStatusCode
	RequestContentType          core.HTTPMediaType
	ExpectedResponseContentType core.HTTPMediaType
}

func (r BoundedRequest[Target]) Validate() error {
	common := commonRequestContract{
		Target: r.Target, Headers: r.Headers, Query: r.Query, Semantics: r.Semantics,
		ExpectedStatus: r.ExpectedStatus, CaptureHeaders: r.CaptureHeaders,
	}
	if err := validateCommonRequest(common); err != nil {
		return err
	}
	if r.Semantics.Replay != core.HTTPReplaySingleAttempt {
		return requestError(core.ErrExchangeContract)
	}
	hasBody := len(r.Body) > 0
	requiresBody := methodRequiresBody(r.Semantics.Method)
	if hasBody != requiresBody {
		return requestError(core.ErrExchangeContract)
	}
	if hasBody {
		if err := r.RequestContentType.Validate(); err != nil {
			return requestError(err)
		}
	} else if r.RequestContentType != core.HTTPMediaTypeUnknown {
		return requestError(core.ErrExchangeContract)
	}
	if r.ExpectedResponseContentType != core.HTTPMediaTypeUnknown {
		if err := r.ExpectedResponseContentType.Validate(); err != nil {
			return requestError(err)
		}
	}
	return nil
}

type BoundedResponse struct {
	Body     []byte
	Headers  core.HTTPHeaders
	Status   core.HTTPStatusCode
	Attempts uint64
}

type StatusError struct {
	Status   core.HTTPStatusCode
	Expected core.HTTPStatusCode
}

func (e StatusError) Error() string {
	return fmt.Sprintf(ErrFmtStatusError, e.Status.Int(), e.Expected.Int())
}

func (e StatusError) Unwrap() error { return core.ErrExchangeResponse }

func SendBounded[Target RequestTarget](ctx context.Context, client Client, request BoundedRequest[Target], policy BoundedPolicy) (BoundedResponse, error) {
	var zero BoundedResponse
	if err := validateClientContext(ctx); err != nil {
		return zero, err
	}
	if err := client.Validate(); err != nil {
		return zero, err
	}
	if err := request.Validate(); err != nil {
		return zero, err
	}
	if err := policy.Validate(); err != nil {
		return zero, err
	}
	if uint64(len(request.Body)) > policy.RequestBodyLimit.Uint64() {
		return zero, errors.Join(core.ErrExchangeRequest, core.ErrExchangeBodyLimit)
	}
	requestContext, cancel := context.WithTimeout(ctx, policy.AttemptTimeout.Duration())
	defer cancel()
	httpRequest, err := buildBoundedHTTPRequest(requestContext, request)
	if err != nil {
		return zero, requestError(err)
	}
	httpResponse, err := configuredHTTPClient(client.http, policy.Redirect).Do(httpRequest) // #nosec G704 -- typed target validation and redirect policy own the destination.
	if err != nil {
		return zero, classifyHTTPDoError[core.HTTPNoBody](ctx, httpResponse, err, 1).Err
	}
	return readBoundedHTTPResponse(requestContext, httpResponse, request, policy.ResponseBodyLimit)
}

func buildBoundedHTTPRequest[Target RequestTarget](ctx context.Context, request BoundedRequest[Target]) (*http.Request, error) {
	target, err := targetWithQuery(request.Target, request.Query)
	if err != nil {
		return nil, err
	}
	var body io.Reader
	if len(request.Body) > 0 {
		body = bytes.NewReader(request.Body)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, request.Semantics.Method.String(), target, body)
	if err != nil {
		return nil, err
	}
	applyBoundedHeaders(httpRequest, request.Headers, request.RequestContentType, request.ExpectedResponseContentType)
	return httpRequest, nil
}

func readBoundedHTTPResponse[Target RequestTarget](ctx context.Context, response *http.Response, request BoundedRequest[Target], limit core.ByteCount) (BoundedResponse, error) {
	if response == nil || response.Body == nil {
		return BoundedResponse{Attempts: 1}, responseError(core.ErrExchangeContract)
	}
	result, resultErr := readBoundedHTTPResponseBody(ctx, response, request, limit)
	return result, closeResponseBody(response.Body, resultErr)
}

func readBoundedHTTPResponseBody[Target RequestTarget](ctx context.Context, response *http.Response, request BoundedRequest[Target], limit core.ByteCount) (BoundedResponse, error) {
	result := BoundedResponse{Attempts: 1}
	status, err := core.NewHTTPStatusCode(response.StatusCode)
	if err != nil {
		return result, responseError(err)
	}
	result.Status = status
	statusErr := rejectedStatus(status, request.ExpectedStatus)
	if err := validateExpectedContentType(response, request.ExpectedResponseContentType); err != nil {
		return result, errors.Join(statusErr, err)
	}
	if err := validateIdentityEncoding(response); err != nil {
		return result, errors.Join(statusErr, err)
	}
	result.Headers, err = captureHeaders(response.Header, request.CaptureHeaders)
	if err != nil {
		return result, err
	}
	maximum, err := limit.Int64()
	if err != nil {
		return result, responseError(err)
	}
	if response.ContentLength > maximum {
		return result, errors.Join(statusErr, core.ErrExchangeResponse, core.ErrExchangeBodyLimit)
	}
	result.Body, err = readBounded(ctx, response.Body, maximum, bodyReadResponse)
	if err != nil {
		return result, errors.Join(statusErr, err)
	}
	if statusErr != nil {
		return result, statusErr
	}
	return result, nil
}

func rejectedStatus(status, expected core.HTTPStatusCode) error {
	if status == expected {
		return nil
	}
	return StatusError{Status: status, Expected: expected}
}

func closeResponseBody(body io.Closer, cause error) error {
	closeErr := body.Close()
	if closeErr == nil {
		return cause
	}
	closeFailure := errors.Join(core.ErrExchangeResponse, core.ErrExchangeTransport, closeErr)
	return errors.Join(cause, closeFailure)
}

type StreamPolicy struct {
	AttemptTimeout core.NanosecondsDuration
	ErrorBodyLimit core.ByteCount
	Redirect       core.HTTPRedirectPolicy
}

func (p StreamPolicy) Validate() error {
	if err := validatePositiveDuration(p.AttemptTimeout); err != nil {
		return responseError(err)
	}
	if err := validateBodyLimit(p.ErrorBodyLimit); err != nil {
		return responseError(err)
	}
	if err := p.Redirect.Validate(); err != nil {
		return responseError(err)
	}
	return nil
}

type StreamUploadRequest[Target RequestTarget] struct {
	Target         Target
	Body           io.Reader
	Semantics      core.HTTPRequestSemantics
	Headers        core.HTTPHeaders
	CaptureHeaders HeaderSelection
	ContentLength  core.ByteLength
	ExpectedStatus core.HTTPStatusCode
}

func (r StreamUploadRequest[Target]) Validate() error {
	common := commonRequestContract{
		Target: r.Target, Headers: r.Headers, Semantics: r.Semantics,
		ExpectedStatus: r.ExpectedStatus, CaptureHeaders: r.CaptureHeaders,
	}
	if err := validateCommonRequest(common); err != nil {
		return err
	}
	if r.Body == nil || !methodRequiresBody(r.Semantics.Method) || r.Semantics.Replay != core.HTTPReplaySingleAttempt {
		return requestError(core.ErrExchangeContract)
	}
	if err := r.ContentLength.Validate(); err != nil {
		return requestError(err)
	}
	length, err := r.ContentLength.Int64()
	if err != nil || length == math.MaxInt64 {
		return requestError(core.ErrExchangeContract)
	}
	return nil
}

type StreamResponse struct {
	Headers      core.HTTPHeaders
	Status       core.HTTPStatusCode
	BytesWritten core.ByteLength
}

func SendStream[Target RequestTarget](ctx context.Context, client Client, request StreamUploadRequest[Target], policy StreamPolicy) (StreamResponse, error) {
	var zero StreamResponse
	if err := validateStreamCall(ctx, client, request.Validate(), policy); err != nil {
		return zero, err
	}
	requestContext, cancel := context.WithTimeout(ctx, policy.AttemptTimeout.Duration())
	defer cancel()
	length, err := request.ContentLength.Int64()
	if err != nil {
		return zero, requestError(err)
	}
	counter := &streamCounter{reader: io.LimitReader(request.Body, length+1)}
	var body io.Reader = counter
	if length == 0 {
		body = http.NoBody
	}
	httpRequest, err := http.NewRequestWithContext(requestContext, request.Semantics.Method.String(), request.Target.String(), body)
	if err != nil {
		return zero, requestError(err)
	}
	httpRequest.ContentLength = length
	applyRawHeaders(httpRequest, request.Headers)
	httpResponse, err := configuredHTTPClient(client.http, policy.Redirect).Do(httpRequest) // #nosec G704 -- typed target validation and redirect policy own the destination.
	if err != nil {
		return zero, classifyHTTPDoError[core.HTTPNoBody](ctx, httpResponse, err, 1).Err
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return zero, responseError(core.ErrExchangeContract)
	}
	input := streamUploadResponseInput[Target]{
		Context: requestContext, Response: httpResponse, Request: request,
		Policy: policy, Counter: counter,
	}
	result, resultErr := readStreamUploadResponse(input)
	return result, closeResponseBody(httpResponse.Body, resultErr)
}

type streamUploadResponseInput[Target RequestTarget] struct {
	Context  context.Context
	Response *http.Response
	Counter  *streamCounter
	Request  StreamUploadRequest[Target]
	Policy   StreamPolicy
}

func readStreamUploadResponse[Target RequestTarget](input streamUploadResponseInput[Target]) (StreamResponse, error) {
	var result StreamResponse
	status, err := core.NewHTTPStatusCode(input.Response.StatusCode)
	if err != nil {
		return result, responseError(err)
	}
	result.Status = status
	result.Headers, err = captureHeaders(input.Response.Header, input.Request.CaptureHeaders)
	if err != nil {
		return result, err
	}
	statusErr := rejectedStatus(status, input.Request.ExpectedStatus)
	if err := validateIdentityEncoding(input.Response); err != nil {
		return result, errors.Join(statusErr, err)
	}
	_, drainErr := copyBounded(input.Context, io.Discard, input.Response.Body, input.Policy.ErrorBodyLimit)
	expected, err := input.Request.ContentLength.Int64()
	if err != nil {
		return result, requestError(err)
	}
	if input.Counter.count != expected {
		return result, errors.Join(statusErr, core.ErrExchangeRequest, io.ErrUnexpectedEOF)
	}
	result.BytesWritten = input.Request.ContentLength
	if statusErr != nil || drainErr != nil {
		return result, errors.Join(statusErr, drainErr)
	}
	return result, nil
}

type StreamDownloadRequest[Target RequestTarget] struct {
	Target                      Target
	Destination                 io.Writer
	Semantics                   core.HTTPRequestSemantics
	Headers                     core.HTTPHeaders
	CaptureHeaders              HeaderSelection
	ResponseBodyLimit           core.ByteCount
	ExpectedStatus              core.HTTPStatusCode
	ExpectedResponseContentType core.HTTPMediaType
}

func (r StreamDownloadRequest[Target]) Validate() error {
	common := commonRequestContract{
		Target: r.Target, Headers: r.Headers, Semantics: r.Semantics,
		ExpectedStatus: r.ExpectedStatus, CaptureHeaders: r.CaptureHeaders,
	}
	if err := validateCommonRequest(common); err != nil {
		return err
	}
	if r.Destination == nil || r.Semantics.Method != core.HTTPMethodGet || r.Semantics.Replay != core.HTTPReplaySingleAttempt {
		return requestError(core.ErrExchangeContract)
	}
	if err := r.ResponseBodyLimit.Validate(); err != nil {
		return requestError(err)
	}
	if r.ExpectedResponseContentType != core.HTTPMediaTypeUnknown {
		if err := r.ExpectedResponseContentType.Validate(); err != nil {
			return requestError(err)
		}
	}
	return nil
}

func ReceiveStream[Target RequestTarget](ctx context.Context, client Client, request StreamDownloadRequest[Target], policy StreamPolicy) (StreamResponse, error) {
	var zero StreamResponse
	if err := validateStreamCall(ctx, client, request.Validate(), policy); err != nil {
		return zero, err
	}
	requestContext, cancel := context.WithTimeout(ctx, policy.AttemptTimeout.Duration())
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, request.Semantics.Method.String(), request.Target.String(), nil)
	if err != nil {
		return zero, requestError(err)
	}
	applyRawHeaders(httpRequest, request.Headers)
	if request.ExpectedResponseContentType != core.HTTPMediaTypeUnknown {
		httpRequest.Header.Set(core.HTTPHeaderAccept, request.ExpectedResponseContentType.String())
	}
	httpResponse, err := configuredHTTPClient(client.http, policy.Redirect).Do(httpRequest) // #nosec G704 -- typed target validation and redirect policy own the destination.
	if err != nil {
		return zero, classifyHTTPDoError[core.HTTPNoBody](ctx, httpResponse, err, 1).Err
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return zero, responseError(core.ErrExchangeContract)
	}
	result, resultErr := readStreamDownloadResponse(requestContext, httpResponse, request, policy)
	return result, closeResponseBody(httpResponse.Body, resultErr)
}

func readStreamDownloadResponse[Target RequestTarget](ctx context.Context, response *http.Response, request StreamDownloadRequest[Target], policy StreamPolicy) (StreamResponse, error) {
	var result StreamResponse
	status, err := core.NewHTTPStatusCode(response.StatusCode)
	if err != nil {
		return result, responseError(err)
	}
	result.Status = status
	result.Headers, err = captureHeaders(response.Header, request.CaptureHeaders)
	if err != nil {
		return result, err
	}
	if status != request.ExpectedStatus {
		drainErr := drainRejectedResponse(ctx, response.Body, policy.ErrorBodyLimit)
		return result, errors.Join(StatusError{Status: status, Expected: request.ExpectedStatus}, drainErr)
	}
	if err := validateExpectedContentType(response, request.ExpectedResponseContentType); err != nil {
		return result, err
	}
	if err := validateIdentityEncoding(response); err != nil {
		return result, err
	}
	maximum, err := request.ResponseBodyLimit.Int64()
	if err != nil {
		return result, responseError(err)
	}
	if response.ContentLength > maximum {
		return result, errors.Join(core.ErrExchangeResponse, core.ErrExchangeBodyLimit)
	}
	written, err := copyBounded(ctx, request.Destination, response.Body, request.ResponseBodyLimit)
	if err != nil {
		return result, err
	}
	if written > 0 {
		result.BytesWritten = core.NewByteLength(uint64(written))
	}
	return result, nil
}

func drainRejectedResponse(ctx context.Context, body io.Reader, limit core.ByteCount) error {
	_, err := copyBounded(ctx, io.Discard, body, limit)
	return err
}

type commonRequestContract struct {
	Target         RequestTarget
	Semantics      core.HTTPRequestSemantics
	Headers        core.HTTPHeaders
	Query          core.HTTPQuery
	CaptureHeaders HeaderSelection
	ExpectedStatus core.HTTPStatusCode
}

func validateCommonRequest(contract commonRequestContract) error {
	if err := contract.Target.Validate(); err != nil {
		return requestError(err)
	}
	if err := contract.Headers.Validate(); err != nil {
		return requestError(err)
	}
	if managedRequestHeaderPresent(contract.Headers) {
		return requestError(core.ErrExchangeContract)
	}
	if err := contract.Query.Validate(); err != nil {
		return requestError(err)
	}
	if err := contract.Semantics.Validate(); err != nil {
		return requestError(err)
	}
	if !validExpectedStatus(contract.ExpectedStatus) {
		return requestError(core.ErrExchangeContract)
	}
	return contract.CaptureHeaders.Validate()
}

func managedRequestHeaderPresent(headers core.HTTPHeaders) bool {
	return headers.Contains(core.HTTPHeaderContentLength) ||
		headers.Contains(core.HTTPHeaderAcceptEncoding) ||
		headers.Contains(core.HTTPHeaderAccept)
}

func validExpectedStatus(status core.HTTPStatusCode) bool {
	success, err := core.HTTPStatusIsSuccess(status)
	return err == nil && success
}

func validateStreamCall(ctx context.Context, client Client, requestErr error, policy StreamPolicy) error {
	if err := validateClientContext(ctx); err != nil {
		return err
	}
	if err := client.Validate(); err != nil {
		return err
	}
	if requestErr != nil {
		return requestErr
	}
	return policy.Validate()
}

func methodRequiresBody(method core.HTTPMethod) bool {
	return method == core.HTTPMethodPost || method == core.HTTPMethodPut || method == core.HTTPMethodPatch
}

func targetWithQuery[Target RequestTarget](target Target, query core.HTTPQuery) (string, error) {
	encoded, err := query.Encode()
	if err != nil {
		return "", err
	}
	if encoded == "" {
		return target.String(), nil
	}
	return target.String() + "?" + encoded, nil
}

func applyBoundedHeaders(request *http.Request, headers core.HTTPHeaders, requestType, responseType core.HTTPMediaType) {
	applyRawHeaders(request, headers)
	if requestType != core.HTTPMediaTypeUnknown {
		request.Header.Set(core.HTTPHeaderContentType, requestType.String())
	}
	if responseType != core.HTTPMediaTypeUnknown {
		request.Header.Set(core.HTTPHeaderAccept, responseType.String())
	}
}

func applyRawHeaders(request *http.Request, headers core.HTTPHeaders) {
	request.Header.Set(core.HTTPHeaderAcceptEncoding, core.HTTPContentEncodingIdentity)
	for _, header := range headers.Values {
		request.Header.Set(header.Name, header.Value)
	}
}

func validateExpectedContentType(response *http.Response, expected core.HTTPMediaType) error {
	if expected == core.HTTPMediaTypeUnknown {
		return nil
	}
	values := response.Header.Values(core.HTTPHeaderContentType)
	if len(values) != 1 || values[0] != expected.String() {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeContentType)
	}
	return nil
}

func validateIdentityEncoding(response *http.Response) error {
	if len(response.Header.Values(core.HTTPHeaderContentEncoding)) != 0 {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeContentType)
	}
	return nil
}

func captureHeaders(source http.Header, selection HeaderSelection) (core.HTTPHeaders, error) {
	result := core.HTTPHeaders{Values: make([]core.HTTPHeader, 0, len(selection.Names))}
	for _, name := range selection.Names {
		values := source.Values(name)
		if len(values) == 0 {
			continue
		}
		if len(values) != 1 {
			return core.HTTPHeaders{}, responseError(core.ErrExchangeContract)
		}
		result.Values = append(result.Values, core.HTTPHeader{Name: name, Value: values[0]})
	}
	if err := result.Validate(); err != nil {
		return core.HTTPHeaders{}, responseError(err)
	}
	return result, nil
}

type streamCounter struct {
	reader io.Reader
	count  int64
}

func (r *streamCounter) Read(buffer []byte) (int, error) {
	read, err := r.reader.Read(buffer)
	r.count += int64(read)
	return read, err
}

func copyBounded(ctx context.Context, destination io.Writer, source io.Reader, limit core.ByteCount) (int64, error) {
	if ctx == nil || destination == nil || source == nil {
		return 0, responseError(core.ErrExchangeContract)
	}
	maximum, err := limit.Int64()
	if err != nil {
		return 0, responseError(err)
	}
	state := boundedCopyState{
		Context: ctx, Destination: destination, Source: source,
		Buffer: make([]byte, streamReadBufferBytes), Maximum: maximum,
	}
	return state.Copy()
}

type boundedCopyState struct {
	Context     context.Context
	Destination io.Writer
	Source      io.Reader
	Buffer      []byte
	Maximum     int64
	Written     int64
	EmptyReads  int
}

func (s *boundedCopyState) Copy() (int64, error) {
	for s.Written < s.Maximum {
		if err := s.contextError(); err != nil {
			return s.Written, err
		}
		readBuffer := s.nextBuffer()
		count, readErr := s.Source.Read(readBuffer)
		if count < 0 || count > len(readBuffer) {
			return s.Written, errors.Join(core.ErrExchangeResponse, core.ErrExchangeTransport, core.ErrExchangeContract)
		}
		if err := s.write(readBuffer[:count]); err != nil {
			return s.Written, err
		}
		done, err := s.finishRead(count, readErr)
		if err != nil || done {
			return s.Written, err
		}
	}
	return s.Written, s.probeOverflow()
}

func (s *boundedCopyState) nextBuffer() []byte {
	remaining := s.Maximum - s.Written
	if int64(len(s.Buffer)) > remaining {
		return s.Buffer[:remaining]
	}
	return s.Buffer
}

func (s *boundedCopyState) write(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	s.EmptyReads = 0
	count, writeErr := s.Destination.Write(data)
	if count < 0 || count > len(data) {
		return errors.Join(core.ErrExchangeWrite, core.ErrExchangeContract)
	}
	s.Written += int64(count)
	if writeErr != nil {
		return errors.Join(core.ErrExchangeWrite, writeErr)
	}
	if count != len(data) {
		return errors.Join(core.ErrExchangeWrite, io.ErrShortWrite)
	}
	return nil
}

func (s *boundedCopyState) finishRead(count int, readErr error) (bool, error) {
	if count == 0 && readErr == nil {
		s.EmptyReads++
		if s.EmptyReads >= streamMaximumEmptyReadRuns {
			return false, errors.Join(core.ErrExchangeTransport, io.ErrNoProgress)
		}
	}
	if readErr == nil {
		return false, nil
	}
	if errors.Is(readErr, io.EOF) {
		return true, nil
	}
	return false, errors.Join(core.ErrExchangeTransport, readErr)
}

func (s *boundedCopyState) probeOverflow() error {
	probe := s.Buffer[:1]
	for {
		if err := s.contextError(); err != nil {
			return err
		}
		count, readErr := s.Source.Read(probe)
		if count < 0 || count > len(probe) {
			return errors.Join(core.ErrExchangeResponse, core.ErrExchangeTransport, core.ErrExchangeContract)
		}
		if count > 0 {
			return errors.Join(core.ErrExchangeResponse, core.ErrExchangeBodyLimit)
		}
		done, err := s.finishRead(count, readErr)
		if err != nil || done {
			return err
		}
	}
}

func (s *boundedCopyState) contextError() error {
	cause := context.Cause(s.Context)
	if cause == nil {
		return nil
	}
	return errors.Join(core.ErrExchangeCancelled, cause)
}
