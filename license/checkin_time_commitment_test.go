package license

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

// goldenTimeCommitment builds the deterministic fixture whose wire bytes were
// captured from the pre-generic named types (BugCheckInTimeCommitmentBody /
// WitnessCheckInTimeCommitmentBody). The literals asserted below are the exact
// pre-refactor output; any drift is a signed wire-contract break.
func goldenTimeCommitmentFields(t *testing.T) (core.DeviceFingerprint, core.LeaseID, CheckInNonce, core.UnixNanoTime) {
	t.Helper()
	device, err := core.ParseDeviceFingerprint(core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	leaseID, err := core.ParseLeaseID("lease-2026-golden")
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := ParseCheckInNonce("0101010101010101010101010101010101010101010101010101010101010101")
	if err != nil {
		t.Fatal(err)
	}
	return device, leaseID, nonce, core.UnixNanoTimeFromInt64(1767225600123456789)
}

const (
	goldenBugTimeCommitmentJSON = `{"schema":"bug-license-check-in-time-commitment-v2026",` +
		`"device_fingerprint":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",` +
		`"lease_id":"lease-2026-golden","lease_generation":7,` +
		`"request_nonce":"0101010101010101010101010101010101010101010101010101010101010101",` +
		`"server_observed_at":1767225600123456789}`
	goldenWitnessTimeCommitmentJSON = `{"schema":"witness-subscription-check-in-time-commitment-v2026",` +
		`"device_fingerprint":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",` +
		`"lease_id":"lease-2026-golden","lease_generation":7,` +
		`"request_nonce":"0101010101010101010101010101010101010101010101010101010101010101",` +
		`"server_observed_at":1767225600123456789}`
	goldenBugTimeCommitmentSignedMessage = "foundation-signed-2026\x00" +
		"bug-check-in-time-commitment-2026\x00server-key-golden\x00" + goldenBugTimeCommitmentJSON
	goldenWitnessTimeCommitmentSignedMessage = "foundation-signed-2026\x00" +
		"witness-check-in-time-commitment-2026\x00server-key-golden\x00" + goldenWitnessTimeCommitmentJSON
)

// TestCheckInTimeCommitmentGoldenWireBytes proves invariant 1 of the generic
// unification: the canonical JSON and the domain-separated signing preimage
// emitted by CheckInTimeCommitmentBody[G] are byte-for-byte identical to what
// the pre-refactor named Bug/Witness types emitted (literals captured before
// the refactor), and a decode of those exact bytes re-encodes stably.
func TestCheckInTimeCommitmentGoldenWireBytes(t *testing.T) {
	t.Parallel()

	device, leaseID, nonce, observed := goldenTimeCommitmentFields(t)
	keyID, err := core.ParseSigningKeyID("server-key-golden")
	if err != nil {
		t.Fatal(err)
	}

	bug := CheckInTimeCommitmentBody[BugCheckInGrant]{
		DeviceFingerprint: device,
		LeaseID:           leaseID,
		RequestNonce:      nonce,
		ServerObservedAt:  observed,
		LeaseGeneration:   7,
		Schema:            core.SchemaBugCheckInTimeCommitment,
	}
	witness := CheckInTimeCommitmentBody[WitnessCheckInGrant]{
		DeviceFingerprint: device,
		LeaseID:           leaseID,
		RequestNonce:      nonce,
		ServerObservedAt:  observed,
		LeaseGeneration:   7,
		Schema:            core.SchemaWitnessCheckInTimeCommitment,
	}

	checkGoldenBytes := func(name string, got []byte, err error, want string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		if string(got) != want {
			t.Fatalf("%s wire drift:\n got  %s\n want %s", name, got, want)
		}
	}

	canonical, err := bug.Canonical(nil)
	checkGoldenBytes("bug Canonical", canonical, err, goldenBugTimeCommitmentJSON)
	marshaled, err := json.Marshal(bug)
	checkGoldenBytes("bug json.Marshal", marshaled, err, goldenBugTimeCommitmentJSON)
	message, err := core.AppendSignedMessage(nil, keyID, bug)
	checkGoldenBytes("bug signing preimage", message, err, goldenBugTimeCommitmentSignedMessage)

	canonical, err = witness.Canonical(nil)
	checkGoldenBytes("witness Canonical", canonical, err, goldenWitnessTimeCommitmentJSON)
	marshaled, err = json.Marshal(witness)
	checkGoldenBytes("witness json.Marshal", marshaled, err, goldenWitnessTimeCommitmentJSON)
	message, err = core.AppendSignedMessage(nil, keyID, witness)
	checkGoldenBytes("witness signing preimage", message, err, goldenWitnessTimeCommitmentSignedMessage)

	var bugDecoded CheckInTimeCommitmentBody[BugCheckInGrant]
	if err := json.Unmarshal([]byte(goldenBugTimeCommitmentJSON), &bugDecoded); err != nil {
		t.Fatalf("bug decode error = %v", err)
	}
	if bugDecoded != bug {
		t.Fatalf("bug decode drift: got %+v, want %+v", bugDecoded, bug)
	}
	var witnessDecoded CheckInTimeCommitmentBody[WitnessCheckInGrant]
	if err := json.Unmarshal([]byte(goldenWitnessTimeCommitmentJSON), &witnessDecoded); err != nil {
		t.Fatalf("witness decode error = %v", err)
	}
	if witnessDecoded != witness {
		t.Fatalf("witness decode drift: got %+v, want %+v", witnessDecoded, witness)
	}
}

// TestCheckInTimeCommitmentDomainSeparation proves invariant 2: a commitment
// signed for one product can never verify or validate as the other product's
// commitment. Every cross-product path — retyping the signed value, decoding
// the wire bytes under the wrong product, and flipping the schema while
// keeping the foreign signature — must fail with the typed contract error.
func TestCheckInTimeCommitmentDomainSeparation(t *testing.T) {
	t.Parallel()

	device, leaseID, nonce, observed := goldenTimeCommitmentFields(t)
	keyID, publicKey, privateKey := testServerSigningKey(t)
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}

	bugSigned := signTestBody(t, keyID, privateKey, CheckInTimeCommitmentBody[BugCheckInGrant]{
		DeviceFingerprint: device,
		LeaseID:           leaseID,
		RequestNonce:      nonce,
		ServerObservedAt:  observed,
		LeaseGeneration:   7,
		Schema:            core.SchemaBugCheckInTimeCommitment,
	})
	witnessSigned := signTestBody(t, keyID, privateKey, CheckInTimeCommitmentBody[WitnessCheckInGrant]{
		DeviceFingerprint: device,
		LeaseID:           leaseID,
		RequestNonce:      nonce,
		ServerObservedAt:  observed,
		LeaseGeneration:   7,
		Schema:            core.SchemaWitnessCheckInTimeCommitment,
	})
	if err := bugSigned.Verify(keyring); err != nil {
		t.Fatalf("bug commitment Verify() error = %v, want nil", err)
	}
	if err := witnessSigned.Verify(keyring); err != nil {
		t.Fatalf("witness commitment Verify() error = %v, want nil", err)
	}

	requireContractFailure := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: cross-product commitment accepted, want typed contract failure", name)
		}
		if !errors.Is(err, core.ErrFoundationContract) {
			t.Fatalf("%s error = %v, want %v", name, err, core.ErrFoundationContract)
		}
	}

	// Attack 1: retype the signed bug value as a witness commitment (exact
	// same body bytes, schema still bug). The witness schema pin must refuse
	// before any signature question is asked, and vice versa.
	bugAsWitness := core.Signed[CheckInTimeCommitmentBody[WitnessCheckInGrant]]{
		Body:      CheckInTimeCommitmentBody[WitnessCheckInGrant](bugSigned.Body),
		KeyID:     bugSigned.KeyID,
		Signature: bugSigned.Signature,
	}
	requireContractFailure("retyped bug->witness Validate", bugAsWitness.Body.Validate())
	requireContractFailure("retyped bug->witness Verify", bugAsWitness.Verify(keyring))
	if !errors.Is(bugAsWitness.Body.Validate(), core.ErrLicenseContract) {
		t.Fatalf("retyped bug->witness Validate must fail the license contract, got %v", bugAsWitness.Body.Validate())
	}
	witnessAsBug := core.Signed[CheckInTimeCommitmentBody[BugCheckInGrant]]{
		Body:      CheckInTimeCommitmentBody[BugCheckInGrant](witnessSigned.Body),
		KeyID:     witnessSigned.KeyID,
		Signature: witnessSigned.Signature,
	}
	requireContractFailure("retyped witness->bug Validate", witnessAsBug.Body.Validate())
	requireContractFailure("retyped witness->bug Verify", witnessAsBug.Verify(keyring))

	// Attack 2: decode the signed bug wire bytes as a witness commitment.
	bugWire, err := json.Marshal(bugSigned)
	if err != nil {
		t.Fatal(err)
	}
	var decodedAsWitness core.Signed[CheckInTimeCommitmentBody[WitnessCheckInGrant]]
	if err := json.Unmarshal(bugWire, &decodedAsWitness); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	requireContractFailure("decoded bug wire as witness Verify", decodedAsWitness.Verify(keyring))

	// Attack 3: flip the schema to the witness value while keeping the bug
	// signature. Validation passes the schema pin, so the domain-separated
	// signature itself must refuse: the preimage was signed under
	// bug-check-in-time-commitment, not witness-check-in-time-commitment.
	flipped := core.Signed[CheckInTimeCommitmentBody[WitnessCheckInGrant]]{
		Body:      CheckInTimeCommitmentBody[WitnessCheckInGrant](bugSigned.Body),
		KeyID:     bugSigned.KeyID,
		Signature: bugSigned.Signature,
	}
	flipped.Body.Schema = core.SchemaWitnessCheckInTimeCommitment
	if err := flipped.Body.Validate(); err != nil {
		t.Fatalf("schema-flipped body must pass validation to reach the signature gate, got %v", err)
	}
	requireContractFailure("schema-flipped bug signature as witness Verify", flipped.Verify(keyring))
	flippedBack := core.Signed[CheckInTimeCommitmentBody[BugCheckInGrant]]{
		Body:      CheckInTimeCommitmentBody[BugCheckInGrant](witnessSigned.Body),
		KeyID:     witnessSigned.KeyID,
		Signature: witnessSigned.Signature,
	}
	flippedBack.Body.Schema = core.SchemaBugCheckInTimeCommitment
	requireContractFailure("schema-flipped witness signature as bug Verify", flippedBack.Verify(keyring))
}
