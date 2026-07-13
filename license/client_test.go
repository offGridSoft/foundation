package license

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	grant, keyID, publicKey := signedBugGrant(t)
	handlerErr := make(chan error, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(OffgridAPICallKeyHeader); got != "public-call-key" {
			handlerErr <- fmt.Errorf("api call key header = %q, want public-call-key", got)
			http.Error(w, "bad api key", http.StatusBadRequest)
			return
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-1"),
			Data: &BugCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		APIKey:   testAPICallKey(t),
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
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
	if response.Grant == nil {
		t.Fatalf("response.Grant = nil, want signed Bug grant")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Grant.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestClientSendsFirstCheckInWithoutUsage(t *testing.T) {
	t.Parallel()
	grant, keyID, publicKey := signedBugGrant(t)
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
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-first"),
			Data: &BugCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
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
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
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
	grant, keyID, publicKey := signedWitnessGrant(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[WitnessCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-witness"),
			Data: &WitnessCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridWitnessCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[WitnessCheckIn, WitnessCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
	}

	response, err := client.Do(context.Background(), testWitnessCheckIn(t))
	if err != nil {
		t.Fatal(err)
	}
	if response.Grant == nil {
		t.Fatalf("response.Grant = nil, want signed Witness grant")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Grant.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsForgedLeaseSignatureHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		kind forgedLeaseKind
	}{
		{name: "bug forged seat lease rejected", kind: forgedLeaseKindSeat},
		{name: "witness forged subscription lease rejected", kind: forgedLeaseKindSubscription},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := runForgedLeaseSignatureCheckIn(t, tc.kind); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("forged lease client.Do error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

type forgedLeaseKind uint8

const (
	forgedLeaseKindSeat forgedLeaseKind = iota + 1
	forgedLeaseKindSubscription
)

func runForgedLeaseSignatureCheckIn(t *testing.T, kind forgedLeaseKind) error {
	t.Helper()
	switch kind {
	case forgedLeaseKindSeat:
		return runForgedSeatLeaseCheckIn(t)
	case forgedLeaseKindSubscription:
		return runForgedSubscriptionLeaseCheckIn(t)
	default:
		return core.ErrLicenseContract
	}
}

func runForgedSeatLeaseCheckIn(t *testing.T) error {
	t.Helper()
	grant, keyID, publicKey := signedBugGrant(t)
	grant.Lease.Signature = testSignatureHex(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-forged-seat"),
			Data: &BugCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
	}
	_, err = client.Do(context.Background(), testBugCheckIn(t))
	return err
}

func runForgedSubscriptionLeaseCheckIn(t *testing.T) error {
	t.Helper()
	grant, keyID, publicKey := signedWitnessGrant(t)
	grant.Lease.Signature = testSignatureHex(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[WitnessCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-forged-subscription"),
			Data: &WitnessCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridWitnessCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[WitnessCheckIn, WitnessCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
	}
	_, err = client.Do(context.Background(), testWitnessCheckIn(t))
	return err
}

func TestClientRejectsRawResponseWithoutEnvelope(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(BugCheckInResponse{Decision: CheckInDecision{
			Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment,
		}})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  testClientKeyring(t),
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
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
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
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Keyring:  testClientKeyring(t),
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

func TestClientRejectsCredentialRedirectHostile(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(OffgridAPICallKeyHeader) == "" {
			http.Error(w, "missing key", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "https://evil.example/check-in", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		APIKey:   testAPICallKey(t),
		Keyring:  testClientKeyring(t),
	}
	if _, err := client.Do(context.Background(), testBugCheckIn(t)); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("client.Do redirect error = %v, want ErrLicenseContract", err)
	}
}

func TestRetryLoopRetriesThenAcceptsEnvelope(t *testing.T) {
	t.Parallel()
	grant, keyID, publicKey := signedBugGrant(t)
	var attempts int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
				RequestID: core.NewAPIRequestID("req-retry"),
				Error: &core.APIErrorBody{
					Code:    core.APICodeServiceUnavailable,
					Message: "try again",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-ok"),
			Data: &BugCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 3},
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
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
	grant, keyID, publicKey := signedWitnessGrant(t)
	attempts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(core.APIEnvelope[WitnessCheckInResponse]{
				RequestID: core.NewAPIRequestID("req-witness-retry"),
				Error: &core.APIErrorBody{
					Code:    core.APICodeServiceUnavailable,
					Message: "try again",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[WitnessCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-witness-ok"),
			Data: &WitnessCheckInResponse{
				Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
				Grant:    grant,
			},
		})
	}))
	defer server.Close()
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridWitnessCheckInPath)
	if err != nil {
		t.Fatal(err)
	}
	client := Client[WitnessCheckIn, WitnessCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 2},
		Keyring:  core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
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
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
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
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     server.Client(),
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
		Keyring:  testClientKeyring(t),
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
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP:     &http.Client{Transport: failingRoundTripper{}},
		Endpoint: endpoint,
		Jitter:   func() float64 { return 1 },
		Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
		Keyring:  testClientKeyring(t),
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
		mutate  func(*Client[BugCheckIn, BugCheckInResponse])
		name    string
		wantErr bool
	}{
		{name: "valid api key and backoff", mutate: func(*Client[BugCheckIn, BugCheckInResponse]) {}},
		{name: "invalid backoff with valid api key", wantErr: true, mutate: func(c *Client[BugCheckIn, BugCheckInResponse]) {
			c.Backoff.MaxAttempts = 0
		}},
		{name: "missing keyring rejected", wantErr: true, mutate: func(c *Client[BugCheckIn, BugCheckInResponse]) {
			c.Keyring = core.SigningKeyring{}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := Client[BugCheckIn, BugCheckInResponse]{
				HTTP:     &http.Client{},
				Endpoint: BugCheckInEndpoint(),
				APIKey:   testAPICallKey(t),
				Backoff:  core.BackoffPolicy{Base: time.Nanosecond, Max: time.Nanosecond, MaxAttempts: 1},
				Keyring:  testClientKeyring(t),
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
	client := Client[BugCheckIn, BugCheckInResponse]{Jitter: func() float64 { return 0 }}
	if got := client.jitterFraction(); got != 1 {
		t.Fatalf("jitterFraction zero = %v, want 1", got)
	}
	if got := conservativeJitterFraction(-0.1); got != 1 {
		t.Fatalf("conservativeJitterFraction negative = %v, want 1", got)
	}
	if got := conservativeJitterFraction(1.1); got != 1 {
		t.Fatalf("conservativeJitterFraction >1 = %v, want 1", got)
	}
	backoff := DefaultCheckInBackoff()
	got, err := backoff.Delay(0, client.jitterFraction())
	if err != nil {
		t.Fatal(err)
	}
	if got != backoff.Base {
		t.Fatalf("zero jitter delay = %s, want %s", got, backoff.Base)
	}
}

func TestCheckInResponseHostileCombinations(t *testing.T) {
	t.Parallel()
	grant := testBugGrantContract(t)
	for _, tc := range []struct {
		name     string
		response BugCheckInResponse
	}{
		{name: "granted missing grant", response: BugCheckInResponse{Decision: CheckInDecision{Granted: true, Refusal: RefusalNone}}},
		{name: "granted with refusal", response: BugCheckInResponse{
			Decision: CheckInDecision{Granted: true, Refusal: RefusalPaymentRequired},
			Grant:    grant,
		}},
		{name: "granted with remediation", response: BugCheckInResponse{
			Decision: CheckInDecision{Granted: true, Remediation: RemediationUpdatePayment},
			Grant:    grant,
		}},
		{name: "refused with grant", response: BugCheckInResponse{
			Decision: CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment},
			Grant:    grant,
		}},
		{name: "refused none refusal", response: BugCheckInResponse{
			Decision: CheckInDecision{Refusal: RefusalNone, Remediation: RemediationUpdatePayment},
		}},
		{name: "refused missing remediation", response: BugCheckInResponse{
			Decision: CheckInDecision{Refusal: RefusalPaymentRequired},
		}},
		{name: "refused mismatched remediation", response: BugCheckInResponse{
			Decision: CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationDeactivateMachine},
		}},
		{name: "refused unknown remediation ordinal", response: BugCheckInResponse{
			Decision: CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: Remediation(RemediationRetryUpload + 1)},
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

func TestCheckInTransportContractLayerTriad(t *testing.T) {
	t.Parallel()
	grant := testBugGrantContract(t)
	tests := []struct {
		body    TransportResponseBody
		name    string
		wantErr bool
	}{
		{name: "positive granted response", body: BugCheckInResponse{Decision: CheckInDecision{Granted: true, Refusal: RefusalNone}, Grant: grant}},
		{name: "negative zero response", body: BugCheckInResponse{}, wantErr: true},
		{name: "neutral refused response", body: BugCheckInResponse{Decision: CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.body.Validate()
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("TransportResponseBody.Validate() error = %v, want ErrLicenseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("TransportResponseBody.Validate() error = %v", err)
			}
		})
	}
}

func TestClientRejectsNilContextWithTypedIdentity(t *testing.T) {
	t.Parallel()

	client := Client[BugCheckIn, BugCheckInResponse]{}
	var nilContext context.Context
	_, err := client.Do(nilContext, BugCheckIn{})
	if !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("Client.Do(nil) error = %v, want ErrNilContext", err)
	}
	if !errors.Is(err, core.ErrLicenseContract) || !errors.Is(err, core.ErrFoundationContract) {
		t.Fatalf("Client.Do(nil) error = %v, want license and foundation identities", err)
	}
}

func TestCompilerOwnedDefaultsCannotBeMutatedGlobally(t *testing.T) {
	t.Parallel()

	mutated := DefaultCheckInBackoff()
	mutated.MaxAttempts = 0
	if err := DefaultCheckInBackoff().Validate(); err != nil {
		t.Fatalf("DefaultCheckInBackoff() after local mutation = %v", err)
	}
	for _, endpoint := range []CheckInEndpoint{BugCheckInEndpoint(), WitnessCheckInEndpoint()} {
		if err := endpoint.Validate(); err != nil {
			t.Fatalf("default endpoint validation = %v", err)
		}
	}
}

func TestCheckInResponseMalformedGrantCarriesLicenseIdentity(t *testing.T) {
	t.Parallel()
	grant := testBugGrantContract(t)
	grant.Lease.Signature = core.Ed25519SignatureHex{}
	response := BugCheckInResponse{
		Decision: CheckInDecision{Granted: true, Refusal: RefusalNone},
		Grant:    grant,
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
		{name: "nil response body", run: func() error {
			_, err := readCappedResponseBody(nil)
			return err
		}},
		{name: "success envelope decode failure", run: func() error {
			reply := &http.Response{Body: io.NopCloser(strings.NewReader("<html>"))}
			_, err := readResponse[BugCheckInGrant](reply, testClientKeyring(t))
			return err
		}},
		{name: "failure envelope decode failure", run: func() error {
			reply := &http.Response{Body: io.NopCloser(strings.NewReader("<html>"))}
			err := readFailureResponse[BugCheckInGrant](reply, http.StatusBadGateway, 0)
			var statusErr CheckInHTTPError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadGateway {
				return fmt.Errorf("status error = %v, want CheckInHTTPError 502", err)
			}
			return err
		}},
		{name: "request build failure", run: func() error {
			client := Client[BugCheckIn, BugCheckInResponse]{
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
	backoff := DefaultCheckInBackoff()
	minBudget := time.Duration(backoff.MaxAttempts-1) * backoff.Max
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
		"184467440737095516160",
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

func signedBugGrant(t *testing.T) (*BugCheckInGrant, core.SigningKeyID, core.Ed25519PublicKeyHex) {
	t.Helper()
	keyID, publicKey, privateKey := testServerSigningKey(t)
	leaseBody := testSeatLeaseBody(t)
	certificateBody := BugWriterCertificateBody{
		Schema:            core.SchemaBugWriterCertificate,
		DeviceFingerprint: leaseBody.DeviceFingerprint,
		Writer:            leaseBody.Writer,
		IssuedAt:          leaseBody.IssuedAt,
		ValidUntil:        leaseBody.WriteGraceUntil(),
	}
	grant := &BugCheckInGrant{
		Lease:             signTestBody(t, keyID, privateKey, leaseBody),
		WriterCertificate: signTestBody(t, keyID, privateKey, certificateBody),
	}
	return grant, keyID, publicKey
}

func signedWitnessGrant(t *testing.T) (*WitnessCheckInGrant, core.SigningKeyID, core.Ed25519PublicKeyHex) {
	t.Helper()
	keyID, publicKey, privateKey := testServerSigningKey(t)
	grant := &WitnessCheckInGrant{Lease: signTestBody(t, keyID, privateKey, testSubscriptionLeaseBody())}
	return grant, keyID, publicKey
}

func testServerSigningKey(t *testing.T) (core.SigningKeyID, core.Ed25519PublicKeyHex, ed25519.PrivateKey) {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	seed[len(seed)-1] = 42
	privateKey := ed25519.NewKeyFromSeed(seed)
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := core.NewEd25519PublicKeyHex(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicKey, privateKey
}

func signTestBody[B core.CanonicalBody](t *testing.T, keyID core.SigningKeyID, privateKey ed25519.PrivateKey, body B) core.Signed[B] {
	t.Helper()
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[B]{Body: body, KeyID: keyID, Signature: signature}
}

func testBugGrantContract(t *testing.T) *BugCheckInGrant {
	t.Helper()
	grant, _, _ := signedBugGrant(t)
	return grant
}

func testClientKeyring(t *testing.T) core.SigningKeyring {
	t.Helper()
	_, keyID, publicKey := signedBugGrant(t)
	return core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
}

func testSignatureHex(t *testing.T) core.Ed25519SignatureHex {
	t.Helper()
	signature, err := core.NewEd25519SignatureHex(make([]byte, ed25519.SignatureSize))
	if err != nil {
		t.Fatal(err)
	}
	return signature
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
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
		Writer:             testBugWriterKey(t),
		WriteGraceDuration: window.WriteGraceDuration,
		Plan:               SeatPlanStandard,
		BillingPeriod:      BillingPeriodMonthly,
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
		Writer:            testBugWriterKey(t),
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
