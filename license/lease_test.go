package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func testTime(nanos int64) core.UnixNanoTime {
	return core.UnixNanoTimeFromInt64(nanos)
}

func testDeveloperKeyID(t *testing.T) DeveloperKeyID {
	t.Helper()
	id, err := ParseDeveloperKeyID("OGS-DEV-alpha")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func testDeviceFingerprint(t *testing.T) core.DeviceFingerprint {
	t.Helper()
	fp, err := core.ParseDeviceFingerprint(core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return fp
}

func TestSeatLeaseCanonicalWireForm(t *testing.T) {
	t.Parallel()
	window := testLeaseWindow(t, SeatPlanStandard, 0)
	body := SeatLeaseBody{
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

	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"bug-license-lease-v2026","lease_id":"lease-2026-test","lease_generation":1,"developer_key_id":"OGS-DEV-alpha",` +
		`"device_fingerprint":"sha256:` + strings.Repeat("a", 64) + `",` +
		`"writer":{"key_id":"` + body.Writer.KeyID.String() + `","public_key":"` + body.Writer.PublicKey.String() + `"},` +
		`"issued_at":1782302400000000000,"paid_until":1784894400000000000,` +
		`"lease_not_after":1785240000000000000,` +
		`"check_in_after":1785153600000000000,"check_in_by":1785240000000000000,` +
		`"write_grace_ns":259200000000000,"plan":"standard","billing_period":"monthly","prepaid_years":0}`
	if string(got) != want {
		t.Fatalf("canonical seat lease\n got: %s\nwant: %s", got, want)
	}
}

func TestSubscriptionLeaseCanonicalWireForm(t *testing.T) {
	t.Parallel()
	window := testSubscriptionLeaseWindow(t, SubscriptionPlanSilver, 0)
	body := SubscriptionLeaseBody{
		LeaseID:            testLeaseID(t),
		Generation:         1,
		DeviceFingerprint:  testDeviceFingerprint(t),
		IssuedAt:           window.IssuedAt,
		PaidUntil:          window.PaidUntil,
		TokenExpiresAt:     window.TokenExpiresAt,
		CheckInAfterAt:     window.CheckInAfterAt,
		CheckInByAt:        window.CheckInByAt,
		WriteGraceDuration: window.WriteGraceDuration,
		Schema:             core.SchemaWitnessSubscription,
		Plan:               SubscriptionPlanSilver,
		BillingPeriod:      BillingPeriodMonthly,
	}

	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"witness-subscription-lease-v2026","lease_id":"lease-2026-test","lease_generation":1,"device_fingerprint":"sha256:` +
		strings.Repeat("a", 64) + `","issued_at":1782302400000000000,` +
		`"paid_until":1784894400000000000,` +
		`"lease_not_after":1785240000000000000,"check_in_after":1785153600000000000,` +
		`"check_in_by":1785240000000000000,"write_grace_ns":259200000000000,` +
		`"plan":"silver","billing_period":"monthly","prepaid_years":0}`
	if string(got) != want {
		t.Fatalf("canonical subscription lease\n got: %s\nwant: %s", got, want)
	}
}

func TestLeaseCanonicalRoundTripTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "seat lease", run: func(t *testing.T) {
			t.Helper()
			original, err := testSeatLeaseBody(t).Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := core.DecodeStrictJSON[SeatLeaseBody](original)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Fatalf("seat lease canonical round trip\n got: %s\nwant: %s", got, original)
			}
		}},
		{name: "subscription lease", run: func(t *testing.T) {
			t.Helper()
			original, err := testSubscriptionLeaseBody().Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := core.DecodeStrictJSON[SubscriptionLeaseBody](original)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Fatalf("subscription lease canonical round trip\n got: %s\nwant: %s", got, original)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestSubscriptionLeaseRejectsDeveloperKeyField(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"device_fingerprint":"sha256:` + strings.Repeat("a", 64) + `",` +
		`"issued_at":1782302400000000000,` +
		`"paid_until":1784894400000000000,"lease_not_after":1785240000000000000,` +
		`"check_in_after":1785153600000000000,"check_in_by":1785240000000000000,` +
		`"write_grace_ns":259200000000000,` +
		`"schema":"witness-subscription-lease-v2026","plan":"silver","billing_period":"monthly",` +
		`"prepaid_years":0,"developer_key_id":"OGS-DEV-alpha"}`)

	if _, err := core.DecodeStrictJSON[SubscriptionLeaseBody](raw); err == nil {
		t.Fatalf("SubscriptionLeaseBody accepted developer_key_id")
	}
}

func TestCrossProductLeaseBodiesRejectEachOther(t *testing.T) {
	t.Parallel()
	seatRaw, err := json.Marshal(testSeatLeaseBody(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.DecodeStrictJSON[SubscriptionLeaseBody](seatRaw); !errors.Is(err, core.ErrJSONContract) {
		t.Fatalf("seat decoded as subscription error = %v, want ErrJSONContract", err)
	}

	subRaw, err := json.Marshal(testSubscriptionLeaseBody())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.DecodeStrictJSON[SeatLeaseBody](subRaw); !errors.Is(err, core.ErrJSONContract) {
		t.Fatalf("subscription decoded as seat error = %v, want ErrJSONContract", err)
	}
}

func TestLeaseBodySchemaMutationRejectsSignedLease(t *testing.T) {
	t.Parallel()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := core.NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	body := testSeatLeaseBody(t)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(private, message))
	if err != nil {
		t.Fatal(err)
	}
	body.Schema = core.SchemaWitnessSubscription
	signed := core.Signed[SeatLeaseBody]{
		KeyID:     keyID,
		Signature: signature,
		Body:      body,
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("Verify schema mutation error = %v, want ErrLicenseContract", err)
	}
}

func TestSeatLeaseBodyWindowHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*SeatLeaseBody)
		name   string
	}{
		{name: "zero issued", mutate: func(b *SeatLeaseBody) { b.IssuedAt = core.UnixNanoTime{} }},
		{name: "expires at issued", mutate: func(b *SeatLeaseBody) { b.TokenExpiresAt = b.IssuedAt }},
		{name: "check-in before issued", mutate: func(b *SeatLeaseBody) { b.CheckInAfterAt = b.IssuedAt.Add(-time.Nanosecond) }},
		{name: "four-week check-in before paid-through", mutate: func(b *SeatLeaseBody) {
			b.CheckInAfterAt = b.PaidUntil.Add(-time.Nanosecond)
		}},
		{name: "check-in after by", mutate: func(b *SeatLeaseBody) { b.CheckInAfterAt = b.CheckInByAt.Add(time.Nanosecond) }},
		{name: "check-in by after expiry", mutate: func(b *SeatLeaseBody) { b.CheckInByAt = b.TokenExpiresAt.Add(time.Nanosecond) }},
		{name: "negative write grace", mutate: func(b *SeatLeaseBody) {
			b.WriteGraceDuration = core.NanosecondsDurationFromInt64(-1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := testSeatLeaseBody(t)
			tc.mutate(&body)
			if err := body.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("SeatLeaseBody.Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestSubscriptionLeaseBodyWindowHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*SubscriptionLeaseBody)
		name   string
	}{
		{name: "zero paid until", mutate: func(b *SubscriptionLeaseBody) { b.PaidUntil = core.UnixNanoTime{} }},
		{name: "zero token expiry", mutate: func(b *SubscriptionLeaseBody) { b.TokenExpiresAt = core.UnixNanoTime{} }},
		{name: "missing check-in after", mutate: func(b *SubscriptionLeaseBody) { b.CheckInAfterAt = core.UnixNanoTime{} }},
		{name: "four-week check-in before paid-through", mutate: func(b *SubscriptionLeaseBody) {
			b.CheckInAfterAt = b.PaidUntil.Add(-time.Nanosecond)
		}},
		{name: "check-in after by", mutate: func(b *SubscriptionLeaseBody) { b.CheckInAfterAt = b.CheckInByAt.Add(time.Nanosecond) }},
		{name: "check-in by after expiry", mutate: func(b *SubscriptionLeaseBody) {
			b.CheckInByAt = b.TokenExpiresAt.Add(time.Nanosecond)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := testSubscriptionLeaseBody()
			tc.mutate(&body)
			if err := body.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("SubscriptionLeaseBody.Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestSignedLeaseBodiesAcceptServerOwnedPolicyTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "seat lease accepts changed connected grace", run: func(t *testing.T) {
			t.Helper()
			body := testSeatLeaseBody(t)
			body.PaidUntil = body.IssuedAt.Add(30 * 24 * time.Hour)
			body.CheckInAfterAt = body.PaidUntil.Add(6 * time.Hour)
			body.CheckInByAt = body.CheckInAfterAt.Add(12 * time.Hour)
			body.TokenExpiresAt = body.CheckInByAt
			body.WriteGraceDuration = core.NewNanosecondsDuration(6 * time.Hour)
			if err := body.Validate(); err != nil {
				t.Fatalf("SeatLeaseBody.Validate() = %v", err)
			}
		}},
		{name: "subscription lease accepts changed connected grace", run: func(t *testing.T) {
			t.Helper()
			body := testSubscriptionLeaseBody()
			body.PaidUntil = body.IssuedAt.Add(30 * 24 * time.Hour)
			body.CheckInAfterAt = body.PaidUntil.Add(6 * time.Hour)
			body.CheckInByAt = body.CheckInAfterAt.Add(12 * time.Hour)
			body.TokenExpiresAt = body.CheckInByAt
			body.WriteGraceDuration = core.NewNanosecondsDuration(6 * time.Hour)
			if err := body.Validate(); err != nil {
				t.Fatalf("SubscriptionLeaseBody.Validate() = %v", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestLeaseBillingTermHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T) error
		name string
	}{
		{name: "four week seat lease rejects prepaid years", run: func(t *testing.T) error {
			t.Helper()
			body := testSeatLeaseBody(t)
			body.PrepaidYears = 1
			return body.Validate()
		}},
		{name: "prepaid seat lease rejects zero years", run: func(t *testing.T) error {
			t.Helper()
			body := testSeatLeaseBody(t)
			body.BillingPeriod = BillingPeriodPrepaidYears
			return body.Validate()
		}},
		{name: "four week subscription lease rejects prepaid years", run: func(t *testing.T) error {
			t.Helper()
			body := testSubscriptionLeaseBody()
			body.PrepaidYears = 1
			return body.Validate()
		}},
		{name: "prepaid subscription lease rejects zero years", run: func(t *testing.T) error {
			t.Helper()
			body := testSubscriptionLeaseBody()
			body.BillingPeriod = BillingPeriodPrepaidYears
			return body.Validate()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(t); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestCheckInDueUsesBodyBoundary(t *testing.T) {
	t.Parallel()
	body := testSubscriptionLeaseBody()
	if CheckInDue(body, body.CheckInAfter().Add(-time.Nanosecond)) {
		t.Fatalf("CheckInDue before boundary = true")
	}
	if !CheckInDue(body, body.CheckInAfter()) {
		t.Fatalf("CheckInDue at boundary = false")
	}
}

func testSubscriptionLeaseBody() SubscriptionLeaseBody {
	window, err := BuildLeaseWindow(testTime(1782302400000000000), mustSubscriptionOffer(SubscriptionPlanSilver), 0)
	if err != nil {
		panic(err)
	}
	return SubscriptionLeaseBody{
		LeaseID:            mustLeaseIDNoT(),
		Generation:         1,
		DeviceFingerprint:  testDeviceFingerprintNoT(),
		IssuedAt:           window.IssuedAt,
		PaidUntil:          window.PaidUntil,
		TokenExpiresAt:     window.TokenExpiresAt,
		CheckInAfterAt:     window.CheckInAfterAt,
		CheckInByAt:        window.CheckInByAt,
		WriteGraceDuration: window.WriteGraceDuration,
		Schema:             core.SchemaWitnessSubscription,
		Plan:               SubscriptionPlanSilver,
		BillingPeriod:      BillingPeriodMonthly,
		PrepaidYears:       0,
	}
}

func mustLeaseIDNoT() core.LeaseID {
	id, err := core.ParseLeaseID("lease-2026-test")
	if err != nil {
		panic(err)
	}
	return id
}

func testLeaseWindow(t *testing.T, plan SeatPlan, prepaidYears uint8) LeaseWindow {
	t.Helper()
	window, err := BuildLeaseWindow(testTime(1782302400000000000), mustSeatOffer(plan), prepaidYears)
	if err != nil {
		t.Fatal(err)
	}
	return window
}

func testSubscriptionLeaseWindow(t *testing.T, plan SubscriptionPlan, prepaidYears uint8) LeaseWindow {
	t.Helper()
	window, err := BuildLeaseWindow(testTime(1782302400000000000), mustSubscriptionOffer(plan), prepaidYears)
	if err != nil {
		t.Fatal(err)
	}
	return window
}

func mustSeatOffer(plan SeatPlan) Offer {
	offer, err := OfferForSeatPlan(plan)
	if err != nil {
		panic(err)
	}
	return offer
}

func mustSubscriptionOffer(plan SubscriptionPlan) Offer {
	offer, err := OfferForSubscriptionPlan(plan)
	if err != nil {
		panic(err)
	}
	return offer
}

func testDeviceFingerprintNoT() core.DeviceFingerprint {
	fp, err := core.ParseDeviceFingerprint(core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
	if err != nil {
		panic(err)
	}
	return fp
}
