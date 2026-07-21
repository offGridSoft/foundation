package peachfuzz

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// SignedRunEvidence is the complete immutable fleet accounting atom. Unlike a
// generic signed body, it binds Machine to the signing public key so one
// producer cannot claim another producer's namespace.
type SignedRunEvidence struct {
	Body      RunEvidence              `json:"body"`
	KeyID     core.SigningKeyID        `json:"key_id"`
	Signature core.Ed25519SignatureHex `json:"signature"`
}

func NewSignedRunEvidence(signed core.Signed[RunEvidence]) (SignedRunEvidence, error) {
	value := SignedRunEvidence{Body: signed.Body, KeyID: signed.KeyID, Signature: signed.Signature}
	return value, value.Validate()
}

func (s SignedRunEvidence) Validate() error {
	signed := s.foundationSigned()
	if err := signed.Validate(); err != nil {
		return signedRunEvidenceError(err)
	}
	publicKey, machine, err := signedRunEvidenceIdentity(s.KeyID)
	if err != nil {
		return signedRunEvidenceError(err)
	}
	if s.Body.Machine != machine {
		return signedRunEvidenceError(ErrContract)
	}
	if err := publicKey.Validate(); err != nil {
		return signedRunEvidenceError(err)
	}
	return nil
}

func (s SignedRunEvidence) Verify() error {
	if err := s.Validate(); err != nil {
		return err
	}
	publicKey, _, err := signedRunEvidenceIdentity(s.KeyID)
	if err != nil {
		return signedRunEvidenceError(err)
	}
	keyring, err := core.NewPinnedAuthorityKeyring(publicKey)
	if err != nil {
		return signedRunEvidenceError(err)
	}
	if err := s.foundationSigned().Verify(keyring); err != nil {
		return signedRunEvidenceError(err)
	}
	return nil
}

// MachineIDFromSigningPublicKey is the sole machine-namespace derivation for
// signed Peachfuzz evidence. The 128-bit prefix is collision-resistant at the
// intended global fleet size and remains inside the existing MachineID shape.
func MachineIDFromSigningPublicKey(publicKey core.Ed25519PublicKeyHex) (MachineID, error) {
	if err := publicKey.Validate(); err != nil {
		return MachineID{}, signedRunEvidenceError(err)
	}
	return ParseMachineID(publicKey.String()[:MachineIDTextBytes])
}

func signedRunEvidenceIdentity(keyID core.SigningKeyID) (core.Ed25519PublicKeyHex, MachineID, error) {
	if err := keyID.Validate(); err != nil {
		return core.Ed25519PublicKeyHex{}, MachineID{}, err
	}
	publicKey, err := core.ParseEd25519PublicKeyHex(keyID.String())
	if err != nil {
		return core.Ed25519PublicKeyHex{}, MachineID{}, err
	}
	machine, err := MachineIDFromSigningPublicKey(publicKey)
	return publicKey, machine, err
}

func (s SignedRunEvidence) foundationSigned() core.Signed[RunEvidence] {
	return core.Signed[RunEvidence]{Body: s.Body, KeyID: s.KeyID, Signature: s.Signature}
}

func signedRunEvidenceError(err error) error {
	return fmt.Errorf(ErrFmtSignedRunEvidence, errors.Join(ErrContract, err))
}
