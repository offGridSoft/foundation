package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	HTTPHeaderContentType               = "Content-Type"
	HTTPHeaderRetryAfter                = "Retry-After"
	HTTPContentTypeJSON                 = "application/json"
	URLSchemeHTTP                       = "http"
	URLSchemeHTTPS                      = "https"
	HostLocalhost                       = "localhost"
	HTTPSURLDefaultMaxRunes             = 2048
	HTTPHeaderNameMaxRunes              = 256
	HTTPHeaderValueMaxRunes             = 8192
	BackoffMaxDuration                  = 24 * time.Hour
	HTTPStatusOK                        = 200
	HTTPStatusNotFound                  = 404
	HTTPStatusTooManyRequests           = 429
	HTTPStatusInternalServerError       = 500
	HTTPStatusServerErrorUpperExclusive = 600
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

func ClassifyHTTPStatus(status int) HTTPOutcome {
	// Offgrid API success responses are pinned to HTTP 200 with an APIEnvelope.
	switch {
	case status == HTTPStatusOK:
		return HTTPOutcomeSuccess
	case status == HTTPStatusTooManyRequests:
		return HTTPOutcomeRetryable
	case status >= HTTPStatusInternalServerError && status < HTTPStatusServerErrorUpperExclusive:
		return HTTPOutcomeRetryable
	default:
		return HTTPOutcomeTerminal
	}
}

type BackoffPolicy struct {
	Base        time.Duration
	Max         time.Duration
	MaxAttempts int
}

func (p BackoffPolicy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf(ErrFmtBackoffAttempts, ErrFoundationContract)
	}
	if p.Base <= 0 || p.Max < p.Base || p.Max > BackoffMaxDuration {
		return fmt.Errorf(ErrFmtBackoffWindow, ErrFoundationContract)
	}
	return nil
}

func (p BackoffPolicy) Delay(attempt int, jitterFrac float64) (time.Duration, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	if attempt < 0 || attempt >= p.MaxAttempts {
		return 0, fmt.Errorf(ErrFmtBackoffAttempts, ErrFoundationContract)
	}
	if !(jitterFrac >= 0 && jitterFrac <= 1) {
		return 0, fmt.Errorf(ErrFmtBackoffWindow, ErrFoundationContract)
	}
	ceiling := p.Base
	for i := 0; i < attempt && ceiling < p.Max; i++ {
		ceiling *= 2
	}
	if ceiling > p.Max {
		ceiling = p.Max
	}
	return time.Duration(jitterFrac * float64(ceiling)), nil
}
