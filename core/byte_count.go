package core

import "fmt"

const ErrFmtByteCount = "core.ByteCount: %w"

type ByteCount struct {
	value uint64
}

func NewByteCount(value uint64) ByteCount {
	return ByteCount{value: value}
}

func (c ByteCount) Uint64() uint64 {
	return c.value
}

func (c ByteCount) Validate() error {
	if c.value == 0 {
		return fmt.Errorf(ErrFmtByteCount, ErrFoundationContract)
	}
	return nil
}

func (c ByteCount) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return appendUint64JSON(c.value), nil
}

//validate:unmarshal_ignore reason="ByteCount validates a temporary before assignment so rejected input cannot mutate the receiver."
func (c *ByteCount) UnmarshalJSON(data []byte) error {
	value, err := parseStrictUint64JSON(data)
	if err != nil {
		return fmt.Errorf(ErrFmtByteCount, ErrFoundationContract)
	}
	decoded := NewByteCount(value)
	if err := decoded.Validate(); err != nil {
		return err
	}
	*c = decoded
	return nil
}
