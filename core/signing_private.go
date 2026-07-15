package core

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

const ErrFmtEd25519SigningKey = "core.Ed25519SigningKey: %w"

// Ed25519SigningKey owns an offline authority's canonical private-key
// contract. It intentionally exposes no private-byte or serialization method.
type Ed25519SigningKey struct {
	value [ed25519.PrivateKeySize]byte
}

// ParseEd25519SigningKeyBase64 accepts exactly one canonical representation:
// standard padded base64 of a complete 64-byte Ed25519 private key.
func ParseEd25519SigningKeyBase64(value string) (Ed25519SigningKey, error) {
	raw, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil || len(raw) != ed25519.PrivateKeySize || base64.StdEncoding.EncodeToString(raw) != value {
		return Ed25519SigningKey{}, fmt.Errorf(ErrFmtEd25519SigningKey, ErrFoundationContract)
	}
	var key Ed25519SigningKey
	copy(key.value[:], raw)
	if err := key.Validate(); err != nil {
		return Ed25519SigningKey{}, err
	}
	return key, nil
}

func (k Ed25519SigningKey) Validate() error {
	derived := ed25519.NewKeyFromSeed(k.value[:ed25519.SeedSize])
	if !bytes.Equal(derived, k.value[:]) {
		return fmt.Errorf(ErrFmtEd25519SigningKey, ErrFoundationContract)
	}
	return nil
}

func (k Ed25519SigningKey) PublicKey() (Ed25519PublicKeyHex, error) {
	if err := k.Validate(); err != nil {
		return Ed25519PublicKeyHex{}, err
	}
	public := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(public, k.value[ed25519.SeedSize:])
	return NewEd25519PublicKeyHex(public)
}

// SignCanonical validates and signs exactly the Foundation canonical message
// consumed by Signed.Verify.
func SignCanonical[B CanonicalBody](key Ed25519SigningKey, body B) (Signed[B], error) {
	public, err := key.PublicKey()
	if err != nil {
		return Signed[B]{}, err
	}
	keyID, err := ParseSigningKeyID(public.String())
	if err != nil {
		return Signed[B]{}, err
	}
	message, err := AppendSignedMessage(nil, keyID, body)
	if err != nil {
		return Signed[B]{}, err
	}
	signature, err := NewEd25519SignatureHex(ed25519.Sign(key.value[:], message))
	if err != nil {
		return Signed[B]{}, err
	}
	signed := Signed[B]{Body: body, KeyID: keyID, Signature: signature}
	if err := signed.Validate(); err != nil {
		return Signed[B]{}, err
	}
	return signed, nil
}
