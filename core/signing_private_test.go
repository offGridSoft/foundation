package core

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestEd25519SigningKeyHostileTable(t *testing.T) {
	t.Parallel()
	valid := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	validText := base64.StdEncoding.EncodeToString(valid)
	inconsistent := append([]byte(nil), valid...)
	inconsistent[len(inconsistent)-1] ^= 1
	cases := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "valid canonical private key", value: validText},
		{name: "empty", value: "", wantError: true},
		{name: "invalid base64", value: "not-base64", wantError: true},
		{name: "unpadded base64", value: strings.TrimRight(validText, "="), wantError: true},
		{name: "seed is not private key", value: base64.StdEncoding.EncodeToString(valid.Seed()), wantError: true},
		{name: "short private key", value: base64.StdEncoding.EncodeToString(valid[:len(valid)-1]), wantError: true},
		{name: "long private key", value: base64.StdEncoding.EncodeToString(append(valid, 0)), wantError: true},
		{name: "inconsistent public suffix", value: base64.StdEncoding.EncodeToString(inconsistent), wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, err := ParseEd25519SigningKeyBase64(tc.value)
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseEd25519SigningKeyBase64() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEd25519SigningKeyBase64() error = %v", err)
			}
			if err := key.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestSignCanonicalBindsBodyAndAuthority(t *testing.T) {
	t.Parallel()
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	key, err := ParseEd25519SigningKeyBase64(base64.StdEncoding.EncodeToString(private))
	if err != nil {
		t.Fatalf("ParseEd25519SigningKeyBase64() error = %v", err)
	}
	body := signedTestBody{Value: "ok", Schema: SchemaReleaseCommandRun}
	signed, err := SignCanonical(key, body)
	if err != nil {
		t.Fatalf("SignCanonical() error = %v", err)
	}
	public, err := key.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey() error = %v", err)
	}
	keyring, err := NewPinnedAuthorityKeyring(public)
	if err != nil {
		t.Fatalf("NewPinnedAuthorityKeyring() error = %v", err)
	}
	if err := signed.Verify(keyring); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	signed.Body.Value = "changed"
	if err := signed.Verify(keyring); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("Verify(mutated body) error = %v, want %v", err, ErrFoundationContract)
	}
}
