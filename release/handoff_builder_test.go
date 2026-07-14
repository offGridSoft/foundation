package release

import (
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBuildDeployPlanDerivesBoundIdentityWithoutAliasing(t *testing.T) {
	t.Parallel()
	manifest := validManifest(t)
	targets := []UploadTarget{validUploadTarget(t)}
	input := DeployPlanInput{Manifest: manifest, Layout: validWitnessReleaseRootLayout(t), Targets: targets}
	if err := input.Validate(); err != nil {
		t.Fatalf("DeployPlanInput.Validate() error = %v", err)
	}
	plan, err := BuildDeployPlan(input)
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}
	if plan.TargetCount != 1 || plan.Product != manifest.Product || plan.ReleaseID != manifest.ReleaseID {
		t.Fatalf("BuildDeployPlan() identity = %+v, manifest = %+v", plan, manifest)
	}
	targets[0] = UploadTarget{}
	if err := plan.Targets[0].Validate(); err != nil {
		t.Fatalf("BuildDeployPlan() retained caller target alias: %v", err)
	}
	manifest.Artifacts[0] = Artifact{}
	if err := plan.Manifest.Artifacts[0].Validate(); err != nil {
		t.Fatalf("BuildDeployPlan() retained caller manifest alias: %v", err)
	}
	plan.ManifestSHA256 = core.SHA256Hex{}
	if !errors.Is(plan.Validate(), core.ErrReleaseContract) {
		t.Fatalf("DeployPlan.Validate() accepted missing derived manifest hash")
	}
}

func TestBuildDownloadIndexSupportsMultipleArtifactsOnOnePlatform(t *testing.T) {
	t.Parallel()

	first := validArtifactWithSize(t, ToolsArchiveName, 12)
	second := validArtifactWithSize(t, "witness", 24)
	second.Kind = KindProductBinary
	second.SHA256 = mustSHA256(t, "c")
	manifest, err := BuildManifest(ManifestInput{
		Product: core.ProductWitness, Version: mustVersion(t), ReleaseID: mustReleaseID(t),
		Date: mustReleaseDate(t), Commit: mustCommit(t), CreatedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Artifacts: []Artifact{first, second},
	})
	if err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
	}
	firstDownload := validDownloadIndex(t).Downloads[0]
	secondURL, err := ParseDownloadURL(strings.Replace(validDownloadURL(t).String(), ToolsArchiveName, "witness", 1))
	if err != nil {
		t.Fatalf("ParseDownloadURL() error = %v", err)
	}
	secondDownload := Download{
		Artifact: second.Name, URL: secondURL, SHA256: second.SHA256, Size: second.Size, Platform: second.Platform,
	}
	index, err := BuildDownloadIndex(DownloadIndexInput{
		Manifest: manifest, Downloads: []Download{firstDownload, secondDownload}, GeneratedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	})
	if err != nil {
		t.Fatalf("BuildDownloadIndex(two same-platform artifacts) error = %v", err)
	}
	if index.DownloadCount != 2 {
		t.Fatalf("BuildDownloadIndex().DownloadCount = %d, want 2", index.DownloadCount)
	}
	duplicate := secondDownload
	duplicate.Artifact = firstDownload.Artifact
	if _, err := BuildDownloadIndex(DownloadIndexInput{
		Manifest: manifest, Downloads: []Download{firstDownload, duplicate}, GeneratedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	}); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("BuildDownloadIndex(duplicate artifact) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func TestBuildDownloadIndexDerivesCountWithoutAliasing(t *testing.T) {
	t.Parallel()
	valid := validDownloadIndex(t)
	manifest := validManifest(t)
	downloads := append([]Download(nil), valid.Downloads...)
	input := DownloadIndexInput{
		Manifest: manifest, Downloads: downloads, GeneratedAt: valid.GeneratedAt,
	}
	if err := input.Validate(); err != nil {
		t.Fatalf("DownloadIndexInput.Validate() error = %v", err)
	}
	index, err := BuildDownloadIndex(input)
	if err != nil {
		t.Fatalf("BuildDownloadIndex() error = %v", err)
	}
	if index.DownloadCount != 1 {
		t.Fatalf("BuildDownloadIndex().DownloadCount = %d, want 1", index.DownloadCount)
	}
	downloads[0] = Download{}
	if err := index.Downloads[0].Validate(); err != nil {
		t.Fatalf("BuildDownloadIndex() retained caller download alias: %v", err)
	}
	drifted := valid.Downloads
	drifted[0].SHA256 = mustSHA256(t, "d")
	if _, err := BuildDownloadIndex(DownloadIndexInput{Manifest: manifest, Downloads: drifted, GeneratedAt: valid.GeneratedAt}); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("BuildDownloadIndex(hash drift) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func mustOtherDatedLayout(t *testing.T, current ReleaseRootLayout) ReleaseRootLayout {
	t.Helper()
	date, err := ParseReleaseDate("2026-07-09")
	if err != nil {
		t.Fatalf("ParseReleaseDate() error = %v", err)
	}
	layout, err := BuildReleaseRootLayout(ReleaseRootInput{
		Product: current.Product, Version: current.Version, ReleaseID: current.ReleaseID, Date: date,
	})
	if err != nil {
		t.Fatalf("BuildReleaseRootLayout() error = %v", err)
	}
	return layout
}
