package license

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const BugWriterKeyIDPrefix = "bug-writer-sha256:"

// BugWriterKey is the machine-held signing identity certified by a Bug seat
// lease. Its identifier is derived from the public key, so a server or client
// cannot pair an attacker-chosen identifier with different key material.
type BugWriterKey struct {
	KeyID     core.SigningKeyID        `json:"key_id"`
	PublicKey core.Ed25519PublicKeyHex `json:"public_key"`
}

func NewBugWriterKey(publicKey core.Ed25519PublicKeyHex) (BugWriterKey, error) {
	publicBytes, err := publicKey.Bytes()
	if err != nil {
		return BugWriterKey{}, fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
	}
	digest := sha256.Sum256(publicBytes)
	keyID, err := core.ParseSigningKeyID(BugWriterKeyIDPrefix + hex.EncodeToString(digest[:]))
	if err != nil {
		return BugWriterKey{}, fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
	}
	return BugWriterKey{KeyID: keyID, PublicKey: publicKey}, nil
}

func (k BugWriterKey) Validate() error {
	derived, err := NewBugWriterKey(k.PublicKey)
	if err != nil || derived.KeyID != k.KeyID {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	return nil
}

func (k BugWriterKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	type wire struct {
		KeyID     core.SigningKeyID        `json:"key_id"`
		PublicKey core.Ed25519PublicKeyHex `json:"public_key"`
	}
	return json.Marshal(wire(k))
}
