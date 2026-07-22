package core

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestHTTPMethodExhaustiveDomain(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name  string
		wire  string
		value HTTPMethod
	}{
		{name: "get is safe retrieval", value: HTTPMethodGet, wire: "GET"},
		{name: "head is safe metadata retrieval", value: HTTPMethodHead, wire: "HEAD"},
		{name: "post can create with replay identity", value: HTTPMethodPost, wire: "POST"},
		{name: "put is idempotent replacement", value: HTTPMethodPut, wire: "PUT"},
		{name: "patch is partial mutation", value: HTTPMethodPatch, wire: "PATCH"},
		{name: "delete is idempotent removal", value: HTTPMethodDelete, wire: "DELETE"},
		{name: "options is safe capability retrieval", value: HTTPMethodOptions, wire: "OPTIONS"},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if gotErr := tc.value.Validate(); gotErr != nil {
				t.Fatalf("HTTPMethod.Validate() error = %v, want nil", gotErr)
			}
			if got := tc.value.String(); got != tc.wire {
				t.Fatalf("HTTPMethod.String() = %q, want %q", got, tc.wire)
			}
			gotParsed, gotErr := ParseHTTPMethod(tc.wire)
			if gotErr != nil || gotParsed != tc.value {
				t.Fatalf("ParseHTTPMethod() = (%v, %v), want (%v, nil)", gotParsed, gotErr, tc.value)
			}
			gotJSON, gotErr := json.Marshal(tc.value)
			if gotErr != nil {
				t.Fatalf("json.Marshal(HTTPMethod) error = %v, want nil", gotErr)
			}
			var gotRoundTrip HTTPMethod
			gotErr = json.Unmarshal(gotJSON, &gotRoundTrip)
			if gotErr != nil || gotRoundTrip != tc.value {
				t.Fatalf("HTTPMethod JSON round trip = (%v, %v), want (%v, nil)", gotRoundTrip, gotErr, tc.value)
			}
		})
	}
	for raw := 0; raw <= math.MaxUint8; raw++ {
		method := HTTPMethod(raw)
		wantValid := wantHTTPMethodValid(method)
		gotErr := method.Validate()
		if (gotErr == nil) != wantValid {
			t.Fatalf("HTTPMethod(%d).Validate() error = %v, want valid %v", raw, gotErr, wantValid)
		}
		if !wantValid && !errors.Is(gotErr, ErrExchangeContract) {
			t.Fatalf("HTTPMethod(%d).Validate() error = %v, want %v", raw, gotErr, ErrExchangeContract)
		}
	}
}

func wantHTTPMethodValid(method HTTPMethod) bool {
	switch method {
	case HTTPMethodGet, HTTPMethodHead, HTTPMethodPost, HTTPMethodPut, HTTPMethodPatch, HTTPMethodDelete, HTTPMethodOptions:
		return true
	default:
		return false
	}
}

func TestHTTPMethodRejectsHostileWireTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wire string
	}{
		{name: "empty token is missing", wire: ""},
		{name: "lowercase token is noncanonical", wire: "get"},
		{name: "mixed case token is noncanonical", wire: "Post"},
		{name: "leading space changes token", wire: " GET"},
		{name: "trailing space changes token", wire: "GET "},
		{name: "embedded tab splits token", wire: "GE\tT"},
		{name: "embedded newline attempts injection", wire: "GET\nPOST"},
		{name: "unknown future token is closed", wire: "CONNECT"},
		{name: "trace token is outside application domain", wire: "TRACE"},
		{name: "oversized token cannot consume unbounded work", wire: strings.Repeat("G", HTTPMethodTokenMaxRunes+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := ParseHTTPMethod(tc.wire)
			if got != HTTPMethodUnknown || !errors.Is(gotErr, ErrExchangeContract) {
				t.Fatalf("ParseHTTPMethod(%q) = (%v, %v), want (%v, %v)", tc.wire, got, gotErr, HTTPMethodUnknown, ErrExchangeContract)
			}
		})
	}
}

func TestHTTPPolicyEnumsExhaustiveDomain(t *testing.T) {
	t.Parallel()

	for raw := 0; raw <= math.MaxUint8; raw++ {
		redirect := HTTPRedirectPolicy(raw)
		gotRedirectErr := redirect.Validate()
		wantRedirectValid := redirect == HTTPRedirectReject || redirect == HTTPRedirectSameOrigin
		if (gotRedirectErr == nil) != wantRedirectValid {
			t.Fatalf("HTTPRedirectPolicy(%d).Validate() error = %v, want valid %v", raw, gotRedirectErr, wantRedirectValid)
		}
		if !wantRedirectValid && !errors.Is(gotRedirectErr, ErrExchangeContract) {
			t.Fatalf("HTTPRedirectPolicy(%d).Validate() error = %v, want %v", raw, gotRedirectErr, ErrExchangeContract)
		}
		if wantRedirectValid {
			encoded, marshalErr := json.Marshal(redirect)
			var decoded HTTPRedirectPolicy
			unmarshalErr := json.Unmarshal(encoded, &decoded)
			if marshalErr != nil || unmarshalErr != nil || decoded != redirect {
				t.Fatalf("HTTPRedirectPolicy(%d) JSON round trip = (%v, %v, %v), want (%v, nil, nil)", raw, decoded, marshalErr, unmarshalErr, redirect)
			}
		}

		replay := HTTPReplaySafety(raw)
		gotReplayErr := replay.Validate()
		wantReplayValid := replay == HTTPReplaySafe || replay == HTTPReplayIdempotent || replay == HTTPReplaySingleAttempt
		if (gotReplayErr == nil) != wantReplayValid {
			t.Fatalf("HTTPReplaySafety(%d).Validate() error = %v, want valid %v", raw, gotReplayErr, wantReplayValid)
		}
		if !wantReplayValid && !errors.Is(gotReplayErr, ErrExchangeContract) {
			t.Fatalf("HTTPReplaySafety(%d).Validate() error = %v, want %v", raw, gotReplayErr, ErrExchangeContract)
		}
		if wantReplayValid {
			encoded, marshalErr := json.Marshal(replay)
			var decoded HTTPReplaySafety
			unmarshalErr := json.Unmarshal(encoded, &decoded)
			if marshalErr != nil || unmarshalErr != nil || decoded != replay {
				t.Fatalf("HTTPReplaySafety(%d) JSON round trip = (%v, %v, %v), want (%v, nil, nil)", raw, decoded, marshalErr, unmarshalErr, replay)
			}
		}
		gotAutomatic, gotAutomaticErr := replay.AllowsAutomaticReplay()
		if wantReplayValid != (gotAutomaticErr == nil) {
			t.Fatalf("HTTPReplaySafety(%d).AllowsAutomaticReplay() error = %v, want valid %v", raw, gotAutomaticErr, wantReplayValid)
		}
		wantAutomatic := replay == HTTPReplaySafe || replay == HTTPReplayIdempotent
		if gotAutomatic != wantAutomatic {
			t.Fatalf("HTTPReplaySafety(%d).AllowsAutomaticReplay() = %v, want %v", raw, gotAutomatic, wantAutomatic)
		}
	}
}

func TestHTTPMediaTypeExhaustiveDomain(t *testing.T) {
	t.Parallel()

	for raw := 0; raw <= math.MaxUint8; raw++ {
		mediaType := HTTPMediaType(raw)
		wantValid := mediaType >= HTTPMediaTypeJSON && mediaType <= HTTPMediaTypeTimestampReply
		gotErr := mediaType.Validate()
		if (gotErr == nil) != wantValid {
			t.Fatalf("HTTPMediaType(%d).Validate() error = %v, want valid %v", raw, gotErr, wantValid)
		}
		if wantValid && mediaType.String() == "" {
			t.Fatalf("HTTPMediaType(%d).String() empty for valid enum", raw)
		}
		if wantValid {
			encoded, marshalErr := json.Marshal(mediaType)
			var decoded HTTPMediaType
			unmarshalErr := json.Unmarshal(encoded, &decoded)
			if marshalErr != nil || unmarshalErr != nil || decoded != mediaType {
				t.Fatalf("HTTPMediaType(%d) JSON round trip = (%v, %v, %v), want (%v, nil, nil)", raw, decoded, marshalErr, unmarshalErr, mediaType)
			}
		}
		if !wantValid && (!errors.Is(gotErr, ErrExchangeContract) || mediaType.String() != "") {
			t.Fatalf("HTTPMediaType(%d) = string %q error %v, want empty/%v", raw, mediaType.String(), gotErr, ErrExchangeContract)
		}
	}
}

func TestHTTPPolicyEnumWireRejectsHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		run  func() (bool, error)
		name string
	}{
		{name: "redirect empty token rejects without mutation", run: func() (bool, error) {
			value := HTTPRedirectReject
			err := json.Unmarshal([]byte(`""`), &value)
			return value == HTTPRedirectReject, err
		}},
		{name: "redirect future token rejects without mutation", run: func() (bool, error) {
			value := HTTPRedirectReject
			err := json.Unmarshal([]byte(`"follow_all"`), &value)
			return value == HTTPRedirectReject, err
		}},
		{name: "redirect numeric type confusion rejects without mutation", run: func() (bool, error) {
			value := HTTPRedirectReject
			err := json.Unmarshal([]byte(`1`), &value)
			return value == HTTPRedirectReject, err
		}},
		{name: "replay empty token rejects without mutation", run: func() (bool, error) {
			value := HTTPReplaySafe
			err := json.Unmarshal([]byte(`""`), &value)
			return value == HTTPReplaySafe, err
		}},
		{name: "replay future token rejects without mutation", run: func() (bool, error) {
			value := HTTPReplaySafe
			err := json.Unmarshal([]byte(`"buffer_maybe"`), &value)
			return value == HTTPReplaySafe, err
		}},
		{name: "replay boolean type confusion rejects without mutation", run: func() (bool, error) {
			value := HTTPReplaySafe
			err := json.Unmarshal([]byte(`true`), &value)
			return value == HTTPReplaySafe, err
		}},
		{name: "media empty token rejects without mutation", run: func() (bool, error) {
			value := HTTPMediaTypeJSON
			err := json.Unmarshal([]byte(`""`), &value)
			return value == HTTPMediaTypeJSON, err
		}},
		{name: "media parameterized value rejects implicit grammar", run: func() (bool, error) {
			value := HTTPMediaTypeJSON
			err := json.Unmarshal([]byte(`"application/json; charset=utf-8"`), &value)
			return value == HTTPMediaTypeJSON, err
		}},
		{name: "media case drift rejects noncanonical token", run: func() (bool, error) {
			value := HTTPMediaTypeJSON
			err := json.Unmarshal([]byte(`"Application/JSON"`), &value)
			return value == HTTPMediaTypeJSON, err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			unchanged, gotErr := tc.run()
			if !unchanged || !errors.Is(gotErr, ErrExchangeContract) {
				t.Fatalf("hostile enum decode = unchanged %v, error %v; want true/%v", unchanged, gotErr, ErrExchangeContract)
			}
		})
	}
}

func TestHTTPRequestSemanticsExhaustiveLattice(t *testing.T) {
	t.Parallel()

	methods := []HTTPMethod{
		HTTPMethodGet,
		HTTPMethodHead,
		HTTPMethodPost,
		HTTPMethodPut,
		HTTPMethodPatch,
		HTTPMethodDelete,
		HTTPMethodOptions,
	}
	replayModes := []HTTPReplaySafety{
		HTTPReplaySafe,
		HTTPReplayIdempotent,
		HTTPReplaySingleAttempt,
	}
	key, err := ParseHTTPIdempotencyKey("request-2026-0001")
	if err != nil {
		t.Fatalf("ParseHTTPIdempotencyKey() error = %v, want nil", err)
	}
	keys := []HTTPIdempotencyKey{{}, key}
	for _, method := range methods {
		for _, replay := range replayModes {
			for _, candidateKey := range keys {
				contract := HTTPRequestSemantics{Method: method, Replay: replay, IdempotencyKey: candidateKey}
				wantValid := wantHTTPRequestSemanticsValid(method, replay, !candidateKey.IsZero())
				gotErr := contract.Validate()
				if (gotErr == nil) != wantValid {
					t.Fatalf("HTTPRequestSemantics{%s,%s,key=%v}.Validate() error = %v, want valid %v", method, replay, !candidateKey.IsZero(), gotErr, wantValid)
				}
				if !wantValid && !errors.Is(gotErr, ErrExchangeContract) {
					t.Fatalf("HTTPRequestSemantics{%s,%s,key=%v}.Validate() error = %v, want %v", method, replay, !candidateKey.IsZero(), gotErr, ErrExchangeContract)
				}
			}
		}
	}
}

func TestHTTPRouteSemanticsExhaustiveLattice(t *testing.T) {
	t.Parallel()

	methods := []HTTPMethod{
		HTTPMethodGet,
		HTTPMethodHead,
		HTTPMethodPost,
		HTTPMethodPut,
		HTTPMethodPatch,
		HTTPMethodDelete,
		HTTPMethodOptions,
	}
	replayModes := []HTTPReplaySafety{
		HTTPReplaySafe,
		HTTPReplayIdempotent,
		HTTPReplaySingleAttempt,
	}
	for _, method := range methods {
		for _, replay := range replayModes {
			contract := HTTPRouteSemantics{Method: method, Replay: replay}
			wantValid := wantHTTPRouteSemanticsValid(method, replay)
			gotErr := contract.Validate()
			if (gotErr == nil) != wantValid {
				t.Fatalf("HTTPRouteSemantics{%s,%s}.Validate() error = %v, want valid %v", method, replay, gotErr, wantValid)
			}
			if !wantValid && !errors.Is(gotErr, ErrExchangeContract) {
				t.Fatalf("HTTPRouteSemantics{%s,%s}.Validate() error = %v, want %v", method, replay, gotErr, ErrExchangeContract)
			}
		}
	}
}

func wantHTTPRouteSemanticsValid(method HTTPMethod, replay HTTPReplaySafety) bool {
	switch method {
	case HTTPMethodGet, HTTPMethodHead, HTTPMethodOptions:
		return replay == HTTPReplaySafe
	case HTTPMethodPut, HTTPMethodDelete:
		return replay == HTTPReplayIdempotent
	case HTTPMethodPost, HTTPMethodPatch:
		return replay == HTTPReplayIdempotent || replay == HTTPReplaySingleAttempt
	default:
		return false
	}
}

func wantHTTPRequestSemanticsValid(method HTTPMethod, replay HTTPReplaySafety, hasKey bool) bool {
	switch method {
	case HTTPMethodGet, HTTPMethodHead, HTTPMethodOptions:
		return (replay == HTTPReplaySafe || replay == HTTPReplaySingleAttempt) && !hasKey
	case HTTPMethodPut, HTTPMethodDelete:
		return (replay == HTTPReplayIdempotent || replay == HTTPReplaySingleAttempt) && !hasKey
	case HTTPMethodPost, HTTPMethodPatch:
		return replay == HTTPReplaySingleAttempt && !hasKey || replay == HTTPReplayIdempotent && hasKey
	default:
		return false
	}
}

func TestHTTPStatusCodeAndClassifierExhaustiveDomain(t *testing.T) {
	t.Parallel()

	for raw := -1; raw <= 700; raw++ {
		got, gotErr := NewHTTPStatusCode(raw)
		wantValid := raw >= HTTPStatusMinimum && raw <= HTTPStatusMaximum
		if (gotErr == nil) != wantValid {
			t.Fatalf("NewHTTPStatusCode(%d) = (%v, %v), want valid %v", raw, got, gotErr, wantValid)
		}
		if !wantValid && !errors.Is(gotErr, ErrExchangeContract) {
			t.Fatalf("NewHTTPStatusCode(%d) error = %v, want %v", raw, gotErr, ErrExchangeContract)
		}
		if wantValid {
			encoded, marshalErr := json.Marshal(got)
			var decoded HTTPStatusCode
			unmarshalErr := json.Unmarshal(encoded, &decoded)
			if marshalErr != nil || unmarshalErr != nil || decoded != got {
				t.Fatalf("HTTPStatusCode(%d) JSON round trip = (%v, %v, %v), want (%v, nil, nil)", raw, decoded, marshalErr, unmarshalErr, got)
			}
		}
	}
	for _, raw := range []int{math.MinInt, math.MaxInt} {
		got, gotErr := NewHTTPStatusCode(raw)
		if got != HTTPStatusCodeUnknown || !errors.Is(gotErr, ErrExchangeContract) {
			t.Fatalf("NewHTTPStatusCode(%d) = (%v, %v), want (%v, %v)", raw, got, gotErr, HTTPStatusCodeUnknown, ErrExchangeContract)
		}
	}

	for expectedRaw := 200; expectedRaw <= 299; expectedRaw++ {
		expected, err := NewHTTPStatusCode(expectedRaw)
		if err != nil {
			t.Fatalf("NewHTTPStatusCode(%d) error = %v, want nil", expectedRaw, err)
		}
		for statusRaw := HTTPStatusMinimum; statusRaw <= HTTPStatusMaximum; statusRaw++ {
			status, err := NewHTTPStatusCode(statusRaw)
			if err != nil {
				t.Fatalf("NewHTTPStatusCode(%d) error = %v, want nil", statusRaw, err)
			}
			got, gotErr := ClassifyHTTPStatus(status, expected)
			want := wantHTTPOutcome(statusRaw, expectedRaw)
			if gotErr != nil || got != want {
				t.Fatalf("ClassifyHTTPStatus(%d, %d) = (%v, %v), want (%v, nil)", statusRaw, expectedRaw, got, gotErr, want)
			}
		}
	}
}

func TestHTTPStatusCodeWireRejectsHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wire string
	}{
		{name: "one below protocol floor", wire: `99`},
		{name: "one above protocol ceiling", wire: `600`},
		{name: "negative status", wire: `-1`},
		{name: "maximum integer", wire: `9223372036854775807`},
		{name: "string type confusion", wire: `"200"`},
		{name: "boolean type confusion", wire: `true`},
		{name: "null type confusion", wire: `null`},
		{name: "fractional numeric status", wire: `200.5`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value := HTTPStatusOK
			gotErr := json.Unmarshal([]byte(tc.wire), &value)
			if value != HTTPStatusOK || !errors.Is(gotErr, ErrExchangeContract) {
				t.Fatalf("HTTPStatusCode.UnmarshalJSON(%s) = (%v, %v), want (%v, %v)", tc.wire, value, gotErr, HTTPStatusOK, ErrExchangeContract)
			}
		})
	}
}

func TestHTTPRetryableStatusCodesAreCanonicalAndImmutable(t *testing.T) {
	t.Parallel()

	want := [HTTPRetryableStatusCount]HTTPStatusCode{
		HTTPStatusRequestTimeout,
		HTTPStatusTooEarly,
		HTTPStatusTooManyRequests,
		HTTPStatusInternalServerError,
		HTTPStatusBadGateway,
		HTTPStatusServiceUnavailable,
		HTTPStatusGatewayTimeout,
	}
	got := HTTPRetryableStatusCodes()
	if got != want {
		t.Fatalf("HTTPRetryableStatusCodes() = %v, want %v", got, want)
	}
	got[0] = HTTPStatusOK
	gotAgain := HTTPRetryableStatusCodes()
	if gotAgain != want {
		t.Fatalf("HTTPRetryableStatusCodes() after caller mutation = %v, want %v", gotAgain, want)
	}
}

func wantHTTPOutcome(status, expected int) HTTPOutcome {
	if status == expected {
		return HTTPOutcomeSuccess
	}
	for _, retryable := range HTTPRetryableStatusCodes() {
		if status == retryable.Int() {
			return HTTPOutcomeRetryable
		}
	}
	return HTTPOutcomeTerminal
}
