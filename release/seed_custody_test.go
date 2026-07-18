package release

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestEffectiveSeedBindsReleaseAndDoesNotAliasCustodyBytes(t *testing.T) {
	t.Parallel()
	seed := mustCustodySeed(t)
	firstID := mustSeedReleaseID(t, "bug-2026-"+strings.Repeat("c", 40))
	secondID := mustSeedReleaseID(t, "bug-2026-"+strings.Repeat("d", 40))
	first, err := DeriveGarbleSeed(seed, firstID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeriveGarbleSeed(seed, secondID)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("EffectiveSeed() first = %q, second = %q, want distinct release-bound seeds", first.String(), second.String())
	}
	if strings.Contains(first.String(), "=") || len(first.String()) != GarbleSeedEncodedBytes {
		t.Fatalf("EffectiveSeed() = %q, want exact %d-byte unpadded base64 contract", first.String(), GarbleSeedEncodedBytes)
	}
	decoded, err := base64.RawStdEncoding.DecodeString(first.String())
	if err != nil || len(decoded) != GarbleSeedBytes {
		t.Fatalf("DecodeString(EffectiveSeed()) bytes = %d, error = %v, want %d and nil", len(decoded), err, GarbleSeedBytes)
	}
}

func validCustodySeedBytes() []byte {
	value := make([]byte, core.GarbleCustodySeedBytes)
	for index := range value {
		value[index] = byte(index + 1)
	}
	return value
}

func mustCustodySeed(t *testing.T) core.GarbleCustodySeed {
	t.Helper()
	seed, err := core.NewGarbleCustodySeed(validCustodySeedBytes())
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

func mustSeedReleaseID(t *testing.T, value string) ReleaseID {
	t.Helper()
	id, err := ParseReleaseID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
