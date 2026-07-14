package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

type signedTestBody struct {
	Value  string   `json:"value"`
	Schema SchemaID `json:"-"`
}

type canonicalWithoutValidationBody struct {
	Valid bool
}

func (b canonicalWithoutValidationBody) Validate() error {
	if !b.Valid {
		return ErrFoundationContract
	}
	return nil
}

func (canonicalWithoutValidationBody) Canonical(dst []byte) ([]byte, error) {
	return append(dst, signedTestCanonicalJSON...), nil
}

func (canonicalWithoutValidationBody) SigningSchema() SchemaID {
	return SchemaReleaseCommandRun
}

const signedTestBodyJSONFieldValue = "value"

const (
	retiredSignedMessageDomainV1  = "foundation-signed-v1"
	retiredSignedMessageDomainV2  = "foundation-signed-v2"
	signedTestCanonicalJSON       = `{"value":"ok"}`
	signedTestKeyIDToken          = "server-key-1"
	signingDomainTruncationOffset = SchemaID(1 << 8)
)

func (b signedTestBody) Validate() error {
	if b.Value == "" {
		return ErrFoundationContract
	}
	return b.Schema.ResolveSigningDomain().Validate()
}

func (b signedTestBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	dst, err := AppendJSONField(dst, signedTestBodyJSONFieldValue, b.Value)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b signedTestBody) SigningSchema() SchemaID {
	return b.Schema
}

func TestSignedVerifyHostileTable(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	otherID, otherPublicKey, _ := signingTestKey(t, "server-key-2")
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature := signTestMessage(t, privateKey, message)
	for _, tc := range []struct {
		name    string
		signed  Signed[signedTestBody]
		keyring SigningKeyring
	}{
		{
			name: "tampered body rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: signature,
				Body:      signedTestBody{Value: "changed", Schema: SchemaReleaseCommandRun},
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "flipped signature rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: flipSignature(t, signature),
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "unknown key id rejected",
			signed: Signed[signedTestBody]{
				KeyID:     otherID,
				Signature: signature,
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
		{
			name: "wrong public key rejected",
			signed: Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: signature,
				Body:      body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: otherPublicKey}}},
		},
		{
			name: "empty signature rejected",
			signed: Signed[signedTestBody]{
				KeyID: keyID,
				Body:  body,
			},
			keyring: SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.signed.Verify(tc.keyring); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("Verify error = %v, want ErrFoundationContract", err)
			}
		})
	}
}

func TestSignedVerifyAcceptsValidSignature(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: signTestMessage(t, privateKey, message),
		Body:      body,
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); err != nil {
		t.Fatal(err)
	}
}

func TestSigningBoundariesOwnBodyValidation(t *testing.T) {
	t.Parallel()

	keyID, publicKey, privateKey := signingTestKey(t, signedTestKeyIDToken)
	invalid := canonicalWithoutValidationBody{}
	if _, err := AppendSignedMessage(nil, keyID, invalid); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("AppendSignedMessage(invalid body) error = %v, want %v", err, ErrFoundationContract)
	}
	message, err := appendSignedMessageValidated(nil, keyID, invalid)
	if err != nil {
		t.Fatalf("appendSignedMessageValidated(test fixture) error = %v", err)
	}
	signature := signTestMessage(t, privateKey, message)
	signed := Signed[canonicalWithoutValidationBody]{Body: invalid, KeyID: keyID, Signature: signature}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("Signed.Verify(invalid body) error = %v, want %v", err, ErrFoundationContract)
	}
}

func TestAppendSignedMessageWireLayout(t *testing.T) {
	t.Parallel()
	keyID, _, _ := signingTestKey(t, signedTestKeyIDToken)
	got, err := AppendSignedMessage(nil, keyID, signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun})
	if err != nil {
		t.Fatal(err)
	}
	separator := string([]byte{SignedMessageSep})
	want := SignedMessageDomain + separator + SigningDomainTokenReleaseCommandRun + separator + signedTestKeyIDToken + separator + signedTestCanonicalJSON
	if string(got) != want {
		t.Fatalf("AppendSignedMessage() = %q, want %q", got, want)
	}
}

func TestSchemaSigningDomainTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		schema SchemaID
		domain SigningDomain
	}{
		{name: "bug usage is not signable", schema: SchemaBugUsage, domain: SigningDomainUnknown},
		{name: "witness usage is not signable", schema: SchemaWitnessUsage, domain: SigningDomainUnknown},
		{name: "bug check-in is not signable", schema: SchemaBugCheckIn, domain: SigningDomainUnknown},
		{name: "bug seat lease owns its domain", schema: SchemaBugSeatLease, domain: SigningDomainBugSeatLease},
		{name: "witness check-in is not signable", schema: SchemaWitnessCheckIn, domain: SigningDomainUnknown},
		{name: "witness subscription owns its domain", schema: SchemaWitnessSubscription, domain: SigningDomainWitnessSubscriptionLease},
		{name: "custody session open request is not signable", schema: SchemaCustodySessionOpenRequest, domain: SigningDomainUnknown},
		{name: "custody session open response is not signable", schema: SchemaCustodySessionOpenResponse, domain: SigningDomainUnknown},
		{name: "custody finalize request is not signable", schema: SchemaCustodyFinalizeRequest, domain: SigningDomainUnknown},
		{name: "custody receipt owns its domain", schema: SchemaCustodyReceipt, domain: SigningDomainWitnessCustodyReceipt},
		{name: "release manifest owns its domain", schema: SchemaReleaseManifest, domain: SigningDomainReleaseManifest},
		{name: "release upload receipt owns its domain", schema: SchemaReleaseUploadReceipt, domain: SigningDomainReleaseUploadReceipt},
		{name: "release download index owns its domain", schema: SchemaReleaseDownloadIndex, domain: SigningDomainReleaseDownloadIndex},
		{name: "release plan owns its domain", schema: SchemaReleasePlan, domain: SigningDomainReleasePlan},
		{name: "release root layout owns its domain", schema: SchemaReleaseRootLayout, domain: SigningDomainReleaseRootLayout},
		{name: "release command run owns its domain", schema: SchemaReleaseCommandRun, domain: SigningDomainReleaseCommandRun},
		{name: "zero schema is not signable", schema: SchemaUnknown, domain: SigningDomainUnknown},
		{name: "future schema cannot truncate to signable ordinal", schema: signingDomainTruncationOffset + SchemaBugSeatLease, domain: SigningDomainUnknown},
		{name: "future schema is not signable", schema: SchemaID(^uint16(0)), domain: SigningDomainUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.schema.ResolveSigningDomain(); got != tc.domain {
				t.Fatalf("SchemaID.ResolveSigningDomain() = %v, want %v", got, tc.domain)
			}
		})
	}
}

func TestSigningDomainJSONHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		token     string
		name      string
		domain    SigningDomain
		wantError bool
	}{
		{name: "bug seat lease domain round trips", domain: SigningDomainBugSeatLease, token: SigningDomainTokenBugSeatLease},
		{name: "witness subscription domain round trips", domain: SigningDomainWitnessSubscriptionLease, token: SigningDomainTokenWitnessSubscriptionLease},
		{name: "custody receipt domain round trips", domain: SigningDomainWitnessCustodyReceipt, token: SigningDomainTokenWitnessCustodyReceipt},
		{name: "release manifest domain round trips", domain: SigningDomainReleaseManifest, token: SigningDomainTokenReleaseManifest},
		{name: "release upload receipt domain round trips", domain: SigningDomainReleaseUploadReceipt, token: SigningDomainTokenReleaseUploadReceipt},
		{name: "release download index domain round trips", domain: SigningDomainReleaseDownloadIndex, token: SigningDomainTokenReleaseDownloadIndex},
		{name: "release plan domain round trips", domain: SigningDomainReleasePlan, token: SigningDomainTokenReleasePlan},
		{name: "release root layout domain round trips", domain: SigningDomainReleaseRootLayout, token: SigningDomainTokenReleaseRootLayout},
		{name: "release command run domain round trips", domain: SigningDomainReleaseCommandRun, token: SigningDomainTokenReleaseCommandRun},
		{name: "zero domain marshal rejected", domain: SigningDomainUnknown, wantError: true},
		{name: "future domain marshal rejected", domain: SigningDomain(^uint16(0)), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := tc.domain.MarshalJSON()
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("SigningDomain.MarshalJSON() error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want, err := json.Marshal(tc.token)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != string(want) {
				t.Fatalf("SigningDomain.MarshalJSON() = %s, want %s", encoded, want)
			}
			var decoded SigningDomain
			if err := decoded.UnmarshalJSON(encoded); err != nil {
				t.Fatal(err)
			}
			if decoded != tc.domain {
				t.Fatalf("SigningDomain.UnmarshalJSON() = %v, want %v", decoded, tc.domain)
			}
		})
	}
}

func TestParseSigningDomainHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		token     string
		want      SigningDomain
		wantError bool
	}{
		{name: "bug seat lease token accepted", token: SigningDomainTokenBugSeatLease, want: SigningDomainBugSeatLease},
		{name: "witness subscription token accepted", token: SigningDomainTokenWitnessSubscriptionLease, want: SigningDomainWitnessSubscriptionLease},
		{name: "custody receipt token accepted", token: SigningDomainTokenWitnessCustodyReceipt, want: SigningDomainWitnessCustodyReceipt},
		{name: "release manifest token accepted", token: SigningDomainTokenReleaseManifest, want: SigningDomainReleaseManifest},
		{name: "release upload receipt token accepted", token: SigningDomainTokenReleaseUploadReceipt, want: SigningDomainReleaseUploadReceipt},
		{name: "release download index token accepted", token: SigningDomainTokenReleaseDownloadIndex, want: SigningDomainReleaseDownloadIndex},
		{name: "release plan token accepted", token: SigningDomainTokenReleasePlan, want: SigningDomainReleasePlan},
		{name: "release root layout token accepted", token: SigningDomainTokenReleaseRootLayout, want: SigningDomainReleaseRootLayout},
		{name: "release command run token accepted", token: SigningDomainTokenReleaseCommandRun, want: SigningDomainReleaseCommandRun},
		{name: "empty token rejected", wantError: true},
		{name: "unknown token rejected", token: "unknown", wantError: true},
		{name: "uppercase token rejected", token: "RELEASE-MANIFEST-2026", wantError: true},
		{name: "legacy numeric suffix rejected", token: "release-manifest-v1", wantError: true},
		{name: "leading whitespace rejected", token: " release-manifest-2026", wantError: true},
		{name: "trailing whitespace rejected", token: "release-manifest-2026 ", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSigningDomain(tc.token)
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseSigningDomain() error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("ParseSigningDomain() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSigningDomainUnmarshalHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "empty JSON rejected", raw: ""},
		{name: "null rejected", raw: `null`},
		{name: "number rejected", raw: `1`},
		{name: "boolean rejected", raw: `true`},
		{name: "object rejected", raw: `{}`},
		{name: "array rejected", raw: `[]`},
		{name: "unknown string rejected", raw: `"unknown"`},
		{name: "truncated string rejected", raw: `"release-manifest-2026`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SigningDomainReleaseManifest
			if err := got.UnmarshalJSON([]byte(tc.raw)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("SigningDomain.UnmarshalJSON() error = %v, want ErrFoundationContract", err)
			}
			if got != SigningDomainReleaseManifest {
				t.Fatalf("SigningDomain.UnmarshalJSON() mutated receiver to %v", got)
			}
		})
	}
}

func TestSignedVerifyRejectsSignatureBoundToDifferentKeyID(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	otherID, _, _ := signingTestKey(t, "server-key-2")
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	message, err := AppendSignedMessage(nil, otherID, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: signTestMessage(t, privateKey, message),
		Body:      body,
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("Verify wrong-key-id-bound signature error = %v, want ErrFoundationContract", err)
	}
}

func TestSignedVerifyRejectsSignatureBoundToDifferentDomain(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, "server-key-1")
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signed := Signed[signedTestBody]{
		KeyID:     keyID,
		Signature: signTestMessage(t, privateKey, message),
		Body:      signedTestBody{Value: body.Value, Schema: SchemaReleasePlan},
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("Verify wrong-domain-bound signature error = %v, want ErrFoundationContract", err)
	}
}

func TestSignedVerifyRejectsRetiredFramingTable(t *testing.T) {
	t.Parallel()
	keyID, publicKey, privateKey := signingTestKey(t, signedTestKeyIDToken)
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	separator := string([]byte{SignedMessageSep})
	for _, tc := range []struct {
		name    string
		message string
	}{
		{
			name:    "retired v1 domainless frame rejected",
			message: retiredSignedMessageDomainV1 + separator + signedTestKeyIDToken + separator + signedTestCanonicalJSON,
		},
		{
			name:    "retired v2 typed frame rejected",
			message: retiredSignedMessageDomainV2 + separator + SigningDomainTokenReleaseCommandRun + separator + signedTestKeyIDToken + separator + signedTestCanonicalJSON,
		},
		{
			name:    "current year frame missing signing domain rejected",
			message: SignedMessageDomain + separator + signedTestKeyIDToken + separator + signedTestCanonicalJSON,
		},
		{
			name:    "current year frame with key and domain reordered rejected",
			message: SignedMessageDomain + separator + signedTestKeyIDToken + separator + SigningDomainTokenReleaseCommandRun + separator + signedTestCanonicalJSON,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			signed := Signed[signedTestBody]{
				KeyID:     keyID,
				Signature: signTestMessage(t, privateKey, []byte(tc.message)),
				Body:      body,
			}
			keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
			if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("Verify retired framing error = %v, want ErrFoundationContract", err)
			}
		})
	}
}

func TestSigningKeyringRejectsDuplicateKeyID(t *testing.T) {
	t.Parallel()
	keyID, publicKey, _ := signingTestKey(t, "server-key-1")
	ring := SigningKeyring{Keys: []SigningPublicKey{
		{ID: keyID, PublicKey: publicKey},
		{ID: keyID, PublicKey: publicKey},
	}}
	if err := ring.Validate(); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("SigningKeyring duplicate Validate error = %v, want ErrFoundationContract", err)
	}
}

func TestSigningKeyringRejectsUnboundedKeySet(t *testing.T) {
	t.Parallel()

	keys := make([]SigningPublicKey, SigningKeyringMaxKeys+1)
	for i := range keys {
		keyID, publicKey, _ := signingTestKey(t, fmt.Sprintf("server-key-%d", i))
		keys[i] = SigningPublicKey{ID: keyID, PublicKey: publicKey}
	}
	if err := (SigningKeyring{Keys: keys}).Validate(); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("SigningKeyring(over cap).Validate() error = %v, want ErrFoundationContract", err)
	}
}

func signingTestKey(t *testing.T, id string) (SigningKeyID, Ed25519PublicKeyHex, ed25519.PrivateKey) {
	t.Helper()
	seed := sha256.Sum256([]byte(id))
	private := ed25519.NewKeyFromSeed(seed[:])
	public := ed25519.PublicKey(private[ed25519.SeedSize:])
	keyID, err := ParseSigningKeyID(id)
	if err != nil {
		t.Fatal(err)
	}
	publicHex, err := NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	return keyID, publicHex, private
}

func signTestMessage(t *testing.T, privateKey ed25519.PrivateKey, message []byte) Ed25519SignatureHex {
	t.Helper()
	signature, err := NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return signature
}

func flipSignature(t *testing.T, signature Ed25519SignatureHex) Ed25519SignatureHex {
	t.Helper()
	raw, err := signature.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte(nil), raw...)
	out[0] ^= 1
	flipped, err := NewEd25519SignatureHex(out)
	if err != nil {
		t.Fatal(err)
	}
	return flipped
}
