package license

import (
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func testBugWriterKey(t *testing.T) BugWriterKey {
	t.Helper()
	return mustTestBugWriterKey()
}

func mustTestBugWriterKey() BugWriterKey {
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	publicKey, err := core.NewEd25519PublicKeyHex(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		panic(err)
	}
	writer, err := NewBugWriterKey(publicKey)
	if err != nil {
		panic(err)
	}
	return writer
}

func TestBugWriterKeyRejectsSubstitution(t *testing.T) {
	t.Parallel()
	trusted := testBugWriterKey(t)
	foreignSeed := make([]byte, ed25519.SeedSize)
	foreignSeed[len(foreignSeed)-1] = 1
	foreignPrivate := ed25519.NewKeyFromSeed(foreignSeed)
	foreignPublic, err := core.NewEd25519PublicKeyHex(foreignPrivate.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	trusted.PublicKey = foreignPublic
	if err := trusted.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("BugWriterKey.Validate(substituted public key) error = %v, want %v", err, core.ErrLicenseContract)
	}
}
