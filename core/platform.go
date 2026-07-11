package core

import (
	"encoding/json"
	"fmt"
	"runtime"
)

type Platform uint8

const (
	PlatformUnknown Platform = iota
	PlatformDarwinAMD64
	PlatformDarwinARM64
	PlatformLinuxAMD64
	PlatformLinuxARM64
	PlatformWindowsAMD64
	PlatformWindowsARM64
)

const (
	PlatformTokenDarwinAMD64  = "darwin-amd64"
	PlatformTokenDarwinARM64  = "darwin-arm64"
	PlatformTokenLinuxAMD64   = "linux-amd64" // #nosec G101 -- GOOS/GOARCH platform token, not a credential.
	PlatformTokenLinuxARM64   = "linux-arm64"
	PlatformTokenWindowsAMD64 = "windows-amd64"
	PlatformTokenWindowsARM64 = "windows-arm64"
)

const (
	GOOSDarwin  = "darwin"
	GOOSLinux   = "linux" // #nosec G101 -- GOOS token, not a credential.
	GOOSWindows = "windows"
	GOARCHAMD64 = "amd64"
	GOARCHARM64 = "arm64"
)

func platformNames() [PlatformWindowsARM64 + 1]string {
	return [...]string{
		PlatformDarwinAMD64:  PlatformTokenDarwinAMD64,
		PlatformDarwinARM64:  PlatformTokenDarwinARM64,
		PlatformLinuxAMD64:   PlatformTokenLinuxAMD64,
		PlatformLinuxARM64:   PlatformTokenLinuxARM64,
		PlatformWindowsAMD64: PlatformTokenWindowsAMD64,
		PlatformWindowsARM64: PlatformTokenWindowsARM64,
	}
}

func (p Platform) String() string {
	if p.IsValid() {
		return platformNames()[p]
	}
	return ""
}

func (p Platform) IsValid() bool {
	return p > PlatformUnknown && int(p) < len(platformNames()) && platformNames()[p] != ""
}

func (p Platform) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtPlatform, ErrFoundationContract)
	}
	return nil
}

func ParsePlatform(token string) (Platform, error) {
	for platform := PlatformDarwinAMD64; int(platform) < len(platformNames()); platform++ {
		if platformNames()[platform] == token {
			return platform, nil
		}
	}
	return PlatformUnknown, fmt.Errorf(ErrFmtPlatform, ErrFoundationContract)
}

func CurrentPlatform() (Platform, error) {
	return ParsePlatform(runtime.GOOS + "-" + runtime.GOARCH)
}

func (p Platform) IsReleaseTarget() bool {
	switch p {
	case PlatformDarwinARM64, PlatformLinuxAMD64, PlatformLinuxARM64, PlatformWindowsAMD64:
		return true
	default:
		return false
	}
}

func (p Platform) GOOS() (string, error) {
	switch p {
	case PlatformDarwinAMD64, PlatformDarwinARM64:
		return GOOSDarwin, nil
	case PlatformLinuxAMD64, PlatformLinuxARM64:
		return GOOSLinux, nil
	case PlatformWindowsAMD64, PlatformWindowsARM64:
		return GOOSWindows, nil
	default:
		return "", fmt.Errorf(ErrFmtPlatform, ErrFoundationContract)
	}
}

func (p Platform) GOARCH() (string, error) {
	switch p {
	case PlatformDarwinAMD64, PlatformLinuxAMD64, PlatformWindowsAMD64:
		return GOARCHAMD64, nil
	case PlatformDarwinARM64, PlatformLinuxARM64, PlatformWindowsARM64:
		return GOARCHARM64, nil
	default:
		return "", fmt.Errorf(ErrFmtPlatform, ErrFoundationContract)
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
		return fmt.Errorf(ErrFmtPlatform, ErrFoundationContract)
	}
	parsed, err := ParsePlatform(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
