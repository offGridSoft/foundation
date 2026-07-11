package core

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

const (
	ErrFmtSigningKeyID    = "core.SigningKeyID: %w"
	ErrFmtSigningKeyring  = "core.SigningKeyring: %w"
	ErrFmtSignedSignature = "core.Signed.Signature: %w"
	SignedMessageDomain   = "foundation-signed-v1"
	SignedMessageSep      = byte(0)
	SigningKeyringMaxKeys = 16
)

type CanonicalBody interface {
	Validatable
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
	if err := id.Validate(); err != nil {
		return err
	}
	return nil
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
	if len(r.Keys) == 0 || len(r.Keys) > SigningKeyringMaxKeys {
		return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
	}
	for index, key := range r.Keys {
		if err := key.Validate(); err != nil {
			return err
		}
		for _, prior := range r.Keys[:index] {
			if prior.ID == key.ID {
				return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
			}
		}
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
	return r.lookupValidated(id)
}

func (r SigningKeyring) lookupValidated(id SigningKeyID) (ed25519.PublicKey, error) {
	for _, key := range r.Keys {
		if key.ID.String() == id.String() {
			return key.PublicKey.Bytes()
		}
	}
	return nil, fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
}

type Signed[B CanonicalBody] struct {
	Body      B                   `json:"body"`
	KeyID     SigningKeyID        `json:"key_id"`
	Signature Ed25519SignatureHex `json:"signature"`
}

func (s Signed[B]) Validate() error {
	if err := s.KeyID.Validate(); err != nil {
		return err
	}
	if err := s.Signature.Validate(); err != nil {
		return fmt.Errorf(ErrFmtSignedSignature, err)
	}
	return s.Body.Validate()
}

func (s Signed[B]) Verify(keyring SigningKeyring) error {
	message, err := AppendSignedMessage(nil, s.KeyID, s.Body)
	if err != nil {
		return err
	}
	if err := s.Signature.Validate(); err != nil {
		return fmt.Errorf(ErrFmtSignedSignature, err)
	}
	if err := keyring.Validate(); err != nil {
		return err
	}
	key, err := keyring.lookupValidated(s.KeyID)
	if err != nil {
		return err
	}
	signature, err := s.Signature.Bytes()
	if err != nil {
		return fmt.Errorf(ErrFmtSignedSignature, err)
	}
	if !ed25519.Verify(key, message, signature) {
		return fmt.Errorf(ErrFmtSignedSignature, ErrFoundationContract)
	}
	return nil
}

func AppendSignedMessage[B CanonicalBody](dst []byte, keyID SigningKeyID, body B) ([]byte, error) {
	if err := keyID.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, SignedMessageDomain...)
	dst = append(dst, SignedMessageSep)
	dst = append(dst, keyID.String()...)
	dst = append(dst, SignedMessageSep)
	return body.Canonical(dst)
}
