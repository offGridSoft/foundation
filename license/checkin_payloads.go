package license

import (
	"fmt"
	"runtime"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

const (
	SchemaBugUsage = "bug-usage-v1"
)

type Platform uint8

const (
	platformInvalid Platform = iota
	PlatformDarwinAMD64
	PlatformDarwinARM64
	PlatformLinuxAMD64
	PlatformLinuxARM64
	PlatformWindowsAMD64
	PlatformWindowsARM64
)

var platformNames = [...]string{
	PlatformDarwinAMD64:  "darwin-amd64",
	PlatformDarwinARM64:  "darwin-arm64",
	PlatformLinuxAMD64:   "linux-amd64",
	PlatformLinuxARM64:   "linux-arm64",
	PlatformWindowsAMD64: "windows-amd64",
	PlatformWindowsARM64: "windows-arm64",
}

func (p Platform) String() string {
	if p.IsValid() {
		return platformNames[p]
	}
	return ""
}

func (p Platform) IsValid() bool {
	return p > platformInvalid && int(p) < len(platformNames) && platformNames[p] != ""
}

func (p Platform) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	return nil
}

func CurrentPlatform() (Platform, error) {
	switch runtime.GOOS + "-" + runtime.GOARCH {
	case PlatformDarwinAMD64.String():
		return PlatformDarwinAMD64, nil
	case PlatformDarwinARM64.String():
		return PlatformDarwinARM64, nil
	case PlatformLinuxAMD64.String():
		return PlatformLinuxAMD64, nil
	case PlatformLinuxARM64.String():
		return PlatformLinuxARM64, nil
	case PlatformWindowsAMD64.String():
		return PlatformWindowsAMD64, nil
	case PlatformWindowsARM64.String():
		return PlatformWindowsARM64, nil
	default:
		return platformInvalid, fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
}

func (p Platform) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *Platform) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	for platform := PlatformDarwinAMD64; int(platform) < len(platformNames); platform++ {
		if platformNames[platform] == token {
			*p = platform
			return nil
		}
	}
	return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
}

type BugUsage struct {
	WindowStart  core.UnixNanoTime `json:"window_start"`
	WindowEnd    core.UnixNanoTime `json:"window_end"`
	Schema       string            `json:"schema"`
	Green        uint32            `json:"green"`
	Start        uint32            `json:"start"`
	Show         uint32            `json:"show"`
	List         uint32            `json:"list"`
	Red          uint32            `json:"red"`
	Audit        uint32            `json:"audit"`
	Verify       uint32            `json:"verify"`
	Dupe         uint32            `json:"dupe"`
	Init         uint32            `json:"init"`
	Languages    uint32            `json:"languages"`
	InstallHooks uint32            `json:"install_hooks"`
	LedgerAdmin  uint32            `json:"ledger_admin"`
	LicenseAdmin uint32            `json:"license_admin"`
}

func (u BugUsage) IsZero() bool {
	return u == BugUsage{}
}

func (u BugUsage) Validate() error {
	if u.IsZero() {
		return nil
	}
	if u.Schema != SchemaBugUsage {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	if u.WindowStart.IsZero() || !u.WindowEnd.After(u.WindowStart) {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrLicenseContract)
	}
	return nil
}

type BugCheckIn struct {
	Schema            string              `json:"schema"`
	DeveloperKey      DeveloperKey        `json:"developer_key"`
	DeviceFingerprint DeviceFingerprint   `json:"device_fingerprint"`
	DeviceLabel       string              `json:"device_label"`
	BinaryVersion     core.ProductVersion `json:"binary_version"`
	BinarySHA256      core.SHA256Hex      `json:"binary_sha256"`
	LeaseID           LeaseID             `json:"lease_id"`
	Usage             BugUsage            `json:"usage"`
	Platform          Platform            `json:"platform"`
}

func (c BugCheckIn) Validate() error {
	if c.Schema != SchemaBugCheckIn {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := c.DeveloperKey.Validate(); err != nil {
		return err
	}
	if err := c.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if err := c.BinaryVersion.Validate(); err != nil {
		return err
	}
	if err := c.BinarySHA256.Validate(); err != nil {
		return err
	}
	if err := c.Platform.Validate(); err != nil {
		return err
	}
	return c.Usage.Validate()
}

type WitnessCheckIn struct {
	Schema            string              `json:"schema"`
	DeviceFingerprint DeviceFingerprint   `json:"device_fingerprint"`
	BinaryVersion     core.ProductVersion `json:"binary_version"`
	BinarySHA256      core.SHA256Hex      `json:"binary_sha256"`
	LeaseID           LeaseID             `json:"lease_id"`
	AccountToken      AccountToken        `json:"account_token"`
	Platform          Platform            `json:"platform"`
}

func (c WitnessCheckIn) Validate() error {
	if c.Schema != SchemaWitnessCheckIn {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := c.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if err := c.BinaryVersion.Validate(); err != nil {
		return err
	}
	if err := c.BinarySHA256.Validate(); err != nil {
		return err
	}
	if err := c.Platform.Validate(); err != nil {
		return err
	}
	return c.AccountToken.Validate()
}

type AccountToken struct {
	value string
}

func ParseAccountToken(value string) (AccountToken, error) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, " \t\r\n") {
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
