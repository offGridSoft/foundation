package release

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestValidatePublicReleaseArtifactsHostileTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mutate func(*testing.T, []Artifact) []Artifact
		name   string
		accept bool
	}{
		{name: "exact four binaries fast gate final certificate and two legal documents accepted", accept: true},
		{name: "missing artifact rejected", mutate: func(_ *testing.T, items []Artifact) []Artifact { return items[:len(items)-1] }},
		{name: "extra artifact rejected", mutate: func(_ *testing.T, items []Artifact) []Artifact { return append(items, items[0]) }},
		{name: "duplicate platform rejects substituted binary", mutate: func(_ *testing.T, items []Artifact) []Artifact { return duplicatePublicBinaryPlatform(items) }},
		{name: "unsupported kind rejected", mutate: func(_ *testing.T, items []Artifact) []Artifact {
			items[0].Kind = KindManifest
			items[0].Platform = core.PlatformUnknown
			return items
		}},
		{name: "unknown gate evidence name rejected", mutate: replacePublicArtifactName(KindGateEvidence, "unknown_gate.json")},
		{name: "binary named as gate cannot substitute gate", mutate: replacePublicArtifactKind(KindGateEvidence, KindProductBinary)},
		{name: "gate named as legal cannot substitute legal", mutate: replacePublicArtifactKind(KindLegalDocument, KindGateEvidence)},
		{name: "unknown legal document name rejected", mutate: replacePublicArtifactName(KindLegalDocument, "COPYING")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			artifacts := validPublicReleaseArtifacts(t)
			if tc.mutate != nil {
				artifacts = tc.mutate(t, artifacts)
			}
			err := ValidatePublicReleaseArtifacts(artifacts)
			if tc.accept && err != nil {
				t.Fatalf("ValidatePublicReleaseArtifacts() error = %v, want nil", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ValidatePublicReleaseArtifacts() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func validPublicReleaseArtifacts(t *testing.T) []Artifact {
	t.Helper()
	artifacts := make([]Artifact, 0, len(BuildPlatforms())+4)
	for _, platform := range BuildPlatforms() {
		artifacts = append(artifacts, validPublicArtifact(t, "bug_"+platform.String(), KindProductBinary, platform))
	}
	artifacts = append(artifacts,
		validPublicArtifact(t, FastGateEvidenceFileName, KindGateEvidence, core.PlatformUnknown),
		validPublicArtifact(t, FinalCertificateFileName, KindGateEvidence, core.PlatformUnknown),
		validPublicArtifact(t, LicenseFileName, KindLegalDocument, core.PlatformUnknown),
		validPublicArtifact(t, ThirdPartyNoticesName, KindLegalDocument, core.PlatformUnknown),
	)
	return artifacts
}

func TestValidatePublicReleaseManifestRequiresCertificateBinding(t *testing.T) {
	t.Parallel()
	base := validManifest(t)
	artifacts := validPublicReleaseArtifacts(t)
	for index := range artifacts {
		if artifacts[index].Name.String() == FinalCertificateFileName {
			artifacts[index].SHA256 = base.Evidence.FinalCertificate.SHA256
		}
	}
	manifest, err := BuildManifest(ManifestInput{
		Product: base.Product, Version: base.Version, ReleaseID: base.ReleaseID,
		Date: base.Date, Commit: base.Commit, Artifacts: artifacts,
		Evidence: base.Evidence, CreatedAt: base.CreatedAt,
	})
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}

	cases := []struct {
		mutate func(*Manifest)
		name   string
	}{
		{name: "certified manifest accepted"},
		{name: "candidate evidence rejected", mutate: func(value *Manifest) {
			value.Evidence.FinalCertificate = CandidateFinalCertificateEvidence()
		}},
		{name: "certificate commit drift rejected", mutate: func(value *Manifest) {
			value.Evidence.FinalCertificate.SubjectCommit = mustOtherCommit(t)
		}},
		{name: "certificate artifact digest drift rejected", mutate: func(value *Manifest) {
			for index := range value.Artifacts {
				if value.Artifacts[index].Name.String() == FinalCertificateFileName {
					value.Artifacts[index].SHA256 = mustSHA256(t, "e")
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			candidate := manifest
			candidate.Artifacts = append([]Artifact(nil), manifest.Artifacts...)
			if tc.mutate != nil {
				tc.mutate(&candidate)
			}
			err := ValidatePublicReleaseManifest(candidate)
			if tc.mutate == nil && err != nil {
				t.Fatalf("ValidatePublicReleaseManifest() error = %v", err)
			}
			if tc.mutate != nil && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ValidatePublicReleaseManifest() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func validPublicArtifact(t *testing.T, rawName string, kind Kind, platform core.Platform) Artifact {
	t.Helper()
	name, err := ParseArtifactName(rawName)
	if err != nil {
		t.Fatalf("ParseArtifactName(%q) error = %v", rawName, err)
	}
	return Artifact{
		Name: name, SHA256: mustSHA256(t, "a"), Size: core.NewByteCount(1), Kind: kind, Platform: platform,
	}
}

func duplicatePublicBinaryPlatform(items []Artifact) []Artifact {
	items[1].Platform = items[0].Platform
	return items
}

func replacePublicArtifactName(kind Kind, rawName string) func(*testing.T, []Artifact) []Artifact {
	return func(t *testing.T, items []Artifact) []Artifact {
		t.Helper()
		name, err := ParseArtifactName(rawName)
		if err != nil {
			t.Fatalf("ParseArtifactName(%q) error = %v", rawName, err)
		}
		for index := range items {
			if items[index].Kind == kind {
				items[index].Name = name
				return items
			}
		}
		return items
	}
}

func replacePublicArtifactKind(from, to Kind) func(*testing.T, []Artifact) []Artifact {
	return func(_ *testing.T, items []Artifact) []Artifact {
		for index := range items {
			if items[index].Kind == from {
				items[index].Kind = to
				if to.RequiresPlatform() {
					items[index].Platform = core.PlatformDarwinARM64
				}
				return items
			}
		}
		return items
	}
}
