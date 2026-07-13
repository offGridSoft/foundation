package license

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBugCheckInResponseBoundaryTable(t *testing.T) {
	t.Parallel()

	grant, keyID, publicKey := signedBugGrant(t)
	_, foreignPublic, _ := testServerSigningKeyWithByte(t, 43)
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	one := signedRevocationDelivery(t, keyID, 1)
	two := signedRevocationDelivery(t, keyID, 2)
	maximum := makeRevocationDelivery(t, keyID, BugWriterRevocationDeliveryMax)
	overMaximum := makeRevocationDelivery(t, keyID, BugWriterRevocationDeliveryMax+1)

	granted := func(values ...core.Signed[BugWriterRevocationBody]) BugCheckInResponse {
		return BugCheckInResponse{
			Decision:          CheckInDecision{Granted: true, Refusal: RefusalNone},
			Grant:             grant,
			WriterRevocations: BugWriterRevocationSet{Values: values},
		}
	}
	refused := func(refusal Refusal, remediation Remediation, values ...core.Signed[BugWriterRevocationBody]) BugCheckInResponse {
		return BugCheckInResponse{
			Decision:          CheckInDecision{Refusal: refusal, Remediation: remediation},
			WriterRevocations: BugWriterRevocationSet{Values: values},
		}
	}

	cases := []struct {
		name     string
		keyring  core.SigningKeyring
		response BugCheckInResponse
		wantErr  bool
	}{
		{name: "grant without revocations", response: granted(), keyring: keyring},
		{name: "grant with one revocation", response: granted(one), keyring: keyring},
		{name: "grant with sorted revocations", response: granted(one, two), keyring: keyring},
		{name: "payment refusal without revocations", response: refused(RefusalPaymentRequired, RemediationUpdatePayment), keyring: keyring},
		{name: "payment refusal carries revocation", response: refused(RefusalPaymentRequired, RemediationUpdatePayment, one), keyring: keyring},
		{name: "revoked refusal carries revocation", response: refused(RefusalKeyRevoked, RemediationContactSupport, one), keyring: keyring},
		{name: "seat refusal carries revocation", response: refused(RefusalSeatLimit, RemediationReduceSeats, one), keyring: keyring},
		{name: "unsupported build refusal carries two", response: refused(RefusalUnsupportedBuild, RemediationInstallSupportedBuild, one, two), keyring: keyring},
		{name: "grant accepts exact revocation maximum", response: granted(maximum...), keyring: keyring},
		{name: "refusal accepts exact revocation maximum", response: refused(RefusalDeviceLimit, RemediationDeactivateMachine, maximum...), keyring: keyring},
		{name: "zero response rejected", response: BugCheckInResponse{}, keyring: keyring, wantErr: true},
		{name: "grant decision missing grant rejected", response: BugCheckInResponse{Decision: CheckInDecision{Granted: true, Refusal: RefusalNone}}, keyring: keyring, wantErr: true},
		{name: "refusal carrying grant rejected", response: BugCheckInResponse{Decision: CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment}, Grant: grant}, keyring: keyring, wantErr: true},
		{name: "refusal remediation mismatch rejected", response: refused(RefusalPaymentRequired, RemediationDeactivateMachine), keyring: keyring, wantErr: true},
		{name: "duplicate writer cutoff rejected", response: granted(one, one), keyring: keyring, wantErr: true},
		{name: "descending writer order rejected", response: granted(two, one), keyring: keyring, wantErr: true},
		{name: "over revocation maximum rejected", response: granted(overMaximum...), keyring: keyring, wantErr: true},
		{name: "empty signed revocation rejected", response: granted(core.Signed[BugWriterRevocationBody]{}), keyring: keyring, wantErr: true},
		{name: "wrong revocation schema rejected", response: granted(mutateRevocation(one, func(body *BugWriterRevocationBody) { body.Schema = core.SchemaBugWriterCertificate })), keyring: keyring, wantErr: true},
		{name: "zero revocation cutoff rejected", response: granted(mutateRevocation(one, func(body *BugWriterRevocationBody) { body.RevokedAt = core.UnixNanoTime{} })), keyring: keyring, wantErr: true},
		{name: "tampered signed cutoff rejected", response: granted(mutateRevocation(one, func(body *BugWriterRevocationBody) { body.RevokedAt = body.RevokedAt.Add(time.Nanosecond) })), keyring: keyring, wantErr: true},
		{name: "foreign verifier rejects response", response: granted(one), keyring: core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: foreignPublic}}}, wantErr: true},
		{name: "missing verifier rejects response", response: granted(one), keyring: core.SigningKeyring{}, wantErr: true},
		{name: "unknown refusal rejected", response: refused(Refusal(RefusalStorageVerificationFailure+1), RemediationRetryUpload), keyring: keyring, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.response.Verify(tc.keyring)
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("BugCheckInResponse.Verify() error = %v, want %v", err, core.ErrLicenseContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("BugCheckInResponse.Verify() error = %v", err)
			}
		})
	}
}

func TestBugWriterRevocationSetMergeBoundaryTable(t *testing.T) {
	t.Parallel()

	keyID, _, privateKey := testServerSigningKey(t)
	one := signedRevocationDelivery(t, keyID, 1)
	two := signedRevocationDelivery(t, keyID, 2)
	conflict := one
	conflict.Body.RevokedAt = conflict.Body.RevokedAt.Add(time.Nanosecond)
	conflict = signTestBody(t, keyID, privateKey, conflict.Body)
	maximum := BugWriterRevocationSet{Values: makeRevocationDelivery(t, keyID, BugWriterRevocationDeliveryMax)}
	cases := []struct {
		name      string
		left      BugWriterRevocationSet
		right     BugWriterRevocationSet
		wantCount int
		wantErr   bool
	}{
		{name: "two empty sets remain empty", wantCount: 0},
		{name: "empty left accepts one", right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "empty right preserves one", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "exact redelivery remains one", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "new greater writer appends", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two}}, wantCount: 2},
		{name: "new lesser writer prepends", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two}}, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 2},
		{name: "overlapping sorted sets preserve union", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one, two}}, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two}}, wantCount: 2},
		{name: "conflicting signed cutoff rejected", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{conflict}}, wantErr: true},
		{name: "descending left rejected", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two, one}}, wantErr: true},
		{name: "duplicate right rejected", right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one, one}}, wantErr: true},
		{name: "merged result above delivery cap rejected", left: maximum, right: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{signedRevocationDelivery(t, keyID, BugWriterRevocationDeliveryMax+1)}}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.left.Merge(tc.right)
			if tc.wantErr {
				if !errors.Is(err, core.ErrLicenseContract) {
					t.Fatalf("BugWriterRevocationSet.Merge() error = %v, want %v", err, core.ErrLicenseContract)
				}
				return
			}
			if err != nil || len(got.Values) != tc.wantCount {
				t.Fatalf("BugWriterRevocationSet.Merge() count = %d, error = %v, want count = %d", len(got.Values), err, tc.wantCount)
			}
		})
	}
}

func makeRevocationDelivery(t *testing.T, keyID core.SigningKeyID, count int) []core.Signed[BugWriterRevocationBody] {
	t.Helper()
	values := make([]core.Signed[BugWriterRevocationBody], 0, count)
	for index := 1; index <= count; index++ {
		values = append(values, signedRevocationDelivery(t, keyID, index))
	}
	return values
}

func signedRevocationDelivery(t *testing.T, keyID core.SigningKeyID, ordinal int) core.Signed[BugWriterRevocationBody] {
	t.Helper()
	_, _, privateKey := testServerSigningKey(t)
	writer, err := core.ParseSigningKeyID(BugWriterKeyIDPrefix + fmt.Sprintf("%064x", ordinal))
	if err != nil {
		t.Fatalf("ParseSigningKeyID() error = %v", err)
	}
	body := BugWriterRevocationBody{
		Schema:      core.SchemaBugWriterRevocation,
		WriterKeyID: writer,
		RevokedAt:   core.NewUnixNanoTime(time.Date(2026, 7, 13, 12, 0, 0, ordinal, time.UTC)),
	}
	return signTestBody(t, keyID, privateKey, body)
}

func testServerSigningKeyWithByte(t *testing.T, value byte) (core.SigningKeyID, core.Ed25519PublicKeyHex, ed25519.PrivateKey) {
	t.Helper()
	keyID, err := core.ParseSigningKeyID("server-key-1")
	if err != nil {
		t.Fatalf("ParseSigningKeyID() error = %v", err)
	}
	seed := make([]byte, ed25519.SeedSize)
	seed[len(seed)-1] = value
	key := ed25519.NewKeyFromSeed(seed)
	parsed, err := core.NewEd25519PublicKeyHex(key.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatalf("NewEd25519PublicKeyHex() error = %v", err)
	}
	return keyID, parsed, key
}

func mutateRevocation(value core.Signed[BugWriterRevocationBody], mutate func(*BugWriterRevocationBody)) core.Signed[BugWriterRevocationBody] {
	mutate(&value.Body)
	return value
}
