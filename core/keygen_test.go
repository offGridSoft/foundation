package core

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
				if !errors.Is(err, ErrKeygenContract) {
					t.Fatalf("Validate() error = %v, want %v", err, ErrKeygenContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestParseGeneratedSigningKeyDerivesPublicHalf(t *testing.T) {
	t.Parallel()
	valid, _ := validSigningKeyFixture(t)
	parsed, err := ParseGeneratedSigningKey(valid.PrivateKeyBase64)
	if err != nil {
		t.Fatalf("ParseGeneratedSigningKey() error = %v", err)
	}
	if parsed != valid {
		t.Fatalf("ParseGeneratedSigningKey() = %#v, want matching generated key", parsed)
	}
}

func TestGeneratedSecretHostileTable(t *testing.T) {
	t.Parallel()
	hexStandard := hex.EncodeToString(make([]byte, SecretByteStandard))
	hexMinimum := hex.EncodeToString(make([]byte, SecretByteMinimum))
	cases := []struct {
		name      string
		secret    GeneratedSecret
		wantError bool
	}{
		{name: "valid standard secret", secret: GeneratedSecret{Hex: hexStandard, ByteLen: SecretByteStandard}},
		{name: "valid minimum secret", secret: GeneratedSecret{Hex: hexMinimum, ByteLen: SecretByteMinimum}},
		{name: "below minimum width", secret: GeneratedSecret{Hex: hex.EncodeToString(make([]byte, SecretByteMinimum-1)), ByteLen: SecretByteMinimum - 1}, wantError: true},
		{name: "above maximum width", secret: GeneratedSecret{Hex: hex.EncodeToString(make([]byte, SecretByteMaximum+1)), ByteLen: SecretByteMaximum + 1}, wantError: true},
		{name: "hex shorter than declared length", secret: GeneratedSecret{Hex: hexMinimum, ByteLen: SecretByteStandard}, wantError: true},
		{name: "uppercase hex", secret: GeneratedSecret{Hex: strings.ToUpper(strings.Repeat("ab", SecretByteStandard)), ByteLen: SecretByteStandard}, wantError: true},
		{name: "non-hex alphabet", secret: GeneratedSecret{Hex: strings.Repeat("zz", SecretByteStandard), ByteLen: SecretByteStandard}, wantError: true},
		{name: "empty hex", secret: GeneratedSecret{Hex: "", ByteLen: SecretByteStandard}, wantError: true},
		{name: "odd hex length", secret: GeneratedSecret{Hex: hexStandard[:len(hexStandard)-1], ByteLen: SecretByteStandard}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.secret.Validate()
			if tc.wantError {
				if !errors.Is(err, ErrKeygenContract) {
					t.Fatalf("Validate() error = %v, want %v", err, ErrKeygenContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
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
		{name: "garble custody", value: KeygenKindTokenGarble, want: KeygenKindGarbleCustody},
		{name: "empty", value: "", wantError: true},
		{name: "uppercase", value: "ED25519", wantError: true},
		{name: "unknown kind", value: "rsa", wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseKeygenKind(tc.value)
			if tc.wantError {
				if !errors.Is(err, ErrKeygenContract) {
					t.Fatalf("ParseKeygenKind(%q) error = %v, want %v", tc.value, err, ErrKeygenContract)
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
			wire, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal(%v) error = %v, want nil", got, err)
			}
			var decoded KeygenKind
			if err := json.Unmarshal(wire, &decoded); err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v, want nil", wire, err)
			}
			if decoded != got {
				t.Fatalf("JSON round trip = %v, want %v", decoded, got)
			}
		})
	}
	if err := KeygenKindInvalid.Validate(); !errors.Is(err, ErrKeygenContract) {
		t.Fatalf("KeygenKindInvalid.Validate() = %v, want %v", err, ErrKeygenContract)
	}
	if _, err := json.Marshal(KeygenKindInvalid); !errors.Is(err, ErrKeygenContract) {
		t.Fatalf("json.Marshal(KeygenKindInvalid) error = %v, want %v", err, ErrKeygenContract)
	}
	var decoded KeygenKind
	if err := json.Unmarshal([]byte(`"unknown"`), &decoded); !errors.Is(err, ErrKeygenContract) {
		t.Fatalf("json.Unmarshal(unknown KeygenKind) error = %v, want %v", err, ErrKeygenContract)
	}
	var nilKind *KeygenKind
	if err := nilKind.UnmarshalJSON([]byte(`"secret"`)); !errors.Is(err, ErrKeygenContract) {
		t.Fatalf("nil KeygenKind.UnmarshalJSON() error = %v, want %v", err, ErrKeygenContract)
	}
}
