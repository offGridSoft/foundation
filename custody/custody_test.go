package custody

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSessionOpenRequestHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*testing.T, *SessionOpenRequest)
		name   string
	}{
		{name: "schema mismatch", mutate: func(_ *testing.T, r *SessionOpenRequest) { r.Schema = core.SchemaCustodyFinalizeRequest }},
		{name: "artifact count mismatch", mutate: func(_ *testing.T, r *SessionOpenRequest) { r.ArtifactCount++ }},
		{name: "total byte mismatch", mutate: func(_ *testing.T, r *SessionOpenRequest) { r.TotalBytes = core.NewByteCount(99) }},
		{name: "duplicate artifact name", mutate: func(_ *testing.T, r *SessionOpenRequest) {
			r.Artifacts = append(r.Artifacts, r.Artifacts[0])
			r.ArtifactCount = 2
			r.TotalBytes = core.NewByteCount(24)
		}},
		{name: "artifact size overflow", mutate: func(t *testing.T, r *SessionOpenRequest) {
			r.Artifacts = []ArtifactDescriptor{
				artifactWithNameAndSize(t, "bundle.tar", math.MaxUint64),
				artifactWithNameAndSize(t, "bundle-2.tar", 2),
			}
			r.ArtifactCount = 2
			r.TotalBytes = core.NewByteCount(1)
		}},
		{name: "artifact with path separator", mutate: func(_ *testing.T, r *SessionOpenRequest) {
			r.Artifacts[0].Name = ArtifactName{}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := validOpenRequest(t)
			tc.mutate(t, &req)
			if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("Validate error = %v, want ErrCustodyContract", err)
			}
		})
	}
}

func TestStorageScalarsRejectHostileInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "object path absolute", run: func() error { _, err := ParseObjectPath("/customer/x"); return err }},
		{name: "object path traversal", run: func() error { _, err := ParseObjectPath("customer/../x"); return err }},
		{name: "object path lone traversal", run: func() error { _, err := ParseObjectPath(".."); return err }},
		{name: "object path trailing traversal", run: func() error { _, err := ParseObjectPath("customer/.."); return err }},
		{name: "object path current directory", run: func() error { _, err := ParseObjectPath("customer/./x"); return err }},
		{name: "object path empty segment", run: func() error { _, err := ParseObjectPath("customer//x"); return err }},
		{name: "object path trailing slash", run: func() error { _, err := ParseObjectPath("customer/x/"); return err }},
		{name: "object path backslash traversal", run: func() error { _, err := ParseObjectPath(`customer\..\x`); return err }},
		{name: "object path control rune", run: func() error { _, err := ParseObjectPath("customer/\x00/x"); return err }},
		{name: "object path too long", run: func() error { _, err := ParseObjectPath(strings.Repeat("a", core.PathTokenMaxRunes+1)); return err }},
		{name: "artifact name control rune", run: func() error { _, err := ParseArtifactName("bundle\x00.tar"); return err }},
		{name: "artifact name too long", run: func() error {
			_, err := ParseArtifactName(strings.Repeat("a", core.FileNameTokenMaxRunes+1))
			return err
		}},
		{name: "artifact name dot rejected", run: func() error { _, err := ParseArtifactName("."); return err }},
		{name: "artifact name dot dot rejected", run: func() error { _, err := ParseArtifactName(".."); return err }},
		{name: "signed url http", run: func() error { _, err := ParseSignedUploadURL("http://storage.example/upload"); return err }},
		{name: "signed url missing path", run: func() error { _, err := ParseSignedUploadURL("https://storage.example"); return err }},
		{name: "signed url userinfo", run: func() error {
			_, err := ParseSignedUploadURL("https://trusted.example@evil.example/upload")
			return err
		}},
		{name: "signed url control rune", run: func() error {
			_, err := ParseSignedUploadURL("https://storage.example/upload\nx")
			return err
		}},
		{name: "customer id lowercase", run: func() error { _, err := ParseCustomerID(strings.Repeat("a", ULIDTextLen)); return err }},
		{name: "customer id illegal rune", run: func() error { _, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZI"); return err }},
		{name: "customer id non-canonical first rune", run: func() error { _, err := ParseCustomerID("81HZZZZZZZZZZZZZZZZZZZZZZZ"); return err }},
		{name: "session id non-canonical first rune", run: func() error { _, err := ParseSessionID("81J00000000000000000000000"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("%s error = %v, want ErrCustodyContract", tc.name, err)
			}
		})
	}
}

func TestUploadMethodWireContract(t *testing.T) {
	t.Parallel()
	method := core.UploadMethodSignedPUT
	if !method.IsValid() || method.String() != "signed_put" {
		t.Fatalf("UploadMethod = %q valid=%v", method.String(), method.IsValid())
	}
	if err := method.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := method.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip core.UploadMethod
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != method {
		t.Fatalf("UploadMethod roundTrip = %s, want %s", roundTrip, method)
	}
	for _, raw := range []string{`"copy"`, `0`, `""`} {
		var parsed core.UploadMethod
		if err := parsed.UnmarshalJSON([]byte(raw)); !errors.Is(err, core.ErrFoundationContract) {
			t.Fatalf("UploadMethod(%s) error = %v, want ErrFoundationContract", raw, err)
		}
	}
}

func TestSessionOpenResponseAndUploadTargetContracts(t *testing.T) {
	t.Parallel()
	target := validUploadTarget(t)
	if err := target.Validate(); err != nil {
		t.Fatal(err)
	}
	response := SessionOpenResponse{
		Schema:    core.SchemaCustodySessionOpenResponse,
		Session:   mustSessionID(t),
		Targets:   []UploadTarget{target},
		Retention: mustRetention(),
		ExpiresAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	}
	if err := response.Validate(); err != nil {
		t.Fatal(err)
	}
	target.Headers[0].Name = "bad\nheader"
	if err := target.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("UploadTarget invalid header error = %v, want ErrCustodyContract", err)
	}
	target = validUploadTarget(t)
	target.Headers[0].Name = " Content-Type "
	if err := target.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("UploadTarget spaced header error = %v, want ErrCustodyContract", err)
	}
	target = validUploadTarget(t)
	target.Headers[0].Value = "application/json\x00"
	if err := target.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("UploadTarget control header value error = %v, want ErrCustodyContract", err)
	}
	response.Targets = append(response.Targets, target)
	if err := response.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("SessionOpenResponse duplicate target error = %v, want ErrCustodyContract", err)
	}
}

func TestSignedUploadURLWireContract(t *testing.T) {
	t.Parallel()
	signedURL := mustSignedUploadURL(t)
	if signedURL.String() == "" {
		t.Fatalf("SignedUploadURL.String empty")
	}
	if err := signedURL.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := signedURL.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip SignedUploadURL
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != signedURL {
		t.Fatalf("SignedUploadURL roundTrip = %s, want %s", roundTrip, signedURL)
	}
}

func TestReceiptCanonicalWireForm(t *testing.T) {
	t.Parallel()
	body := validReceipt(t)
	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"release":{"version":"2026.0.0",` +
		`"commit":"` + strings.Repeat("a", 40) + `","tool_manifest_sha":"` + strings.Repeat("b", 64) + `"},` +
		`"schema":"witness-custody-receipt-v1",` +
		`"customer_id":"01HZZZZZZZZZZZZZZZZZZZZZZZ",` +
		`"session_id":"01J00000000000000000000000","chain_hash":"` + strings.Repeat("e", 64) + `",` +
		`"objects":[{"artifact":"bundle.tar","object":"customers/01HZZZZZZZZZZZZZZZZZZZZZZZ/witness/2026/07/07/01J00000000000000000000000/bundle.tar",` +
		`"generation":"1710000000000000","sha256":"` + strings.Repeat("c", 64) + `","blake3":"` + strings.Repeat("d", 64) + `","size_bytes":12}],` +
		`"retention":{"retain_until":1815000000000000000,"class":"conditional"},` +
		`"issued_at":1782302400000000000,"ledger_seq":7,"provider":"gcs"}`
	if string(got) != want {
		t.Fatalf("receipt canonical\n got: %s\nwant: %s", got, want)
	}
}

func TestReceiptCanonicalRoundTripTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body ReceiptBody
	}{
		{name: "receipt", body: validReceipt(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original, err := tc.body.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := core.DecodeStrictJSON[ReceiptBody](original)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Fatalf("receipt canonical round trip\n got: %s\nwant: %s", got, original)
			}
		})
	}
}

func TestReceiptRejectsNegativeLedgerSeq(t *testing.T) {
	t.Parallel()
	body := validReceipt(t)
	body.LedgerSeq = LedgerSeq(-7)
	if err := body.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("ReceiptBody.Validate error = %v, want ErrCustodyContract", err)
	}
}

func TestUploadTargetRejectsDuplicateHeadersHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		headers []UploadHeader
	}{
		{name: "duplicate exact header name", headers: []UploadHeader{
			{Name: core.HTTPHeaderContentType, Value: core.HTTPContentTypeJSON},
			{Name: core.HTTPHeaderContentType, Value: "application/octet-stream"},
		}},
		{name: "duplicate case-folded header name", headers: []UploadHeader{
			{Name: "Content-Type", Value: core.HTTPContentTypeJSON},
			{Name: "content-type", Value: "application/octet-stream"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := validUploadTarget(t)
			target.Headers = tc.headers
			if err := target.Validate(); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("UploadTarget.Validate() error = %v, want ErrCustodyContract", err)
			}
		})
	}
}

func TestReceiptRejectsDuplicateObjects(t *testing.T) {
	t.Parallel()
	body := validReceipt(t)
	body.Objects = append(body.Objects, body.Objects[0])
	if err := body.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("ReceiptBody duplicate object error = %v, want ErrCustodyContract", err)
	}
}

func TestFinalizeRejectsEmptyObjects(t *testing.T) {
	t.Parallel()
	req := FinalizeRequest{
		Schema:  core.SchemaCustodyFinalizeRequest,
		Session: mustSessionID(t),
	}
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest error = %v, want ErrCustodyContract", err)
	}
}

func TestFinalizeRejectsDuplicateObjects(t *testing.T) {
	t.Parallel()
	object := mustUploadedObject(t)
	req := FinalizeRequest{
		Schema:  core.SchemaCustodyFinalizeRequest,
		Session: mustSessionID(t),
		Objects: []UploadedObject{object, object},
	}
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest duplicate object error = %v, want ErrCustodyContract", err)
	}
}

func TestGenerationRejectsControlToken(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"", " generation", "generation\n1", strings.Repeat("a", core.OpaqueTokenDefaultMaxRunes+1)} {
		if _, err := ParseGeneration(raw); !errors.Is(err, core.ErrCustodyContract) {
			t.Fatalf("ParseGeneration(%q) error = %v, want ErrCustodyContract", raw, err)
		}
	}
}

func validOpenRequest(t *testing.T) SessionOpenRequest {
	t.Helper()
	return SessionOpenRequest{
		Schema:        core.SchemaCustodySessionOpenRequest,
		Customer:      mustCustomerID(t),
		Lease:         mustOpenLeaseRef(t),
		Release:       mustRelease(t),
		Artifacts:     []ArtifactDescriptor{mustArtifact(t)},
		TotalBytes:    core.NewByteCount(12),
		ArtifactCount: 1,
	}
}

func validReceipt(t *testing.T) ReceiptBody {
	t.Helper()
	return ReceiptBody{
		Schema:    core.SchemaCustodyReceipt,
		Customer:  mustCustomerID(t),
		Session:   mustSessionID(t),
		Release:   mustRelease(t),
		Objects:   []UploadedObject{mustUploadedObject(t)},
		Retention: mustRetention(),
		Provider:  core.StorageProviderGCS,
		LedgerSeq: mustLedgerSeq(t, 7),
		ChainHash: mustSHA256(t, "e"),
		IssuedAt:  core.UnixNanoTimeFromInt64(1782302400000000000),
	}
}

func mustOpenLeaseRef(t *testing.T) OpenLeaseRef {
	t.Helper()
	leaseID, err := core.ParseLeaseID("lease-1")
	if err != nil {
		t.Fatal(err)
	}
	fp, err := core.ParseDeviceFingerprint(core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return OpenLeaseRef{LeaseID: leaseID, DeviceFingerprint: fp}
}

func mustRelease(t *testing.T) ReleaseIdentity {
	t.Helper()
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := core.ParseBuildCommit(strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	return ReleaseIdentity{
		Version:     version,
		Commit:      commit,
		ManifestSHA: mustSHA256(t, "b"),
	}
}

func mustArtifact(t *testing.T) ArtifactDescriptor {
	t.Helper()
	return artifactWithNameAndSize(t, "bundle.tar", 12)
}

func artifactWithNameAndSize(t *testing.T, name string, size uint64) ArtifactDescriptor {
	t.Helper()
	return ArtifactDescriptor{
		Name:   mustArtifactNameValue(t, name),
		Size:   core.NewByteCount(size),
		SHA256: mustSHA256(t, "c"),
		BLAKE3: mustBLAKE3(t, "d"),
	}
}

func mustUploadedObject(t *testing.T) UploadedObject {
	t.Helper()
	object, err := ParseObjectPath("customers/01HZZZZZZZZZZZZZZZZZZZZZZZ/witness/2026/07/07/01J00000000000000000000000/bundle.tar")
	if err != nil {
		t.Fatal(err)
	}
	return UploadedObject{
		Artifact:   mustArtifactNameValue(t, "bundle.tar"),
		Object:     object,
		Generation: mustGeneration(t),
		Size:       core.NewByteCount(12),
		SHA256:     mustSHA256(t, "c"),
		BLAKE3:     mustBLAKE3(t, "d"),
	}
}

func validUploadTarget(t *testing.T) UploadTarget {
	t.Helper()
	return UploadTarget{
		Artifact: mustArtifactNameValue(t, "bundle.tar"),
		Object:   mustObjectPath(t),
		URL:      mustSignedUploadURL(t),
		Headers: []UploadHeader{{
			Name:  core.HTTPHeaderContentType,
			Value: core.HTTPContentTypeJSON,
		}},
		Provider: core.StorageProviderGCS,
		Method:   core.UploadMethodSignedPUT,
	}
}

func mustObjectPath(t *testing.T) ObjectPath {
	t.Helper()
	object, err := ParseObjectPath("customers/01HZZZZZZZZZZZZZZZZZZZZZZZ/witness/2026/07/07/01J00000000000000000000000/bundle.tar")
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func mustSignedUploadURL(t *testing.T) SignedUploadURL {
	t.Helper()
	signedURL, err := ParseSignedUploadURL("https://storage.googleapis.com/offgrid-custody/bundle.tar?signature=abc")
	if err != nil {
		t.Fatal(err)
	}
	return signedURL
}

func mustGeneration(t *testing.T) Generation {
	t.Helper()
	generation, err := ParseGeneration("1710000000000000")
	if err != nil {
		t.Fatal(err)
	}
	return generation
}

func mustLedgerSeq(t *testing.T, value int64) LedgerSeq {
	t.Helper()
	seq, err := NewLedgerSeq(value)
	if err != nil {
		t.Fatal(err)
	}
	return seq
}

func mustRetention() RetentionPolicy {
	return RetentionPolicy{
		Class:       RetentionClassConditional,
		RetainUntil: core.NewUnixNanoTime(time.Unix(0, 1815000000000000000)),
	}
}

func mustCustomerID(t *testing.T) CustomerID {
	t.Helper()
	id, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZZ")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustSessionID(t *testing.T) SessionID {
	t.Helper()
	id, err := ParseSessionID("01J00000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustArtifactNameValue(t *testing.T, value string) ArtifactName {
	t.Helper()
	name, err := ParseArtifactName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func mustSHA256(t *testing.T, digit string) core.SHA256Hex {
	t.Helper()
	sum, err := core.ParseSHA256Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}

func mustBLAKE3(t *testing.T, digit string) core.BLAKE3Hex {
	t.Helper()
	sum, err := core.ParseBLAKE3Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
