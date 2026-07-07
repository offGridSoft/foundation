package custody

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/core"
)

func TestSessionOpenRequestHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*SessionOpenRequest)
		name   string
	}{
		{name: "schema mismatch", mutate: func(r *SessionOpenRequest) { r.Schema = SchemaFinalizeRequest }},
		{name: "artifact count mismatch", mutate: func(r *SessionOpenRequest) { r.ArtifactCount++ }},
		{name: "total byte mismatch", mutate: func(r *SessionOpenRequest) { r.TotalBytes = core.NewByteCount(99) }},
		{name: "artifact with path separator", mutate: func(r *SessionOpenRequest) {
			r.Artifacts[0].Name = ArtifactName{}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := validOpenRequest(t)
			tc.mutate(&req)
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
		{name: "object path empty segment", run: func() error { _, err := ParseObjectPath("customer//x"); return err }},
		{name: "signed url http", run: func() error { _, err := ParseSignedUploadURL("http://storage.example/upload"); return err }},
		{name: "customer id lowercase", run: func() error { _, err := ParseCustomerID(strings.Repeat("a", ULIDTextLen)); return err }},
		{name: "customer id illegal rune", run: func() error { _, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZI"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("%s error = %v, want ErrCustodyContract", tc.name, err)
			}
		})
	}
}

func TestReceiptCanonicalWireForm(t *testing.T) {
	t.Parallel()
	body := validReceipt(t)
	got, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"retention":{"retain_until":1815000000000000000,"class":"conditional"},` +
		`"issued_at":1782302400000000000,"release":{"version":"1.0.0",` +
		`"commit":"` + strings.Repeat("a", 40) + `","tool_manifest_sha":"` + strings.Repeat("b", 64) + `"},` +
		`"schema":"witness-custody-receipt-v1","customer_id":"01HZZZZZZZZZZZZZZZZZZZZZZZ",` +
		`"session_id":"01J00000000000000000000000","chain_hash":"` + strings.Repeat("e", 64) + `",` +
		`"objects":[{"artifact":"bundle.tar","object":"customers/01HZZZZZZZZZZZZZZZZZZZZZZZ/witness/2026/07/07/01J00000000000000000000000/bundle.tar",` +
		`"generation":"1710000000000000","sha256":"` + strings.Repeat("c", 64) + `","blake3":"` + strings.Repeat("d", 64) + `","size_bytes":12}],` +
		`"ledger_seq":7,"provider":"gcs"}`
	if string(got) != want {
		t.Fatalf("receipt canonical\n got: %s\nwant: %s", got, want)
	}
}

func TestFinalizeRejectsEmptyObjects(t *testing.T) {
	t.Parallel()
	req := FinalizeRequest{
		Schema:  SchemaFinalizeRequest,
		Session: mustSessionID(t),
	}
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest error = %v, want ErrCustodyContract", err)
	}
}

func validOpenRequest(t *testing.T) SessionOpenRequest {
	t.Helper()
	return SessionOpenRequest{
		Schema:        SchemaSessionOpenRequest,
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
		Schema:    SchemaReceipt,
		Customer:  mustCustomerID(t),
		Session:   mustSessionID(t),
		Release:   mustRelease(t),
		Objects:   []UploadedObject{mustUploadedObject(t)},
		Retention: mustRetention(),
		Provider:  StorageProviderGCS,
		LedgerSeq: 7,
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
	version, err := core.ParseProductVersion("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	commit, err := ParseBuildCommit(strings.Repeat("a", 40))
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
	return ArtifactDescriptor{
		Name:   mustArtifactNameValue("bundle.tar"),
		Size:   core.NewByteCount(12),
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
		Artifact:   mustArtifactNameValue("bundle.tar"),
		Object:     object,
		Generation: mustGeneration(t),
		Size:       core.NewByteCount(12),
		SHA256:     mustSHA256(t, "c"),
		BLAKE3:     mustBLAKE3(t, "d"),
	}
}

func mustGeneration(t *testing.T) Generation {
	t.Helper()
	generation, err := ParseGeneration("1710000000000000")
	if err != nil {
		t.Fatal(err)
	}
	return generation
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

func mustArtifactNameValue(value string) ArtifactName {
	name, err := ParseArtifactName(value)
	if err != nil {
		panic(err)
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
