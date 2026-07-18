package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	GarbleCustodySeedBytes     = 64
	GarbleCustodySeedTextBytes = 88
	ErrFmtGarbleCustodySeed    = "core.GarbleCustodySeed: %w"
)

// GarbleCustodySeed is the long-lived, product-scoped root from which release
// packages derive short-lived tool material. Core owns its persistence shape
// so key generation, release orchestration, and applications share one type.
type GarbleCustodySeed struct {
	value [GarbleCustodySeedBytes]byte
}

func ParseGarbleCustodySeed(value string) (GarbleCustodySeed, error) {
	if len(value) != GarbleCustodySeedTextBytes {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtGarbleCustodySeed, ErrFoundationContract)
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) != GarbleCustodySeedBytes || base64.StdEncoding.EncodeToString(raw) != value || allZeroGarbleCustodySeed(raw) {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtGarbleCustodySeed, ErrFoundationContract)
	}
	return NewGarbleCustodySeed(raw)
}

func NewGarbleCustodySeed(value []byte) (GarbleCustodySeed, error) {
	if len(value) != GarbleCustodySeedBytes || allZeroGarbleCustodySeed(value) {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtGarbleCustodySeed, ErrFoundationContract)
	}
	var seed GarbleCustodySeed
	copy(seed.value[:], value)
	return seed, nil
}

func allZeroGarbleCustodySeed(value []byte) bool {
	for _, part := range value {
		if part != 0 {
			return false
		}
	}
	return true
}

func (s GarbleCustodySeed) Validate() error {
	_, err := ParseGarbleCustodySeed(base64.StdEncoding.EncodeToString(s.value[:]))
	return err
}

func (s GarbleCustodySeed) Bytes() []byte {
	out := make([]byte, len(s.value))
	copy(out, s.value[:])
	return out
}

// MarshalText emits the one canonical Secret Manager representation accepted
// by ParseGarbleCustodySeed.
func (s GarbleCustodySeed) MarshalText() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(s.value[:])), nil
}

func (s GarbleCustodySeed) SHA256() SHA256Hex {
	return NewSHA256Hex(sha256.Sum256(s.value[:]))
}

func (s GarbleCustodySeed) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(s.value[:]))
}

//validate:unmarshal_ignore reason="ParseGarbleCustodySeed validates a temporary before assignment so rejected input cannot mutate the receiver."
func (s *GarbleCustodySeed) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGarbleCustodySeed, ErrFoundationContract)
	}
	parsed, err := ParseGarbleCustodySeed(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

var _ Validatable = GarbleCustodySeed{}
