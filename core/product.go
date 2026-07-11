package core

import (
	"encoding/json"
	"fmt"
)

type ProductVersion struct {
	value string
}

// FoundationVersion2026 is this module's compiler-owned source version token
// for the 2026 release line.
const FoundationVersion2026 = "2026.0.0"

func ParseProductVersion(value string) (ProductVersion, error) {
	if err := ValidateOpaqueToken(value, OpaqueTokenDefaultMaxRunes); err != nil {
		return ProductVersion{}, fmt.Errorf(ErrFmtProductVersion, ErrFoundationContract)
	}
	return ProductVersion{value: value}, nil
}

func (v ProductVersion) String() string {
	return v.value
}

func (v ProductVersion) Validate() error {
	_, err := ParseProductVersion(v.value)
	return err
}

func (v ProductVersion) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v.value)
}

//validate:unmarshal_ignore reason="ParseProductVersion validates a temporary before assignment so rejected input cannot mutate the receiver."
func (v *ProductVersion) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtProductVersion, ErrFoundationContract)
	}
	parsed, err := ParseProductVersion(value)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}
