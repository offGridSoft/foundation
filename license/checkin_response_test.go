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
	maximum := makeRevocationDelivery(t, keyID, int(BugWriterRevocationDeliveryMaximum))
	overMaximum := makeRevocationDelivery(t, keyID, int(BugWriterRevocationDeliveryMaximum)+1)

	granted := func(values ...core.Signed[BugWriterRevocationBody]) BugCheckInResponse {
		body := BugCheckInResponseBody{
			Schema:            core.SchemaBugCheckInResponse,
			RequestNonce:      testCheckInNonce(t),
			Decision:          CheckInDecision{Granted: true, Refusal: RefusalNone},
			Grant:             grant,
			WriterRevocations: BugWriterRevocationDelivery{Values: values},
		}
		if body.Validate() == nil {
			return signBugCheckInResponse(t, body)
		}
		response := signBugCheckInResponse(t, BugCheckInResponseBody{
			Schema:       core.SchemaBugCheckInResponse,
			RequestNonce: testCheckInNonce(t),
			Decision:     CheckInDecision{Granted: true, Refusal: RefusalNone},
			Grant:        grant,
		})
		response.Authority.Body = body
		return response
	}
	refused := func(refusal Refusal, remediation Remediation, values ...core.Signed[BugWriterRevocationBody]) BugCheckInResponse {
		valid := signBugCheckInResponse(t, BugCheckInResponseBody{
			Schema:       core.SchemaBugCheckInResponse,
			RequestNonce: testCheckInNonce(t),
			Decision:     CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment},
		})
		valid.Authority.Body.Decision = CheckInDecision{Refusal: refusal, Remediation: remediation}
		valid.Authority.Body.WriterRevocations = BugWriterRevocationDelivery{Values: values}
		if valid.Authority.Body.Validate() == nil {
			keyID, _, privateKey := testServerSigningKey(t)
			valid.Authority = signTestBody(t, keyID, privateKey, valid.Authority.Body)
		}
		return valid
	}
	grantMissing := granted()
	grantMissing.Authority.Body.Grant = nil
	refusalWithGrant := refused(RefusalPaymentRequired, RemediationUpdatePayment)
	refusalWithGrant.Authority.Body.Grant = grant
	zeroResponse := BugCheckInResponse{}

	cases := []struct {
		response BugCheckInResponse
		name     string
		keyring  core.SigningKeyring
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
		{name: "zero response rejected", response: zeroResponse, keyring: keyring, wantErr: true},
		{name: "grant decision missing grant rejected", response: grantMissing, keyring: keyring, wantErr: true},
		{name: "refusal carrying grant rejected", response: refusalWithGrant, keyring: keyring, wantErr: true},
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

func TestUnsignedRefusalCannotCrossVerificationBoundary(t *testing.T) {
	t.Parallel()

	keyID, publicKey, _ := testServerSigningKey(t)
	response := BugCheckInResponse{Authority: core.Signed[BugCheckInResponseBody]{
		Body: BugCheckInResponseBody{
			Schema:       core.SchemaBugCheckInResponse,
			RequestNonce: testCheckInNonce(t),
			Decision:     CheckInDecision{Refusal: RefusalPaymentRequired, Remediation: RemediationUpdatePayment},
		},
		KeyID: keyID,
	}}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := response.Verify(keyring); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugCheckInResponse.Verify(unsigned refusal) error = %v, want %v", err, core.ErrLicenseContract)
	}
}

func TestRetainedWriterRevocationsVerifyEveryPresentAuthoritySignature(t *testing.T) {
	t.Parallel()

	keyID, publicKey, _ := testServerSigningKey(t)
	revocation := signedRevocationDelivery(t, keyID, 1)
	set := BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{revocation}}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := set.Verify(keyring); err != nil {
		t.Fatalf("BugWriterRevocationSet.Verify() error = %v", err)
	}
	tampered := set
	tampered.Values = append([]core.Signed[BugWriterRevocationBody](nil), set.Values...)
	tampered.Values[0].Body.RevokedAt = tampered.Values[0].Body.RevokedAt.Add(-time.Nanosecond)
	if err := tampered.Verify(keyring); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugWriterRevocationSet.Verify(tampered) error = %v, want %v", err, core.ErrLicenseContract)
	}
}

func TestBugWriterRevocationSetMergeBoundaryTable(t *testing.T) {
	t.Parallel()

	keyID, _, privateKey := testServerSigningKey(t)
	one := signedRevocationDelivery(t, keyID, 1)
	two := signedRevocationDelivery(t, keyID, 2)
	later := one
	later.Body.RevokedAt = later.Body.RevokedAt.Add(time.Nanosecond)
	later = signTestBody(t, keyID, privateKey, later.Body)
	earlier := one
	earlier.Body.RevokedAt = earlier.Body.RevokedAt.Add(-time.Nanosecond)
	earlier = signTestBody(t, keyID, privateKey, earlier.Body)
	maximum := BugWriterRevocationSet{Values: makeRevocationDelivery(t, keyID, int(BugWriterRevocationDeliveryMaximum))}
	cases := []struct {
		name       string
		left       BugWriterRevocationSet
		right      BugWriterRevocationDelivery
		wantCount  int
		wantCutoff core.UnixNanoTime
		wantErr    bool
	}{
		{name: "two empty sets remain empty", wantCount: 0},
		{name: "empty left accepts one", right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "empty right preserves one", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "exact redelivery remains one", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 1},
		{name: "new greater writer appends", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{two}}, wantCount: 2},
		{name: "new lesser writer prepends", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{one}}, wantCount: 2},
		{name: "overlapping sorted sets preserve union", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one, two}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{two}}, wantCount: 2},
		{name: "earlier signed cutoff broadens existing revocation", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{earlier}}, wantCount: 1, wantCutoff: earlier.Body.RevokedAt},
		{name: "later signed cutoff cannot narrow existing revocation", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{one}}, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{later}}, wantCount: 1, wantCutoff: one.Body.RevokedAt},
		{name: "descending left rejected", left: BugWriterRevocationSet{Values: []core.Signed[BugWriterRevocationBody]{two, one}}, wantErr: true},
		{name: "duplicate right rejected", right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{one, one}}, wantErr: true},
		{name: "retained set grows beyond one delivery", left: maximum, right: BugWriterRevocationDelivery{Values: []core.Signed[BugWriterRevocationBody]{signedRevocationDelivery(t, keyID, int(BugWriterRevocationDeliveryMaximum)+1)}}, wantCount: int(BugWriterRevocationDeliveryMaximum) + 1},
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
			if !tc.wantCutoff.IsZero() && got.Values[0].Body.RevokedAt != tc.wantCutoff {
				t.Fatalf("BugWriterRevocationSet.Merge() cutoff = %v, want %v", got.Values[0].Body.RevokedAt, tc.wantCutoff)
			}
		})
	}
	tooLarge := BugWriterRevocationSet{Values: make([]core.Signed[BugWriterRevocationBody], int(BugWriterRevocationPersistenceMaximum)+1)}
	if !errors.Is(tooLarge.Validate(), core.ErrLicenseContract) {
		t.Fatalf("BugWriterRevocationSet.Validate() accepted %d retained revocations", len(tooLarge.Values))
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
