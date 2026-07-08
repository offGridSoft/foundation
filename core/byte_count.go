package core

import (
	"fmt"

	"encoding/json"
)

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
	return json.Marshal(c.value)
}

func (c *ByteCount) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtByteCount, ErrFoundationContract)
	}
	*c = NewByteCount(value)
	return c.Validate()
}
