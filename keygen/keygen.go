// Package keygen owns CSPRNG-backed generation of Foundation key contracts.
// Pure key shapes and validators remain in core; entropy enters only here at
// the outer generation boundary.
package keygen

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/release"
)

// GenerateEd25519SigningKey mints a fresh signing key from the system CSPRNG
// and returns Foundation's validated persistence contract.
func GenerateEd25519SigningKey() (core.GeneratedSigningKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return core.GeneratedSigningKey{}, fmt.Errorf(core.ErrFmtGeneratedSigningKey, errors.Join(core.ErrKeygenEntropy, err))
	}
	defer clear(private)
	publicHex, err := core.NewEd25519PublicKeyHex(public)
	if err != nil {
		return core.GeneratedSigningKey{}, fmt.Errorf(core.ErrFmtGeneratedSigningKey, errors.Join(core.ErrKeygenEntropy, err))
	}
	out := core.GeneratedSigningKey{
		PrivateKeyBase64: base64.StdEncoding.EncodeToString(private),
		PublicKeyHex:     publicHex,
	}
	if err := out.Validate(); err != nil {
		return core.GeneratedSigningKey{}, err
	}
	return out, nil
}

// GenerateGarbleCustodySeed mints Foundation's validated long-lived custody
// seed. Release-specific Garble seeds are derived from this value and a typed
// release identity; callers persist only the custody seed in their secret
// manager and never invent an encoding outside Foundation.
func GenerateGarbleCustodySeed() (release.GarbleCustodySeed, error) {
	raw := make([]byte, release.GarbleCustodySeedBytes)
	defer clear(raw)
	if _, err := rand.Read(raw); err != nil {
		return release.GarbleCustodySeed{}, fmt.Errorf(core.ErrFmtGeneratedSecret, errors.Join(core.ErrKeygenEntropy, err))
	}
	seed, err := release.NewGarbleCustodySeed(raw)
	if err != nil {
		return release.GarbleCustodySeed{}, fmt.Errorf(core.ErrFmtGeneratedSecret, errors.Join(core.ErrKeygenContract, err))
	}
	return seed, nil
}

// GenerateSecretHex mints a bounded symmetric secret from the system CSPRNG
// and returns Foundation's validated lowercase-hex persistence contract.
func GenerateSecretHex(byteLen int) (core.GeneratedSecret, error) {
	if byteLen < core.SecretByteMinimum || byteLen > core.SecretByteMaximum {
		return core.GeneratedSecret{}, fmt.Errorf(core.ErrFmtGeneratedSecret, core.ErrKeygenContract)
	}
	raw := make([]byte, byteLen)
	defer clear(raw)
	if _, err := rand.Read(raw); err != nil {
		return core.GeneratedSecret{}, fmt.Errorf(core.ErrFmtGeneratedSecret, errors.Join(core.ErrKeygenEntropy, err))
	}
	out := core.GeneratedSecret{Hex: hex.EncodeToString(raw), ByteLen: byteLen}
	if err := out.Validate(); err != nil {
		return core.GeneratedSecret{}, err
	}
	return out, nil
}
