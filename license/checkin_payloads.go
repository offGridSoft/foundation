package license

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

type BugUsage struct {
	WindowEnd    core.UnixNanoTime `json:"window_end"`
	WindowStart  core.UnixNanoTime `json:"window_start"`
	Green        uint32            `json:"green"`
	Verify       uint32            `json:"verify"`
	Start        uint32            `json:"start"`
	Show         uint32            `json:"show"`
	List         uint32            `json:"list"`
	Red          uint32            `json:"red"`
	LicenseAdmin uint32            `json:"license_admin"`
	Audit        uint32            `json:"audit"`
	Dupe         uint32            `json:"dupe"`
	Init         uint32            `json:"init"`
	Languages    uint32            `json:"languages"`
	InstallHooks uint32            `json:"install_hooks"`
	LedgerAdmin  uint32            `json:"ledger_admin"`
	Schema       core.SchemaID     `json:"schema"`
}

func (u BugUsage) IsZero() bool {
	return u == BugUsage{}
}

func (u BugUsage) Validate() error {
	if u.IsZero() {
		return nil
	}
	if u.Schema != core.SchemaBugUsage {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	if u.WindowStart.IsZero() || !u.WindowEnd.After(u.WindowStart) {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	return nil
}

func (u BugUsage) MarshalJSON() ([]byte, error) {
	if u.IsZero() {
		return []byte("{}"), nil
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	type bugUsageJSON BugUsage
	return json.Marshal(bugUsageJSON(u))
}

type BugCheckIn struct {
	DeveloperKey      DeveloperKey           `json:"developer_key"`
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	DeviceLabel       DeviceLabel            `json:"device_label"`
	BinaryVersion     core.ProductVersion    `json:"binary_version"`
	BinarySHA256      core.SHA256Hex         `json:"binary_sha256"`
	LeaseID           core.LeaseID           `json:"lease_id"`
	Usage             BugUsage               `json:"usage"`
	Schema            core.SchemaID          `json:"schema"`
	Platform          core.Platform          `json:"platform"`
}

func (c BugCheckIn) Validate() error {
	if c.Schema != core.SchemaBugCheckIn {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := c.DeveloperKey.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.DeviceFingerprint.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.DeviceLabel.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.BinaryVersion.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.BinarySHA256.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.LeaseID.ValidateOptional(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.Platform.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.Usage.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	return nil
}

type WitnessCheckIn struct {
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	BinaryVersion     core.ProductVersion    `json:"binary_version"`
	BinarySHA256      core.SHA256Hex         `json:"binary_sha256"`
	LeaseID           core.LeaseID           `json:"lease_id"`
	AccountToken      AccountToken           `json:"account_token"`
	Schema            core.SchemaID          `json:"schema"`
	Platform          core.Platform          `json:"platform"`
}

func (c WitnessCheckIn) Validate() error {
	if c.Schema != core.SchemaWitnessCheckIn {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := c.DeviceFingerprint.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.BinaryVersion.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.BinarySHA256.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.LeaseID.ValidateOptional(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.Platform.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	if err := c.AccountToken.Validate(); err != nil {
		return checkInPayloadError(err)
	}
	return nil
}

func checkInPayloadError(err error) error {
	return fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
}

type AccountToken struct {
	value string
}

func ParseAccountToken(value string) (AccountToken, error) {
	if err := core.ValidateOpaqueToken(value, core.OpaqueTokenDefaultMaxRunes); err != nil {
		return AccountToken{}, fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	if strings.ContainsFunc(value, unicode.IsSpace) {
		return AccountToken{}, fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	return AccountToken{value: value}, nil
}

func (t AccountToken) String() string {
	return t.value
}

func (t AccountToken) Validate() error {
	_, err := ParseAccountToken(t.value)
	return err
}

func (t AccountToken) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t.value)
}

func (t *AccountToken) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	parsed, err := ParseAccountToken(value)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
