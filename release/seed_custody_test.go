package release

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSeedGrantCanonicalRoundTripAndRequestProjection(t *testing.T) {
	t.Parallel()
	body := validSeedGrant(t)
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, canonical) {
		t.Fatalf("json.Marshal(SeedGrantBody) = %s, want canonical %s", encoded, canonical)
	}
	decoded, err := core.DecodeStrictJSON[SeedGrantBody](canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded SeedGrantBody.Validate() error = %v", err)
	}
	request := body.Request()
	if err := request.Validate(); err != nil {
		t.Fatalf("SeedGrantBody.Request().Validate() error = %v", err)
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(requestJSON) == 0 {
		t.Fatalf("json.Marshal(SeedRequest) bytes = %d, want non-zero", len(requestJSON))
	}
}

func TestSeedGrantHostileBindingTable(t *testing.T) {
	t.Parallel()
	valid := validSeedGrant(t)
	for _, tc := range []struct {
		mutate func(*SeedGrantBody)
		name   string
		valid  bool
	}{
		{name: "valid grant", valid: true, mutate: func(*SeedGrantBody) {}},
		{name: "wrong seed digest", mutate: func(g *SeedGrantBody) { g.SeedSHA256 = mustSeedSHA256(t, strings.Repeat("a", 64)) }},
		{name: "missing release ID", mutate: func(g *SeedGrantBody) { g.ReleaseID = ReleaseID{} }},
		{name: "missing commit", mutate: func(g *SeedGrantBody) { g.Commit = core.BuildCommit{} }},
		{name: "missing issued time", mutate: func(g *SeedGrantBody) { g.IssuedAt = core.UnixNanoTime{} }},
		{name: "wrong schema", mutate: func(g *SeedGrantBody) { g.Schema = core.SchemaReleaseManifest }},
		{name: "wrong product", mutate: func(g *SeedGrantBody) { g.Product = core.ProductWitness }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grant := valid
			tc.mutate(&grant)
			err := grant.Validate()
			if tc.valid {
				if err != nil {
					t.Fatalf("SeedGrantBody.Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("SeedGrantBody.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestSeedGrantSignatureAndReleaseBinding(t *testing.T) {
	t.Parallel()
	body := validSeedGrant(t)
	keyID, err := core.ParseSigningKeyID("release-seed-test")
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicHex, err := core.NewEd25519PublicKeyHex(public)
	if err != nil {
		t.Fatal(err)
	}
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(private, message))
	if err != nil {
		t.Fatal(err)
	}
	signed := core.Signed[SeedGrantBody]{Body: body, KeyID: keyID, Signature: signature}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicHex}}}
	request := body.Request()
	if err := VerifySeedGrant(signed, request, keyring); err != nil {
		t.Fatalf("VerifySeedGrant() error = %v", err)
	}
	otherRequest := request
	otherRequest.Commit = mustSeedCommit(t, strings.Repeat("d", 40))
	otherRequest.ReleaseID = mustSeedReleaseID(t, "bug-2026-"+otherRequest.Commit.String())
	if err := VerifySeedGrant(signed, otherRequest, keyring); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("VerifySeedGrant(other valid request) error = %v, want %v", err, core.ErrReleaseContract)
	}
	signed.Body.ReleaseID = mustSeedReleaseID(t, "bug-2026-"+strings.Repeat("d", 40))
	if err := VerifySeedGrant(signed, request, keyring); err == nil {
		t.Fatal("VerifySeedGrant(swapped release) error = nil, want refusal")
	}
}

func TestBuildSeedGrantBodyDerivesOwnedFields(t *testing.T) {
	t.Parallel()
	want := validSeedGrant(t)
	got, err := BuildSeedGrantBody(want.Request(), want.Seed, want.IssuedAt)
	if err != nil {
		t.Fatalf("BuildSeedGrantBody() error = %v", err)
	}
	if got != want {
		t.Fatalf("BuildSeedGrantBody() = %+v, want %+v", got, want)
	}
	if _, err := BuildSeedGrantBody(SeedRequest{}, want.Seed, want.IssuedAt); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("BuildSeedGrantBody(empty request) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func TestEffectiveSeedBindsReleaseAndDoesNotAliasCustodyBytes(t *testing.T) {
	t.Parallel()
	first := validSeedGrant(t)
	second := first
	second.Commit = mustSeedCommit(t, strings.Repeat("d", 40))
	second.ReleaseID = mustSeedReleaseID(t, "bug-2026-"+second.Commit.String())
	firstSeed, err := first.EffectiveSeed()
	if err != nil {
		t.Fatal(err)
	}
	secondSeed, err := second.EffectiveSeed()
	if err != nil {
		t.Fatal(err)
	}
	if firstSeed == secondSeed {
		t.Fatalf("EffectiveSeed() first = %q, second = %q, want distinct release-bound seeds", firstSeed.String(), secondSeed.String())
	}
	if strings.Contains(firstSeed.String(), "=") || len(firstSeed.String()) != GarbleSeedEncodedBytes {
		t.Fatalf("EffectiveSeed() = %q, want Garble's exact %d-byte unpadded base64 contract", firstSeed.String(), GarbleSeedEncodedBytes)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(firstSeed.String())
	if err != nil || len(decoded) != GarbleSeedBytes {
		t.Fatalf("RawStdEncoding.DecodeString(EffectiveSeed()) bytes = %d, error = %v, want %d bytes and nil", len(decoded), err, GarbleSeedBytes)
	}
	copyBytes := first.Seed.Bytes()
	copyBytes[0] ^= 0xff
	if first.Seed.SHA256() != first.SeedSHA256 {
		t.Fatalf("GarbleCustodySeed.SHA256() = %q after copy mutation, want %q", first.Seed.SHA256().String(), first.SeedSHA256.String())
	}
}

func TestGarbleCustodySeedHostileTable(t *testing.T) {
	t.Parallel()
	valid := make([]byte, GarbleCustodySeedBytes)
	for index := range valid {
		valid[index] = byte(index + 1)
	}
	if _, err := NewGarbleCustodySeed(valid); err != nil {
		t.Fatalf("NewGarbleCustodySeed(valid) error = %v", err)
	}
	for _, hostile := range [][]byte{nil, make([]byte, GarbleCustodySeedBytes-1), make([]byte, GarbleCustodySeedBytes), make([]byte, GarbleCustodySeedBytes+1)} {
		if _, err := NewGarbleCustodySeed(hostile); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("NewGarbleCustodySeed(len=%d) error = %v, want %v", len(hostile), err, core.ErrReleaseContract)
		}
	}
}

func validSeedGrant(t *testing.T) SeedGrantBody {
	t.Helper()
	seedBytes := make([]byte, GarbleCustodySeedBytes)
	for index := range seedBytes {
		seedBytes[index] = byte(index + 1)
	}
	seed, err := NewGarbleCustodySeed(seedBytes)
	if err != nil {
		t.Fatal(err)
	}
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := core.ParseBuildCommit(strings.Repeat("c", 40))
	if err != nil {
		t.Fatal(err)
	}
	return SeedGrantBody{
		Version: version, ReleaseID: mustSeedReleaseID(t, "bug-2026-"+commit.String()), Commit: commit,
		Seed: seed, SeedSHA256: seed.SHA256(), IssuedAt: core.NewUnixNanoTime(time.Unix(1_800_000_000, 1)),
		Schema: core.SchemaReleaseSeedGrant, Product: core.ProductBug,
	}
}

func mustSeedReleaseID(t *testing.T, value string) ReleaseID {
	t.Helper()
	id, err := ParseReleaseID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustSeedSHA256(t *testing.T, value string) core.SHA256Hex {
	t.Helper()
	digest, err := core.ParseSHA256Hex(value)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func mustSeedCommit(t *testing.T, value string) core.BuildCommit {
	t.Helper()
	commit, err := core.ParseBuildCommit(value)
	if err != nil {
		t.Fatal(err)
	}
	return commit
}
