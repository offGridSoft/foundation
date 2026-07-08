package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

func testProductVersion(t *testing.T) core.ProductVersion {
	t.Helper()
	version, err := core.ParseProductVersion("1.0.0")
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(OffgridAPICallKeyHeader); got != "public-call-key" {
			t.Fatalf("api call key header = %q", got)
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
	if response.Lease == nil {
		t.Fatalf("response.Lease = nil, want signed seat lease")
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

// witness:waiver test/parallel/default -- mutates package-global CheckInBackoff to prove retry sequencing with nanosecond waits.
func TestRetryLoopRetriesThenAcceptsEnvelope(t *testing.T) {
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
	restoreBackoff := setTestBackoff(core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 3})
	defer restoreBackoff()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
	}

	if _, err := client.Do(context.Background(), testBugCheckIn(t)); err != nil {
		t.Fatalf("client.Do retry success: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

// witness:waiver test/parallel/default -- mutates package-global CheckInBackoff to keep retry-exhaustion proof deterministic and fast.
func TestRetryExhaustionCarriesServerRetryAfter(t *testing.T) {
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
	restoreBackoff := setTestBackoff(core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1})
	defer restoreBackoff()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, SeatLeaseBody]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
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

func TestCheckInBudgetCoversWorstCaseRetryAfterSchedule(t *testing.T) {
	t.Parallel()
	minBudget := time.Duration(CheckInBackoff.MaxAttempts-1) * CheckInBackoff.Max
	if CheckInBudget < minBudget {
		t.Fatalf("CheckInBudget = %s, want at least %s", CheckInBudget, minBudget)
	}
}

// witness:waiver test/parallel/default -- reads package-global CheckInBackoff while serial retry tests mutate it.
func TestRetryAfterClampedToBackoffMax(t *testing.T) {
	now := time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC)
	for _, header := range []string{
		"8640000",
		"9223372036854775807",
		now.Add(24 * time.Hour).Format(http.TimeFormat),
	} {
		if got := parseRetryAfter(header, now); got.Wait != CheckInBackoff.Max {
			t.Fatalf("parseRetryAfter(%q).Wait = %s, want %s", header, got.Wait, CheckInBackoff.Max)
		}
	}
	for _, header := range []string{
		"-1",
		now.Add(-time.Hour).Format(http.TimeFormat),
	} {
		if got := parseRetryAfter(header, now); got.Wait != 0 || got.Hint != 0 {
			t.Fatalf("parseRetryAfter(%q) = %+v, want zero decision", header, got)
		}
	}
}

func setTestBackoff(policy core.BackoffPolicy) func() {
	previous := CheckInBackoff
	CheckInBackoff = policy
	return func() { CheckInBackoff = previous }
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

func testSeatLeaseBody(t *testing.T) SeatLeaseBody {
	t.Helper()
	return SeatLeaseBody{
		IssuedAt:           testTime(1782302400000000000),
		TokenExpiresAt:     testTime(1784894400000000000),
		CheckInAfterAt:     testTime(1783252800000000000),
		CheckInByAt:        testTime(1784030400000000000),
		Schema:             SchemaBugSeatLease,
		DeveloperKeyID:     testDeveloperKeyID(t),
		DeviceFingerprint:  testDeviceFingerprint(t),
		WriteGraceDuration: core.NewNanosecondsDuration(72 * time.Hour),
		Plan:               SeatPlanStandard,
	}
}

func testBugCheckIn(t *testing.T) BugCheckIn {
	t.Helper()
	return BugCheckIn{
		Schema:            SchemaBugCheckIn,
		DeveloperKey:      testDeveloperKey(t),
		DeviceFingerprint: testDeviceFingerprint(t),
		DeviceLabel:       testDeviceLabel(t),
		BinaryVersion:     testProductVersion(t),
		BinarySHA256:      testSHA256(t),
		Platform:          PlatformDarwinARM64,
		Usage: BugUsage{
			Schema:      SchemaBugUsage,
			WindowStart: testTime(1782302400000000000),
			WindowEnd:   testTime(1782302401000000000),
		},
	}
}
