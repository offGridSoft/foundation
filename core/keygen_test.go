package core

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// validSigningKeyFixture builds a deterministic, canonically-consistent
// GeneratedSigningKey plus the raw private bytes the hostile cases mutate.
func validSigningKeyFixture(t *testing.T) (GeneratedSigningKey, ed25519.PrivateKey) {
	t.Helper()
	priv := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	pubHex, err := NewEd25519PublicKeyHex(ed25519.PublicKey(priv[ed25519.SeedSize:]))
	if err != nil {
		t.Fatalf("NewEd25519PublicKeyHex() error = %v", err)
	}
	return GeneratedSigningKey{
		PrivateKeyBase64: base64.StdEncoding.EncodeToString(priv),
		PublicKeyHex:     pubHex,
	}, priv
}

func TestGeneratedSigningKeyHostileTable(t *testing.T) {
	t.Parallel()
	valid, priv := validSigningKeyFixture(t)

	inconsistent := append(ed25519.PrivateKey(nil), priv...)
	inconsistent[len(inconsistent)-1] ^= 1

	otherPriv := ed25519.NewKeyFromSeed(append(make([]byte, ed25519.SeedSize-1), 1))
	otherPubHex, err := NewEd25519PublicKeyHex(ed25519.PublicKey(otherPriv[ed25519.SeedSize:]))
	if err != nil {
		t.Fatalf("NewEd25519PublicKeyHex(other) error = %v", err)
	}

	cases := []struct {
		name      string
		key       GeneratedSigningKey
		wantError bool
	}{
		{name: "valid canonical generated key", key: valid},
		{name: "empty private", key: GeneratedSigningKey{PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "invalid base64 private", key: GeneratedSigningKey{PrivateKeyBase64: "not-base64", PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "unpadded base64 private", key: GeneratedSigningKey{PrivateKeyBase64: strings.TrimRight(valid.PrivateKeyBase64, "="), PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "seed is not private key", key: GeneratedSigningKey{PrivateKeyBase64: base64.StdEncoding.EncodeToString(priv.Seed()), PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "short private", key: GeneratedSigningKey{PrivateKeyBase64: base64.StdEncoding.EncodeToString(priv[:len(priv)-1]), PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "long private", key: GeneratedSigningKey{PrivateKeyBase64: base64.StdEncoding.EncodeToString(append(append([]byte(nil), priv...), 0)), PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "inconsistent private public suffix", key: GeneratedSigningKey{PrivateKeyBase64: base64.StdEncoding.EncodeToString(inconsistent), PublicKeyHex: valid.PublicKeyHex}, wantError: true},
		{name: "public does not match private", key: GeneratedSigningKey{PrivateKeyBase64: valid.PrivateKeyBase64, PublicKeyHex: otherPubHex}, wantError: true},
		{name: "zero public", key: GeneratedSigningKey{PrivateKeyBase64: valid.PrivateKeyBase64}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.key.Validate()
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("Validate() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestGenerateEd25519SigningKeyProducesIngressReadyKey proves the generator's
// output validates, its private half is accepted by the one canonical ingress
// parser, and the derived public equals the stated public.
func TestGenerateEd25519SigningKeyProducesIngressReadyKey(t *testing.T) {
	t.Parallel()
	key, err := GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() error = %v", err)
	}
	if err := key.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	parsed, err := ParseEd25519SigningKeyBase64(key.PrivateKeyBase64)
	if err != nil {
		t.Fatalf("ParseEd25519SigningKeyBase64() error = %v, want ingress-ready key", err)
	}
	derived, err := parsed.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey() error = %v", err)
	}
	if derived != key.PublicKeyHex {
		t.Fatalf("derived public = %v, want %v", derived, key.PublicKeyHex)
	}
}

// TestGenerateEd25519SigningKeyIsUnpredictable proves two mints never collide,
// i.e. the generator draws real entropy and is not seeded from a constant.
func TestGenerateEd25519SigningKeyIsUnpredictable(t *testing.T) {
	t.Parallel()
	first, err := GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() first error = %v", err)
	}
	second, err := GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() second error = %v", err)
	}
	if first.PrivateKeyBase64 == second.PrivateKeyBase64 {
		t.Fatal("two generated keys share a private key; generator is not drawing entropy")
	}
	if first.PublicKeyHex == second.PublicKeyHex {
		t.Fatal("two generated keys share a public key; generator is not drawing entropy")
	}
}

func TestGeneratedSecretHostileTable(t *testing.T) {
	t.Parallel()
	hex32 := hex.EncodeToString(make([]byte, 32))
	hex16 := hex.EncodeToString(make([]byte, 16))
	cases := []struct {
		name      string
		secret    GeneratedSecret
		wantError bool
	}{
		{name: "valid 32-byte secret", secret: GeneratedSecret{Hex: hex32, ByteLen: 32}},
		{name: "valid 16-byte minimum", secret: GeneratedSecret{Hex: hex16, ByteLen: 16}},
		{name: "below minimum width", secret: GeneratedSecret{Hex: hex.EncodeToString(make([]byte, 8)), ByteLen: 8}, wantError: true},
		{name: "hex shorter than declared length", secret: GeneratedSecret{Hex: hex16, ByteLen: 32}, wantError: true},
		{name: "uppercase hex", secret: GeneratedSecret{Hex: strings.ToUpper(strings.Repeat("ab", 32)), ByteLen: 32}, wantError: true},
		{name: "non-hex alphabet", secret: GeneratedSecret{Hex: strings.Repeat("zz", 32), ByteLen: 32}, wantError: true},
		{name: "empty hex", secret: GeneratedSecret{Hex: "", ByteLen: 32}, wantError: true},
		{name: "odd hex length", secret: GeneratedSecret{Hex: hex32[:len(hex32)-1], ByteLen: 32}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.secret.Validate()
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("Validate() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestGenerateSecretHexProducesValidSecret(t *testing.T) {
	t.Parallel()
	s, err := GenerateSecretHex(32)
	if err != nil {
		t.Fatalf("GenerateSecretHex(32) error = %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if s.ByteLen != 32 || len(s.Hex) != 64 {
		t.Fatalf("GenerateSecretHex(32) = {ByteLen:%d, len(Hex):%d}, want {32, 64}", s.ByteLen, len(s.Hex))
	}
	if _, err := GenerateSecretHex(SecretByteMinimum - 1); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("GenerateSecretHex(too small) error = %v, want %v", err, ErrFoundationContract)
	}
	other, err := GenerateSecretHex(32)
	if err != nil {
		t.Fatalf("GenerateSecretHex(32) second error = %v", err)
	}
	if s.Hex == other.Hex {
		t.Fatal("two generated secrets are identical; generator is not drawing entropy")
	}
}

func TestKeygenKindHostileTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		value     string
		want      KeygenKind
		wantError bool
	}{
		{name: "ed25519", value: KeygenKindTokenEd25519, want: KeygenKindEd25519},
		{name: "secret", value: KeygenKindTokenSecret, want: KeygenKindSecret},
		{name: "empty", value: "", wantError: true},
		{name: "uppercase", value: "ED25519", wantError: true},
		{name: "unknown kind", value: "rsa", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseKeygenKind(tc.value)
			if tc.wantError {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseKeygenKind(%q) error = %v, want %v", tc.value, err, ErrFoundationContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKeygenKind(%q) error = %v", tc.value, err)
			}
			if got != tc.want {
				t.Fatalf("ParseKeygenKind(%q) = %v, want %v", tc.value, got, tc.want)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got.String() != tc.value {
				t.Fatalf("String() = %q, want %q", got.String(), tc.value)
			}
		})
	}
	if err := KeygenKindInvalid.Validate(); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("KeygenKindInvalid.Validate() = %v, want %v", err, ErrFoundationContract)
	}
}
