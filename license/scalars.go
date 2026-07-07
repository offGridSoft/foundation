package license

import (
	"fmt"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

const (
	DeveloperKeyPrefix       = "OGS-DEV-"
	DeveloperKeyMinRunes     = 20
	DeveloperKeyPreviewRunes = 12
)

type DeveloperKey struct {
	value string
}

func ParseDeveloperKey(value string) (DeveloperKey, error) {
	if !strings.HasPrefix(value, DeveloperKeyPrefix) || len([]rune(value)) < DeveloperKeyMinRunes {
		return DeveloperKey{}, fmt.Errorf(ErrFmtDeveloperKey, core.ErrLicenseContract)
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return DeveloperKey{}, fmt.Errorf(ErrFmtDeveloperKey, core.ErrLicenseContract)
	}
	return DeveloperKey{value: value}, nil
}

func (k DeveloperKey) String() string {
	return k.value
}

func (k DeveloperKey) IsZero() bool {
	return k.value == ""
}

func (k DeveloperKey) Validate() error {
	_, err := ParseDeveloperKey(k.value)
	return err
}

func (k DeveloperKey) Preview() string {
	if err := k.Validate(); err != nil {
		return ""
	}
	runes := []rune(k.value)
	return string(runes[:DeveloperKeyPreviewRunes]) + "..."
}

func (k DeveloperKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.value)
}

func (k *DeveloperKey) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeveloperKey, core.ErrLicenseContract)
	}
	parsed, err := ParseDeveloperKey(value)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

type DeveloperKeyID struct {
	value string
}

func ParseDeveloperKeyID(value string) (DeveloperKeyID, error) {
	if strings.TrimSpace(value) == "" {
		return DeveloperKeyID{}, fmt.Errorf(ErrFmtDeveloperKeyID, core.ErrLicenseContract)
	}
	return DeveloperKeyID{value: value}, nil
}

func (id DeveloperKeyID) String() string {
	return id.value
}

func (id DeveloperKeyID) Validate() error {
	_, err := ParseDeveloperKeyID(id.value)
	return err
}

func (id DeveloperKeyID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *DeveloperKeyID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeveloperKeyID, core.ErrLicenseContract)
	}
	parsed, err := ParseDeveloperKeyID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type APICallKey struct {
	value string
}

// APICallKey is a public bot-filter value shipped in released binaries. It is
// not authentication and grants no entitlement; it lets the edge reject traffic
// that is not even presenting itself as an Offgrid tool before JSON decode or
// store work.
func ParseAPICallKey(value string) (APICallKey, error) {
	if strings.TrimSpace(value) == "" {
		return APICallKey{}, fmt.Errorf(ErrFmtAPICallKey, core.ErrLicenseContract)
	}
	return APICallKey{value: value}, nil
}

func (k APICallKey) String() string {
	return k.value
}

func (k APICallKey) IsZero() bool {
	return k.value == ""
}

func (k APICallKey) Validate() error {
	_, err := ParseAPICallKey(k.value)
	return err
}

func (k APICallKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.value)
}

func (k *APICallKey) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtAPICallKey, core.ErrLicenseContract)
	}
	parsed, err := ParseAPICallKey(value)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}
