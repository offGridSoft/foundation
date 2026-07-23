package exchange

import (
	"errors"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestClientConstructionClosesRetryImplementationSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		http      *http.Client
		name      string
		wantError bool
	}{
		{name: "default client is accepted as the complete public dependency", http: http.DefaultClient},
		{name: "dedicated zero-value HTTP client is accepted", http: &http.Client{}},
		{name: "nil HTTP client is rejected at construction", wantError: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(testCase.http)
			if testCase.wantError {
				if client != (Client{}) || !errors.Is(err, core.ErrExchangeResponse) {
					t.Fatalf("NewClient() = (%+v,%v), want zero/ErrExchangeResponse", client, err)
				}
				return
			}
			if err != nil || client.Validate() != nil {
				t.Fatalf("NewClient() = (%+v,%v), Validate=%v; want valid/nil", client, err, client.Validate())
			}
		})
	}
	if err := (Client{}).Validate(); !errors.Is(err, core.ErrExchangeResponse) {
		t.Fatalf("zero Client.Validate() error = %v, want ErrExchangeResponse", err)
	}
}

func TestClientPolicyValidateHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	baseline := clientTestPolicy(3)
	cases := []struct {
		mutate  func(*ClientPolicy)
		wantErr error
		name    string
	}{
		{name: "baseline accepts ordinary bounded policy"},
		{name: "attempt timeout accepts one nanosecond floor", mutate: func(p *ClientPolicy) { p.AttemptTimeout = core.NewNanosecondsDuration(time.Nanosecond) }},
		{name: "attempt timeout accepts exact twenty four hour ceiling", mutate: func(p *ClientPolicy) { p.AttemptTimeout = core.NewNanosecondsDuration(core.BackoffMaxDuration) }},
		{name: "request body accepts one byte floor", mutate: func(p *ClientPolicy) { p.RequestBodyLimit = core.NewByteCount(1) }},
		{name: "request body accepts exact json ceiling", mutate: func(p *ClientPolicy) { p.RequestBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes) }},
		{name: "response body accepts one byte floor", mutate: func(p *ClientPolicy) { p.ResponseBodyLimit = core.NewByteCount(1) }},
		{name: "response body accepts exact json ceiling", mutate: func(p *ClientPolicy) { p.ResponseBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes) }},
		{name: "retry after accepts one nanosecond floor", mutate: func(p *ClientPolicy) {
			p.Retry.MaximumRetryAfter = core.NewNanosecondsDuration(time.Nanosecond)
			p.Retry.RetryWaitLimit = core.NewNanosecondsDuration(time.Nanosecond)
		}},
		{name: "retry after accepts exact twenty four hour ceiling", mutate: func(p *ClientPolicy) {
			p.Retry.MaximumRetryAfter = core.NewNanosecondsDuration(core.BackoffMaxDuration)
		}},
		{name: "retry wait accepts one nanosecond floor", mutate: func(p *ClientPolicy) { p.Retry.RetryWaitLimit = core.NewNanosecondsDuration(time.Nanosecond) }},
		{name: "retry wait accepts exact retry hint ceiling", mutate: func(p *ClientPolicy) { p.Retry.RetryWaitLimit = p.Retry.MaximumRetryAfter }},
		{name: "backoff accepts one attempt floor", mutate: func(p *ClientPolicy) { p.Retry.Backoff.MaxAttempts = 1 }},
		{name: "backoff accepts maximum attempt domain", mutate: func(p *ClientPolicy) { p.Retry.Backoff.MaxAttempts = math.MaxUint64 }},
		{name: "same origin redirect is explicit and accepted", mutate: func(p *ClientPolicy) { p.Redirect = core.HTTPRedirectSameOrigin }},
		{name: "zero attempt timeout rejects missing deadline", mutate: func(p *ClientPolicy) { p.AttemptTimeout = core.NanosecondsDuration{} }, wantErr: core.ErrExchangeResponse},
		{name: "negative attempt timeout rejects one below floor", mutate: func(p *ClientPolicy) { p.AttemptTimeout = core.NanosecondsDurationFromInt64(-1) }, wantErr: core.ErrExchangeResponse},
		{name: "attempt timeout one nanosecond above ceiling rejects", mutate: func(p *ClientPolicy) {
			p.AttemptTimeout = core.NewNanosecondsDuration(core.BackoffMaxDuration + time.Nanosecond)
		}, wantErr: core.ErrExchangeResponse},
		{name: "attempt timeout maximum duration rejects hostile upper end", mutate: func(p *ClientPolicy) { p.AttemptTimeout = core.NewNanosecondsDuration(time.Duration(math.MaxInt64)) }, wantErr: core.ErrExchangeResponse},
		{name: "zero request body limit rejects missing bound", mutate: func(p *ClientPolicy) { p.RequestBodyLimit = core.ByteCount{} }, wantErr: core.ErrExchangeResponse},
		{name: "request body one byte above json ceiling rejects", mutate: func(p *ClientPolicy) { p.RequestBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1) }, wantErr: core.ErrExchangeBodyLimit},
		{name: "request body maximum count rejects hostile upper end", mutate: func(p *ClientPolicy) { p.RequestBodyLimit = core.NewByteCount(math.MaxUint64) }, wantErr: core.ErrExchangeBodyLimit},
		{name: "zero response body limit rejects missing bound", mutate: func(p *ClientPolicy) { p.ResponseBodyLimit = core.ByteCount{} }, wantErr: core.ErrExchangeResponse},
		{name: "response body one byte above json ceiling rejects", mutate: func(p *ClientPolicy) { p.ResponseBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1) }, wantErr: core.ErrExchangeBodyLimit},
		{name: "response body maximum count rejects hostile upper end", mutate: func(p *ClientPolicy) { p.ResponseBodyLimit = core.NewByteCount(math.MaxUint64) }, wantErr: core.ErrExchangeBodyLimit},
		{name: "zero retry after rejects missing server cap", mutate: func(p *ClientPolicy) { p.Retry.MaximumRetryAfter = core.NanosecondsDuration{} }, wantErr: core.ErrExchangeResponse},
		{name: "negative retry after rejects one below floor", mutate: func(p *ClientPolicy) { p.Retry.MaximumRetryAfter = core.NanosecondsDurationFromInt64(-1) }, wantErr: core.ErrExchangeResponse},
		{name: "retry after one nanosecond above ceiling rejects", mutate: func(p *ClientPolicy) {
			p.Retry.MaximumRetryAfter = core.NewNanosecondsDuration(core.BackoffMaxDuration + time.Nanosecond)
		}, wantErr: core.ErrExchangeResponse},
		{name: "retry after maximum duration rejects hostile upper end", mutate: func(p *ClientPolicy) {
			p.Retry.MaximumRetryAfter = core.NewNanosecondsDuration(time.Duration(math.MaxInt64))
		}, wantErr: core.ErrExchangeResponse},
		{name: "zero retry wait rejects missing foreground cap", mutate: func(p *ClientPolicy) { p.Retry.RetryWaitLimit = core.NanosecondsDuration{} }, wantErr: core.ErrExchangeResponse},
		{name: "negative retry wait rejects one below floor", mutate: func(p *ClientPolicy) { p.Retry.RetryWaitLimit = core.NanosecondsDurationFromInt64(-1) }, wantErr: core.ErrExchangeResponse},
		{name: "retry wait above advertised hint ceiling rejects inverted policy", mutate: func(p *ClientPolicy) {
			p.Retry.RetryWaitLimit = core.NewNanosecondsDuration(p.Retry.MaximumRetryAfter.Duration() + time.Nanosecond)
		}, wantErr: core.ErrExchangeResponse},
		{name: "zero retry policy rejects every missing field", mutate: func(p *ClientPolicy) { p.Retry = core.HTTPRetryPolicy{} }, wantErr: core.ErrExchangeResponse},
		{name: "zero backoff policy rejects every missing field", mutate: func(p *ClientPolicy) { p.Retry.Backoff = core.BackoffPolicy{} }, wantErr: core.ErrExchangeResponse},
		{name: "zero backoff attempts rejects exact floor", mutate: func(p *ClientPolicy) { p.Retry.Backoff.MaxAttempts = 0 }, wantErr: core.ErrExchangeResponse},
		{name: "zero backoff base rejects missing delay", mutate: func(p *ClientPolicy) { p.Retry.Backoff.Base = core.NanosecondsDuration{} }, wantErr: core.ErrExchangeResponse},
		{name: "negative backoff base rejects hostile duration", mutate: func(p *ClientPolicy) { p.Retry.Backoff.Base = core.NanosecondsDurationFromInt64(-1) }, wantErr: core.ErrExchangeResponse},
		{name: "backoff maximum one below base rejects inverted window", mutate: func(p *ClientPolicy) {
			p.Retry.Backoff.Max = core.NewNanosecondsDuration(p.Retry.Backoff.Base.Duration() - time.Nanosecond)
		}, wantErr: core.ErrExchangeResponse},
		{name: "backoff maximum one above global ceiling rejects", mutate: func(p *ClientPolicy) {
			p.Retry.Backoff.Max = core.NewNanosecondsDuration(core.BackoffMaxDuration + time.Nanosecond)
		}, wantErr: core.ErrExchangeResponse},
		{name: "unknown redirect policy rejects zero enum", mutate: func(p *ClientPolicy) { p.Redirect = core.HTTPRedirectUnknown }, wantErr: core.ErrExchangeResponse},
		{name: "future redirect policy rejects maximum enum", mutate: func(p *ClientPolicy) { p.Redirect = core.HTTPRedirectPolicy(math.MaxUint8) }, wantErr: core.ErrExchangeResponse},
		{name: "multiple invalid fields still reject typed", mutate: func(p *ClientPolicy) {
			p.AttemptTimeout = core.NanosecondsDuration{}
			p.RequestBodyLimit = core.ByteCount{}
			p.Redirect = core.HTTPRedirectUnknown
		}, wantErr: core.ErrExchangeResponse},
		{name: "request and response maximums cannot bypass json ceiling", mutate: func(p *ClientPolicy) {
			p.RequestBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1)
			p.ResponseBodyLimit = core.NewByteCount(core.StrictJSONMaxBytes + 1)
		}, wantErr: core.ErrExchangeBodyLimit},
		{name: "all duration zero values reject closed policy", mutate: func(p *ClientPolicy) {
			p.AttemptTimeout = core.NanosecondsDuration{}
			p.Retry.MaximumRetryAfter = core.NanosecondsDuration{}
			p.Retry.RetryWaitLimit = core.NanosecondsDuration{}
			p.Retry.Backoff.Base = core.NanosecondsDuration{}
		}, wantErr: core.ErrExchangeResponse},
		{name: "all duration maximum values reject upper abuse", mutate: func(p *ClientPolicy) {
			maximum := core.NewNanosecondsDuration(time.Duration(math.MaxInt64))
			p.AttemptTimeout = maximum
			p.Retry.MaximumRetryAfter = maximum
			p.Retry.RetryWaitLimit = maximum
			p.Retry.Backoff.Max = maximum
		}, wantErr: core.ErrExchangeResponse},
		{name: "body maximums reject before allocation", mutate: func(p *ClientPolicy) {
			p.RequestBodyLimit = core.NewByteCount(math.MaxUint64)
			p.ResponseBodyLimit = core.NewByteCount(math.MaxUint64)
		}, wantErr: core.ErrExchangeBodyLimit},
		{name: "future redirect plus exhausted backoff rejects", mutate: func(p *ClientPolicy) {
			p.Redirect = core.HTTPRedirectPolicy(math.MaxUint8)
			p.Retry.Backoff.MaxAttempts = 0
		}, wantErr: core.ErrExchangeResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			policy := baseline
			if tc.mutate != nil {
				tc.mutate(&policy)
			}
			gotErr := policy.Validate()
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("ClientPolicy.Validate() error = %v, want %v", gotErr, tc.wantErr)
			}
		})
	}
}

func TestParseRetryAfterHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	nowTime := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	now := core.NewUnixNanoTime(nowTime)
	maximum := core.NewNanosecondsDuration(time.Minute)
	cases := []struct {
		wantErr error
		name    string
		values  []string
		now     core.UnixNanoTime
		maximum core.NanosecondsDuration
		want    time.Duration
	}{
		{name: "absent header leaves backoff authoritative", now: now, maximum: maximum},
		{name: "zero seconds permits immediate backoff", values: []string{"0"}, now: now, maximum: maximum},
		{name: "one second accepts exact delta floor", values: []string{"1"}, now: now, maximum: maximum, want: time.Second},
		{name: "fifty nine seconds accepts one below cap", values: []string{"59"}, now: now, maximum: maximum, want: 59 * time.Second},
		{name: "sixty seconds accepts exact cap", values: []string{"60"}, now: now, maximum: maximum, want: time.Minute},
		{name: "sixty one seconds clamps one above cap", values: []string{"61"}, now: now, maximum: maximum, want: time.Minute},
		{name: "maximum integer seconds clamps without overflow", values: []string{"9223372036854775807"}, now: now, maximum: maximum, want: time.Minute},
		{name: "leading zero delta remains valid rfc digits", values: []string{"01"}, now: now, maximum: maximum, want: time.Second},
		{name: "future date one second accepts exact floor", values: []string{nowTime.Add(time.Second).Format(http.TimeFormat)}, now: now, maximum: maximum, want: time.Second},
		{name: "future date one below cap remains exact", values: []string{nowTime.Add(59 * time.Second).Format(http.TimeFormat)}, now: now, maximum: maximum, want: 59 * time.Second},
		{name: "future date exact cap remains exact", values: []string{nowTime.Add(time.Minute).Format(http.TimeFormat)}, now: now, maximum: maximum, want: time.Minute},
		{name: "future date one above cap clamps", values: []string{nowTime.Add(time.Minute + time.Second).Format(http.TimeFormat)}, now: now, maximum: maximum, want: time.Minute},
		{name: "past date becomes neutral hint", values: []string{nowTime.Add(-time.Second).Format(http.TimeFormat)}, now: now, maximum: maximum},
		{name: "present date becomes neutral hint", values: []string{nowTime.Format(http.TimeFormat)}, now: now, maximum: maximum},
		{name: "one nanosecond maximum clamps one second delta", values: []string{"1"}, now: now, maximum: core.NewNanosecondsDuration(time.Nanosecond), want: time.Nanosecond},
		{name: "one second maximum accepts one second delta", values: []string{"1"}, now: now, maximum: core.NewNanosecondsDuration(time.Second), want: time.Second},
		{name: "twenty four hour maximum accepts exact boundary", values: []string{"86400"}, now: now, maximum: core.NewNanosecondsDuration(core.BackoffMaxDuration), want: core.BackoffMaxDuration},
		{name: "twenty four hour maximum clamps one above", values: []string{"86401"}, now: now, maximum: core.NewNanosecondsDuration(core.BackoffMaxDuration), want: core.BackoffMaxDuration},
		{name: "date cap uses injected clock not wall clock", values: []string{nowTime.Add(30 * time.Second).Format(http.TimeFormat)}, now: now, maximum: maximum, want: 30 * time.Second},
		{name: "zero unix nanoseconds is a valid injected clock", values: []string{time.Unix(1, 0).UTC().Format(http.TimeFormat)}, now: core.UnixNanoTimeFromInt64(0), maximum: maximum, want: time.Second},
		{name: "empty header value rejects missing digits", values: []string{""}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "duplicate headers reject ambiguous precedence", values: []string{"1", "2"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "leading space rejects noncanonical delta", values: []string{" 1"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "trailing space rejects noncanonical delta", values: []string{"1 "}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "negative delta rejects sign", values: []string{"-1"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "positive sign rejects noncanonical delta", values: []string{"+1"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "fractional delta rejects noninteger hint", values: []string{"1.5"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "hex delta rejects alternate number grammar", values: []string{"0x10"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "alphabetic hint rejects unknown format", values: []string{"later"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "integer above uint63 rejects parser overflow", values: []string{"9223372036854775808"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "maximum uint string rejects parser overflow", values: []string{"18446744073709551615"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "truncated http date rejects", values: []string{"Wed, 22 Jul 2026 12:00"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "invalid date day rejects", values: []string{"Wed, 32 Jul 2026 12:00:00 GMT"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "invalid date month rejects", values: []string{"Wed, 22 Xxx 2026 12:00:00 GMT"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "zero clock rejects date arithmetic", values: []string{nowTime.Add(time.Second).Format(http.TimeFormat)}, now: core.UnixNanoTime{}, maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "negative clock rejects date arithmetic", values: []string{nowTime.Add(time.Second).Format(http.TimeFormat)}, now: core.UnixNanoTimeFromInt64(-1), maximum: maximum, wantErr: core.ErrExchangeResponse},
		{name: "zero maximum rejects unbounded parser policy", values: []string{"1"}, now: now, maximum: core.NanosecondsDuration{}, wantErr: core.ErrExchangeResponse},
		{name: "negative maximum rejects hostile parser policy", values: []string{"1"}, now: now, maximum: core.NanosecondsDurationFromInt64(-1), wantErr: core.ErrExchangeResponse},
		{name: "maximum one above global cap rejects parser policy", values: []string{"1"}, now: now, maximum: core.NewNanosecondsDuration(core.BackoffMaxDuration + time.Nanosecond), wantErr: core.ErrExchangeResponse},
		{name: "newline injection rejects both number and date grammar", values: []string{"1\n2"}, now: now, maximum: maximum, wantErr: core.ErrExchangeResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := parseRetryAfter(tc.values, tc.now, tc.maximum)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("parseRetryAfter() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr == nil && got.Duration() != tc.want {
				t.Fatalf("parseRetryAfter() = %v, want %v", got.Duration(), tc.want)
			}
		})
	}
}
