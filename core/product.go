package core

import (
	"fmt"

	json "github.com/goccy/go-json"
)

type ProductVersion struct {
	value string
}

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
	return v.Validate()
}
