package exchange

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	streamReadBufferBytes      = 32 << 10
	streamMaximumEmptyReadRuns = 100
)

type ServerPolicy struct {
	RequestBodyLimit core.ByteCount
}

func (p ServerPolicy) Validate() error {
	if err := p.RequestBodyLimit.Validate(); err != nil {
		return requestError(err)
	}
	if p.RequestBodyLimit.Uint64() > core.StrictJSONMaxBytes {
		return requestError(core.ErrExchangeBodyLimit)
	}
	return nil
}

type Received[T core.Validatable] struct {
	Body           T
	IdempotencyKey core.HTTPIdempotencyKey
}

func (r Received[T]) Validate() error {
	if err := r.Body.Validate(); err != nil {
		return requestError(err)
	}
	if !r.IdempotencyKey.IsZero() {
		if err := r.IdempotencyKey.Validate(); err != nil {
			return requestError(err)
		}
	}
	return nil
}

func ReceiveJSON[T core.Validatable](request *http.Request, route core.HTTPRouteSemantics, policy ServerPolicy) (Received[T], error) {
	var zero Received[T]
	key, limit, err := validateReceiveJSONCall(request, route, policy)
	if err != nil {
		return zero, err
	}
	return decodeReceivedJSON[T](request, key, limit)
}

// JSONProjector completes a decoded wire structure with typed request state
// before the owning body validates the complete ingress value. Typical
// projections copy path, query, or authenticated principal values into Body.
type JSONProjector[Body any, BodyPtr interface {
	*Body
	core.Validatable
}] func(context.Context, *http.Request, BodyPtr) error

// ReceiveProjectedJSON decodes a strict bounded JSON object, applies one typed
// structure-to-structure projection, and validates the completed body. The
// unvalidated intermediate never crosses this package boundary.
func ReceiveProjectedJSON[Body any, BodyPtr interface {
	*Body
	core.Validatable
}](request *http.Request, route core.HTTPRouteSemantics, policy ServerPolicy, project JSONProjector[Body, BodyPtr]) (Received[BodyPtr], error) {
	var zero Received[BodyPtr]
	if project == nil {
		return zero, requestError(core.ErrFoundationContract)
	}
	key, limit, err := validateReceiveJSONCall(request, route, policy)
	if err != nil {
		return zero, err
	}
	data, err := readRequestBody(request, limit)
	if err != nil {
		return zero, err
	}
	body, err := core.DecodeStrictJSONStructure[Body](data)
	if err != nil {
		return zero, requestError(err)
	}
	bodyPtr := BodyPtr(&body)
	if err := project(request.Context(), request, bodyPtr); err != nil {
		return zero, requestError(err)
	}
	if err := bodyPtr.Validate(); err != nil {
		return zero, requestError(err)
	}
	received := Received[BodyPtr]{Body: bodyPtr, IdempotencyKey: key}
	if err := received.Validate(); err != nil {
		return zero, err
	}
	return received, nil
}

// ReceiveNoBody validates the complete HTTP metadata boundary for a route
// whose method never carries an exchange body. It rejects smuggled bodies,
// body metadata, encodings, and replay headers before application dispatch.
func ReceiveNoBody(request *http.Request, route core.HTTPRouteSemantics) (Received[core.HTTPNoBody], error) {
	var zero Received[core.HTTPNoBody]
	key, err := validateReceiveNoBodyCall(request, route)
	if err != nil {
		return zero, err
	}
	received := Received[core.HTTPNoBody]{Body: core.HTTPNoBody{}, IdempotencyKey: key}
	if err := received.Validate(); err != nil {
		return zero, err
	}
	return received, nil
}

func validateReceiveNoBodyCall(request *http.Request, route core.HTTPRouteSemantics) (core.HTTPIdempotencyKey, error) {
	if request == nil {
		return core.HTTPIdempotencyKey{}, requestError(core.ErrFoundationContract)
	}
	if err := route.Validate(); err != nil {
		return core.HTTPIdempotencyKey{}, requestError(err)
	}
	if err := validateRequestContext(request.Context()); err != nil {
		return core.HTTPIdempotencyKey{}, err
	}
	if err := validateNoBodyRequestMetadata(request, route); err != nil {
		return core.HTTPIdempotencyKey{}, err
	}
	return requestIdempotencyKey(request, route)
}

func validateReceiveJSONCall(request *http.Request, route core.HTTPRouteSemantics, policy ServerPolicy) (core.HTTPIdempotencyKey, int64, error) {
	if request == nil {
		return core.HTTPIdempotencyKey{}, 0, requestError(core.ErrFoundationContract)
	}
	if err := policy.Validate(); err != nil {
		return core.HTTPIdempotencyKey{}, 0, err
	}
	if err := route.Validate(); err != nil {
		return core.HTTPIdempotencyKey{}, 0, requestError(err)
	}
	if !methodRequiresBody(route.Method) {
		return core.HTTPIdempotencyKey{}, 0, requestError(core.ErrExchangeContract)
	}
	if err := validateRequestContext(request.Context()); err != nil {
		return core.HTTPIdempotencyKey{}, 0, err
	}
	if err := validateRequestMetadata(request, route); err != nil {
		return core.HTTPIdempotencyKey{}, 0, err
	}
	key, err := requestIdempotencyKey(request, route)
	if err != nil {
		return core.HTTPIdempotencyKey{}, 0, err
	}
	limit, err := policy.RequestBodyLimit.Int64()
	if err != nil {
		return core.HTTPIdempotencyKey{}, 0, requestError(err)
	}
	return key, limit, nil
}

func decodeReceivedJSON[T core.Validatable](request *http.Request, key core.HTTPIdempotencyKey, limit int64) (Received[T], error) {
	var zero Received[T]
	data, err := readRequestBody(request, limit)
	if err != nil {
		return zero, err
	}
	body, err := core.DecodeStrictJSON[T](data)
	if err != nil {
		return zero, requestError(err)
	}
	if err := body.Validate(); err != nil {
		return zero, requestError(err)
	}
	received := Received[T]{Body: body, IdempotencyKey: key}
	if err := received.Validate(); err != nil {
		return zero, err
	}
	return received, nil
}

func validateRequestContext(ctx context.Context) error {
	if ctx == nil {
		return requestError(core.ErrNilContext)
	}
	select {
	case <-ctx.Done():
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeCancelled, context.Cause(ctx))
	default:
		return nil
	}
}

func validateRequestMetadata(request *http.Request, route core.HTTPRouteSemantics) error {
	if err := validateRequestMethod(request, route); err != nil {
		return err
	}
	contentTypes := request.Header.Values(core.HTTPHeaderContentType)
	if len(contentTypes) != 1 || contentTypes[0] != core.HTTPContentTypeJSON {
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeContentType)
	}
	if len(request.Header.Values(core.HTTPHeaderContentEncoding)) != 0 {
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeContentType)
	}
	return nil
}

func validateNoBodyRequestMetadata(request *http.Request, route core.HTTPRouteSemantics) error {
	if err := validateRequestMethod(request, route); err != nil {
		return err
	}
	if request.ContentLength != 0 || request.Body != nil && request.Body != http.NoBody {
		return requestError(core.ErrFoundationContract)
	}
	if len(request.Header.Values(core.HTTPHeaderContentType)) != 0 || len(request.Header.Values(core.HTTPHeaderContentEncoding)) != 0 {
		return errors.Join(core.ErrExchangeRequest, core.ErrExchangeContentType)
	}
	if len(request.TransferEncoding) != 0 {
		return requestError(core.ErrFoundationContract)
	}
	return nil
}

func validateRequestMethod(request *http.Request, route core.HTTPRouteSemantics) error {
	method, err := core.ParseHTTPMethod(request.Method)
	if err != nil || method != route.Method {
		return requestError(core.ErrFoundationContract)
	}
	return nil
}

type ServerResponse[T core.Validatable] struct {
	Envelope   core.APIEnvelope[T]
	RetryAfter core.HTTPRetryDirective
	Status     core.HTTPStatusCode
}

func (r ServerResponse[T]) Validate() error {
	success, err := core.HTTPStatusIsSuccess(r.Status)
	if err != nil {
		return responseError(err)
	}
	if success {
		return r.validateSuccess()
	}
	return r.validateFailure()
}

func (r ServerResponse[T]) validateSuccess() error {
	if err := r.Envelope.ValidateSuccess(); err != nil {
		return responseError(err)
	}
	if !r.RetryAfter.Delay.IsZero() {
		return responseError(core.ErrExchangeContract)
	}
	return nil
}

func (r ServerResponse[T]) validateFailure() error {
	if err := r.Envelope.ValidateFailure(); err != nil {
		return responseError(err)
	}
	retryable, err := core.HTTPStatusIsRetryable(r.Status)
	if err != nil {
		return responseError(err)
	}
	if retryable {
		if err := r.RetryAfter.Validate(); err != nil {
			return responseError(err)
		}
		return nil
	}
	if !r.RetryAfter.Delay.IsZero() {
		return responseError(core.ErrExchangeContract)
	}
	return nil
}

func WriteJSON[T core.Validatable](writer http.ResponseWriter, response ServerResponse[T]) error {
	if writer == nil {
		return responseError(core.ErrFoundationContract)
	}
	if err := response.Validate(); err != nil {
		return err
	}
	body, err := core.EncodeValidatedJSON(response.Envelope)
	if err != nil {
		return responseError(err)
	}
	writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	writer.Header().Set(core.HTTPHeaderContentLength, strconv.Itoa(len(body)))
	if !response.RetryAfter.Delay.IsZero() {
		header, headerErr := response.RetryAfter.HeaderValue()
		if headerErr != nil {
			return responseError(headerErr)
		}
		writer.Header().Set(core.HTTPHeaderRetryAfter, header)
	}
	writer.WriteHeader(response.Status.Int())
	written, err := writer.Write(body)
	if err != nil {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeWrite, err)
	}
	if written != len(body) {
		return errors.Join(core.ErrExchangeResponse, core.ErrExchangeWrite, io.ErrShortWrite)
	}
	return nil
}

func requestIdempotencyKey(request *http.Request, route core.HTTPRouteSemantics) (core.HTTPIdempotencyKey, error) {
	values := request.Header.Values(core.HTTPHeaderIdempotencyKey)
	requiresKey := route.Replay == core.HTTPReplayIdempotent && (route.Method == core.HTTPMethodPost || route.Method == core.HTTPMethodPatch)
	if !requiresKey {
		if len(values) != 0 {
			return core.HTTPIdempotencyKey{}, requestError(core.ErrFoundationContract)
		}
		return core.HTTPIdempotencyKey{}, nil
	}
	if len(values) != 1 {
		return core.HTTPIdempotencyKey{}, requestError(core.ErrFoundationContract)
	}
	key, err := core.ParseHTTPIdempotencyKey(values[0])
	if err != nil {
		return core.HTTPIdempotencyKey{}, requestError(err)
	}
	return key, nil
}

func readRequestBody(request *http.Request, limit int64) ([]byte, error) {
	if request.Body == nil {
		return nil, requestError(core.ErrFoundationContract)
	}
	body := request.Body
	data, readErr := readRequestBodyData(request, body, limit)
	closeErr := body.Close()
	if closeErr != nil {
		return nil, errors.Join(readErr, requestError(closeErr))
	}
	return data, readErr
}

func readRequestBodyData(request *http.Request, body io.Reader, limit int64) ([]byte, error) {
	if request.ContentLength > limit {
		return nil, errors.Join(core.ErrExchangeRequest, core.ErrExchangeBodyLimit)
	}
	data, err := readBounded(request.Context(), body, limit, bodyReadRequest)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type bodyReadPhase uint8

const (
	bodyReadUnknown bodyReadPhase = iota
	bodyReadRequest
	bodyReadResponse
)

func readBounded(ctx context.Context, reader io.Reader, limit int64, phase bodyReadPhase) ([]byte, error) {
	if ctx == nil || reader == nil || limit < 1 || limit > core.StrictJSONMaxBytes {
		return nil, bodyReadError(phase, core.ErrFoundationContract)
	}
	capacity := min(limit, streamReadBufferBytes)
	state := boundedReadState{
		Context: ctx, Reader: reader, Limit: limit, Phase: phase,
		Chunk: make([]byte, capacity),
	}
	state.Buffer.Grow(int(capacity))
	return state.Read()
}

type boundedReadState struct {
	Context    context.Context
	Reader     io.Reader
	Chunk      []byte
	Buffer     bytes.Buffer
	Limit      int64
	EmptyReads int
	Phase      bodyReadPhase
}

func (s *boundedReadState) Read() ([]byte, error) {
	for {
		if err := validateBodyReadContext(s.Context, s.Phase); err != nil {
			return nil, err
		}
		readBuffer := s.nextBuffer()
		count, readErr := s.Reader.Read(readBuffer)
		readSize := len(readBuffer)
		if count < 0 || count > int(readSize) {
			return nil, bodyReadError(s.Phase, core.ErrFoundationContract)
		}
		if err := s.store(readBuffer[:count]); err != nil {
			return nil, err
		}
		done, err := s.finishRead(count, readErr)
		if err != nil {
			return nil, err
		}
		if done {
			return s.Buffer.Bytes(), nil
		}
	}
}

func (s *boundedReadState) nextBuffer() []byte {
	remaining := s.Limit - int64(s.Buffer.Len())
	readSize := min(int64(len(s.Chunk)), remaining+1)
	return s.Chunk[:readSize]
}

func (s *boundedReadState) store(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	s.EmptyReads = 0
	if int64(s.Buffer.Len())+int64(len(data)) > s.Limit {
		return bodyReadLimitError(s.Phase)
	}
	if _, err := s.Buffer.Write(data); err != nil {
		return bodyReadError(s.Phase, err)
	}
	return nil
}

func (s *boundedReadState) finishRead(count int, readErr error) (bool, error) {
	if count == 0 && readErr == nil {
		s.EmptyReads++
		if s.EmptyReads >= streamMaximumEmptyReadRuns {
			return false, bodyReadError(s.Phase, io.ErrNoProgress)
		}
	}
	if readErr == nil {
		return false, nil
	}
	if errors.Is(readErr, io.EOF) {
		return true, nil
	}
	if _, ok := errors.AsType[*http.MaxBytesError](readErr); ok {
		return false, bodyReadLimitError(s.Phase)
	}
	return false, bodyReadError(s.Phase, readErr)
}

func validateBodyReadContext(ctx context.Context, phase bodyReadPhase) error {
	select {
	case <-ctx.Done():
		return errors.Join(bodyReadIdentity(phase), core.ErrExchangeCancelled, context.Cause(ctx))
	default:
		return nil
	}
}

func bodyReadError(phase bodyReadPhase, cause error) error {
	return errors.Join(bodyReadIdentity(phase), cause)
}

func bodyReadLimitError(phase bodyReadPhase) error {
	return errors.Join(bodyReadIdentity(phase), core.ErrExchangeBodyLimit)
}

func bodyReadIdentity(phase bodyReadPhase) error {
	if phase == bodyReadResponse {
		return core.ErrExchangeResponse
	}
	return core.ErrExchangeRequest
}

func requestError(cause error) error {
	return fmt.Errorf("exchange request contract: %w", errors.Join(core.ErrExchangeRequest, cause))
}

func responseError(cause error) error {
	return fmt.Errorf("exchange response contract: %w", errors.Join(core.ErrExchangeResponse, cause))
}
