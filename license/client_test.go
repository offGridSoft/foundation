package license

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func checkInTestBackoff(maxAttempts uint64) core.BackoffPolicy {
	delay := core.NewNanosecondsDuration(time.Nanosecond)
	return core.BackoffPolicy{Base: delay, Max: delay, MaxAttempts: maxAttempts}
}

func TestMaximalBugCheckInResponseFitsTransportBoundary(t *testing.T) {
	t.Parallel()

	grant, keyID, publicKey := signedBugGrant(t)
	response := signBugCheckInResponse(t, BugCheckInResponseBody{
		Schema:       core.SchemaBugCheckInResponse,
		RequestNonce: testCheckInNonce(t),
		Decision:     CheckInDecision{Granted: true, Refusal: RefusalNone},
		Grant:        grant,
		WriterRevocations: BugWriterRevocationDelivery{Values: makeRevocationDelivery(
			t, keyID, int(BugWriterRevocationDeliveryMaximum),
		)},
	})
	body, err := json.Marshal(core.APIEnvelope[BugCheckInResponse]{
		RequestID: core.NewAPIRequestID("req-maximal-bug-check-in"),
		Data:      &response,
	})
	if err != nil {
		t.Fatalf("json.Marshal(maximal response) error = %v", err)
	}
	if len(body) > CheckInResponseByteCap {
		t.Fatalf("maximal legal response bytes = %d, cap = %d", len(body), CheckInResponseByteCap)
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	decoded, err := core.DecodeStrictJSON[core.APIEnvelope[BugCheckInResponse]](body)
	if err != nil || decoded.Data == nil {
		t.Fatalf("DecodeStrictJSON(maximal response) = (%+v, %v), want typed response", decoded, err)
	}
	if err := decoded.Data.Verify(keyring); err != nil {
		t.Fatalf("maximal response Verify() error = %v", err)
	}
}

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
			Data:      new(testGrantedBugResponse(t, grant)),
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
	if response.Authority.Body.Grant == nil {
		t.Fatalf("response.Grant = nil, want signed Bug grant")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Authority.Body.Grant.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsSignedResponseForDifferentRequestNonce(t *testing.T) {
	t.Parallel()

	grant, keyID, publicKey := signedBugGrant(t)
	foreignNonce, err := ParseCheckInNonce("0202020202020202020202020202020202020202020202020202020202020202")
	if err != nil {
		t.Fatal(err)
	}
	response := signBugCheckInResponse(t, BugCheckInResponseBody{
		Schema:       core.SchemaBugCheckInResponse,
		RequestNonce: foreignNonce,
		Decision:     CheckInDecision{Granted: true, Refusal: RefusalNone},
		Grant:        grant,
	})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_ = json.NewEncoder(w).Encode(core.APIEnvelope[BugCheckInResponse]{
			RequestID: core.NewAPIRequestID("req-replayed-response"),
			Data:      &response,
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
	if _, err := client.Do(context.Background(), testBugCheckIn(t)); !errors.Is(err, core.ErrCheckInNonce) {
		t.Fatalf("Client.Do(replayed response) error = %v, want %v", err, core.ErrCheckInNonce)
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
			Data:      new(testGrantedBugResponse(t, grant)),
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
			Data:      new(testGrantedWitnessResponse(t, grant)),
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
	if response.Authority.Body.Grant == nil {
		t.Fatalf("response.Grant = nil, want signed Witness grant")
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Authority.Body.Grant.Verify(keyring); err != nil {
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
			Data:      new(testGrantedBugResponse(t, grant)),
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
			Data:      new(testGrantedWitnessResponse(t, grant)),
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
		_ = json.NewEncoder(w).Encode(testRefusedBugResponse(t))
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
			Data:      new(testGrantedBugResponse(t, grant)),
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
		Backoff:  checkInTestBackoff(3),
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
			Data:      new(testGrantedWitnessResponse(t, grant)),
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
		Backoff:  checkInTestBackoff(2),
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
		Backoff:  checkInTestBackoff(1),
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
		Backoff:  checkInTestBackoff(1),
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
				Endpoint: mustDefaultCheckInEndpoint(t, core.ProductBug),
				APIKey:   testAPICallKey(t),
				Backoff:  checkInTestBackoff(1),
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

func TestCheckInResponseHostileCombinations(t *testing.T) {
	t.Parallel()
	grant := testBugGrantContract(t)
	valid := testGrantedBugResponse(t, grant)
	unchecked := func(decision CheckInDecision, candidate *BugCheckInGrant) BugCheckInResponse {
		response := valid
		response.Authority.Body.Decision = decision
		response.Authority.Body.Grant = candidate
		return response
	}
	for _, tc := range []struct {
		name     string
		response BugCheckInResponse
	}{
		{name: "granted missing grant", response: unchecked(CheckInDecision{Granted: true, Refusal: RefusalNone}, nil)},
		{name: "granted with refusal", response: unchecked(CheckInDecision{Granted: true, Refusal: RefusalPaymentRequired}, grant)},
		{name: "granted with remediation", response: unchecked(CheckInDecision{Granted: true, Remediation: RemediationUpdatePayment}, grant)},
		{name: "refused with grant", response: unchecked(CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment}, grant)},
		{name: "refused none refusal", response: unchecked(CheckInDecision{Refusal: RefusalNone, Remediation: RemediationUpdatePayment}, nil)},
		{name: "refused missing remediation", response: unchecked(CheckInDecision{Refusal: RefusalPaymentRequired}, nil)},
		{name: "refused mismatched remediation", response: unchecked(CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationDeactivateMachine}, nil)},
		{name: "refused unknown remediation ordinal", response: unchecked(CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: Remediation(RemediationRetryUpload + 1)}, nil)},
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
		body    core.APIBody
		name    string
		wantErr bool
	}{
		{name: "positive granted response", body: testGrantedBugResponse(t, grant)},
		{name: "negative zero response", body: BugCheckInResponse{}, wantErr: true},
		{name: "neutral refused response", body: testRefusedBugResponse(t)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.body.Validate()
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("APIBody.Validate() error = %v, want ErrLicenseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("APIBody.Validate() error = %v", err)
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

	mutated := core.DefaultHTTPBackoffPolicy()
	mutated.MaxAttempts = 0
	if err := core.DefaultHTTPBackoffPolicy().Validate(); err != nil {
		t.Fatalf("DefaultHTTPBackoffPolicy() after local mutation = %v", err)
	}
	for _, endpoint := range []core.APIEndpoint{
		mustDefaultCheckInEndpoint(t, core.ProductBug),
		mustDefaultCheckInEndpoint(t, core.ProductWitness),
	} {
		if err := endpoint.Validate(); err != nil {
			t.Fatalf("default endpoint validation = %v", err)
		}
	}
}

func TestCheckInResponseMalformedGrantCarriesLicenseIdentity(t *testing.T) {
	t.Parallel()
	response := testGrantedBugResponse(t, testBugGrantContract(t))
	response.Authority.Body.Grant.Lease.Signature = core.Ed25519SignatureHex{}
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
		{name: "success envelope decode failure", run: func() error {
			return runMalformedCheckInResponse(t, core.HTTPStatusOK)
		}},
		{name: "failure envelope decode failure", run: func() error {
			status, err := core.NewHTTPStatusCode(http.StatusBadRequest)
			if err != nil {
				return err
			}
			return runMalformedCheckInResponse(t, status)
		}},
		{name: "network transport failure", run: func() error {
			endpoint := mustDefaultCheckInEndpoint(t, core.ProductBug)
			client := Client[BugCheckIn, BugCheckInResponse]{
				HTTP: &http.Client{Transport: failingRoundTripper{}}, Endpoint: endpoint,
				Backoff: checkInTestBackoff(1), Keyring: testClientKeyring(t),
			}
			_, err := client.Do(t.Context(), testBugCheckIn(t))
			return err
		}},
		{name: "request build failure", run: func() error {
			client := Client[BugCheckIn, BugCheckInResponse]{
				HTTP:     http.DefaultClient,
				Endpoint: core.APIEndpoint{},
			}
			return client.Validate()
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

func runMalformedCheckInResponse(t *testing.T, status core.HTTPStatusCode) error {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		writer.WriteHeader(status.Int())
		_, _ = writer.Write([]byte(`{"request_id":`))
	}))
	t.Cleanup(server.Close)
	endpoint, err := ParseCheckInEndpoint(server.URL + OffgridBugCheckInPath)
	if err != nil {
		return err
	}
	client := Client[BugCheckIn, BugCheckInResponse]{
		HTTP: server.Client(), Endpoint: endpoint,
		Backoff: checkInTestBackoff(1), Keyring: testClientKeyring(t),
	}
	_, err = client.Do(t.Context(), testBugCheckIn(t))
	return err
}

func TestCheckInBudgetCoversWorstCaseRetryAfterSchedule(t *testing.T) {
	t.Parallel()
	backoff := core.DefaultHTTPBackoffPolicy()
	minBudget := time.Duration(backoff.MaxAttempts-1) * backoff.Max.Duration()
	if CheckInBudget < minBudget {
		t.Fatalf("CheckInBudget = %s, want at least %s", CheckInBudget, minBudget)
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

func testCheckInNonce(t *testing.T) CheckInNonce {
	t.Helper()
	nonce, err := ParseCheckInNonce("0101010101010101010101010101010101010101010101010101010101010101")
	if err != nil {
		t.Fatal(err)
	}
	return nonce
}

func testLeaseID(t *testing.T) core.LeaseID {
	t.Helper()
	id, err := core.ParseLeaseID("lease-2026-test")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func signBugCheckInResponse(t *testing.T, body BugCheckInResponseBody) BugCheckInResponse {
	t.Helper()
	keyID, _, privateKey := testServerSigningKey(t)
	if body.Grant != nil && body.TimeCommitment == nil {
		lease := body.Grant.Lease.Body
		commitment := signTestBody(t, keyID, privateKey, CheckInTimeCommitmentBody[BugCheckInGrant]{
			Schema:            core.SchemaBugCheckInTimeCommitment,
			DeviceFingerprint: lease.DeviceFingerprint,
			LeaseID:           lease.LeaseID,
			LeaseGeneration:   lease.Generation,
			RequestNonce:      body.RequestNonce,
			ServerObservedAt:  core.UnixNanoTimeFromInt64(1_800_000_000_000_000_000),
		})
		body.TimeCommitment = &commitment
	}
	return BugCheckInResponse{Authority: signTestBody(t, keyID, privateKey, body)}
}

func signWitnessCheckInResponse(t *testing.T, body WitnessCheckInResponseBody) WitnessCheckInResponse {
	t.Helper()
	keyID, _, privateKey := testServerSigningKey(t)
	if body.Grant != nil && body.TimeCommitment == nil {
		lease := body.Grant.Lease.Body
		commitment := signTestBody(t, keyID, privateKey, CheckInTimeCommitmentBody[WitnessCheckInGrant]{
			Schema:            core.SchemaWitnessCheckInTimeCommitment,
			DeviceFingerprint: lease.DeviceFingerprint,
			LeaseID:           lease.LeaseID,
			LeaseGeneration:   lease.Generation,
			RequestNonce:      body.RequestNonce,
			ServerObservedAt:  core.UnixNanoTimeFromInt64(1_800_000_000_000_000_000),
		})
		body.TimeCommitment = &commitment
	}
	return WitnessCheckInResponse{Authority: signTestBody(t, keyID, privateKey, body)}
}

func testGrantedBugResponse(t *testing.T, grant *BugCheckInGrant) BugCheckInResponse {
	t.Helper()
	return signBugCheckInResponse(t, BugCheckInResponseBody{
		Schema:       core.SchemaBugCheckInResponse,
		RequestNonce: testCheckInNonce(t),
		Decision:     CheckInDecision{Granted: true, Refusal: RefusalNone},
		Grant:        grant,
	})
}

func testRefusedBugResponse(t *testing.T) BugCheckInResponse {
	t.Helper()
	return signBugCheckInResponse(t, BugCheckInResponseBody{
		Schema:       core.SchemaBugCheckInResponse,
		RequestNonce: testCheckInNonce(t),
		Decision:     CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment},
	})
}

func testGrantedWitnessResponse(t *testing.T, grant *WitnessCheckInGrant) WitnessCheckInResponse {
	t.Helper()
	return signWitnessCheckInResponse(t, WitnessCheckInResponseBody{
		Schema:       core.SchemaWitnessCheckInResponse,
		RequestNonce: testCheckInNonce(t),
		Decision:     CheckInDecision{Granted: true, Refusal: RefusalNone},
		Grant:        grant,
	})
}

//go:fix inline

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

func testSeatLeaseBody(t *testing.T) SeatLeaseBody {
	t.Helper()
	window, err := BuildLeaseWindow(testTime(1782302400000000000), mustTestSeatOffer(t, SeatPlanStandard), 0)
	if err != nil {
		t.Fatal(err)
	}
	return SeatLeaseBody{
		LeaseID:            testLeaseID(t),
		Generation:         1,
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
	price := testOfferAmount(testPaidOfferMinorUnits)
	if plan == SeatPlanOSS {
		price = testOfferAmount(0)
	}
	offer, err := OfferForSeatPlan(plan, price)
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
		RequestNonce:      testCheckInNonce(t),
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
	token, err := ParseAccountToken("acct_1234567890abcdef")
	if err != nil {
		t.Fatal(err)
	}
	return WitnessCheckIn{
		Schema:            core.SchemaWitnessCheckIn,
		DeviceFingerprint: testDeviceFingerprint(t),
		BinaryVersion:     testProductVersion(t),
		BinarySHA256:      testSHA256(t),
		RequestNonce:      testCheckInNonce(t),
		Platform:          core.PlatformDarwinARM64,
		AccountToken:      token,
	}
}
