package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	FastGateEvidenceFileName  = "fast_gate.json"
	FinalGateEvidenceFileName = "final_gate.json"
)

type publicArtifactCoverage struct {
	darwinARM64  bool
	linuxAMD64   bool
	linuxARM64   bool
	windowsAMD64 bool
	fastGate     bool
	finalGate    bool
	license      bool
	notices      bool
}

func ValidatePublicReleaseArtifacts(artifacts []Artifact) error {
	wantCount := len(BuildPlatforms()) + 4
	if len(artifacts) != wantCount {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	var coverage publicArtifactCoverage
	for _, artifact := range artifacts {
		if err := artifact.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtManifest, err)
		}
		if err := coverage.accept(artifact); err != nil {
			return err
		}
	}
	if !coverage.complete() {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	return nil
}

func (c *publicArtifactCoverage) accept(artifact Artifact) error {
	switch artifact.Kind {
	case KindProductBinary:
		return c.acceptPlatform(artifact.Platform)
	case KindGateEvidence:
		return c.acceptGateEvidence(artifact.Name.String())
	case KindLegalDocument:
		return c.acceptLegalDocument(artifact.Name.String())
	default:
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
}

func (c *publicArtifactCoverage) acceptPlatform(platform core.Platform) error {
	var seen *bool
	switch platform {
	case core.PlatformDarwinARM64:
		seen = &c.darwinARM64
	case core.PlatformLinuxAMD64:
		seen = &c.linuxAMD64
	case core.PlatformLinuxARM64:
		seen = &c.linuxARM64
	case core.PlatformWindowsAMD64:
		seen = &c.windowsAMD64
	default:
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	return acceptOnce(seen)
}

func (c *publicArtifactCoverage) acceptGateEvidence(name string) error {
	switch name {
	case FastGateEvidenceFileName:
		return acceptOnce(&c.fastGate)
	case FinalGateEvidenceFileName:
		return acceptOnce(&c.finalGate)
	default:
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
}

func (c *publicArtifactCoverage) acceptLegalDocument(name string) error {
	switch name {
	case LicenseFileName:
		return acceptOnce(&c.license)
	case ThirdPartyNoticesName:
		return acceptOnce(&c.notices)
	default:
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
}

func acceptOnce(seen *bool) error {
	if *seen {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	*seen = true
	return nil
}

func (c publicArtifactCoverage) complete() bool {
	return c.darwinARM64 && c.linuxAMD64 && c.linuxARM64 && c.windowsAMD64 &&
		c.fastGate && c.finalGate && c.license && c.notices
}
