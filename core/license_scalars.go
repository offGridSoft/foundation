package core

import (
	"fmt"
	"strings"

	"encoding/json"
)

const (
	DeviceFingerprintPrefixSHA256 = "sha256:"
	ErrFmtDeviceFingerprint       = "core.DeviceFingerprint: %w"
	ErrFmtLeaseID                 = "core.LeaseID: %w"
)

type DeviceFingerprint struct {
	value string
}

func ParseDeviceFingerprint(value string) (DeviceFingerprint, error) {
	if !strings.HasPrefix(value, DeviceFingerprintPrefixSHA256) {
		return DeviceFingerprint{}, fmt.Errorf(ErrFmtDeviceFingerprint, ErrFoundationContract)
	}
	digest := strings.TrimPrefix(value, DeviceFingerprintPrefixSHA256)
	if _, err := ParseSHA256Hex(digest); err != nil {
		return DeviceFingerprint{}, fmt.Errorf(ErrFmtDeviceFingerprint, err)
	}
	return DeviceFingerprint{value: value}, nil
}

func (f DeviceFingerprint) String() string {
	return f.value
}

func (f DeviceFingerprint) Validate() error {
	_, err := ParseDeviceFingerprint(f.value)
	return err
}

func (f DeviceFingerprint) MarshalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(f.value)
}

func (f *DeviceFingerprint) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDeviceFingerprint, ErrFoundationContract)
	}
	parsed, err := ParseDeviceFingerprint(value)
	if err != nil {
		return err
	}
	*f = parsed
	return f.Validate()
}

type LeaseID struct {
	value string
}

func ParseLeaseID(value string) (LeaseID, error) {
	if err := ValidateOpaqueToken(value, OpaqueTokenDefaultMaxRunes); err != nil {
		return LeaseID{}, fmt.Errorf(ErrFmtLeaseID, ErrFoundationContract)
	}
	return LeaseID{value: value}, nil
}

func (id LeaseID) String() string {
	return id.value
}

func (id LeaseID) IsZero() bool {
	return id.value == ""
}

func (id LeaseID) Validate() error {
	_, err := ParseLeaseID(id.value)
	return err
}

func (id LeaseID) MarshalJSON() ([]byte, error) {
	if !id.IsZero() {
		if err := id.Validate(); err != nil {
			return nil, err
		}
	}
	return json.Marshal(id.value)
}

//validate:unmarshal_ignore reason="LeaseID is nullable on first check-in; owning structs validate requiredness."
func (id *LeaseID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtLeaseID, ErrFoundationContract)
	}
	if value == "" {
		*id = LeaseID{}
		return nil
	}
	parsed, err := ParseLeaseID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return id.Validate()
}
