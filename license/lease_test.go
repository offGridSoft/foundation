package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
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
	body := SeatLeaseBody{
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

	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"bug-license-lease-v1","developer_key_id":"OGS-DEV-alpha",` +
		`"device_fingerprint":"sha256:` + strings.Repeat("a", 64) + `",` +
		`"issued_at":1782302400000000000,"expires_at":1784894400000000000,` +
		`"check_in_after":1783252800000000000,"check_in_by":1784030400000000000,` +
		`"write_grace_ns":259200000000000,"plan":"standard"}`
	if string(got) != want {
		t.Fatalf("canonical seat lease\n got: %s\nwant: %s", got, want)
	}
}

func TestSubscriptionLeaseCanonicalWireForm(t *testing.T) {
	t.Parallel()
	body := SubscriptionLeaseBody{
		PaidUntil:      testTime(1784894400000000000),
		TokenExpiresAt: testTime(1784030400000000000),
		CheckInAfterAt: testTime(1783252800000000000),
		CheckInByAt:    testTime(1784030400000000000),
		Schema:         SchemaWitnessSubscription,
		Plan:           SubscriptionPlanSilver,
		BillingPeriod:  BillingPeriodMonthly,
	}

	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"witness-subscription-lease-v1","paid_until":1784894400000000000,` +
		`"lease_not_after":1784030400000000000,"check_in_after":1783252800000000000,` +
		`"check_in_by":1784030400000000000,"plan":"silver","billing_period":"monthly"}`
	if string(got) != want {
		t.Fatalf("canonical subscription lease\n got: %s\nwant: %s", got, want)
	}
}

func TestSubscriptionLeaseRejectsDeviceFingerprintField(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"paid_until":1784894400000000000,"lease_not_after":1784894400000000000,` +
		`"check_in_after":1783252800000000000,"check_in_by":1784030400000000000,` +
		`"schema":"witness-subscription-lease-v1","plan":"silver","billing_period":"monthly",` +
		`"device_fingerprint":"sha256:` + strings.Repeat("a", 64) + `"}`)

	if _, err := core.DecodeStrictJSON[SubscriptionLeaseBody](raw); err == nil {
		t.Fatalf("SubscriptionLeaseBody accepted device_fingerprint")
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
	body.Schema = SchemaWitnessSubscription
	signed := core.Signed[SeatLeaseBody]{
		KeyID:     keyID,
		Signature: ed25519.Sign(private, message),
		Body:      body,
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("Verify schema mutation error = %v, want ErrLicenseContract", err)
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
	return SubscriptionLeaseBody{
		PaidUntil:      testTime(1784894400000000000),
		TokenExpiresAt: testTime(1784030400000000000),
		CheckInAfterAt: testTime(1783252800000000000),
		CheckInByAt:    testTime(1784030400000000000),
		Schema:         SchemaWitnessSubscription,
		Plan:           SubscriptionPlanSilver,
		BillingPeriod:  BillingPeriodMonthly,
	}
}
