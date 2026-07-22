package exchange

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type fixedClock struct {
	now core.UnixNanoTime
}

func (c fixedClock) Now() core.UnixNanoTime {
	return c.now
}

type fixedJitter struct {
	fraction float64
}

func (j fixedJitter) Fraction() float64 {
	return j.fraction
}

type recordingWaiter struct {
	delays []core.NanosecondsDuration
	mu     sync.Mutex
}

func (w *recordingWaiter) Wait(ctx context.Context, delay core.NanosecondsDuration) error {
	if ctx == nil {
		return core.ErrNilContext
	}
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
	}
	w.mu.Lock()
	w.delays = append(w.delays, delay)
	w.mu.Unlock()
	return nil
}

func (w *recordingWaiter) Values() []core.NanosecondsDuration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]core.NanosecondsDuration(nil), w.delays...)
}

func TestSendJSONClientServerLayerTriad(t *testing.T) {
	t.Parallel()

	t.Run("positive idempotent request retries server busy then succeeds", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Uint64
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			received, err := ReceiveJSON[receiveFixture](request,
				core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent},
				ServerPolicy{RequestBodyLimit: core.NewByteCount(receiveFixtureBodyLimit)},
			)
			if err != nil || received.Body != (receiveFixture{Name: "peachfuzz", Count: 7}) || received.IdempotencyKey.IsZero() {
				t.Errorf("ReceiveJSON() = (%+v, %v), want typed request", received, err)
				return
			}
			if attempts.Add(1) < 3 {
				response := ServerResponse[responseFixture]{
					Status:     core.HTTPStatusServiceUnavailable,
					Envelope:   failureResponseEnvelope(t),
					RetryAfter: core.HTTPRetryDirective{Delay: core.NewNanosecondsDuration(time.Nanosecond)},
				}
				if err := WriteJSON(writer, response); err != nil {
					t.Errorf("WriteJSON(retry) error = %v, want nil", err)
				}
				return
			}
			if err := WriteJSON(writer, ServerResponse[responseFixture]{Status: core.HTTPStatusOK, Envelope: successResponseEnvelope(t)}); err != nil {
				t.Errorf("WriteJSON(success) error = %v, want nil", err)
			}
		}))
		defer server.Close()

		waiter := &recordingWaiter{}
		client := Client{HTTP: server.Client(), Clock: fixedClock{now: core.UnixNanoTimeFromInt64(1)}, Jitter: fixedJitter{fraction: 1}, Waiter: waiter}
		request := idempotentClientRequest(t, server.URL)
		got, gotErr := SendJSON[receiveFixture, responseFixture](context.Background(), client, request, clientTestPolicy(3))
		if gotErr != nil {
			t.Fatalf("SendJSON() error = %v, server attempts = %d, endpoint = %s, want nil", gotErr, attempts.Load(), request.Endpoint.String())
		}
		if got.Attempts != 3 || got.Status != core.HTTPStatusOK || got.Envelope.Data == nil || got.Envelope.Data.Value != "accepted" {
			t.Fatalf("SendJSON() = %+v, want third-attempt accepted response", got)
		}
		if gotAttempts := attempts.Load(); gotAttempts != 3 {
			t.Fatalf("server attempts = %d, want 3", gotAttempts)
		}
		gotDelays := waiter.Values()
		if len(gotDelays) != 2 || gotDelays[0].Duration() != time.Second || gotDelays[1].Duration() != time.Second {
			t.Fatalf("retry delays = %v, want two one-second server-rounded waits", gotDelays)
		}
	})

	t.Run("negative single attempt mutation never replays retryable response", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Uint64
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			attempts.Add(1)
			response := ServerResponse[responseFixture]{
				Status:     core.HTTPStatusTooManyRequests,
				Envelope:   failureResponseEnvelope(t),
				RetryAfter: core.HTTPRetryDirective{Delay: core.NewNanosecondsDuration(time.Second)},
			}
			if err := WriteJSON(writer, response); err != nil {
				t.Errorf("WriteJSON() error = %v, want nil", err)
			}
		}))
		defer server.Close()

		endpoint, err := core.ParseAPIEndpoint(server.URL + "/v1/exchange")
		if err != nil {
			t.Fatalf("ParseAPIEndpoint() error = %v, want nil", err)
		}
		body := receiveFixture{Name: "single", Count: 1}
		request := Request[receiveFixture]{
			Body:           &body,
			Endpoint:       endpoint,
			Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt},
			ExpectedStatus: core.HTTPStatusOK,
		}
		client := Client{HTTP: server.Client(), Clock: fixedClock{now: core.UnixNanoTimeFromInt64(1)}, Jitter: fixedJitter{fraction: 1}, Waiter: &recordingWaiter{}}
		got, gotErr := SendJSON[receiveFixture, responseFixture](context.Background(), client, request, clientTestPolicy(4))
		if !errors.Is(gotErr, core.ErrExchangeResponse) || errors.Is(gotErr, core.ErrExchangeRetryExhausted) {
			t.Fatalf("SendJSON() error = %v, want terminal response without retry exhaustion", gotErr)
		}
		if got.Attempts != 1 || attempts.Load() != 1 {
			t.Fatalf("attempts = client %d/server %d, want 1/1", got.Attempts, attempts.Load())
		}
	})

	t.Run("neutral cancelled context sends no request", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Uint64
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			attempts.Add(1)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := Client{HTTP: server.Client()}
		got, gotErr := SendJSON[receiveFixture, responseFixture](ctx, client, idempotentClientRequest(t, server.URL), clientTestPolicy(3))
		if !errors.Is(gotErr, core.ErrExchangeCancelled) {
			t.Fatalf("SendJSON() error = %v, want %v", gotErr, core.ErrExchangeCancelled)
		}
		if got != (Response[responseFixture]{}) {
			t.Fatalf("SendJSON() response = %+v, want zero", got)
		}
		if gotAttempts := attempts.Load(); gotAttempts != 0 {
			t.Fatalf("server attempts = %d, want 0", gotAttempts)
		}
	})
}

func idempotentClientRequest(t *testing.T, baseURL string) Request[receiveFixture] {
	t.Helper()
	endpoint, err := core.ParseAPIEndpoint(baseURL + "/v1/exchange")
	if err != nil {
		t.Fatalf("ParseAPIEndpoint() error = %v, want nil", err)
	}
	key, err := core.ParseHTTPIdempotencyKey("request-peachfuzz-0001")
	if err != nil {
		t.Fatalf("ParseHTTPIdempotencyKey() error = %v, want nil", err)
	}
	body := receiveFixture{Name: "peachfuzz", Count: 7}
	return Request[receiveFixture]{
		Body:           &body,
		Endpoint:       endpoint,
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent, IdempotencyKey: key},
		ExpectedStatus: core.HTTPStatusOK,
	}
}

func clientTestPolicy(maxAttempts uint64) ClientPolicy {
	retry := core.DefaultHTTPRetryPolicy()
	retry.Backoff = core.BackoffPolicy{Base: core.NewNanosecondsDuration(time.Nanosecond), Max: core.NewNanosecondsDuration(time.Nanosecond), MaxAttempts: maxAttempts}
	retry.MaximumRetryAfter = core.NewNanosecondsDuration(time.Minute)
	retry.RetryWaitLimit = core.NewNanosecondsDuration(time.Minute)
	return ClientPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(5 * time.Second),
		RequestBodyLimit:  core.NewByteCount(receiveFixtureBodyLimit),
		ResponseBodyLimit: core.NewByteCount(receiveFixtureBodyLimit),
		Retry:             retry,
		Redirect:          core.HTTPRedirectReject,
	}
}
