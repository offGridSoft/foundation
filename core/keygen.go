package core

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	ErrFmtGeneratedSigningKey = "core.GeneratedSigningKey: %w"
	ErrFmtGeneratedSecret     = "core.GeneratedSecret: %w"
	ErrFmtKeygenKind          = "core.KeygenKind: %w"

	// SecretByteMinimum is the smallest symmetric-secret width Foundation mints:
	// 128 bits. HMAC-SHA256 keys, AEAD keys, CSRF secrets, and password peppers
	// use 32 bytes (256 bits); nothing narrower than 16 bytes is a real secret.
	SecretByteMinimum = 16
)

// KeygenKind is the closed set of key material the generator can mint. The zero
// value is invalid so an unparsed kind can never select a generator.
type KeygenKind uint8

const (
	KeygenKindInvalid KeygenKind = iota
	KeygenKindEd25519
	KeygenKindSecret
	keygenKindSentinel
)

const (
	KeygenKindTokenEd25519 = "ed25519"
	KeygenKindTokenSecret  = "secret"
)

func (k KeygenKind) Valid() bool { return k > KeygenKindInvalid && k < keygenKindSentinel }

func (k KeygenKind) Validate() error {
	if !k.Valid() {
		return fmt.Errorf(ErrFmtKeygenKind, ErrFoundationContract)
	}
	return nil
}

func (k KeygenKind) String() string {
	switch k {
	case KeygenKindEd25519:
		return KeygenKindTokenEd25519
	case KeygenKindSecret:
		return KeygenKindTokenSecret
	default:
		return ""
	}
}

func ParseKeygenKind(value string) (KeygenKind, error) {
	switch value {
	case KeygenKindTokenEd25519:
		return KeygenKindEd25519, nil
	case KeygenKindTokenSecret:
		return KeygenKindSecret, nil
	default:
		return KeygenKindInvalid, fmt.Errorf(ErrFmtKeygenKind, ErrFoundationContract)
	}
}

// GeneratedSigningKey is the output contract of Ed25519 signing-key generation:
// the canonical standard-padded base64 of the complete 64-byte private key that
// an operator persists (1Password / Secret Manager / env ingress), paired with
// its derived public hex. It carries the base64 because Ed25519SigningKey
// deliberately exposes no serialization; Validate proves the private half
// round-trips through the one canonical parser and derives exactly the stated
// public, so no caller can emit a key the ingress would reject or whose public
// half disagrees with its private.
type GeneratedSigningKey struct {
	PrivateKeyBase64 string
	PublicKeyHex     Ed25519PublicKeyHex
}

// Validate re-parses the private base64 through the single canonical contract
// and requires its derived public to equal the stated public.
func (g GeneratedSigningKey) Validate() error {
	key, err := ParseEd25519SigningKeyBase64(g.PrivateKeyBase64)
	if err != nil {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, err)
	}
	public, err := key.PublicKey()
	if err != nil {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, err)
	}
	if public != g.PublicKeyHex {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, ErrFoundationContract)
	}
	return nil
}

// GenerateEd25519SigningKey mints a fresh offline-authority key from the system
// CSPRNG and returns its validated output contract. The private bytes leave only
// as the base64 field the caller persists; they are never otherwise exposed.
func GenerateEd25519SigningKey() (GeneratedSigningKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return GeneratedSigningKey{}, fmt.Errorf(ErrFmtGeneratedSigningKey, err)
	}
	publicHex, err := NewEd25519PublicKeyHex(public)
	if err != nil {
		return GeneratedSigningKey{}, fmt.Errorf(ErrFmtGeneratedSigningKey, err)
	}
	out := GeneratedSigningKey{
		PrivateKeyBase64: base64.StdEncoding.EncodeToString(private),
		PublicKeyHex:     publicHex,
	}
	if err := out.Validate(); err != nil {
		return GeneratedSigningKey{}, err
	}
	return out, nil
}

// GeneratedSecret is the output contract of symmetric-secret generation: the
// lowercase hex of ByteLen random bytes (ByteLen >= SecretByteMinimum). It is
// the right shape for HMAC keys, AEAD keys, CSRF secrets, and password peppers —
// anything the server both mints and verifies, where no external party needs a
// public half and asymmetric signing would be the wrong primitive.
type GeneratedSecret struct {
	Hex     string
	ByteLen int
}

// Validate requires a real secret width and exact fixed-length lowercase hex.
func (s GeneratedSecret) Validate() error {
	if s.ByteLen < SecretByteMinimum {
		return fmt.Errorf(ErrFmtGeneratedSecret, ErrFoundationContract)
	}
	if err := validateFixedHex(s.Hex, s.ByteLen); err != nil {
		return fmt.Errorf(ErrFmtGeneratedSecret, ErrFoundationContract)
	}
	return nil
}

// GenerateSecretHex mints byteLen random bytes from the system CSPRNG as
// lowercase hex. byteLen must be at least SecretByteMinimum.
func GenerateSecretHex(byteLen int) (GeneratedSecret, error) {
	if byteLen < SecretByteMinimum {
		return GeneratedSecret{}, fmt.Errorf(ErrFmtGeneratedSecret, ErrFoundationContract)
	}
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return GeneratedSecret{}, fmt.Errorf(ErrFmtGeneratedSecret, err)
	}
	out := GeneratedSecret{Hex: hex.EncodeToString(raw), ByteLen: byteLen}
	if err := out.Validate(); err != nil {
		return GeneratedSecret{}, err
	}
	return out, nil
}
