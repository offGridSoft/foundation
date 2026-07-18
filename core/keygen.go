package core

import (
	"encoding/json"
	"errors"
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
	// SecretByteStandard is the canonical width for HMAC-SHA256 keys, AEAD keys,
	// CSRF secrets, and password peppers.
	SecretByteStandard = 32
	// SecretByteMaximum bounds caller-controlled allocation while covering every
	// symmetric secret Foundation owns. Current HMAC, AEAD, CSRF, and pepper
	// contracts use 32 bytes; 64 bytes preserves room for wider keyed hashes.
	SecretByteMaximum = 64
)

// KeygenKind is the closed set of key material the generator can mint. The zero
// value is invalid so an unparsed kind can never select a generator.
type KeygenKind uint8

const (
	KeygenKindInvalid KeygenKind = iota
	KeygenKindEd25519
	KeygenKindSecret
	KeygenKindGarbleCustody
	keygenKindSentinel
)

const (
	KeygenKindTokenEd25519       = "ed25519"
	KeygenKindTokenSecret        = "secret"
	KeygenKindTokenGarbleCustody = "garble-custody"
)

func (k KeygenKind) IsValid() bool { return k > KeygenKindInvalid && k < keygenKindSentinel }

func (k KeygenKind) Validate() error {
	if !k.IsValid() {
		return fmt.Errorf(ErrFmtKeygenKind, ErrKeygenContract)
	}
	return nil
}

func (k KeygenKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.String())
}

func (k *KeygenKind) UnmarshalJSON(data []byte) error {
	if k == nil {
		return fmt.Errorf(ErrFmtKeygenKind, ErrKeygenContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtKeygenKind, ErrKeygenContract)
	}
	parsed, err := ParseKeygenKind(token)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

func (k KeygenKind) String() string {
	switch k {
	case KeygenKindEd25519:
		return KeygenKindTokenEd25519
	case KeygenKindSecret:
		return KeygenKindTokenSecret
	case KeygenKindGarbleCustody:
		return KeygenKindTokenGarbleCustody
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
	case KeygenKindTokenGarbleCustody:
		return KeygenKindGarbleCustody, nil
	default:
		return KeygenKindInvalid, fmt.Errorf(ErrFmtKeygenKind, ErrKeygenContract)
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
	if err := g.PublicKeyHex.Validate(); err != nil {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, errors.Join(ErrKeygenContract, err))
	}
	key, err := ParseEd25519SigningKeyBase64(g.PrivateKeyBase64)
	if err != nil {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, errors.Join(ErrKeygenContract, err))
	}
	public, err := key.PublicKey()
	if err != nil {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, errors.Join(ErrKeygenContract, err))
	}
	if public != g.PublicKeyHex {
		return fmt.Errorf(ErrFmtGeneratedSigningKey, ErrKeygenContract)
	}
	return nil
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
	if s.ByteLen < SecretByteMinimum || s.ByteLen > SecretByteMaximum {
		return fmt.Errorf(ErrFmtGeneratedSecret, ErrKeygenContract)
	}
	if err := validateFixedHex(s.Hex, s.ByteLen); err != nil {
		return fmt.Errorf(ErrFmtGeneratedSecret, ErrKeygenContract)
	}
	return nil
}
