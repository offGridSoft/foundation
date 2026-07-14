package license

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestDeveloperKeyHostileTable(t *testing.T) {
	t.Parallel()
	valid, err := ParseDeveloperKey("OGS-DEV-123456789012")
	if err != nil {
		t.Fatalf("ParseDeveloperKey(valid) error = %v", err)
	}
	if got := valid.Preview(); got != "OGS-DEV-1234..." {
		t.Fatalf("DeveloperKey.Preview() = %q, want masked prefix", got)
	}
	if got := (DeveloperKey{}).Preview(); got != "" {
		t.Fatalf("zero DeveloperKey.Preview() = %q, want empty", got)
	}
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "blank", value: ""},
		{name: "wrong prefix", value: "BAD-DEV-123456789012"},
		{name: "too short", value: "OGS-DEV-short"},
		{name: "contains space", value: "OGS-DEV-123456 789012"},
		{name: "contains newline", value: "OGS-DEV-123456789012\n"},
		{name: "contains nul", value: "OGS-DEV-123456\x00789012"},
		{name: "too long", value: DeveloperKeyPrefix + strings.Repeat("a", DeveloperKeyMaxRunes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseDeveloperKey(tc.value)
			if !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("ParseDeveloperKey error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestPlanAndRefusalHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "seat plan rejects foreign token", run: func() error { _, err := ParseSeatPlan("trial"); return err }},
		{name: "subscription plan rejects seat token", run: func() error { _, err := ParseSubscriptionPlan("enterprise"); return err }},
		{name: "billing period rejects annual", run: func() error { _, err := ParseBillingPeriod("annual"); return err }},
		{name: "refusal rejects empty", run: func() error { _, err := ParseRefusal(""); return err }},
		{name: "refusal rejects unknown", run: func() error { _, err := ParseRefusal("denied"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("%s error = %v, want ErrLicenseContract", tc.name, err)
			}
		})
	}
}

func TestPlatformRejectsUnknownAndOrdinal(t *testing.T) {
	t.Parallel()
	if platform, err := core.CurrentPlatform(); err != nil {
		t.Fatal(err)
	} else if err := platform.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`"linux-riscv64"`, `"darwin"`, `1`, `""`} {
		var platform core.Platform
		if err := json.Unmarshal([]byte(raw), &platform); !errors.Is(err, core.ErrFoundationContract) {
			t.Fatalf("Unmarshal(%s) error = %v, want ErrFoundationContract", raw, err)
		}
	}
}

func TestAccountTokenWireContract(t *testing.T) {
	t.Parallel()
	token, err := ParseAccountToken("acct_123456789")
	if err != nil {
		t.Fatal(err)
	}
	if token.String() != "acct_123456789" {
		t.Fatalf("AccountToken.String = %s", token)
	}
	data, err := token.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip AccountToken
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != token {
		t.Fatalf("AccountToken roundTrip = %s, want %s", roundTrip, token)
	}
	for _, raw := range []string{"", " acct", "acct 123", "acct\n123", strings.Repeat("a", core.OpaqueTokenDefaultMaxRunes+1)} {
		if _, err := ParseAccountToken(raw); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseAccountToken(%q) error = %v, want ErrLicenseContract", raw, err)
		}
	}
}

func TestAPICallKeyWireContract(t *testing.T) {
	t.Parallel()
	key, err := ParseAPICallKey("public-bot-filter")
	if err != nil {
		t.Fatal(err)
	}
	data, err := key.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip APICallKey
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != key {
		t.Fatalf("APICallKey roundTrip = %s, want %s", roundTrip, key)
	}
	for _, raw := range []string{"", " key", "key\n1", strings.Repeat("a", core.OpaqueTokenDefaultMaxRunes+1)} {
		if _, err := ParseAPICallKey(raw); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseAPICallKey(%q) error = %v, want ErrLicenseContract", raw, err)
		}
	}
}

func TestDeviceLabelWireContract(t *testing.T) {
	t.Parallel()
	label, err := ParseDeviceLabel(" developer laptop ")
	if err != nil {
		t.Fatal(err)
	}
	if label.String() != "developer laptop" {
		t.Fatalf("DeviceLabel.String = %q, want trimmed label", label.String())
	}
	data, err := label.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip DeviceLabel
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != label {
		t.Fatalf("DeviceLabel roundTrip = %s, want %s", roundTrip, label)
	}
	for _, raw := range []string{"", " \t ", "dev\nlaptop", strings.Repeat("a", DeviceLabelMaxRunes+1)} {
		if _, err := ParseDeviceLabel(raw); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseDeviceLabel error = %v, want ErrLicenseContract", err)
		}
	}
}

func TestDeveloperKeyIDRejectsControlToken(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", " key", "key\x00id", "key\nid", strings.Repeat("a", core.OpaqueTokenDefaultMaxRunes+1)} {
		if _, err := ParseDeveloperKeyID(raw); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseDeveloperKeyID(%q) error = %v, want ErrLicenseContract", raw, err)
		}
	}
}

func TestCheckInEndpointWireContract(t *testing.T) {
	t.Parallel()
	endpoint := mustDefaultCheckInEndpoint(t, core.ProductBug)
	data, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip core.APIEndpoint
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != endpoint {
		t.Fatalf("CheckInEndpoint roundTrip = %s, want %s", roundTrip, endpoint)
	}
	for _, raw := range []string{
		"http://api.offgridsoftware.ca/v2026/bug/check_in",
		"https://",
		"https://api.offgridsoftware.ca",
		"https://api.offgridsoftware.ca@evil.example/v2026/bug/check_in",
		"https://api.offgridsoftware.ca/v2026/bug/check_in\nx",
		strings.Repeat("a", core.HTTPSURLDefaultMaxRunes+1),
		"://bad",
	} {
		if _, err := ParseCheckInEndpoint(raw); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseCheckInEndpoint(%q) error = %v, want ErrLicenseContract", raw, err)
		}
	}
}

func TestCheckInEndpointForBaseURLHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseURL   string
		want      string
		product   core.Product
		wantError bool
	}{
		{name: "production bug endpoint", baseURL: core.OffgridAPIBaseURL, product: core.ProductBug, want: core.OffgridAPIBaseURL + OffgridBugCheckInPath},
		{name: "production witness endpoint", baseURL: core.OffgridAPIBaseURL, product: core.ProductWitness, want: core.OffgridAPIBaseURL + OffgridWitnessCheckInPath},
		{name: "production trailing slash trimmed", baseURL: core.OffgridAPIBaseURL + "/", product: core.ProductBug, want: core.OffgridAPIBaseURL + OffgridBugCheckInPath},
		{name: "dev localhost bug endpoint", baseURL: "http://localhost:40123", product: core.ProductBug, want: "http://localhost:40123" + OffgridBugCheckInPath},
		{name: "dev loopback witness endpoint", baseURL: "http://127.0.0.1:40123", product: core.ProductWitness, want: "http://127.0.0.1:40123" + OffgridWitnessCheckInPath},
		{name: "dev ipv6 loopback endpoint", baseURL: "http://[::1]:40123", product: core.ProductWitness, want: "http://[::1]:40123" + OffgridWitnessCheckInPath},
		{name: "empty base rejected", baseURL: "", product: core.ProductBug, wantError: true},
		{name: "http production rejected", baseURL: "http://api.offgridsoftware.ca", product: core.ProductBug, wantError: true},
		{name: "http localhost lookalike rejected", baseURL: "http://localhost.evil.example:40123", product: core.ProductBug, wantError: true},
		{name: "base path rejected", baseURL: core.OffgridAPIBaseURL + "/api", product: core.ProductBug, wantError: true},
		{name: "query rejected", baseURL: core.OffgridAPIBaseURL + "?x=1", product: core.ProductBug, wantError: true},
		{name: "fragment rejected", baseURL: core.OffgridAPIBaseURL + "#x", product: core.ProductBug, wantError: true},
		{name: "userinfo rejected", baseURL: "https://user@api.offgridsoftware.ca", product: core.ProductBug, wantError: true},
		{name: "oversized base rejected", baseURL: "https://" + strings.Repeat("a", core.HTTPSURLDefaultMaxRunes), product: core.ProductBug, wantError: true},
		{name: "unknown product rejected", baseURL: core.OffgridAPIBaseURL, product: core.ProductUnknown, wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := CheckInEndpointForBaseURL(tc.baseURL, tc.product)
			if tc.wantError {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("CheckInEndpointForBaseURL() error = %v, want ErrLicenseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckInEndpointForBaseURL() error = %v", err)
			}
			if got.String() != tc.want {
				t.Fatalf("CheckInEndpointForBaseURL() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLocalCheckInEndpointHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "localhost http accepted", value: "http://localhost:40123" + OffgridBugCheckInPath},
		{name: "ipv4 loopback http accepted", value: "http://127.0.0.1:40123" + OffgridBugCheckInPath},
		{name: "ipv6 loopback http accepted", value: "http://[::1]:40123" + OffgridWitnessCheckInPath},
		{name: "http production rejected", value: "http://api.offgridsoftware.ca" + OffgridBugCheckInPath, wantError: true},
		{name: "http non-loopback rejected", value: "http://192.0.2.1:40123" + OffgridBugCheckInPath, wantError: true},
		{name: "local userinfo rejected", value: "http://user@localhost:40123" + OffgridBugCheckInPath, wantError: true},
		{name: "local missing path rejected", value: "http://localhost:40123", wantError: true},
		{name: "local query rejected", value: "http://localhost:40123" + OffgridBugCheckInPath + "?x=1", wantError: true},
		{name: "local fragment rejected", value: "http://localhost:40123" + OffgridBugCheckInPath + "#x", wantError: true},
		{name: "local whitespace rejected", value: "http://localhost:40123" + OffgridBugCheckInPath + "\n", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseCheckInEndpoint(tc.value)
			if tc.wantError {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("ParseCheckInEndpoint(%q) error = %v, want ErrLicenseContract", tc.value, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCheckInEndpoint(%q) error = %v", tc.value, err)
			}
		})
	}
}

func TestCoreDeviceFingerprintHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "missing prefix", value: strings.Repeat("a", 64)},
		{name: "uppercase digest", value: core.DeviceFingerprintPrefixSHA256 + strings.Repeat("A", 64)},
		{name: "short digest", value: core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 63)},
		{name: "non hex digest", value: core.DeviceFingerprintPrefixSHA256 + strings.Repeat("g", 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := core.ParseDeviceFingerprint(tc.value)
			if !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("ParseDeviceFingerprint error = %v, want ErrFoundationContract", err)
			}
		})
	}
}

func TestBugUsageHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*BugUsage)
		name   string
	}{
		{name: "bad schema", mutate: func(u *BugUsage) { u.Schema = core.SchemaUnknown }},
		{name: "zero start", mutate: func(u *BugUsage) { u.WindowStart = core.UnixNanoTime{} }},
		{name: "end before start", mutate: func(u *BugUsage) { u.WindowEnd = u.WindowStart.Add(-1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			usage := goodBugUsage()
			tc.mutate(&usage)
			if err := usage.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("BugUsage.Validate error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestBugUsageMarshalTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		want    string
		usage   BugUsage
		wantErr bool
	}{
		{name: "zero usage marshals as nullable first check-in body", usage: BugUsage{}, want: `{}`},
		{name: "schema usage marshals typed usage body", usage: goodBugUsage(), want: `{"window_end":1782302401000000000,"window_start":1782302400000000000,"green":0,"reattest":0,"verify":0,"start":0,"show":0,"list":0,"red":0,"license_admin":0,"audit":0,"dupe":0,"init":0,"languages":0,"install_hooks":0,"ledger_admin":0,"schema":"bug-usage-v2026"}`},
		{name: "invalid nonzero usage returns license contract", usage: BugUsage{
			Schema:      core.SchemaUnknown,
			WindowStart: testTime(1782302400000000000),
			WindowEnd:   testTime(1782302401000000000),
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.usage)
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("json.Marshal(BugUsage) error = %v, want ErrLicenseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("json.Marshal(BugUsage) = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestBugCheckInHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*BugCheckIn)
		name   string
	}{
		{name: "bad schema", mutate: func(c *BugCheckIn) { c.Schema = core.SchemaWitnessCheckIn }},
		{name: "bad developer key", mutate: func(c *BugCheckIn) { c.DeveloperKey = DeveloperKey{} }},
		{name: "bad device fingerprint", mutate: func(c *BugCheckIn) { c.DeviceFingerprint = core.DeviceFingerprint{} }},
		{name: "bad device label", mutate: func(c *BugCheckIn) { c.DeviceLabel = DeviceLabel{} }},
		{name: "bad version", mutate: func(c *BugCheckIn) { c.BinaryVersion = core.ProductVersion{} }},
		{name: "bad sha", mutate: func(c *BugCheckIn) { c.BinarySHA256 = core.SHA256Hex{} }},
		{name: "bad platform", mutate: func(c *BugCheckIn) { c.Platform = core.PlatformUnknown }},
		{name: "bad usage", mutate: func(c *BugCheckIn) { c.Usage.Schema = core.SchemaUnknown }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := goodBugCheckIn(t)
			tc.mutate(&payload)
			if err := payload.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("BugCheckIn.Validate error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestWitnessCheckInHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*WitnessCheckIn)
		name   string
	}{
		{name: "bad schema", mutate: func(c *WitnessCheckIn) { c.Schema = core.SchemaBugCheckIn }},
		{name: "bad device fingerprint", mutate: func(c *WitnessCheckIn) { c.DeviceFingerprint = core.DeviceFingerprint{} }},
		{name: "bad version", mutate: func(c *WitnessCheckIn) { c.BinaryVersion = core.ProductVersion{} }},
		{name: "bad sha", mutate: func(c *WitnessCheckIn) { c.BinarySHA256 = core.SHA256Hex{} }},
		{name: "bad platform", mutate: func(c *WitnessCheckIn) { c.Platform = core.PlatformUnknown }},
		{name: "blank account token", mutate: func(c *WitnessCheckIn) { c.AccountToken = AccountToken{} }},
		{name: "bad usage", mutate: func(c *WitnessCheckIn) { c.Usage.Schema = core.SchemaUnknown }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := goodWitnessCheckIn(t)
			tc.mutate(&payload)
			if err := payload.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("WitnessCheckIn.Validate error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestCheckInLeaseProgressionReferenceHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		leaseID    core.LeaseID
		generation LeaseGeneration
		wantErr    bool
	}{
		{name: "first check-in has no progression reference"},
		{name: "signed lease identity and generation travel together", leaseID: testLeaseID(t), generation: 1},
		{name: "lease identity without generation rejected", leaseID: testLeaseID(t), wantErr: true},
		{name: "generation without lease identity rejected", generation: 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := goodBugCheckIn(t)
			payload.LeaseID = tc.leaseID
			payload.LeaseGeneration = tc.generation
			err := payload.Validate()
			if tc.wantErr {
				if !errors.Is(err, core.ErrLeaseGeneration) {
					t.Fatalf("BugCheckIn.Validate() error = %v, want %v", err, core.ErrLeaseGeneration)
				}
				return
			}
			if err != nil {
				t.Fatalf("BugCheckIn.Validate() error = %v", err)
			}
		})
	}
}

func goodBugCheckIn(t *testing.T) BugCheckIn {
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
		Usage:             goodBugUsage(),
	}
}

func goodWitnessCheckIn(t *testing.T) WitnessCheckIn {
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
		RequestNonce:      testCheckInNonce(t),
		Platform:          core.PlatformDarwinARM64,
		AccountToken:      token,
		Usage:             goodWitnessUsage(),
	}
}

func goodWitnessUsage() WitnessUsage {
	return WitnessUsage{
		Schema:      core.SchemaWitnessUsage,
		WindowStart: testTime(1782302400000000000),
		WindowEnd:   testTime(1782302401000000000),
		Quiz:        1,
		Test:        2,
		Midterm:     3,
		Final:       4,
		Store:       5,
		Verify:      6,
	}
}

func TestWitnessUsageHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*WitnessUsage)
		name   string
	}{
		{name: "bad schema", mutate: func(u *WitnessUsage) { u.Schema = core.SchemaUnknown }},
		{name: "unset window start", mutate: func(u *WitnessUsage) { u.WindowStart = core.UnixNanoTime{} }},
		{name: "equal window rejected", mutate: func(u *WitnessUsage) { u.WindowEnd = u.WindowStart }},
		{name: "reversed window rejected", mutate: func(u *WitnessUsage) { u.WindowEnd = u.WindowStart.Add(-1) }},
		{name: "negative window start rejected", mutate: func(u *WitnessUsage) { u.WindowStart = core.UnixNanoTimeFromInt64(-1) }},
		{name: "negative window end rejected", mutate: func(u *WitnessUsage) { u.WindowEnd = core.UnixNanoTimeFromInt64(-1) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			usage := goodWitnessUsage()
			tc.mutate(&usage)
			if err := usage.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("WitnessUsage.Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func goodBugUsage() BugUsage {
	return BugUsage{
		Schema:      core.SchemaBugUsage,
		WindowStart: testTime(1782302400000000000),
		WindowEnd:   testTime(1782302401000000000),
	}
}
