package release

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBuildManifestDerivesArtifactSetWithoutAliasingInput(t *testing.T) {
	t.Parallel()
	artifacts := []Artifact{
		validArtifactWithSize(t, "bug-linux-amd64", 12),
		validArtifactWithSize(t, "bug-darwin-arm64", 18),
	}
	input := ManifestInput{
		Product: core.ProductBug, Version: mustVersion(t), ReleaseID: mustReleaseID(t),
		Date: mustReleaseDate(t), Commit: mustCommit(t), CreatedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Artifacts: artifacts,
	}
	manifest, err := BuildManifest(input)
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	if manifest.ArtifactCount != uint32(len(artifacts)) {
		t.Fatalf("BuildManifest().ArtifactCount = %d, want %d", manifest.ArtifactCount, len(artifacts))
	}
	if manifest.TotalBytes.Uint64() != 30 {
		t.Fatalf("BuildManifest().TotalBytes = %d, want 30", manifest.TotalBytes.Uint64())
	}
	artifacts[0] = validArtifactWithSize(t, "mutated-after-build", 1)
	if manifest.Artifacts[0].Name.String() != "bug-linux-amd64" {
		t.Fatalf("BuildManifest() retained caller slice alias: %q", manifest.Artifacts[0].Name)
	}
}

func TestManifestInputRejectsHostileArtifactSets(t *testing.T) {
	t.Parallel()
	valid := ManifestInput{
		Product: core.ProductBug, Version: mustVersion(t), ReleaseID: mustReleaseID(t),
		Date: mustReleaseDate(t), Commit: mustCommit(t), CreatedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Artifacts: []Artifact{validArtifactWithSize(t, "bug-linux-amd64", 12)},
	}
	cases := []struct {
		mutate func(*ManifestInput)
		name   string
	}{
		{name: "missing product", mutate: func(i *ManifestInput) { i.Product = core.ProductUnknown }},
		{name: "missing version", mutate: func(i *ManifestInput) { i.Version = core.ProductVersion{} }},
		{name: "missing release id", mutate: func(i *ManifestInput) { i.ReleaseID = ReleaseID{} }},
		{name: "missing date", mutate: func(i *ManifestInput) { i.Date = ReleaseDate{} }},
		{name: "missing commit", mutate: func(i *ManifestInput) { i.Commit = core.BuildCommit{} }},
		{name: "missing created at", mutate: func(i *ManifestInput) { i.CreatedAt = core.UnixNanoTime{} }},
		{name: "empty artifacts", mutate: func(i *ManifestInput) { i.Artifacts = nil }},
		{name: "duplicate artifact", mutate: func(i *ManifestInput) { i.Artifacts = append(i.Artifacts, i.Artifacts[0]) }},
		{name: "zero artifact size", mutate: func(i *ManifestInput) { i.Artifacts[0].Size = core.ByteCount{} }},
		{name: "missing artifact hash", mutate: func(i *ManifestInput) { i.Artifacts[0].SHA256 = core.SHA256Hex{} }},
		{name: "missing artifact name", mutate: func(i *ManifestInput) { i.Artifacts[0].Name = ArtifactName{} }},
		{name: "wrong artifact platform", mutate: func(i *ManifestInput) { i.Artifacts[0].Platform = core.PlatformUnknown }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := valid
			input.Artifacts = append([]Artifact(nil), valid.Artifacts...)
			tc.mutate(&input)
			if err := input.Validate(); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ManifestInput.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}
