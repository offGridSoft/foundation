package core

import (
	"errors"
	"testing"
)

// Contract ratchet: production was correct before the consolidation, but the
// 2xx success range was restated as raw literals in three places. This pins
// the single core-owned gate and its parity with the classifier's
// expected-status acceptance.
func TestHTTPStatusIsSuccessExhaustiveDomainAndClassifierParity(t *testing.T) {
	t.Parallel()

	for raw := HTTPStatusMinimum; raw <= HTTPStatusMaximum; raw++ {
		status := HTTPStatusCode(raw)
		gotSuccess, err := HTTPStatusIsSuccess(status)
		if err != nil {
			t.Fatalf("HTTPStatusIsSuccess(%d) error = %v, want nil inside valid domain", raw, err)
		}
		wantSuccess := raw >= 200 && raw <= 299
		if gotSuccess != wantSuccess {
			t.Fatalf("HTTPStatusIsSuccess(%d) = %v, want %v", raw, gotSuccess, wantSuccess)
		}
		_, classifyErr := ClassifyHTTPStatus(HTTPStatusOK, status)
		if gotSuccess != (classifyErr == nil) {
			t.Fatalf("classifier status acceptance for %d = %v, want parity with success gate %v", raw, classifyErr, gotSuccess)
		}
	}
	for _, status := range []HTTPStatusCode{HTTPStatusCodeUnknown, 99, 600, 65535} {
		if _, err := HTTPStatusIsSuccess(status); !errors.Is(err, ErrExchangeContract) {
			t.Fatalf("HTTPStatusIsSuccess(%d) error = %v, want %v outside valid domain", status, err, ErrExchangeContract)
		}
	}
	if HTTPStatusSuccessMinimum != 200 || HTTPStatusSuccessMaximum != 299 {
		t.Fatalf("success range = [%d, %d], want pinned [200, 299]", HTTPStatusSuccessMinimum, HTTPStatusSuccessMaximum)
	}
}
