package core

import (
	"fmt"
	"time"
	"unicode"

	json "github.com/goccy/go-json"
)

const (
	HTTPHeaderContentType = "Content-Type"
	HTTPHeaderRetryAfter  = "Retry-After"
	HTTPContentTypeJSON   = "application/json"
)

func ValidateHTTPHeaderName(value string) error {
	if value == "" {
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

var httpOutcomeNames = [...]string{
	HTTPOutcomeSuccess:   "success",
	HTTPOutcomeRetryable: "retryable",
	HTTPOutcomeTerminal:  "terminal",
}

func (o HTTPOutcome) String() string {
	if o.IsValid() {
		return httpOutcomeNames[o]
	}
	return ""
}

func (o HTTPOutcome) IsValid() bool {
	return o > httpOutcomeInvalid && int(o) < len(httpOutcomeNames) && httpOutcomeNames[o] != ""
}

func (o HTTPOutcome) MarshalJSON() ([]byte, error) {
	if !o.IsValid() {
		return nil, fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
	}
	return json.Marshal(o.String())
}

func (o *HTTPOutcome) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
	}
	for outcome := HTTPOutcomeSuccess; int(outcome) < len(httpOutcomeNames); outcome++ {
		if httpOutcomeNames[outcome] == token {
			*o = outcome
			return nil
		}
	}
	return fmt.Errorf(ErrFmtHTTPOutcome, ErrFoundationContract)
}

func ClassifyHTTPStatus(status int) HTTPOutcome {
	// Offgrid API success responses are pinned to HTTP 200 with an APIEnvelope.
	switch {
	case status == 200:
		return HTTPOutcomeSuccess
	case status == 429:
		return HTTPOutcomeRetryable
	case status >= 500 && status <= 599:
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
	if p.Base <= 0 || p.Max < p.Base {
		return fmt.Errorf(ErrFmtBackoffWindow, ErrFoundationContract)
	}
	return nil
}

func (p BackoffPolicy) Delay(attempt int, jitterFrac float64) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	ceiling := p.Base
	for i := 0; i < attempt && ceiling < p.Max; i++ {
		ceiling *= 2
	}
	if ceiling > p.Max {
		ceiling = p.Max
	}
	return time.Duration(clampUnit(jitterFrac) * float64(ceiling))
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
