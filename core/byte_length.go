package core

import (
	"fmt"
	"math"
)

const ErrFmtByteLength = "core.ByteLength: %w"

// ByteLength is an exact non-negative byte extent. Unlike ByteCount, whose
// zero value means a missing positive allocation, a zero ByteLength is a
// valid representation of an empty file, object, or wire body.
type ByteLength struct {
	value uint64
}

func NewByteLength(value uint64) ByteLength {
	return ByteLength{value: value}
}

func (l ByteLength) Uint64() uint64 {
	return l.value
}

func (l ByteLength) Validate() error {
	return nil
}

// Int64 returns the checked signed representation required by filesystem and
// streaming APIs.
func (l ByteLength) Int64() (int64, error) {
	if l.value > math.MaxInt64 {
		return 0, fmt.Errorf(ErrFmtByteLength, ErrFoundationContract)
	}
	return int64(l.value), nil
}

func (l ByteLength) MarshalJSON() ([]byte, error) {
	return appendUint64JSON(l.value), nil
}

//validate:unmarshal_ignore reason="ByteLength parses into a temporary before assignment so rejected input cannot mutate the receiver."
func (l *ByteLength) UnmarshalJSON(data []byte) error {
	value, err := parseStrictUint64JSON(data)
	if err != nil {
		return fmt.Errorf(ErrFmtByteLength, ErrFoundationContract)
	}
	l.value = value
	return nil
}
