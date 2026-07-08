package core

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"encoding/json"
)

const BLAKE3DigestBytes = 32

type SHA256Hex struct {
	value string
}

func NewSHA256Hex(sum [sha256.Size]byte) SHA256Hex {
	return SHA256Hex{value: hex.EncodeToString(sum[:])}
}

func ParseSHA256Hex(value string) (SHA256Hex, error) {
	if err := validateFixedHex(value, sha256.Size); err != nil {
		return SHA256Hex{}, fmt.Errorf(ErrFmtSHA256, ErrFoundationContract)
	}
	return SHA256Hex{value: value}, nil
}

func (h SHA256Hex) String() string {
	return h.value
}

func (h SHA256Hex) IsZero() bool {
	return h.value == ""
}

func (h SHA256Hex) Validate() error {
	if h.IsZero() {
		return fmt.Errorf(ErrFmtSHA256, ErrFoundationContract)
	}
	return validateFixedHex(h.value, sha256.Size)
}

func (h SHA256Hex) MarshalJSON() ([]byte, error) {
	if !h.IsZero() {
		if err := h.Validate(); err != nil {
			return nil, err
		}
	}
	return json.Marshal(h.String())
}

func (h *SHA256Hex) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSHA256, ErrFoundationContract)
	}
	if value == "" {
		*h = SHA256Hex{}
		return nil
	}
	parsed, err := ParseSHA256Hex(value)
	if err != nil {
		return err
	}
	*h = parsed
	return h.Validate()
}

type BLAKE3Hex struct {
	value string
}

func ParseBLAKE3Hex(value string) (BLAKE3Hex, error) {
	if err := validateFixedHex(value, BLAKE3DigestBytes); err != nil {
		return BLAKE3Hex{}, fmt.Errorf(ErrFmtBLAKE3, ErrFoundationContract)
	}
	return BLAKE3Hex{value: value}, nil
}

func (h BLAKE3Hex) String() string {
	return h.value
}

func (h BLAKE3Hex) IsZero() bool {
	return h.value == ""
}

func (h BLAKE3Hex) Validate() error {
	if h.IsZero() {
		return fmt.Errorf(ErrFmtBLAKE3, ErrFoundationContract)
	}
	return validateFixedHex(h.value, BLAKE3DigestBytes)
}

func (h BLAKE3Hex) MarshalJSON() ([]byte, error) {
	if !h.IsZero() {
		if err := h.Validate(); err != nil {
			return nil, err
		}
	}
	return json.Marshal(h.String())
}

func (h *BLAKE3Hex) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBLAKE3, ErrFoundationContract)
	}
	if value == "" {
		*h = BLAKE3Hex{}
		return nil
	}
	parsed, err := ParseBLAKE3Hex(value)
	if err != nil {
		return err
	}
	*h = parsed
	return h.Validate()
}

type Ed25519PublicKeyHex struct {
	value string
}

func NewEd25519PublicKeyHex(key ed25519.PublicKey) (Ed25519PublicKeyHex, error) {
	if len(key) != ed25519.PublicKeySize {
		return Ed25519PublicKeyHex{}, fmt.Errorf(ErrFmtEd25519PublicKey, ErrFoundationContract)
	}
	return Ed25519PublicKeyHex{value: hex.EncodeToString(key)}, nil
}

func ParseEd25519PublicKeyHex(value string) (Ed25519PublicKeyHex, error) {
	if err := validateFixedHex(value, ed25519.PublicKeySize); err != nil {
		return Ed25519PublicKeyHex{}, fmt.Errorf(ErrFmtEd25519PublicKey, ErrFoundationContract)
	}
	return Ed25519PublicKeyHex{value: value}, nil
}

func (h Ed25519PublicKeyHex) Bytes() (ed25519.PublicKey, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(h.value)
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(raw), nil
}

func (h Ed25519PublicKeyHex) String() string {
	return h.value
}

func (h Ed25519PublicKeyHex) IsZero() bool {
	return h.value == ""
}

func (h Ed25519PublicKeyHex) Validate() error {
	if h.IsZero() {
		return fmt.Errorf(ErrFmtEd25519PublicKey, ErrFoundationContract)
	}
	return validateFixedHex(h.value, ed25519.PublicKeySize)
}

func (h Ed25519PublicKeyHex) MarshalJSON() ([]byte, error) {
	if !h.IsZero() {
		if err := h.Validate(); err != nil {
			return nil, err
		}
	}
	return json.Marshal(h.String())
}

func (h *Ed25519PublicKeyHex) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtEd25519PublicKey, ErrFoundationContract)
	}
	if value == "" {
		*h = Ed25519PublicKeyHex{}
		return nil
	}
	parsed, err := ParseEd25519PublicKeyHex(value)
	if err != nil {
		return err
	}
	*h = parsed
	return h.Validate()
}

type Ed25519SignatureHex struct {
	value string
}

func NewEd25519SignatureHex(signature []byte) (Ed25519SignatureHex, error) {
	if len(signature) != ed25519.SignatureSize {
		return Ed25519SignatureHex{}, fmt.Errorf(ErrFmtEd25519Signature, ErrFoundationContract)
	}
	return Ed25519SignatureHex{value: hex.EncodeToString(signature)}, nil
}

func ParseEd25519SignatureHex(value string) (Ed25519SignatureHex, error) {
	if err := validateFixedHex(value, ed25519.SignatureSize); err != nil {
		return Ed25519SignatureHex{}, fmt.Errorf(ErrFmtEd25519Signature, ErrFoundationContract)
	}
	return Ed25519SignatureHex{value: value}, nil
}

func (h Ed25519SignatureHex) Bytes() ([]byte, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return hex.DecodeString(h.value)
}

func (h Ed25519SignatureHex) String() string {
	return h.value
}

func (h Ed25519SignatureHex) IsZero() bool {
	return h.value == ""
}

func (h Ed25519SignatureHex) Validate() error {
	if h.IsZero() {
		return fmt.Errorf(ErrFmtEd25519Signature, ErrFoundationContract)
	}
	return validateFixedHex(h.value, ed25519.SignatureSize)
}

func (h Ed25519SignatureHex) MarshalJSON() ([]byte, error) {
	if !h.IsZero() {
		if err := h.Validate(); err != nil {
			return nil, err
		}
	}
	return json.Marshal(h.String())
}

func (h *Ed25519SignatureHex) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf(ErrFmtEd25519Signature, ErrFoundationContract)
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtEd25519Signature, ErrFoundationContract)
	}
	if value == "" {
		*h = Ed25519SignatureHex{}
		return nil
	}
	parsed, err := ParseEd25519SignatureHex(value)
	if err != nil {
		return err
	}
	*h = parsed
	return h.Validate()
}

func validateFixedHex(value string, byteLen int) error {
	if len(value) != byteLen*2 {
		return fmt.Errorf("%w: hex length", ErrFoundationContract)
	}
	if !IsLowerHex(value) {
		return fmt.Errorf("%w: hex alphabet", ErrFoundationContract)
	}
	return nil
}

func IsLowerHex(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
