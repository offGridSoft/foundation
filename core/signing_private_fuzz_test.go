package core

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func FuzzEd25519SigningKeyIngress(f *testing.F) {
	valid := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	f.Add(base64.StdEncoding.EncodeToString(valid))
	f.Add("")
	f.Add("not-base64")
	f.Fuzz(func(t *testing.T, value string) {
		key, err := ParseEd25519SigningKeyBase64(value)
		if err != nil {
			return
		}
		if err := key.Validate(); err != nil {
			t.Fatalf("accepted key failed Validate(): %v", err)
		}
		public, err := key.PublicKey()
		if err != nil {
			t.Fatalf("accepted key failed PublicKey(): %v", err)
		}
		body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
		signed, err := SignCanonical(key, body)
		if err != nil {
			t.Fatalf("accepted key failed SignCanonical(): %v", err)
		}
		keyring, err := NewPinnedAuthorityKeyring(public)
		if err != nil {
			t.Fatalf("accepted key failed NewPinnedAuthorityKeyring(): %v", err)
		}
		if err := signed.Verify(keyring); err != nil {
			t.Fatalf("signed value failed Verify(): %v", err)
		}
	})
}
