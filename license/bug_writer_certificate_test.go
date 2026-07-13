package license

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBugCheckInGrantRejectsCertificateAndLeaseSubstitutionHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *BugCheckInGrant)
		name   string
	}{
		{name: "missing lease rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.Lease = core.Signed[SeatLeaseBody]{} }},
		{name: "missing certificate rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.WriterCertificate = core.Signed[BugWriterCertificateBody]{} }},
		{name: "lease signature shape rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.Lease.Signature = core.Ed25519SignatureHex{} }},
		{name: "certificate signature shape rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.WriterCertificate.Signature = core.Ed25519SignatureHex{} }},
		{name: "lease signing key missing rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.Lease.KeyID = core.SigningKeyID{} }},
		{name: "certificate signing key missing rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) { g.WriterCertificate.KeyID = core.SigningKeyID{} }},
		{name: "device substitution rejected", mutate: func(t *testing.T, g *BugCheckInGrant) {
			g.WriterCertificate.Body.DeviceFingerprint = mustDeviceFingerprintValue(t, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		}},
		{name: "writer key substitution rejected", mutate: func(t *testing.T, g *BugCheckInGrant) { g.WriterCertificate.Body.Writer = differentBugWriterKey(t) }},
		{name: "certificate issued after lease rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) {
			g.WriterCertificate.Body.IssuedAt = g.Lease.Body.IssuedAt.Add(time.Nanosecond)
		}},
		{name: "certificate expires before write grace rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) {
			g.WriterCertificate.Body.ValidUntil = g.Lease.Body.WriteGraceUntil().Add(-time.Nanosecond)
		}},
		{name: "certificate schema substitution rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) {
			g.WriterCertificate.Body.Schema = core.SchemaBugWriterAttestation
		}},
		{name: "certificate inverted window rejected", mutate: func(_ *testing.T, g *BugCheckInGrant) {
			g.WriterCertificate.Body.ValidUntil = g.WriterCertificate.Body.IssuedAt
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grant, _, _ := signedBugGrant(t)
			tc.mutate(t, grant)
			if err := grant.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("BugCheckInGrant.Validate() error = %v, want %v", err, core.ErrLicenseContract)
			}
		})
	}
}

func TestBugCheckInGrantVerificationLayerTriad(t *testing.T) {
	t.Parallel()

	grant, keyID, publicKey := signedBugGrant(t)
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := grant.Verify(keyring); err != nil {
		t.Fatalf("BugCheckInGrant.Verify(valid) error = %v", err)
	}

	forged := *grant
	forged.WriterCertificate.Body.ValidUntil = forged.WriterCertificate.Body.ValidUntil.Add(time.Nanosecond)
	if err := forged.Verify(keyring); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugCheckInGrant.Verify(forged certificate) error = %v, want %v", err, core.ErrLicenseContract)
	}

	refused := CheckInResponse[BugCheckInGrant]{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment}
	if err := refused.Validate(); err != nil {
		t.Fatalf("CheckInResponse.Validate(refused neutral) error = %v", err)
	}
}

func TestBugWriterAttestationBodyRejectsHostileSubstitutionTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*BugWriterAttestationBody)
		name   string
	}{
		{name: "missing schema rejected", mutate: func(b *BugWriterAttestationBody) { b.Schema = core.SchemaUnknown }},
		{name: "certificate schema rejected", mutate: func(b *BugWriterAttestationBody) { b.Schema = core.SchemaBugWriterCertificate }},
		{name: "missing device rejected", mutate: func(b *BugWriterAttestationBody) { b.DeviceFingerprint = core.DeviceFingerprint{} }},
		{name: "missing writer rejected", mutate: func(b *BugWriterAttestationBody) { b.WriterKeyID = core.SigningKeyID{} }},
		{name: "missing operation digest rejected", mutate: func(b *BugWriterAttestationBody) { b.OperationDigest = core.SHA256Hex{} }},
		{name: "missing occurrence time rejected", mutate: func(b *BugWriterAttestationBody) { b.OccurredAt = core.UnixNanoTime{} }},
		{name: "zero body rejected", mutate: func(b *BugWriterAttestationBody) { *b = BugWriterAttestationBody{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := testWriterAttestationBody(t)
			tc.mutate(&body)
			if err := body.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("BugWriterAttestationBody.Validate() error = %v, want %v", err, core.ErrLicenseContract)
			}
		})
	}
}

func TestBugWriterAttestationBodyCanonicalLayerTriad(t *testing.T) {
	t.Parallel()

	body := testWriterAttestationBody(t)
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatalf("BugWriterAttestationBody.Canonical() error = %v", err)
	}
	marshaled, err := body.MarshalJSON()
	if err != nil {
		t.Fatalf("BugWriterAttestationBody.MarshalJSON() error = %v", err)
	}
	if !bytes.Equal(canonical, marshaled) {
		t.Fatalf("BugWriterAttestationBody canonical bytes = %q, MarshalJSON bytes = %q", canonical, marshaled)
	}
	if body.SigningSchema() != core.SchemaBugWriterAttestation {
		t.Fatalf("BugWriterAttestationBody.SigningSchema() = %v, want %v", body.SigningSchema(), core.SchemaBugWriterAttestation)
	}

	invalid := body
	invalid.OperationDigest = core.SHA256Hex{}
	if _, err := invalid.Canonical(nil); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugWriterAttestationBody.Canonical(invalid) error = %v, want %v", err, core.ErrLicenseContract)
	}

	var absent BugWriterAttestationBody
	if err := absent.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugWriterAttestationBody.Validate(absent) error = %v, want %v", err, core.ErrLicenseContract)
	}
}

func testWriterAttestationBody(t *testing.T) BugWriterAttestationBody {
	t.Helper()
	grant, _, _ := signedBugGrant(t)
	digest, err := core.ParseSHA256Hex(strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return BugWriterAttestationBody{
		Schema:            core.SchemaBugWriterAttestation,
		DeviceFingerprint: grant.WriterCertificate.Body.DeviceFingerprint,
		WriterKeyID:       grant.WriterCertificate.Body.Writer.KeyID,
		OperationDigest:   digest,
		OccurredAt:        grant.WriterCertificate.Body.IssuedAt,
	}
}

func differentBugWriterKey(t *testing.T) BugWriterKey {
	t.Helper()
	seed := make([]byte, 32)
	seed[len(seed)-1] = 1
	return bugWriterKeyFromSeed(t, seed)
}

func bugWriterKeyFromSeed(t *testing.T, seed []byte) BugWriterKey {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey, err := core.NewEd25519PublicKeyHex(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewBugWriterKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return writer
}

func mustDeviceFingerprintValue(t *testing.T, value string) core.DeviceFingerprint {
	t.Helper()
	fingerprint, err := core.ParseDeviceFingerprint(value)
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}
