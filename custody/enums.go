package custody

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type RetentionClass uint8

const (
	retentionClassInvalid RetentionClass = iota
	RetentionClassConditional
	RetentionClassPrepaid
)

func retentionClassNames() [RetentionClassPrepaid + 1]string {
	return [...]string{
		RetentionClassConditional: "conditional",
		RetentionClassPrepaid:     "prepaid",
	}
}

func (c RetentionClass) String() string {
	if c.IsValid() {
		return retentionClassNames()[c]
	}
	return ""
}

func (c RetentionClass) IsValid() bool {
	return c > retentionClassInvalid && int(c) < len(retentionClassNames()) && retentionClassNames()[c] != ""
}

func (c RetentionClass) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	return nil
}

func (c RetentionClass) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func ParseRetentionClass(token string) (RetentionClass, error) {
	for class := RetentionClassConditional; int(class) < len(retentionClassNames()); class++ {
		if retentionClassNames()[class] == token {
			return class, nil
		}
	}
	return retentionClassInvalid, fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
}

func (c *RetentionClass) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	parsed, err := ParseRetentionClass(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
