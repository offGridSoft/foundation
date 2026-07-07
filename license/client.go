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

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

const (
	CheckInResponseByteCap = 1 << 16
	CheckInBudget          = 3 * time.Second
	CheckInMinInterval     = 24 * time.Hour
	WarnWindow             = 7 * 24 * time.Hour
	ClockSkewAllowance     = 60 * time.Minute
)

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
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			wait := maxDuration(policy.Delay(attempt-1, c.jitterFraction()), retryAfter)
			if err := sleepContext(ctx, wait); err != nil {
				return CheckInResponse[B]{}, transportError(err)
			}
		}
		response, outcome, nextRetryAfter, err := c.attempt(ctx, body)
		retryAfter = nextRetryAfter
		switch outcome {
		case core.HTTPOutcomeSuccess:
			if verr := response.Validate(); verr != nil {
				return CheckInResponse[B]{}, verr
			}
			return response, nil
		case core.HTTPOutcomeRetryable:
			lastErr = err
		default:
			return CheckInResponse[B]{}, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract)
	}
	return CheckInResponse[B]{}, lastErr
}

func (c Client[P, B]) attempt(
	ctx context.Context,
	body []byte,
) (CheckInResponse[B], core.HTTPOutcome, time.Duration, error) {
	request, err := c.buildRequest(ctx, body)
	if err != nil {
		return CheckInResponse[B]{}, core.HTTPOutcomeTerminal, 0, err
	}
	reply, err := c.HTTP.Do(request)
	if err != nil {
		return CheckInResponse[B]{}, core.HTTPOutcomeRetryable, 0, transportError(err)
	}
	if reply == nil {
		return CheckInResponse[B]{}, core.HTTPOutcomeTerminal, 0, fmt.Errorf(ErrFmtCheckInTransport, core.ErrLicenseContract)
	}
	defer func() { _ = reply.Body.Close() }()
	outcome := core.ClassifyHTTPStatus(reply.StatusCode)
	if outcome != core.HTTPOutcomeSuccess {
		retryAfter := parseRetryAfter(reply.Header.Get(core.HTTPHeaderRetryAfter), time.Now().UTC())
		return CheckInResponse[B]{}, outcome, retryAfter, fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
	}
	decoded, err := readResponse[B](reply)
	if err != nil {
		return CheckInResponse[B]{}, core.HTTPOutcomeTerminal, 0, err
	}
	return decoded, core.HTTPOutcomeSuccess, 0, nil
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
	data, err := io.ReadAll(io.LimitReader(reply.Body, CheckInResponseByteCap+1))
	if err != nil {
		return CheckInResponse[B]{}, fmt.Errorf(ErrFmtCheckInTransport, err)
	}
	if len(data) > CheckInResponseByteCap {
		return CheckInResponse[B]{}, fmt.Errorf(ErrFmtCheckInResponse, core.ErrLicenseContract)
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

func (c Client[P, B]) jitterFraction() float64 {
	if c.Jitter != nil {
		return c.Jitter()
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 0
	}
	return float64(binary.BigEndian.Uint64(raw[:])>>11) / float64(uint64(1)<<53)
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

func parseRetryAfter(header string, now time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		return retryAfterSeconds(seconds)
	}
	if _, err := strconv.ParseUint(header, 10, 64); err == nil {
		return CheckInBackoff.Max
	}
	when, err := http.ParseTime(header)
	if err != nil || !when.After(now) {
		return 0
	}
	return clampRetryAfter(when.Sub(now))
}

func clampRetryAfter(d time.Duration) time.Duration {
	if d > CheckInBackoff.Max {
		return CheckInBackoff.Max
	}
	return d
}

func retryAfterSeconds(seconds int64) time.Duration {
	if seconds <= 0 {
		return 0
	}
	if seconds > int64(CheckInBackoff.Max/time.Second) {
		return CheckInBackoff.Max
	}
	return time.Duration(seconds) * time.Second
}
