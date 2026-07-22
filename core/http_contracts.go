package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	HTTPHeaderContentType                  = "Content-Type"
	HTTPHeaderAccept                       = "Accept"
	HTTPHeaderAuthorization                = "Authorization"
	HTTPHeaderRetryAfter                   = "Retry-After"
	HTTPHeaderContentLength                = "Content-Length"
	HTTPHeaderContentEncoding              = "Content-Encoding"
	HTTPHeaderAcceptEncoding               = "Accept-Encoding"
	HTTPContentTypeJSON                    = "application/json"
	HTTPContentTypeOctetStream             = "application/octet-stream"
	HTTPContentTypeTextPlain               = "text/plain"
	HTTPContentTypeTimestampQuery          = "application/timestamp-query"
	HTTPContentTypeTimestampReply          = "application/timestamp-reply"
	HTTPContentEncodingIdentity            = "identity"
	HTTPAuthorizationBearerPrefix          = "Bearer "
	URLSchemeHTTP                          = "http"
	URLSchemeHTTPS                         = "https"
	HostLocalhost                          = "localhost"
	HTTPSURLDefaultMaxRunes                = 2048
	HTTPHeaderNameMaxRunes                 = 256
	HTTPHeaderValueMaxRunes                = 8192
	BackoffMaxDuration                     = 24 * time.Hour
	HTTPRetryDefaultBase                   = 100 * time.Millisecond
	HTTPRetryDefaultMaximum                = 2 * time.Second
	HTTPRetryDefaultMaximumAttempts uint64 = 4
)

type HTTPSURLPolicy struct {
	MaxRunes      int
	RequirePath   bool
	AllowQuery    bool
	AllowFragment bool
}

func ValidateHTTPSURL(value string, policy HTTPSURLPolicy) error {
	if err := validateURLLength(value, policy.MaxRunes); err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	return validateHTTPSURLParts(parsed, policy)
}

func validateURLLength(value string, maxRunes int) error {
	if maxRunes < 1 {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	if err := ValidateOpaqueToken(value, maxRunes); err != nil {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	return nil
}

func validateHTTPSURLParts(parsed *url.URL, policy HTTPSURLPolicy) error {
	if parsed.Scheme != URLSchemeHTTPS || parsed.Host == "" {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	if parsed.User != nil {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	if policy.RequirePath && strings.TrimSpace(parsed.Path) == "" {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	if !policy.AllowQuery && parsed.RawQuery != "" {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	if !policy.AllowFragment && parsed.Fragment != "" {
		return fmt.Errorf(ErrFmtHTTPSURL, ErrFoundationContract)
	}
	return nil
}

func ValidateHTTPHeaderName(value string) error {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > HTTPHeaderNameMaxRunes {
		return fmt.Errorf(ErrFmtHTTPHeaderName, ErrFoundationContract)
	}
	for _, r := range value {
		if !validHTTPTokenRune(r) {
			return fmt.Errorf(ErrFmtHTTPHeaderName, ErrFoundationContract)
		}
	}
	return nil
}

func ValidateHTTPHeaderValue(value string) error {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > HTTPHeaderValueMaxRunes {
		return fmt.Errorf(ErrFmtHTTPHeaderValue, ErrFoundationContract)
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\t' {
			return fmt.Errorf(ErrFmtHTTPHeaderValue, ErrFoundationContract)
		}
	}
	return nil
}

func validHTTPTokenRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	default:
		return stringsContainsHTTPTokenSpecial(r)
	}
}

func stringsContainsHTTPTokenSpecial(r rune) bool {
	switch r {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}

type HTTPOutcome uint8

const (
	httpOutcomeInvalid HTTPOutcome = iota
	HTTPOutcomeSuccess
	HTTPOutcomeRetryable
	HTTPOutcomeTerminal
)

func httpOutcomeNames() [HTTPOutcomeTerminal + 1]string {
	return [...]string{
		HTTPOutcomeSuccess:   "success",
		HTTPOutcomeRetryable: "retryable",
		HTTPOutcomeTerminal:  "terminal",
	}
}

func (o HTTPOutcome) String() string {
	if o.IsValid() {
		return httpOutcomeNames()[o]
	}
	return ""
}

func (o HTTPOutcome) IsValid() bool {
	return o > httpOutcomeInvalid && int(o) < len(httpOutcomeNames()) && httpOutcomeNames()[o] != ""
}

func (o HTTPOutcome) Validate() error {
	if !o.IsValid() {
		return fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
	}
	return nil
}

func (o HTTPOutcome) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}

func ParseHTTPOutcome(token string) (HTTPOutcome, error) {
	for outcome := HTTPOutcomeSuccess; int(outcome) < len(httpOutcomeNames()); outcome++ {
		if httpOutcomeNames()[outcome] == token {
			return outcome, nil
		}
	}
	return httpOutcomeInvalid, fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
}

func (o *HTTPOutcome) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
	}
	parsed, err := ParseHTTPOutcome(token)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

type BackoffPolicy struct {
	Base        NanosecondsDuration
	Max         NanosecondsDuration
	MaxAttempts uint64
}

func DefaultHTTPBackoffPolicy() BackoffPolicy {
	return BackoffPolicy{
		Base:        NewNanosecondsDuration(HTTPRetryDefaultBase),
		Max:         NewNanosecondsDuration(HTTPRetryDefaultMaximum),
		MaxAttempts: HTTPRetryDefaultMaximumAttempts,
	}
}

// HTTPRetryPolicy is the transport-neutral retry contract used by Go clients,
// browser policy projections, and durable delivery adapters. Status
// classification remains in HTTPRetryableStatusCodes so no consumer can widen
// the retry set independently.
type HTTPRetryPolicy struct {
	Backoff           BackoffPolicy
	MaximumRetryAfter NanosecondsDuration
	RetryWaitLimit    NanosecondsDuration
}

func DefaultHTTPRetryPolicy() HTTPRetryPolicy {
	return HTTPRetryPolicy{
		Backoff:           DefaultHTTPBackoffPolicy(),
		MaximumRetryAfter: NewNanosecondsDuration(BackoffMaxDuration),
		RetryWaitLimit:    NewNanosecondsDuration(HTTPRetryDefaultMaximum),
	}
}

func httpRetryPolicyError(cause error) error {
	return fmt.Errorf(ErrFmtHTTPRetryPolicy, errors.Join(ErrExchangeContract, cause))
}

func (p HTTPRetryPolicy) Validate() error {
	if err := p.Backoff.Validate(); err != nil {
		return httpRetryPolicyError(err)
	}
	if err := validateHTTPRetryDuration(p.MaximumRetryAfter); err != nil {
		return httpRetryPolicyError(err)
	}
	if err := validateHTTPRetryDuration(p.RetryWaitLimit); err != nil {
		return httpRetryPolicyError(err)
	}
	if p.RetryWaitLimit.Duration() > p.MaximumRetryAfter.Duration() {
		return httpRetryPolicyError(ErrFoundationContract)
	}
	return nil
}

func validateHTTPRetryDuration(value NanosecondsDuration) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.IsZero() || value.Duration() > BackoffMaxDuration {
		return ErrExchangeContract
	}
	return nil
}

func (p BackoffPolicy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf(ErrFmtBackoffAttempts, ErrFoundationContract)
	}
	if err := p.Base.Validate(); err != nil {
		return fmt.Errorf(ErrFmtBackoffWindow, err)
	}
	if err := p.Max.Validate(); err != nil {
		return fmt.Errorf(ErrFmtBackoffWindow, err)
	}
	base := p.Base.Duration()
	maximum := p.Max.Duration()
	if base <= 0 || maximum < base || maximum > BackoffMaxDuration {
		return fmt.Errorf(ErrFmtBackoffWindow, ErrFoundationContract)
	}
	return nil
}

func (p BackoffPolicy) Delay(attempt uint64, jitterFrac float64) (NanosecondsDuration, error) {
	if err := p.Validate(); err != nil {
		return NanosecondsDuration{}, err
	}
	if attempt >= p.MaxAttempts {
		return NanosecondsDuration{}, fmt.Errorf(ErrFmtBackoffAttempts, ErrFoundationContract)
	}
	if !(jitterFrac >= 0 && jitterFrac <= 1) {
		return NanosecondsDuration{}, fmt.Errorf(ErrFmtBackoffWindow, ErrFoundationContract)
	}
	ceiling := p.Base.Duration()
	maximum := p.Max.Duration()
	for i := uint64(0); i < attempt && ceiling < maximum; i++ {
		ceiling *= 2
	}
	if ceiling > maximum {
		ceiling = maximum
	}
	return NewNanosecondsDuration(time.Duration(jitterFrac * float64(ceiling))), nil
}
