package peachfuzz

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// MachineEvidenceIdentity is one installation's complete private persistence
// contract. Its machine namespace is derived from, and inseparable from, the
// signing keypair that seals immutable run evidence.
type MachineEvidenceIdentity struct {
	Machine    MachineID                `json:"machine"`
	SigningKey core.GeneratedSigningKey `json:"signing_key"`
}

func NewMachineEvidenceIdentity(signingKey core.GeneratedSigningKey) (MachineEvidenceIdentity, error) {
	if err := signingKey.Validate(); err != nil {
		return MachineEvidenceIdentity{}, machineEvidenceIdentityError(err)
	}
	machine, err := MachineIDFromSigningPublicKey(signingKey.PublicKeyHex)
	if err != nil {
		return MachineEvidenceIdentity{}, machineEvidenceIdentityError(err)
	}
	identity := MachineEvidenceIdentity{Machine: machine, SigningKey: signingKey}
	return identity, identity.Validate()
}

func (i MachineEvidenceIdentity) Validate() error {
	if err := i.Machine.Validate(); err != nil {
		return machineEvidenceIdentityError(err)
	}
	if err := i.SigningKey.Validate(); err != nil {
		return machineEvidenceIdentityError(err)
	}
	machine, err := MachineIDFromSigningPublicKey(i.SigningKey.PublicKeyHex)
	if err != nil || machine != i.Machine {
		return machineEvidenceIdentityError(errors.Join(ErrContract, err))
	}
	return nil
}

func (i MachineEvidenceIdentity) PrivateSigningKey() (core.Ed25519SigningKey, error) {
	if err := i.Validate(); err != nil {
		return core.Ed25519SigningKey{}, err
	}
	key, err := core.ParseEd25519SigningKeyBase64(i.SigningKey.PrivateKeyBase64)
	if err != nil {
		return core.Ed25519SigningKey{}, machineEvidenceIdentityError(err)
	}
	return key, nil
}

func machineEvidenceIdentityError(err error) error {
	return fmt.Errorf(ErrFmtMachineEvidenceIdentity, errors.Join(ErrContract, err))
}
