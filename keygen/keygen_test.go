package keygen

import (
	"bytes"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestGenerateEd25519SigningKeyProducesDistinctValidatedContracts(t *testing.T) {
	t.Parallel()

	first, err := GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() first error = %v", err)
	}
	second, err := GenerateEd25519SigningKey()
	if err != nil {
		t.Fatalf("GenerateEd25519SigningKey() second error = %v", err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("first.Validate() error = %v", err)
	}
	if first.PrivateKeyBase64 == second.PrivateKeyBase64 {
		t.Fatal("private key equality = true, want false for independent CSPRNG mints")
	}
	if first.PublicKeyHex == second.PublicKeyHex {
		t.Fatal("public key equality = true, want false for independent CSPRNG mints")
	}
}

func TestGenerateGarbleCustodySeedProducesDistinctValidatedContracts(t *testing.T) {
	t.Parallel()

	first, err := GenerateGarbleCustodySeed()
	if err != nil {
		t.Fatalf("GenerateGarbleCustodySeed() first error = %v", err)
	}
	second, err := GenerateGarbleCustodySeed()
	if err != nil {
		t.Fatalf("GenerateGarbleCustodySeed() second error = %v", err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("first.Validate() error = %v", err)
	}
	if bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("custody seed equality = true, want false for independent CSPRNG mints")
	}
	encoded, err := first.MarshalText()
	if err != nil {
		t.Fatalf("first.MarshalText() error = %v", err)
	}
	if _, err := core.ParseGarbleCustodySeed(string(encoded)); err != nil {
		t.Fatalf("ParseGarbleCustodySeed(MarshalText()) error = %v", err)
	}
}

func TestGenerateSecretHexHostileBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		byteLen int
		wantErr bool
	}{
		{name: "minimum", byteLen: core.SecretByteMinimum},
		{name: "standard", byteLen: core.SecretByteStandard},
		{name: "maximum", byteLen: core.SecretByteMaximum},
		{name: "below minimum", byteLen: core.SecretByteMinimum - 1, wantErr: true},
		{name: "above maximum", byteLen: core.SecretByteMaximum + 1, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := GenerateSecretHex(tc.byteLen)
			if tc.wantErr {
				if !errors.Is(err, core.ErrKeygenContract) {
					t.Fatalf("GenerateSecretHex(%d) error = %v, want errors.Is(..., %v)", tc.byteLen, err, core.ErrKeygenContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateSecretHex(%d) error = %v, want nil", tc.byteLen, err)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("GenerateSecretHex(%d).Validate() error = %v, want nil", tc.byteLen, err)
			}
		})
	}
}
