package exchange

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type responseFixture struct {
	Value string `json:"value"`
}

func (f responseFixture) Validate() error {
	if f.Value == "" {
		return core.ErrFoundationContract
	}
	return nil
}

func TestServerResponseValidateExhaustiveStatusEnvelopeRetryMatrix(t *testing.T) {
	t.Parallel()

	success := successResponseEnvelope(t)
	failure := failureResponseEnvelope(t)
	retry := core.HTTPRetryDirective{Delay: core.NewNanosecondsDuration(time.Second)}
	for raw := core.HTTPStatusMinimum; raw <= core.HTTPStatusMaximum; raw++ {
		status, err := core.NewHTTPStatusCode(raw)
		if err != nil {
			t.Fatalf("NewHTTPStatusCode(%d) error = %v, want nil", raw, err)
		}
		retryable, err := core.HTTPStatusIsRetryable(status)
		if err != nil {
			t.Fatalf("HTTPStatusIsRetryable(%d) error = %v, want nil", raw, err)
		}
		for _, hasSuccess := range []bool{false, true} {
			for _, hasRetry := range []bool{false, true} {
				envelope := failure
				if hasSuccess {
					envelope = success
				}
				directive := core.HTTPRetryDirective{}
				if hasRetry {
					directive = retry
				}
				response := ServerResponse[responseFixture]{Status: status, Envelope: envelope, RetryAfter: directive}
				wantValid := raw >= 200 && raw <= 299 && hasSuccess && !hasRetry || (raw < 200 || raw > 299) && !hasSuccess && retryable == hasRetry
				gotErr := response.Validate()
				if (gotErr == nil) != wantValid {
					t.Fatalf("ServerResponse{status=%d,success=%v,retry=%v}.Validate() error = %v, want valid %v", raw, hasSuccess, hasRetry, gotErr, wantValid)
				}
				if !wantValid && !errors.Is(gotErr, core.ErrExchangeResponse) {
					t.Fatalf("ServerResponse{status=%d,success=%v,retry=%v}.Validate() error = %v, want %v", raw, hasSuccess, hasRetry, gotErr, core.ErrExchangeResponse)
				}
			}
		}
	}
}

func TestWriteJSONLayerTriad(t *testing.T) {
	t.Parallel()

	t.Run("positive success writes exact validated envelope", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		response := ServerResponse[responseFixture]{Status: core.HTTPStatusOK, Envelope: successResponseEnvelope(t)}
		gotErr := WriteJSON(recorder, response)
		if gotErr != nil {
			t.Fatalf("WriteJSON() error = %v, want nil", gotErr)
		}
		if got := recorder.Code; got != core.HTTPStatusOK.Int() {
			t.Fatalf("response status = %d, want %d", got, core.HTTPStatusOK.Int())
		}
		if got := recorder.Header().Get(core.HTTPHeaderContentType); got != core.HTTPContentTypeJSON {
			t.Fatalf("Content-Type = %q, want %q", got, core.HTTPContentTypeJSON)
		}
		if got := recorder.Header().Get(core.HTTPHeaderContentLength); got != strconv.Itoa(recorder.Body.Len()) {
			t.Fatalf("Content-Length = %q, want %q", got, strconv.Itoa(recorder.Body.Len()))
		}
		decoded, err := core.DecodeStrictJSON[core.APIEnvelope[responseFixture]](recorder.Body.Bytes())
		if err != nil {
			t.Fatalf("DecodeStrictJSON(response) error = %v, want nil", err)
		}
		if err := decoded.ValidateSuccess(); err != nil || decoded.Data == nil || decoded.Data.Value != "accepted" {
			t.Fatalf("written envelope = %+v, validation error = %v", decoded, err)
		}
	})

	t.Run("negative short write preserves typed failure", func(t *testing.T) {
		t.Parallel()

		writer := &shortResponseWriter{header: make(http.Header)}
		response := ServerResponse[responseFixture]{Status: core.HTTPStatusOK, Envelope: successResponseEnvelope(t)}
		gotErr := WriteJSON(writer, response)
		if !errors.Is(gotErr, core.ErrExchangeWrite) || !errors.Is(gotErr, io.ErrShortWrite) {
			t.Fatalf("WriteJSON() error = %v, want %v and %v", gotErr, core.ErrExchangeWrite, io.ErrShortWrite)
		}
	})

	t.Run("neutral terminal rejection omits retry header", func(t *testing.T) {
		t.Parallel()

		status, err := core.NewHTTPStatusCode(http.StatusBadRequest)
		if err != nil {
			t.Fatalf("NewHTTPStatusCode() error = %v, want nil", err)
		}
		recorder := httptest.NewRecorder()
		response := ServerResponse[responseFixture]{Status: status, Envelope: failureResponseEnvelope(t)}
		gotErr := WriteJSON(recorder, response)
		if gotErr != nil {
			t.Fatalf("WriteJSON() error = %v, want nil", gotErr)
		}
		if got := recorder.Header().Get(core.HTTPHeaderRetryAfter); got != "" {
			t.Fatalf("Retry-After = %q, want absent", got)
		}
	})
}

func successResponseEnvelope(t *testing.T) core.APIEnvelope[responseFixture] {
	t.Helper()
	body := responseFixture{Value: "accepted"}
	return core.APIEnvelope[responseFixture]{RequestID: core.NewAPIRequestID("request-success"), Data: &body}
}

func failureResponseEnvelope(t *testing.T) core.APIEnvelope[responseFixture] {
	t.Helper()
	code, err := core.ParseAPICode(core.APICodeTokenServiceUnavailable)
	if err != nil {
		t.Fatalf("ParseAPICode() error = %v, want nil", err)
	}
	body := core.APIErrorBody{Code: code, Message: "temporarily unavailable"}
	return core.APIEnvelope[responseFixture]{RequestID: core.NewAPIRequestID("request-failure"), Error: &body}
}

type shortResponseWriter struct {
	header http.Header
}

func (w *shortResponseWriter) Header() http.Header {
	return w.header
}

func (*shortResponseWriter) WriteHeader(int) {}

func (*shortResponseWriter) Write(body []byte) (int, error) {
	return len(body) - 1, nil
}
