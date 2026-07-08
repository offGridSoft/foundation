package core

import (
	"crypto/ed25519"
	"fmt"

	"encoding/json"
)

const (
	ErrFmtSigningKeyID    = "core.SigningKeyID: %w"
	ErrFmtSigningKeyring  = "core.SigningKeyring: %w"
	ErrFmtSignedSignature = "core.Signed.Signature: %w"
	SignedMessageDomain   = "foundation-signed-v1"
	SignedMessageSep      = byte(0)
)

type CanonicalBody interface {
	Validate() error
	Canonical(dst []byte) ([]byte, error)
}

type SigningKeyID struct {
	value string
}

func ParseSigningKeyID(value string) (SigningKeyID, error) {
	if err := ValidateOpaqueToken(value, OpaqueTokenDefaultMaxRunes); err != nil {
		return SigningKeyID{}, fmt.Errorf(ErrFmtSigningKeyID, ErrFoundationContract)
	}
	return SigningKeyID{value: value}, nil
}

func (id SigningKeyID) String() string {
	return id.value
}

func (id SigningKeyID) Validate() error {
	_, err := ParseSigningKeyID(id.value)
	return err
}

func (id SigningKeyID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *SigningKeyID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSigningKeyID, ErrFoundationContract)
	}
	parsed, err := ParseSigningKeyID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return id.Validate()
}

type SigningPublicKey struct {
	ID        SigningKeyID
	PublicKey Ed25519PublicKeyHex
}

func (k SigningPublicKey) Validate() error {
	if err := k.ID.Validate(); err != nil {
		return err
	}
	return k.PublicKey.Validate()
}

type SigningKeyring struct {
	Keys []SigningPublicKey
}

func (r SigningKeyring) Validate() error {
	if len(r.Keys) == 0 {
		return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
	}
	seen := make(map[string]struct{}, len(r.Keys))
	for _, key := range r.Keys {
		if err := key.Validate(); err != nil {
			return err
		}
		id := key.ID.String()
		if _, exists := seen[id]; exists {
			return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (r SigningKeyring) Lookup(id SigningKeyID) (ed25519.PublicKey, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	for _, key := range r.Keys {
		if key.ID.String() == id.String() {
			return key.PublicKey.Bytes()
		}
	}
	return nil, fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
}

type Signed[B CanonicalBody] struct {
	Body      B            `json:"body"`
	KeyID     SigningKeyID `json:"key_id"`
	Signature []byte       `json:"signature"`
}

func (s Signed[B]) Validate() error {
	if err := s.KeyID.Validate(); err != nil {
		return err
	}
	if len(s.Signature) != ed25519.SignatureSize {
		return fmt.Errorf(ErrFmtSignedSignature, ErrFoundationContract)
	}
	return s.Body.Validate()
}

func (s Signed[B]) Verify(keyring SigningKeyring) error {
	if err := s.Validate(); err != nil {
		return err
	}
	key, err := keyring.Lookup(s.KeyID)
	if err != nil {
		return err
	}
	message, err := AppendSignedMessage(nil, s.KeyID, s.Body)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, message, s.Signature) {
		return fmt.Errorf(ErrFmtSignedSignature, ErrFoundationContract)
	}
	return nil
}

func AppendSignedMessage[B CanonicalBody](dst []byte, keyID SigningKeyID, body B) ([]byte, error) {
	if err := keyID.Validate(); err != nil {
		return nil, err
	}
	canonical, err := body.Canonical(nil)
	if err != nil {
		return nil, err
	}
	dst = append(dst, SignedMessageDomain...)
	dst = append(dst, SignedMessageSep)
	dst = append(dst, keyID.String()...)
	dst = append(dst, SignedMessageSep)
	return append(dst, canonical...), nil
}
