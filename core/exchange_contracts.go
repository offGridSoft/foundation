package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	HTTPMethodTokenMaxRunes    = 7
	HTTPIdempotencyKeyMaxRunes = 256
	HTTPHeaderMaximumCount     = 32
	HTTPQueryMaximumCount      = 32
	HTTPQueryNameMaxRunes      = 128
	HTTPQueryValueMaxRunes     = 2048
	HTTPStatusMinimum          = 100
	HTTPStatusMaximum          = 599
	HTTPHeaderIdempotencyKey   = "Idempotency-Key"
)

type HTTPMethod uint8

const (
	HTTPMethodUnknown HTTPMethod = iota
	HTTPMethodGet
	HTTPMethodHead
	HTTPMethodPost
	HTTPMethodPut
	HTTPMethodPatch
	HTTPMethodDelete
	HTTPMethodOptions
)

const (
	httpMethodTokenGet     = "GET"
	httpMethodTokenHead    = "HEAD"
	httpMethodTokenPost    = "POST"
	httpMethodTokenPut     = "PUT"
	httpMethodTokenPatch   = "PATCH"
	httpMethodTokenDelete  = "DELETE"
	httpMethodTokenOptions = "OPTIONS"
)

func httpMethodNames() [HTTPMethodOptions + 1]string {
	return [...]string{
		HTTPMethodGet:     httpMethodTokenGet,
		HTTPMethodHead:    httpMethodTokenHead,
		HTTPMethodPost:    httpMethodTokenPost,
		HTTPMethodPut:     httpMethodTokenPut,
		HTTPMethodPatch:   httpMethodTokenPatch,
		HTTPMethodDelete:  httpMethodTokenDelete,
		HTTPMethodOptions: httpMethodTokenOptions,
	}
}

func (m HTTPMethod) IsValid() bool {
	return m > HTTPMethodUnknown && int(m) < len(httpMethodNames()) && httpMethodNames()[m] != ""
}

func (m HTTPMethod) String() string {
	if !m.IsValid() {
		return ""
	}
	return httpMethodNames()[m]
}

func (m HTTPMethod) Validate() error {
	if !m.IsValid() {
		return fmt.Errorf(ErrFmtHTTPMethod, ErrExchangeContract)
	}
	return nil
}

func ParseHTTPMethod(value string) (HTTPMethod, error) {
	if err := ValidateOpaqueToken(value, HTTPMethodTokenMaxRunes); err != nil {
		return HTTPMethodUnknown, fmt.Errorf(ErrFmtHTTPMethod, ErrExchangeContract)
	}
	for method := HTTPMethodGet; int(method) < len(httpMethodNames()); method++ {
		if method.String() == value {
			return method, nil
		}
	}
	return HTTPMethodUnknown, fmt.Errorf(ErrFmtHTTPMethod, ErrExchangeContract)
}

func (m HTTPMethod) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}

func (m *HTTPMethod) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPMethod, ErrExchangeContract)
	}
	parsed, err := ParseHTTPMethod(value)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

type HTTPRedirectPolicy uint8

const (
	HTTPRedirectUnknown HTTPRedirectPolicy = iota
	HTTPRedirectReject
	HTTPRedirectSameOrigin
)

func (p HTTPRedirectPolicy) IsValid() bool {
	return p == HTTPRedirectReject || p == HTTPRedirectSameOrigin
}

func (p HTTPRedirectPolicy) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtHTTPRedirectPolicy, ErrExchangeContract)
	}
	return nil
}

func (p HTTPRedirectPolicy) String() string {
	switch p {
	case HTTPRedirectReject:
		return "reject"
	case HTTPRedirectSameOrigin:
		return "same_origin"
	default:
		return ""
	}
}

func ParseHTTPRedirectPolicy(value string) (HTTPRedirectPolicy, error) {
	for policy := HTTPRedirectReject; policy <= HTTPRedirectSameOrigin; policy++ {
		if policy.String() == value {
			return policy, nil
		}
	}
	return HTTPRedirectUnknown, fmt.Errorf(ErrFmtHTTPRedirectPolicy, ErrExchangeContract)
}

func (p HTTPRedirectPolicy) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *HTTPRedirectPolicy) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPRedirectPolicy, ErrExchangeContract)
	}
	parsed, err := ParseHTTPRedirectPolicy(value)
	if err != nil {
		return err
	}
	*p = parsed
	return p.Validate()
}

type HTTPReplaySafety uint8

const (
	HTTPReplayUnknown HTTPReplaySafety = iota
	HTTPReplaySafe
	HTTPReplayIdempotent
	HTTPReplaySingleAttempt
)

func (s HTTPReplaySafety) IsValid() bool {
	return s > HTTPReplayUnknown && s <= HTTPReplaySingleAttempt
}

func (s HTTPReplaySafety) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtHTTPReplaySafety, ErrExchangeContract)
	}
	return nil
}

func (s HTTPReplaySafety) String() string {
	switch s {
	case HTTPReplaySafe:
		return "safe"
	case HTTPReplayIdempotent:
		return "idempotent"
	case HTTPReplaySingleAttempt:
		return "single_attempt"
	default:
		return ""
	}
}

func ParseHTTPReplaySafety(value string) (HTTPReplaySafety, error) {
	for safety := HTTPReplaySafe; safety <= HTTPReplaySingleAttempt; safety++ {
		if safety.String() == value {
			return safety, nil
		}
	}
	return HTTPReplayUnknown, fmt.Errorf(ErrFmtHTTPReplaySafety, ErrExchangeContract)
}

func (s HTTPReplaySafety) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func (s *HTTPReplaySafety) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPReplaySafety, ErrExchangeContract)
	}
	parsed, err := ParseHTTPReplaySafety(value)
	if err != nil {
		return err
	}
	*s = parsed
	return s.Validate()
}

// AllowsAutomaticReplay reports whether the caller has declared that the
// complete request can be reconstructed and sent again. HTTP method semantics
// alone are not sufficient: an idempotent PUT can still carry a one-shot
// stream.
func (s HTTPReplaySafety) AllowsAutomaticReplay() (bool, error) {
	if err := s.Validate(); err != nil {
		return false, err
	}
	return s != HTTPReplaySingleAttempt, nil
}

type HTTPIdempotencyKey struct {
	value string
}

func ParseHTTPIdempotencyKey(value string) (HTTPIdempotencyKey, error) {
	key := HTTPIdempotencyKey{value: value}
	if err := key.Validate(); err != nil {
		return HTTPIdempotencyKey{}, err
	}
	return key, nil
}

func (k HTTPIdempotencyKey) String() string {
	return k.value
}

func (k HTTPIdempotencyKey) IsZero() bool {
	return k.value == ""
}

func (k HTTPIdempotencyKey) Validate() error {
	if err := ValidateOpaqueToken(k.value, HTTPIdempotencyKeyMaxRunes); err != nil {
		return fmt.Errorf(ErrFmtHTTPIdempotencyKey, ErrExchangeContract)
	}
	if err := ValidateHTTPHeaderValue(k.value); err != nil {
		return fmt.Errorf(ErrFmtHTTPIdempotencyKey, ErrExchangeContract)
	}
	return nil
}

func (k HTTPIdempotencyKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.value)
}

func (k *HTTPIdempotencyKey) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPIdempotencyKey, ErrExchangeContract)
	}
	parsed, err := ParseHTTPIdempotencyKey(value)
	if err != nil {
		return err
	}
	*k = parsed
	return k.Validate()
}

type HTTPRequestSemantics struct {
	IdempotencyKey HTTPIdempotencyKey
	Method         HTTPMethod
	Replay         HTTPReplaySafety
}

type HTTPNoBody struct{}

func (HTTPNoBody) Validate() error {
	return nil
}

func (HTTPNoBody) APIBody() {}

type HTTPIdempotentBody interface {
	Validatable
	HTTPIdempotencyKey() (HTTPIdempotencyKey, error)
}

type HTTPRouteSemantics struct {
	Method HTTPMethod
	Replay HTTPReplaySafety
}

func (s HTTPRouteSemantics) Validate() error {
	if err := s.Method.Validate(); err != nil {
		return fmt.Errorf(ErrFmtHTTPRouteSemantics, err)
	}
	if err := s.Replay.Validate(); err != nil {
		return fmt.Errorf(ErrFmtHTTPRouteSemantics, err)
	}
	if !s.validCombination() {
		return fmt.Errorf(ErrFmtHTTPRouteSemantics, ErrExchangeContract)
	}
	return nil
}

func (s HTTPRouteSemantics) validCombination() bool {
	switch s.Method {
	case HTTPMethodGet, HTTPMethodHead, HTTPMethodOptions:
		return s.Replay == HTTPReplaySafe
	case HTTPMethodPut, HTTPMethodDelete:
		return s.Replay == HTTPReplayIdempotent
	case HTTPMethodPost, HTTPMethodPatch:
		return s.Replay == HTTPReplayIdempotent || s.Replay == HTTPReplaySingleAttempt
	default:
		return false
	}
}

type HTTPHeader struct {
	Name  string
	Value string
}

func (h HTTPHeader) Validate() error {
	if err := ValidateHTTPHeaderName(h.Name); err != nil {
		return fmt.Errorf(ErrFmtHTTPHeader, ErrExchangeContract)
	}
	if err := ValidateHTTPHeaderValue(h.Value); err != nil {
		return fmt.Errorf(ErrFmtHTTPHeader, ErrExchangeContract)
	}
	return nil
}

type HTTPHeaders struct {
	Values []HTTPHeader
}

type HTTPQueryParameter struct {
	Name  string
	Value string
}

func (p HTTPQueryParameter) Validate() error {
	if err := ValidateHTTPHeaderName(p.Name); err != nil || utf8.RuneCountInString(p.Name) > HTTPQueryNameMaxRunes {
		return fmt.Errorf(ErrFmtHTTPQueryParameter, ErrExchangeContract)
	}
	if err := ValidateOpaqueToken(p.Value, HTTPQueryValueMaxRunes); err != nil {
		return fmt.Errorf(ErrFmtHTTPQueryParameter, ErrExchangeContract)
	}
	return nil
}

type HTTPQuery struct {
	Parameters []HTTPQueryParameter
}

func (q HTTPQuery) Validate() error {
	if len(q.Parameters) > HTTPQueryMaximumCount {
		return fmt.Errorf(ErrFmtHTTPQuery, ErrExchangeContract)
	}
	for index, parameter := range q.Parameters {
		if err := parameter.Validate(); err != nil {
			return fmt.Errorf(ErrFmtHTTPQuery, err)
		}
		for prior := range index {
			if q.Parameters[prior].Name == parameter.Name {
				return fmt.Errorf(ErrFmtHTTPQuery, ErrExchangeContract)
			}
		}
	}
	return nil
}

func (q HTTPQuery) Encode() (string, error) {
	if err := q.Validate(); err != nil {
		return "", err
	}
	var encoded strings.Builder
	for index, parameter := range q.Parameters {
		if index > 0 {
			encoded.WriteByte('&')
		}
		encoded.WriteString(url.QueryEscape(parameter.Name))
		encoded.WriteByte('=')
		encoded.WriteString(url.QueryEscape(parameter.Value))
	}
	return encoded.String(), nil
}

func (h HTTPHeaders) Contains(name string) bool {
	for _, header := range h.Values {
		if equalHTTPHeaderName(header.Name, name) {
			return true
		}
	}
	return false
}

func (h HTTPHeaders) Get(name string) (string, bool) {
	for _, header := range h.Values {
		if equalHTTPHeaderName(header.Name, name) {
			return header.Value, true
		}
	}
	return "", false
}

func (h HTTPHeaders) Validate() error {
	if len(h.Values) > HTTPHeaderMaximumCount {
		return fmt.Errorf(ErrFmtHTTPHeaders, ErrExchangeContract)
	}
	for index, header := range h.Values {
		if err := header.Validate(); err != nil {
			return fmt.Errorf(ErrFmtHTTPHeaders, err)
		}
		for prior := range index {
			if equalHTTPHeaderName(h.Values[prior].Name, header.Name) {
				return fmt.Errorf(ErrFmtHTTPHeaders, ErrExchangeContract)
			}
		}
	}
	return nil
}

func equalHTTPHeaderName(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range len(left) {
		leftByte := left[index]
		rightByte := right[index]
		if leftByte >= 'A' && leftByte <= 'Z' {
			leftByte += 'a' - 'A'
		}
		if rightByte >= 'A' && rightByte <= 'Z' {
			rightByte += 'a' - 'A'
		}
		if leftByte != rightByte {
			return false
		}
	}
	return true
}

func (s HTTPRequestSemantics) Validate() error {
	if err := s.Method.Validate(); err != nil {
		return fmt.Errorf(ErrFmtHTTPRequestSemantics, err)
	}
	if err := s.Replay.Validate(); err != nil {
		return fmt.Errorf(ErrFmtHTTPRequestSemantics, err)
	}
	if !s.IdempotencyKey.IsZero() {
		if err := s.IdempotencyKey.Validate(); err != nil {
			return fmt.Errorf(ErrFmtHTTPRequestSemantics, err)
		}
	}
	if !s.validCombination() {
		return fmt.Errorf(ErrFmtHTTPRequestSemantics, ErrExchangeContract)
	}
	return nil
}

func (s HTTPRequestSemantics) validCombination() bool {
	hasKey := !s.IdempotencyKey.IsZero()
	if s.Replay == HTTPReplaySingleAttempt {
		return s.Method.IsValid() && !hasKey
	}
	if s.Replay == HTTPReplaySafe {
		return methodIsSafe(s.Method) && !hasKey
	}
	return methodHasIdempotentIdentity(s.Method, hasKey)
}

func methodIsSafe(method HTTPMethod) bool {
	return method == HTTPMethodGet || method == HTTPMethodHead || method == HTTPMethodOptions
}

func methodHasIdempotentIdentity(method HTTPMethod, hasKey bool) bool {
	if hasKey {
		return method == HTTPMethodPost || method == HTTPMethodPatch
	}
	return method == HTTPMethodPut || method == HTTPMethodDelete
}

type HTTPMediaType uint8

const (
	HTTPMediaTypeUnknown HTTPMediaType = iota
	HTTPMediaTypeJSON
	HTTPMediaTypeOctetStream
	HTTPMediaTypeTextPlain
	HTTPMediaTypeTimestampQuery
	HTTPMediaTypeTimestampReply
)

func (m HTTPMediaType) IsValid() bool {
	return m > HTTPMediaTypeUnknown && m <= HTTPMediaTypeTimestampReply
}

func (m HTTPMediaType) Validate() error {
	if !m.IsValid() {
		return fmt.Errorf(ErrFmtHTTPMediaType, ErrExchangeContract)
	}
	return nil
}

func (m HTTPMediaType) String() string {
	switch m {
	case HTTPMediaTypeJSON:
		return HTTPContentTypeJSON
	case HTTPMediaTypeOctetStream:
		return HTTPContentTypeOctetStream
	case HTTPMediaTypeTextPlain:
		return HTTPContentTypeTextPlain
	case HTTPMediaTypeTimestampQuery:
		return HTTPContentTypeTimestampQuery
	case HTTPMediaTypeTimestampReply:
		return HTTPContentTypeTimestampReply
	default:
		return ""
	}
}

func ParseHTTPMediaType(value string) (HTTPMediaType, error) {
	for mediaType := HTTPMediaTypeJSON; mediaType <= HTTPMediaTypeTimestampReply; mediaType++ {
		if mediaType.String() == value {
			return mediaType, nil
		}
	}
	return HTTPMediaTypeUnknown, fmt.Errorf(ErrFmtHTTPMediaType, ErrExchangeContract)
}

func (m HTTPMediaType) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}

func (m *HTTPMediaType) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPMediaType, ErrExchangeContract)
	}
	parsed, err := ParseHTTPMediaType(value)
	if err != nil {
		return err
	}
	*m = parsed
	return m.Validate()
}

type HTTPStatusCode uint16

const HTTPRetryableStatusCount = 7

const (
	HTTPStatusCodeUnknown         HTTPStatusCode = 0
	HTTPStatusOK                  HTTPStatusCode = 200
	HTTPStatusNotFound            HTTPStatusCode = 404
	HTTPStatusRequestTimeout      HTTPStatusCode = 408
	HTTPStatusTooEarly            HTTPStatusCode = 425
	HTTPStatusTooManyRequests     HTTPStatusCode = 429
	HTTPStatusInternalServerError HTTPStatusCode = 500
	HTTPStatusBadGateway          HTTPStatusCode = 502
	HTTPStatusServiceUnavailable  HTTPStatusCode = 503
	HTTPStatusGatewayTimeout      HTTPStatusCode = 504
)

func NewHTTPStatusCode(value int) (HTTPStatusCode, error) {
	if value < HTTPStatusMinimum || value > HTTPStatusMaximum {
		return HTTPStatusCodeUnknown, fmt.Errorf(ErrFmtHTTPStatusCode, ErrExchangeContract)
	}
	return HTTPStatusCode(value), nil
}

func (c HTTPStatusCode) Int() int {
	return int(c)
}

func (c HTTPStatusCode) IsValid() bool {
	return c >= HTTPStatusMinimum && c <= HTTPStatusMaximum
}

func (c HTTPStatusCode) String() string {
	if !c.IsValid() {
		return ""
	}
	return strconv.Itoa(c.Int())
}

func (c HTTPStatusCode) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtHTTPStatusCode, ErrExchangeContract)
	}
	return nil
}

func (c HTTPStatusCode) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.Int())
}

func (c *HTTPStatusCode) UnmarshalJSON(data []byte) error {
	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtHTTPStatusCode, ErrExchangeContract)
	}
	parsed, err := NewHTTPStatusCode(value)
	if err != nil {
		return err
	}
	*c = parsed
	return c.Validate()
}

type HTTPRetryDirective struct {
	Delay NanosecondsDuration
}

func (d HTTPRetryDirective) Validate() error {
	if err := d.Delay.Validate(); err != nil {
		return fmt.Errorf(ErrFmtHTTPRetryDirective, err)
	}
	if d.Delay.IsZero() || d.Delay.Duration() > BackoffMaxDuration {
		return fmt.Errorf(ErrFmtHTTPRetryDirective, ErrExchangeContract)
	}
	return nil
}

func (d HTTPRetryDirective) HeaderValue() (string, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}
	nanoseconds := d.Delay.Nanoseconds()
	seconds := nanoseconds / int64(time.Second)
	if nanoseconds%int64(time.Second) != 0 {
		seconds++
	}
	return strconv.FormatInt(seconds, 10), nil
}

const (
	HTTPStatusSuccessMinimum HTTPStatusCode = 200
	HTTPStatusSuccessMaximum HTTPStatusCode = 299
)

// HTTPStatusIsSuccess is the single compiler-owned success-range gate; client
// expected-status validation, server envelope routing, and the classifier all
// project from it instead of restating the 2xx literals.
func HTTPStatusIsSuccess(status HTTPStatusCode) (bool, error) {
	if err := status.Validate(); err != nil {
		return false, err
	}
	return status >= HTTPStatusSuccessMinimum && status <= HTTPStatusSuccessMaximum, nil
}

func HTTPStatusIsRetryable(status HTTPStatusCode) (bool, error) {
	if err := status.Validate(); err != nil {
		return false, err
	}
	for _, retryable := range HTTPRetryableStatusCodes() {
		if status == retryable {
			return true, nil
		}
	}
	return false, nil
}

// HTTPRetryableStatusCodes is the single compiler-owned retry status lattice.
// The fixed-size return value gives generated browser projections an exact
// shape while returning a copy prevents callers from mutating shared state.
func HTTPRetryableStatusCodes() [HTTPRetryableStatusCount]HTTPStatusCode {
	return [...]HTTPStatusCode{
		HTTPStatusRequestTimeout,
		HTTPStatusTooEarly,
		HTTPStatusTooManyRequests,
		HTTPStatusInternalServerError,
		HTTPStatusBadGateway,
		HTTPStatusServiceUnavailable,
		HTTPStatusGatewayTimeout,
	}
}

func ClassifyHTTPStatus(status, expected HTTPStatusCode) (HTTPOutcome, error) {
	if err := status.Validate(); err != nil {
		return httpOutcomeInvalid, err
	}
	if success, err := HTTPStatusIsSuccess(expected); err != nil || !success {
		return httpOutcomeInvalid, fmt.Errorf(ErrFmtHTTPStatusCode, ErrExchangeContract)
	}
	if status == expected {
		return HTTPOutcomeSuccess, nil
	}
	retryable, err := HTTPStatusIsRetryable(status)
	if err != nil {
		return httpOutcomeInvalid, err
	}
	if retryable {
		return HTTPOutcomeRetryable, nil
	}
	return HTTPOutcomeTerminal, nil
}
