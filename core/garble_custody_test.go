package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

func TestGarbleCustodySeedOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	valid := validGarbleCustodySeedBytes()
	parsed, err := NewGarbleCustodySeed(valid)
	if err != nil {
		t.Fatalf("NewGarbleCustodySeed(valid) error = %v", err)
	}
	encoded, err := parsed.MarshalText()
	if err != nil {
		t.Fatalf("GarbleCustodySeed.MarshalText() error = %v", err)
	}
	if len(encoded) != GarbleCustodySeedTextBytes {
		t.Fatalf("GarbleCustodySeed.MarshalText() bytes = %d, want %d", len(encoded), GarbleCustodySeedTextBytes)
	}
	for _, test := range []struct {
		name  string
		value []byte
	}{
		{name: "nil"},
		{name: "short", value: make([]byte, GarbleCustodySeedBytes-1)},
		{name: "all zero", value: make([]byte, GarbleCustodySeedBytes)},
		{name: "long", value: make([]byte, GarbleCustodySeedBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewGarbleCustodySeed(test.value); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("NewGarbleCustodySeed() error = %v, want %v", err, ErrFoundationContract)
			}
		})
	}
}

func TestGarbleCustodySeedOGSWireAndCopyBoundaries(t *testing.T) {
	t.Parallel()
	seed, err := NewGarbleCustodySeed(validGarbleCustodySeedBytes())
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(seed)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	decoded, err := DecodeStrictJSON[GarbleCustodySeed](wire)
	if err != nil {
		t.Fatalf("DecodeStrictJSON() error = %v", err)
	}
	if decoded.SHA256() != seed.SHA256() {
		t.Fatalf("decoded digest = %q, want %q", decoded.SHA256().String(), seed.SHA256().String())
	}
	if _, err := ParseGarbleCustodySeed(base64.StdEncoding.EncodeToString(make([]byte, GarbleCustodySeedBytes))); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("ParseGarbleCustodySeed(all zero) error = %v, want %v", err, ErrFoundationContract)
	}
	wantDigest := seed.SHA256()
	copyBytes := seed.Bytes()
	copyBytes[0] ^= 0xff
	if seed.SHA256() != wantDigest {
		t.Fatalf("GarbleCustodySeed.SHA256() = %q after copy mutation, want %q", seed.SHA256().String(), wantDigest.String())
	}
}

func validGarbleCustodySeedBytes() []byte {
	value := make([]byte, GarbleCustodySeedBytes)
	for index := range value {
		value[index] = byte(index + 1)
	}
	return value
}
