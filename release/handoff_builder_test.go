package release

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBuildUploadReceiptBindsManifestWithoutAliasing(t *testing.T) {
	t.Parallel()
	manifest := validManifest(t)
	objects := append([]UploadedArtifact(nil), validUploadReceipt(t).Objects...)
	receipt, err := BuildUploadReceipt(UploadReceiptInput{
		Manifest: manifest, Objects: objects, AttemptID: validUploadAttemptID(t),
		UploadedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	})
	if err != nil {
		t.Fatalf("BuildUploadReceipt() error = %v", err)
	}
	canonical, err := manifest.Canonical(nil)
	if err != nil {
		t.Fatalf("Manifest.Canonical() error = %v", err)
	}
	wantHash := core.NewSHA256Hex(sha256.Sum256(canonical))
	if receipt.ManifestSHA256 != wantHash || receipt.Commit != manifest.Commit {
		t.Fatalf("BuildUploadReceipt() identity = %+v", receipt)
	}
	driftedManifest := manifest
	driftedManifest.Artifacts = append([]Artifact(nil), manifest.Artifacts...)
	driftedManifest.Artifacts[0].SHA256 = mustSHA256(t, "d")
	if err := receipt.VerifyManifest(driftedManifest); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("UploadReceipt.VerifyManifest(drift) error = %v, want %v", err, core.ErrReleaseContract)
	}
	objects[0] = UploadedArtifact{}
	if err := receipt.Objects[0].Validate(); err != nil {
		t.Fatalf("BuildUploadReceipt() retained caller slice alias: %v", err)
	}
}

func TestBuildUploadReceiptRejectsManifestObjectDriftTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*UploadReceiptInput)
		name   string
	}{
		{name: "missing object", mutate: func(i *UploadReceiptInput) { i.Objects = nil }},
		{name: "artifact name drift", mutate: func(i *UploadReceiptInput) { i.Objects[0].Artifact = mustArtifactName(t, "other") }},
		{name: "artifact hash drift", mutate: func(i *UploadReceiptInput) { i.Objects[0].SHA256 = mustSHA256(t, "d") }},
		{name: "artifact size drift", mutate: func(i *UploadReceiptInput) { i.Objects[0].Size = core.NewByteCount(13) }},
		{name: "missing upload time", mutate: func(i *UploadReceiptInput) { i.UploadedAt = core.UnixNanoTime{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := UploadReceiptInput{
				Manifest: validManifest(t), Objects: append([]UploadedArtifact(nil), validUploadReceipt(t).Objects...),
				AttemptID:  validUploadAttemptID(t),
				UploadedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
			}
			tc.mutate(&input)
			if err := input.Validate(); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UploadReceiptInput.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
			if _, err := BuildUploadReceipt(input); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("BuildUploadReceipt() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestReleaseHandoffSupportsCompleteGCSAndS3Mirrors(t *testing.T) {
	t.Parallel()
	manifest := validManifest(t)
	attemptID := validUploadAttemptID(t)
	gcsObject := validUploadedArtifactFor(t, manifest, core.StorageProviderGCS, attemptID)
	s3Object := validUploadedArtifactFor(t, manifest, core.StorageProviderS3, attemptID)
	receipt, err := BuildUploadReceipt(UploadReceiptInput{
		Manifest: manifest, Objects: []UploadedArtifact{s3Object, gcsObject},
		AttemptID:  attemptID,
		UploadedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	})
	if err != nil {
		t.Fatalf("BuildUploadReceipt(two providers) error = %v", err)
	}
	if receipt.ObjectCount != 2 || receipt.Objects[0].Provider != core.StorageProviderGCS || receipt.Objects[1].Provider != core.StorageProviderS3 {
		t.Fatalf("BuildUploadReceipt(two providers) = %+v", receipt.Objects)
	}

	gcsDownload := validDownloadIndex(t).Downloads[0]
	s3Download := gcsDownload
	s3Download.Provider = core.StorageProviderS3
	index, err := BuildDownloadIndex(DownloadIndexInput{
		Manifest: manifest, Downloads: []Download{s3Download, gcsDownload},
		GeneratedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	})
	if err != nil {
		t.Fatalf("BuildDownloadIndex(two providers) error = %v", err)
	}
	if index.DownloadCount != 2 || index.Downloads[0].Provider != core.StorageProviderGCS || index.Downloads[1].Provider != core.StorageProviderS3 {
		t.Fatalf("BuildDownloadIndex(two providers) = %+v", index.Downloads)
	}

	gcsTarget := validUploadTargetFor(t, manifest, core.StorageProviderGCS, attemptID)
	s3Target := validUploadTargetFor(t, manifest, core.StorageProviderS3, attemptID)
	plan, err := BuildDeployPlan(DeployPlanInput{
		Manifest: manifest, Layout: validWitnessReleaseRootLayout(t), Targets: []UploadTarget{s3Target, gcsTarget},
		AttemptID: attemptID,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan(two providers) error = %v", err)
	}
	if plan.TargetCount != 2 || plan.Targets[0].Provider != core.StorageProviderGCS || plan.Targets[1].Provider != core.StorageProviderS3 {
		t.Fatalf("BuildDeployPlan(two providers) = %+v", plan.Targets)
	}
}

func TestDeployPlanRejectsIncompleteOrUnauthorisedTargetsTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*DeployPlanInput)
		name   string
	}{
		{name: "missing signed upload url", mutate: func(i *DeployPlanInput) { i.Targets[0].URL = core.SignedUploadURL{} }},
		{name: "missing expiry", mutate: func(i *DeployPlanInput) { i.Targets[0].ExpiresAt = core.UnixNanoTime{} }},
		{name: "duplicate header", mutate: func(i *DeployPlanInput) {
			i.Targets[0].Headers = append(i.Targets[0].Headers, i.Targets[0].Headers[0])
		}},
		{name: "artifact absent from manifest", mutate: func(i *DeployPlanInput) {
			i.Targets[0].Artifact = mustArtifactName(t, "other")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := DeployPlanInput{
				Manifest: validManifest(t), Layout: validWitnessReleaseRootLayout(t), Targets: []UploadTarget{validUploadTarget(t)},
				AttemptID: validUploadAttemptID(t),
			}
			tc.mutate(&input)
			if _, err := BuildDeployPlan(input); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("BuildDeployPlan() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestBuildDeployPlanDerivesBoundIdentityWithoutAliasing(t *testing.T) {
	t.Parallel()
	manifest := validManifest(t)
	targets := []UploadTarget{validUploadTarget(t)}
	input := DeployPlanInput{
		Manifest: manifest, Layout: validWitnessReleaseRootLayout(t), Targets: targets,
		AttemptID: validUploadAttemptID(t),
	}
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
	targets[0].Headers[0] = core.UploadHeader{}
	if err := plan.Targets[0].Headers[0].Validate(); err != nil {
		t.Fatalf("BuildDeployPlan() retained caller header alias: %v", err)
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
		Provider: core.StorageProviderGCS,
	}
	index, err := BuildDownloadIndex(DownloadIndexInput{
		Manifest: manifest, Downloads: []Download{secondDownload, firstDownload}, GeneratedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	})
	if err != nil {
		t.Fatalf("BuildDownloadIndex(two same-platform artifacts) error = %v", err)
	}
	if index.DownloadCount != 2 {
		t.Fatalf("BuildDownloadIndex().DownloadCount = %d, want 2", index.DownloadCount)
	}
	if err := index.VerifyManifest(manifest); err != nil {
		t.Fatalf("DownloadIndex.VerifyManifest() error = %v", err)
	}
	driftedManifest := manifest
	driftedManifest.Artifacts = append([]Artifact(nil), manifest.Artifacts...)
	driftedManifest.Artifacts[0].SHA256 = mustSHA256(t, "d")
	if err := index.VerifyManifest(driftedManifest); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("DownloadIndex.VerifyManifest(drift) error = %v, want %v", err, core.ErrReleaseContract)
	}
	if index.Downloads[0].Artifact.String() >= index.Downloads[1].Artifact.String() {
		t.Fatalf("BuildDownloadIndex() order = %q, %q", index.Downloads[0].Artifact, index.Downloads[1].Artifact)
	}
	unsorted := index
	unsorted.Downloads[0], unsorted.Downloads[1] = unsorted.Downloads[1], unsorted.Downloads[0]
	if err := unsorted.Validate(); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("DownloadIndex.Validate(unsorted) error = %v, want %v", err, core.ErrReleaseContract)
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
		Product: current.Product, Version: current.Version, ReleaseID: current.ReleaseID, Date: date, Commit: current.Commit,
	})
	if err != nil {
		t.Fatalf("BuildReleaseRootLayout() error = %v", err)
	}
	return layout
}
