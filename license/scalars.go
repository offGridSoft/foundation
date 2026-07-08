package license

import (
	"fmt"
	"strings"
	"unicode"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

const (
	DeveloperKeyPrefix       = "OGS-DEV-"
	DeveloperKeyMinRunes     = 20
	DeveloperKeyMaxRunes     = 128
	DeveloperKeyPreviewRunes = 12
	DeviceLabelMaxRunes      = 80
)

type DeveloperKey struct {
	value string
}

func ParseDeveloperKey(value string) (DeveloperKey, error) {
	if err := core.ValidateOpaqueToken(value, DeveloperKeyMaxRunes); err != nil {
		return DeveloperKey{}, fmt.Errorf(ErrFmtDeveloperKey, core.ErrLicenseContract)
	}
	if !strings.HasPrefix(value, DeveloperKeyPrefix) || len([]rune(value)) < DeveloperKeyMinRunes {
		return DeveloperKey{}, fmt.Errorf(ErrFmtDeveloperKey, core.ErrLicenseContract)
	}
	if strings.ContainsFunc(value, unicode.IsSpace) {
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
	if err := core.ValidateOpaqueToken(value, core.OpaqueTokenDefaultMaxRunes); err != nil {
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

type DeviceLabel struct {
	value string
}

func ParseDeviceLabel(value string) (DeviceLabel, error) {
	trimmed := strings.TrimSpace(value)
	if err := core.ValidateOpaqueToken(trimmed, DeviceLabelMaxRunes); err != nil {
		return DeviceLabel{}, fmt.Errorf(ErrFmtDeviceLabel, core.ErrLicenseContract)
	}
	return DeviceLabel{value: trimmed}, nil
}

func (l DeviceLabel) String() string {
	return l.value
}

func (l DeviceLabel) Validate() error {
	_, err := ParseDeviceLabel(l.value)
	return err
}

func (l DeviceLabel) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(l.value)
}

func (l *DeviceLabel) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeviceLabel, core.ErrLicenseContract)
	}
	parsed, err := ParseDeviceLabel(value)
	if err != nil {
		return err
	}
	*l = parsed
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
	if err := core.ValidateOpaqueToken(value, core.OpaqueTokenDefaultMaxRunes); err != nil {
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
