package core

import (
	"encoding/json"
	"fmt"
)

type BuildCommit struct {
	value string
}

func ParseBuildCommit(value string) (BuildCommit, error) {
	if !validBuildCommitHex(value) {
		return BuildCommit{}, fmt.Errorf(ErrFmtBuildCommit, ErrFoundationContract)
	}
	return BuildCommit{value: value}, nil
}

func (c BuildCommit) String() string {
	return c.value
}

func (c BuildCommit) Validate() error {
	_, err := ParseBuildCommit(c.value)
	return err
}

func (c BuildCommit) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.value)
}

//validate:unmarshal_ignore reason="ParseBuildCommit validates a temporary before assignment so rejected input cannot mutate the receiver."
func (c *BuildCommit) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBuildCommit, ErrFoundationContract)
	}
	parsed, err := ParseBuildCommit(value)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func validBuildCommitHex(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	return IsLowerHex(value)
}
