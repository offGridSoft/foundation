package license

import (
	"strings"
	"testing"
	"time"

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

func testDeviceFingerprint(t *testing.T) DeviceFingerprint {
	t.Helper()
	fp, err := ParseDeviceFingerprint(DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
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
	want := `{"issued_at":1782302400000000000,"expires_at":1784894400000000000,` +
		`"check_in_after":1783252800000000000,"check_in_by":1784030400000000000,` +
		`"schema":"bug-license-lease-v1","developer_key_id":"OGS-DEV-alpha",` +
		`"device_fingerprint":"sha256:` + strings.Repeat("a", 64) + `",` +
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
	want := `{"paid_until":1784894400000000000,"lease_not_after":1784030400000000000,` +
		`"check_in_after":1783252800000000000,"check_in_by":1784030400000000000,` +
		`"schema":"witness-subscription-lease-v1","plan":"silver","billing_period":"monthly"}`
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
