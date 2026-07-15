package custody

import (
	"bytes"
	"crypto/ed25519"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type testFatalHelper interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

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
		Schema:     core.SchemaCustodySessionOpenResponse,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Upload: &SessionUploadGrant{
			Session:   mustSessionID(t),
			Targets:   validUploadTargets(t),
			Retention: mustRetention(),
			ExpiresAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		},
		Disposition: SessionOpenDispositionUploadRequired,
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
	target = validUploadTarget(t)
	target.Headers = make([]core.UploadHeader, core.HTTPHeaderMaximumDefault+1)
	if err := target.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("UploadTarget oversized headers error = %v, want ErrCustodyContract", err)
	}
	response.Upload.Targets = append(response.Upload.Targets, response.Upload.Targets[0])
	if err := response.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("SessionOpenResponse duplicate target error = %v, want ErrCustodyContract", err)
	}
	response.Upload.Targets = make([]UploadTarget, core.CollectionMaximumDefault+1)
	if err := response.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("SessionOpenResponse oversized targets error = %v, want ErrCustodyContract", err)
	}
	response.Upload.Targets = validUploadTargets(t)
	response.Upload.Targets[0].Provider = core.StorageProviderS3
	if err := response.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("SessionOpenResponse non-GCS target error = %v, want ErrCustodyContract", err)
	}
}

func TestSessionOpenResponseReusesSignedReceipt(t *testing.T) {
	t.Parallel()

	signed := mustSignedReceipt(t)
	response := SessionOpenResponse{
		Schema:          core.SchemaCustodySessionOpenResponse,
		Customer:        signed.Body.Customer,
		BundleRoot:      signed.Body.BundleRoot,
		ExistingReceipt: &signed,
		Disposition:     SessionOpenDispositionReceiptReused,
	}
	if err := response.Validate(); err != nil {
		t.Fatalf("SessionOpenResponse.Validate() error = %v, want nil", err)
	}

	response.Upload = &SessionUploadGrant{Targets: validUploadTargets(t)}
	if err := response.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("receipt reuse with upload targets error = %v, want ErrCustodyContract", err)
	}
}

func TestSessionOpenResponseVerifyRejectsForgedReusedReceipt(t *testing.T) {
	t.Parallel()
	signed, keyring := verifiedSignedReceipt(t)
	body := signed.Body
	response := SessionOpenResponse{
		Schema: core.SchemaCustodySessionOpenResponse, Customer: body.Customer, BundleRoot: body.BundleRoot,
		ExistingReceipt: &signed, Disposition: SessionOpenDispositionReceiptReused,
	}
	if err := response.Verify(keyring); err != nil {
		t.Fatalf("SessionOpenResponse.Verify() error = %v", err)
	}
	response.ExistingReceipt.Body.LedgerSeq++
	if err := response.Verify(keyring); !errors.Is(err, core.ErrFoundationContract) {
		t.Fatalf("SessionOpenResponse.Verify(tampered) error = %v, want ErrFoundationContract", err)
	}
}

func verifiedSignedReceipt(t testFatalHelper) (core.Signed[ReceiptBody], core.SigningKeyring) {
	t.Helper()
	keyID, publicKey, privateKey := receiptSigningKey(t)
	body := validReceipt(t)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[ReceiptBody]{Body: body, KeyID: keyID, Signature: signature},
		core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
}

func receiptSigningKey(t testFatalHelper) (core.SigningKeyID, core.Ed25519PublicKeyHex, ed25519.PrivateKey) {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, ed25519.SeedSize))
	publicKey, err := core.NewEd25519PublicKeyHex(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := core.ParseSigningKeyID("custody-receipt-key-2026")
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicKey, privateKey
}

func TestSignedUploadURLWireContract(t *testing.T) {
	t.Parallel()
	signedURL := mustSignedUploadURL(t)
	if signedURL.String() == "" {
		t.Fatalf("core.SignedUploadURL.String empty")
	}
	if err := signedURL.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := signedURL.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip core.SignedUploadURL
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != signedURL {
		t.Fatalf("core.SignedUploadURL roundTrip = %s, want %s", roundTrip, signedURL)
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
		`"schema":"witness-custody-receipt-v2026","receipt_id":"01J00000000000000000000001",` +
		`"customer_id":"01HZZZZZZZZZZZZZZZZZZZZZZZ","bundle_root":"` + strings.Repeat("f", 64) + `",` +
		`"session_id":"01J00000000000000000000000","chain_hash":"` + strings.Repeat("e", 64) + `",` +
		`"objects":[{"artifact":"bundle.tar","object":"` + witnessObjectPath(t) + `",` +
		`"generation":"1710000000000000","sha256":"` + strings.Repeat("c", 64) + `","blake3":"` + strings.Repeat("d", 64) + `","size_bytes":12,"provider":"gcs"}],` +
		`"timestamp":{"authority":"freetsa","bundle_root":"` + strings.Repeat("f", 64) + `","imprint_sha256":"` + body.Timestamp.ImprintSHA256.String() + `",` +
		`"token":"` + body.Timestamp.Token.String() + `","response":"` + body.Timestamp.Response.String() + `","timestamped_at":1782302399500000000},` +
		`"retention":{"retain_until":1815000000000000000,"maximum_retain_until":1878000000000000000,"class":"conditional"},` +
		`"issued_at":1782302400000000000,"accepted_at":1782302399000000000,"ledger_seq":7}`
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

func TestTimestampProofAndGCSOnlyCustodyHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *ReceiptBody)
		name   string
		accept bool
	}{
		{name: "exact GCS object and embedded RFC3161 token accepted", accept: true},
		{name: "S3 object refused", mutate: func(_ *testing.T, body *ReceiptBody) {
			body.Objects[0].Provider = core.StorageProviderS3
		}},
		{name: "timestamp bundle root swap refused", mutate: func(t *testing.T, body *ReceiptBody) {
			body.Timestamp.BundleRoot = mustBLAKE3(t, "a")
		}},
		{name: "timestamp imprint swap refused", mutate: func(t *testing.T, body *ReceiptBody) {
			body.Timestamp.ImprintSHA256 = mustSHA256(t, "a")
		}},
		{name: "token absent from response refused", mutate: func(t *testing.T, body *ReceiptBody) {
			body.Timestamp.Token = mustAlternateTimestampToken(t)
		}},
		{name: "timestamp before acceptance refused", mutate: func(_ *testing.T, body *ReceiptBody) {
			body.Timestamp.TimestampedAt = body.AcceptedAt.Add(-time.Nanosecond)
		}},
		{name: "timestamp after receipt issuance refused", mutate: func(_ *testing.T, body *ReceiptBody) {
			body.Timestamp.TimestampedAt = body.IssuedAt.Add(time.Nanosecond)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := validReceipt(t)
			if tc.mutate != nil {
				tc.mutate(t, &body)
			}
			err := body.Validate()
			if tc.accept {
				if err != nil {
					t.Fatalf("ReceiptBody.Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("ReceiptBody.Validate() error = %v, want %v", err, core.ErrCustodyContract)
			}
		})
	}
}

func TestRFC3161DERBoundaryHostileTable(t *testing.T) {
	t.Parallel()

	proof := mustTimestampProof(t)
	validToken := proof.Token.String()
	validResponse := proof.Response.String()
	wrongOID, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	missingSignedData, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2})
	if err != nil {
		t.Fatal(err)
	}
	oversized := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, RFC3161DERMaximumBytes+1))
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "empty token refused", value: ""},
		{name: "invalid base64 token refused", value: "%%%"},
		{name: "noncanonical base64 token refused", value: "TQ"},
		{name: "non-DER token refused", value: "YQ=="},
		{name: "empty DER sequence token refused", value: "MAA="},
		{name: "wrong content type token refused", value: base64.StdEncoding.EncodeToString(testDERSequence(t, append(wrongOID, []byte{0xa0, 0x02, 0x30, 0x00}...)))},
		{name: "missing signed data token refused", value: base64.StdEncoding.EncodeToString(testDERSequence(t, missingSignedData))},
		{name: "trailing token bytes refused", value: validToken + "AA=="},
		{name: "oversized token refused", value: oversized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseRFC3161Token(tc.value); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("ParseRFC3161Token() error = %v, want %v", err, core.ErrCustodyContract)
			}
		})
	}

	failedStatus, err := asn1.Marshal(struct{ Status int }{Status: 2})
	if err != nil {
		t.Fatal(err)
	}
	tokenBytes, err := proof.Token.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{name: "empty response refused", raw: nil},
		{name: "non-DER response refused", raw: []byte("response")},
		{name: "failure status response refused", raw: testDERSequence(t, append(failedStatus, tokenBytes...))},
		{name: "response without token refused", raw: testDERSequence(t, []byte{0x30, 0x03, 0x02, 0x01, 0x00})},
		{name: "response with trailing bytes refused", raw: append(mustTimestampResponseBytes(t, validResponse), 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewRFC3161Response(tc.raw); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("NewRFC3161Response() error = %v, want %v", err, core.ErrCustodyContract)
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
		headers []core.UploadHeader
	}{
		{name: "duplicate exact header name", headers: []core.UploadHeader{
			{Name: core.HTTPHeaderContentType, Value: core.HTTPContentTypeJSON},
			{Name: core.HTTPHeaderContentType, Value: "application/octet-stream"},
		}},
		{name: "duplicate case-folded header name", headers: []core.UploadHeader{
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
		Schema:     core.SchemaCustodyFinalizeRequest,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Session:    mustSessionID(t),
	}
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest error = %v, want ErrCustodyContract", err)
	}
}

func TestFinalizeRejectsDuplicateObjects(t *testing.T) {
	t.Parallel()
	object := mustUploadedObject(t)
	req := FinalizeRequest{
		Schema:     core.SchemaCustodyFinalizeRequest,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Session:    mustSessionID(t),
		Objects:    []UploadedObject{object, object},
	}
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest duplicate object error = %v, want ErrCustodyContract", err)
	}
}

func TestFinalizeGCSOnlyContract(t *testing.T) {
	t.Parallel()

	req := FinalizeRequest{
		Schema:     core.SchemaCustodyFinalizeRequest,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Session:    mustSessionID(t),
		Objects:    mustUploadedObjects(t),
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("FinalizeRequest.Validate() error = %v, want nil", err)
	}

	req.Objects[0].Provider = core.StorageProviderS3
	if err := req.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("FinalizeRequest provider error = %v, want ErrCustodyContract", err)
	}
}

func TestReceiptRejectsCustodyIdentityDrift(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *ReceiptBody)
		name   string
	}{
		{name: "object byte count invalid", mutate: func(_ *testing.T, b *ReceiptBody) { b.Objects[0].Size = core.NewByteCount(0) }},
		{name: "retention exceeds plan ceiling", mutate: func(_ *testing.T, b *ReceiptBody) {
			b.Retention.RetainUntil = b.Retention.MaximumRetainUntil.Add(time.Nanosecond)
		}},
		{name: "issued before accepted", mutate: func(_ *testing.T, b *ReceiptBody) { b.IssuedAt = b.AcceptedAt.Add(-time.Nanosecond) }},
		{name: "retention ends before issuance", mutate: func(_ *testing.T, b *ReceiptBody) {
			b.Retention.RetainUntil = b.IssuedAt.Add(-time.Nanosecond)
		}},
		{name: "bundle root differs from object key", mutate: func(t *testing.T, b *ReceiptBody) { b.BundleRoot = mustBLAKE3(t, "a") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := validReceipt(t)
			tc.mutate(t, &body)
			if err := body.Validate(); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("ReceiptBody.Validate() error = %v, want ErrCustodyContract", err)
			}
		})
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

func validOpenRequest(t testFatalHelper) SessionOpenRequest {
	t.Helper()
	return SessionOpenRequest{
		Schema:        core.SchemaCustodySessionOpenRequest,
		Customer:      mustCustomerID(t),
		BundleRoot:    mustBundleRoot(t),
		Lease:         mustOpenLeaseRef(t),
		Release:       mustRelease(t),
		Artifacts:     []ArtifactDescriptor{mustArtifact(t)},
		TotalBytes:    core.NewByteCount(12),
		ArtifactCount: 1,
	}
}

func validReceipt(t testFatalHelper) ReceiptBody {
	t.Helper()
	return ReceiptBody{
		Schema:     core.SchemaCustodyReceipt,
		ReceiptID:  mustReceiptID(t),
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Session:    mustSessionID(t),
		Release:    mustRelease(t),
		Objects:    mustUploadedObjects(t),
		Timestamp:  mustTimestampProof(t),
		Retention:  mustRetention(),
		LedgerSeq:  mustLedgerSeq(t, 7),
		ChainHash:  mustSHA256(t, "e"),
		AcceptedAt: core.UnixNanoTimeFromInt64(1782302399000000000),
		IssuedAt:   core.UnixNanoTimeFromInt64(1782302400000000000),
	}
}

func mustOpenLeaseRef(t testFatalHelper) OpenLeaseRef {
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

func mustRelease(t testFatalHelper) ReleaseIdentity {
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

func mustArtifact(t testFatalHelper) ArtifactDescriptor {
	t.Helper()
	return artifactWithNameAndSize(t, "bundle.tar", 12)
}

func artifactWithNameAndSize(t testFatalHelper, name string, size uint64) ArtifactDescriptor {
	t.Helper()
	return ArtifactDescriptor{
		Name:   mustArtifactNameValue(t, name),
		Size:   core.NewByteCount(size),
		SHA256: mustSHA256(t, "c"),
		BLAKE3: mustBLAKE3(t, "d"),
	}
}

func mustUploadedObject(t testFatalHelper) UploadedObject {
	t.Helper()
	object, err := ParseObjectPath(witnessObjectPath(t))
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
		Provider:   core.StorageProviderGCS,
	}
}

func mustUploadedObjects(t testFatalHelper) []UploadedObject {
	t.Helper()
	return []UploadedObject{mustUploadedObject(t)}
}

func validUploadTarget(t testFatalHelper) UploadTarget {
	t.Helper()
	return UploadTarget{
		Artifact: mustArtifactNameValue(t, "bundle.tar"),
		Object:   mustObjectPath(t),
		URL:      mustSignedUploadURL(t),
		Headers: []core.UploadHeader{{
			Name:  core.HTTPHeaderContentType,
			Value: core.HTTPContentTypeJSON,
		}},
		Provider: core.StorageProviderGCS,
		Method:   core.UploadMethodSignedPUT,
	}
}

func validUploadTargets(t testFatalHelper) []UploadTarget {
	t.Helper()
	return []UploadTarget{validUploadTarget(t)}
}

func mustTimestampProof(t testFatalHelper) TimestampProof {
	t.Helper()
	status, err := asn1.Marshal(struct{ Status int }{Status: 0})
	if err != nil {
		t.Fatal(err)
	}
	contentType, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2})
	if err != nil {
		t.Fatal(err)
	}
	tokenBytes := testDERSequence(t, append(contentType, []byte{0xa0, 0x02, 0x30, 0x00}...))
	responseBytes := testDERSequence(t, append(status, tokenBytes...))
	token, err := NewRFC3161Token(tokenBytes)
	if err != nil {
		t.Fatal(err)
	}
	response, err := NewRFC3161Response(responseBytes)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := BuildTimestampProof(TimestampProofInput{
		Authority: TimestampAuthorityFreeTSA, BundleRoot: mustBundleRoot(t), Token: token, Response: response,
		TimestampedAt: core.UnixNanoTimeFromInt64(1782302399500000000),
	})
	if err != nil {
		t.Fatal(err)
	}
	return proof
}

func mustAlternateTimestampToken(t testFatalHelper) RFC3161Token {
	t.Helper()
	contentType, err := asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2})
	if err != nil {
		t.Fatal(err)
	}
	token, err := NewRFC3161Token(testDERSequence(t, append(contentType, []byte{0xa0, 0x03, 0x02, 0x01, 0x01}...)))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func mustTimestampResponseBytes(t testFatalHelper, value string) []byte {
	t.Helper()
	response, err := ParseRFC3161Response(value)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := response.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testDERSequence(t testFatalHelper, content []byte) []byte {
	t.Helper()
	if len(content) >= 128 {
		t.Fatalf("test DER content bytes = %d, want < 128", len(content))
	}
	return append([]byte{0x30, byte(len(content))}, content...)
}

func mustObjectPath(t testFatalHelper) ObjectPath {
	t.Helper()
	object, err := ParseObjectPath(witnessObjectPath(t))
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func mustSignedUploadURL(t testFatalHelper) core.SignedUploadURL {
	t.Helper()
	signedURL, err := core.ParseSignedUploadURL("https://storage.googleapis.com/offgrid-custody/bundle.tar?signature=abc")
	if err != nil {
		t.Fatal(err)
	}
	return signedURL
}

func mustGeneration(t testFatalHelper) Generation {
	t.Helper()
	generation, err := ParseGeneration("1710000000000000")
	if err != nil {
		t.Fatal(err)
	}
	return generation
}

func mustLedgerSeq(t testFatalHelper, value int64) LedgerSeq {
	t.Helper()
	seq, err := NewLedgerSeq(value)
	if err != nil {
		t.Fatal(err)
	}
	return seq
}

func mustRetention() RetentionPolicy {
	return RetentionPolicy{
		Class:              RetentionClassConditional,
		RetainUntil:        core.NewUnixNanoTime(time.Unix(0, 1815000000000000000)),
		MaximumRetainUntil: core.NewUnixNanoTime(time.Unix(0, 1878000000000000000)),
	}
}

func mustCustomerID(t testFatalHelper) CustomerID {
	t.Helper()
	id, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZZ")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustSessionID(t testFatalHelper) SessionID {
	t.Helper()
	id, err := ParseSessionID("01J00000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustReceiptID(t testFatalHelper) ReceiptID {
	t.Helper()
	id, err := ParseReceiptID("01J00000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustBundleRoot(t testFatalHelper) core.BLAKE3Hex {
	t.Helper()
	return mustBLAKE3(t, "f")
}

func mustSignedReceipt(t testFatalHelper) core.Signed[ReceiptBody] {
	t.Helper()
	keyID, err := core.ParseSigningKeyID("custody-receipt-key-1")
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.ParseEd25519SignatureHex(strings.Repeat("a", 128))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[ReceiptBody]{Body: validReceipt(t), KeyID: keyID, Signature: signature}
}

func witnessObjectPath(t testFatalHelper) string {
	t.Helper()
	return core.WitnessCustodyPathRoot + "/" + mustCustomerID(t).String() + "/2026/07/" + mustBundleRoot(t).String() + "/bundle.tar"
}

func mustArtifactNameValue(t testFatalHelper, value string) ArtifactName {
	t.Helper()
	name, err := ParseArtifactName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func mustSHA256(t testFatalHelper, digit string) core.SHA256Hex {
	t.Helper()
	sum, err := core.ParseSHA256Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}

func mustBLAKE3(t testFatalHelper, digit string) core.BLAKE3Hex {
	t.Helper()
	sum, err := core.ParseBLAKE3Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
