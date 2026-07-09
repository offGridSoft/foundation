package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"encoding/json"
	"github.com/offGridSoft/foundation/v2026/core"
)

func testProductVersion(t *testing.T) core.ProductVersion {
	t.Helper()
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func testSHA256(t *testing.T) core.SHA256Hex {
	t.Helper()
	sum, err := core.ParseSHA256Hex(strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}

func testDeveloperKey(t *testing.T) DeveloperKey {
	t.Helper()
	key, err := ParseDeveloperKey("OGS-DEV-123456789012")
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testDeviceLabel(t *testing.T) DeviceLabel {
	t.Helper()
	label, err := ParseDeviceLabel("developer laptop")
	if err != nil {
		t.Fatal(err)
	}
	return label
}

func testAPICallKey(t *testing.T) APICallKey {
	t.Helper()
	key, err := ParseAPICallKey("public-call-key")
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestClientDecodesOffgridEnvelope(t *testing.T) {
	t.Parallel()
	keyID, publicKey, signature := signedSeatLeaseParts(t)
	lease := &core.Signed[SeatLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      testSeatLeaseBody(t),
	}
	handlerErr := make(chan error, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(OffgridAPICallKeyHeader); got != "public-call-key" {
			handlerErr <- fmt.Errorf("api call key header = %q, want public-call-key", got)
			http.Error(w, "bad api key", http.StatusBadRequest)
			return
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-1"),
			Data: &CheckInResponse[SeatLeaseBody]{
				Granted: true,
				Refusal: RefusalNone,
				Lease:   lease,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		APIKey:   testAPICallKey(t),
	}

	response, err := client.Do(context.Background(), testBugCheckIn(t))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerErr:
		t.Fatal(err)
	default:
	}
	if response.Lease == nil {
		t.Fatalf("response.Lease = nil, want signed seat lease")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Lease.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestClientSendsFirstCheckInWithoutUsage(t *testing.T) {
	t.Parallel()
	keyID, _, signature := signedSeatLeaseParts(t)
	lease := &core.Signed[SeatLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      testSeatLeaseBody(t),
	}
	handlerErr := make(chan error, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload BugCheckIn
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			handlerErr <- err
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}
		if !payload.Usage.IsZero() {
			handlerErr <- fmt.Errorf("first check-in usage = %+v, want zero usage", payload.Usage)
			http.Error(w, "bad usage", http.StatusBadRequest)
			return
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-first"),
			Data: &CheckInResponse[SeatLeaseBody]{
				Granted: true,
				Refusal: RefusalNone,
				Lease:   lease,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	payload := testBugCheckIn(t)
	payload.Usage = BugUsage{}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
	}

	if _, err := client.Do(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerErr:
		t.Fatal(err)
	default:
	}
}

func TestWitnessClientDecodesSubscriptionEnvelope(t *testing.T) {
	t.Parallel()
	keyID, publicKey, signature := signedSubscriptionLeaseParts(t)
	lease := &core.Signed[SubscriptionLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      testSubscriptionLeaseBody(),
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SubscriptionLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-witness"),
			Data: &CheckInResponse[SubscriptionLeaseBody]{
				Granted: true,
				Refusal: RefusalNone,
				Lease:   lease,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridWitnessCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[WitnessCheckIn, SubscriptionLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
	}

	response, err := client.Do(context.Background(), testWitnessCheckIn(t))
	if err != nil {
		t.Fatal(err)
	}
	if response.Lease == nil {
		t.Fatalf("response.Lease = nil, want signed subscription lease")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Lease.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsRawResponseWithoutEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(CheckInResponse[SeatLeaseBody]{
			Granted:     false,
			Refusal:     RefusalPaymentRequired,
			Remediation: "update payment",
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
	}

	if _, err := client.Do(context.Background(), testBugCheckIn(t)); err == nil {
		t.Fatalf("client accepted raw response without Offgrid API envelope")
	}
}

func TestClientDecodesTerminalErrorEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-denied"),
			Error: &core.APIErrorBody{
				Code:    core.APICodeForbidden,
				Message: "payment required",
				Tip:     "update billing",
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
	}

	_, err = client.Do(context.Background(), testBugCheckIn(t))
	var apiErr CheckInAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("client.Do error = %v, want CheckInAPIError", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Body.Code != core.APICodeForbidden || apiErr.Body.Message != "payment required" {
		t.Fatalf("CheckInAPIError = %+v, want decoded status/code/message", apiErr)
	}
	if !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("CheckInAPIError errors.Is ErrLicenseContract = false: %v", err)
	}
}

func TestRetryLoopRetriesThenAcceptsEnvelope(t *testing.T) {
	t.Parallel()
	keyID, _, signature := signedSeatLeaseParts(t)
	lease := &core.Signed[SeatLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      testSeatLeaseBody(t),
	}
	var attempts int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
				RequestID: core.NewAPIRequestID("req-retry"),
				Error: &core.APIErrorBody{
					Code:    core.APICodeServiceUnavailable,
					Message: "try again",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-ok"),
			Data: &CheckInResponse[SeatLeaseBody]{
				Granted: true,
				Refusal: RefusalNone,
				Lease:   lease,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 3},
	}

	if _, err := client.Do(context.Background(), testBugCheckIn(t)); err != nil {
		t.Fatalf("client.Do retry success: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestWitnessClientRetryThenAcceptsSubscriptionEnvelope(t *testing.T) {
	t.Parallel()
	keyID, _, signature := signedSubscriptionLeaseParts(t)
	lease := &core.Signed[SubscriptionLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      testSubscriptionLeaseBody(),
	}
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SubscriptionLeaseBody]]{
				RequestID: core.NewAPIRequestID("req-witness-retry"),
				Error: &core.APIErrorBody{
					Code:    core.APICodeServiceUnavailable,
					Message: "try again",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SubscriptionLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-witness-ok"),
			Data: &CheckInResponse[SubscriptionLeaseBody]{
				Granted: true,
				Refusal: RefusalNone,
				Lease:   lease,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridWitnessCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[WitnessCheckIn, SubscriptionLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 2},
	}

	if _, err := client.Do(context.Background(), testWitnessCheckIn(t)); err != nil {
		t.Fatalf("witness client retry success: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRetryExhaustionCarriesServerRetryAfter(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		w.Header().Set(core.HTTPHeaderRetryAfter, "600")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[CheckInResponse[SeatLeaseBody]]{
			RequestID: core.NewAPIRequestID("req-later"),
			Error: &core.APIErrorBody{
				Code:    core.APICodeServiceUnavailable,
				Message: "slow down",
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
	}

	_, err = client.Do(context.Background(), testBugCheckIn(t))
	var retryErr CheckInRetryExhaustedError
	if !errors.As(err, &retryErr) {
		t.Fatalf("client.Do error = %v, want CheckInRetryExhaustedError", err)
	}
	if retryErr.RetryAfter != 10*time.Minute {
		t.Fatalf("RetryAfter = %s, want 10m", retryErr.RetryAfter)
	}
	var apiErr CheckInAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("client.Do error = %v, want nested CheckInAPIError", err)
	}
	if apiErr.RetryAfter != 10*time.Minute {
		t.Fatalf("CheckInAPIError.RetryAfter = %s, want 10m", apiErr.RetryAfter)
	}
}

func TestRetryExhaustedTransportErrorCarriesLicenseIdentity(t *testing.T) {
	t.Parallel()
	endpoint, err := ParseCheckInEndpoint("https://api.offgridsoftware.ca" + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     &http.Client{Transport: failingRoundTripper{}},
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
	}

	_, err = client.Do(context.Background(), testBugCheckIn(t))
	var retryErr CheckInRetryExhaustedError
	if !errors.As(err, &retryErr) {
		t.Fatalf("client.Do error = %v, want CheckInRetryExhaustedError", err)
	}
	if !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("client.Do error = %v, want ErrLicenseContract identity", err)
	}
}

func TestClientValidateChecksAPIKeyAndBackoffTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*Client[BugCheckIn, SeatLeaseBody])
		name    string
		wantErr bool
	}{
		{name: "valid api key and backoff", mutate: func(*Client[BugCheckIn, SeatLeaseBody]) {}},
		{name: "invalid backoff with valid api key", wantErr: true, mutate: func(c *Client[BugCheckIn, SeatLeaseBody]) {
			c.Backoff.MaxAttempts = 0
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := Client[BugCheckIn, SeatLeaseBody]{
				HTTP:     &http.Client{},
				Endpoint: BugCheckInEndpoint,
				APIKey:   testAPICallKey(t),
				Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
			}
			tc.mutate(&client)
			err := client.Validate()
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) && !errors.Is(err, core.ErrFoundationContract) {
					t.Fatalf("Client.Validate() error = %v, want contract identity", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Client.Validate() = %v", err)
			}
		})
	}
}

func TestJitterFractionZeroFailsToMaxDelay(t *testing.T) {
	t.Parallel()
	client := Client[BugCheckIn, SeatLeaseBody]{Jitter: func() float64 { return 0 }}
	if got := client.jitterFraction(); got != 1 {
		t.Fatalf("jitterFraction zero = %v, want 1", got)
	}
	if got := conservativeJitterFraction(-0.1); got != 1 {
		t.Fatalf("conservativeJitterFraction negative = %v, want 1", got)
	}
	if got := conservativeJitterFraction(1.1); got != 1 {
		t.Fatalf("conservativeJitterFraction >1 = %v, want 1", got)
	}
	if got := CheckInBackoff.Delay(0, client.jitterFraction()); got != CheckInBackoff.Base {
		t.Fatalf("zero jitter delay = %s, want %s", got, CheckInBackoff.Base)
	}
}

func TestCheckInResponseHostileCombinations(t *testing.T) {
	t.Parallel()
	lease := &core.Signed[SeatLeaseBody]{
		KeyID:     mustSigningKeyID(t),
		Signature: make([]byte, ed25519.SignatureSize),
		Body:      testSeatLeaseBody(t),
	}
	for _, tc := range []struct {
		name     string
		response CheckInResponse[SeatLeaseBody]
	}{
		{name: "granted missing lease", response: CheckInResponse[SeatLeaseBody]{Granted: true, Refusal: RefusalNone}},
		{name: "granted with refusal", response: CheckInResponse[SeatLeaseBody]{
			Granted: true,
			Refusal: RefusalPaymentRequired,
			Lease:   lease,
		}},
		{name: "granted with remediation", response: CheckInResponse[SeatLeaseBody]{
			Granted:     true,
			Refusal:     RefusalNone,
			Remediation: "pay",
			Lease:       lease,
		}},
		{name: "refused with lease", response: CheckInResponse[SeatLeaseBody]{
			Refusal:     RefusalPaymentRequired,
			Remediation: "pay",
			Lease:       lease,
		}},
		{name: "refused none refusal", response: CheckInResponse[SeatLeaseBody]{
			Refusal:     RefusalNone,
			Remediation: "pay",
		}},
		{name: "refused blank remediation", response: CheckInResponse[SeatLeaseBody]{
			Refusal:     RefusalPaymentRequired,
			Remediation: " \t ",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.response.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("CheckInResponse.Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestCheckInResponseMalformedLeaseCarriesLicenseIdentity(t *testing.T) {
	t.Parallel()
	response := CheckInResponse[SeatLeaseBody]{
		Granted: true,
		Refusal: RefusalNone,
		Lease: &core.Signed[SeatLeaseBody]{
			KeyID:     mustSigningKeyID(t),
			Signature: nil,
			Body:      testSeatLeaseBody(t),
		},
	}
	if err := response.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("CheckInResponse.Validate() error = %v, want ErrLicenseContract", err)
	}
}

func TestClientBoundaryErrorsCarryLicenseIdentityTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "body read failure", run: func() error {
			_, err := readCappedResponseBody(failingReader{})
			return err
		}},
		{name: "success envelope decode failure", run: func() error {
			reply := &http.Response{Body: io.NopCloser(strings.NewReader("<html>"))}
			_, err := readResponse[SeatLeaseBody](reply)
			return err
		}},
		{name: "failure envelope decode failure", run: func() error {
			reply := &http.Response{Body: io.NopCloser(strings.NewReader("<html>"))}
			return readFailureResponse[SeatLeaseBody](reply, http.StatusBadGateway, 0)
		}},
		{name: "request build failure", run: func() error {
			client := Client[BugCheckIn, SeatLeaseBody]{
				Endpoint: CheckInEndpoint{value: "https://%zz"},
			}
			_, err := client.buildRequest(context.Background(), nil)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("%s error = %v, want ErrLicenseContract", tc.name, err)
			}
		})
	}
}

func TestCheckInBudgetCoversWorstCaseRetryAfterSchedule(t *testing.T) {
	t.Parallel()
	minBudget := time.Duration(CheckInBackoff.MaxAttempts-1) * CheckInBackoff.Max
	if CheckInBudget < minBudget {
		t.Fatalf("CheckInBudget = %s, want at least %s", CheckInBudget, minBudget)
	}
}

func TestRetryAfterClampedToBackoffMax(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC)
	maxWait := 17 * time.Millisecond
	for _, header := range []string{
		"8640000",
		"9223372036854775807",
		now.Add(24 * time.Hour).Format(http.TimeFormat),
	} {
		if got := parseRetryAfter(header, now, maxWait); got.Wait != maxWait {
			t.Fatalf("parseRetryAfter(%q).Wait = %s, want %s", header, got.Wait, maxWait)
		}
	}
	for _, header := range []string{
		"-1",
		now.Add(-time.Hour).Format(http.TimeFormat),
	} {
		if got := parseRetryAfter(header, now, maxWait); got.Wait != 0 || got.Hint != 0 {
			t.Fatalf("parseRetryAfter(%q) = %+v, want zero decision", header, got)
		}
	}
}

func signedSeatLeaseParts(t *testing.T) (core.SigningKeyID, core.Ed25519PublicKeyHex, []byte) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatal(err)
	}
	publicHex, err := core.NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	body := testSeatLeaseBody(t)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicHex, ed25519.Sign(private, message)
}

func signedSubscriptionLeaseParts(t *testing.T) (core.SigningKeyID, core.Ed25519PublicKeyHex, []byte) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatal(err)
	}
	publicHex, err := core.NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	body := testSubscriptionLeaseBody()
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicHex, ed25519.Sign(private, message)
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func mustSigningKeyID(t *testing.T) core.SigningKeyID {
	t.Helper()
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatal(err)
	}
	return keyID
}

func testSeatLeaseBody(t *testing.T) SeatLeaseBody {
	t.Helper()
	window, err := BuildLeaseWindow(testTime(1782302400000000000), mustTestSeatOffer(t, SeatPlanStandard), 0)
	if err != nil {
		t.Fatal(err)
	}
	return SeatLeaseBody{
		IssuedAt:           window.IssuedAt,
		PaidUntil:          window.PaidUntil,
		TokenExpiresAt:     window.TokenExpiresAt,
		CheckInAfterAt:     window.CheckInAfterAt,
		CheckInByAt:        window.CheckInByAt,
		Schema:             core.SchemaBugSeatLease,
		DeveloperKeyID:     testDeveloperKeyID(t),
		DeviceFingerprint:  testDeviceFingerprint(t),
		WriteGraceDuration: window.WriteGraceDuration,
		Plan:               SeatPlanStandard,
		BillingPeriod:      BillingPeriodFourWeeks,
	}
}

func mustTestSeatOffer(t *testing.T, plan SeatPlan) Offer {
	t.Helper()
	offer, err := OfferForSeatPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	return offer
}

func testBugCheckIn(t *testing.T) BugCheckIn {
	t.Helper()
	return BugCheckIn{
		Schema:            core.SchemaBugCheckIn,
		DeveloperKey:      testDeveloperKey(t),
		DeviceFingerprint: testDeviceFingerprint(t),
		DeviceLabel:       testDeviceLabel(t),
		BinaryVersion:     testProductVersion(t),
		BinarySHA256:      testSHA256(t),
		Platform:          core.PlatformDarwinARM64,
		Usage: BugUsage{
			Schema:      core.SchemaBugUsage,
			WindowStart: testTime(1782302400000000000),
			WindowEnd:   testTime(1782302401000000000),
		},
	}
}

func testWitnessCheckIn(t *testing.T) WitnessCheckIn {
	t.Helper()
	token, err := ParseAccountToken("acct_123456789")
	if err != nil {
		t.Fatal(err)
	}
	return WitnessCheckIn{
		Schema:            core.SchemaWitnessCheckIn,
		DeviceFingerprint: testDeviceFingerprint(t),
		BinaryVersion:     testProductVersion(t),
		BinarySHA256:      testSHA256(t),
		Platform:          core.PlatformDarwinARM64,
		AccountToken:      token,
	}
}
