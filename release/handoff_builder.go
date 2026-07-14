package release

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.Validatable = DeployPlanInput{}
	_ core.Validatable = DownloadIndexInput{}
	_ core.Validatable = UploadReceiptInput{}
)

type UploadReceiptInput struct {
	AttemptID  UploadAttemptID
	Objects    []UploadedArtifact
	Manifest   Manifest
	UploadedAt core.UnixNanoTime
}

func (i UploadReceiptInput) Validate() error {
	receipt, err := uploadReceiptFromInput(i)
	if err != nil {
		return err
	}
	return receipt.VerifyManifest(i.Manifest)
}

func BuildUploadReceipt(input UploadReceiptInput) (UploadReceipt, error) {
	if err := input.Validate(); err != nil {
		return UploadReceipt{}, err
	}
	receipt, err := uploadReceiptFromInput(input)
	if err != nil {
		return UploadReceipt{}, err
	}
	return receipt, nil
}

func uploadReceiptFromInput(input UploadReceiptInput) (UploadReceipt, error) {
	canonical, err := input.Manifest.Canonical(nil)
	if err != nil {
		return UploadReceipt{}, wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	set, err := core.BuildArtifactSet(input.Objects)
	if err != nil {
		return UploadReceipt{}, wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := validateUploadedArtifactsMatchManifest(input.Manifest, set.Items); err != nil {
		return UploadReceipt{}, err
	}
	return UploadReceipt{
		Version:        input.Manifest.Version,
		ReleaseID:      input.Manifest.ReleaseID,
		Commit:         input.Manifest.Commit,
		ManifestSHA256: core.NewSHA256Hex(sha256.Sum256(canonical)),
		Objects:        set.Items,
		AttemptID:      input.AttemptID,
		UploadedAt:     input.UploadedAt,
		TotalBytes:     set.TotalBytes,
		ObjectCount:    set.Count,
		Schema:         core.SchemaReleaseUploadReceipt,
		Product:        input.Manifest.Product,
	}, nil
}

func validateUploadedArtifactsMatchManifest(manifest Manifest, objects []UploadedArtifact) error {
	if len(objects) < len(manifest.Artifacts) || len(objects)%len(manifest.Artifacts) != 0 {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	for _, object := range objects {
		if !manifestHasUploadedArtifact(manifest, object) {
			return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
		}
	}
	for _, object := range objects {
		if !providerUploadedEveryManifestArtifact(objects, object.Provider, manifest) {
			return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
		}
	}
	return nil
}

func manifestHasUploadedArtifact(manifest Manifest, object UploadedArtifact) bool {
	for _, artifact := range manifest.Artifacts {
		if object.Artifact == artifact.Name && object.SHA256 == artifact.SHA256 && object.Size == artifact.Size {
			return true
		}
	}
	return false
}

func providerUploadedEveryManifestArtifact(objects []UploadedArtifact, provider core.StorageProvider, manifest Manifest) bool {
	for _, artifact := range manifest.Artifacts {
		if !hasProviderUploadedArtifact(objects, provider, artifact.Name) {
			return false
		}
	}
	return true
}

func hasProviderUploadedArtifact(objects []UploadedArtifact, provider core.StorageProvider, artifact ArtifactName) bool {
	for _, object := range objects {
		if object.Provider == provider && object.Artifact == artifact {
			return true
		}
	}
	return false
}

type DeployPlanInput struct {
	AttemptID UploadAttemptID
	Layout    ReleaseRootLayout
	Targets   []UploadTarget
	Manifest  Manifest
}

func (i DeployPlanInput) Validate() error {
	plan, err := deployPlanFromInput(i)
	if err != nil {
		return err
	}
	return plan.Validate()
}

func BuildDeployPlan(input DeployPlanInput) (DeployPlan, error) {
	plan, err := deployPlanFromInput(input)
	if err != nil {
		return DeployPlan{}, err
	}
	if err := plan.Validate(); err != nil {
		return DeployPlan{}, err
	}
	return plan, nil
}

func deployPlanFromInput(input DeployPlanInput) (DeployPlan, error) {
	canonical, err := input.Manifest.Canonical(nil)
	if err != nil {
		return DeployPlan{}, wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	count, err := core.DeriveCollectionCount(len(input.Targets), 1, core.CollectionMaximumDefault)
	if err != nil {
		return DeployPlan{}, wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	manifest := input.Manifest
	manifest.Artifacts = append([]Artifact(nil), input.Manifest.Artifacts...)
	return DeployPlan{
		Schema:         core.SchemaReleaseDeployPlan,
		Product:        input.Manifest.Product,
		Version:        input.Manifest.Version,
		ReleaseID:      input.Manifest.ReleaseID,
		ManifestSHA256: core.NewSHA256Hex(sha256.Sum256(canonical)),
		Layout:         input.Layout,
		Targets:        cloneAndSortUploadTargets(input.Targets),
		Manifest:       manifest,
		AttemptID:      input.AttemptID,
		TargetCount:    count,
	}, nil
}

func cloneAndSortUploadTargets(targets []UploadTarget) []UploadTarget {
	copied := append([]UploadTarget(nil), targets...)
	for index := range copied {
		copied[index].Headers = append([]core.UploadHeader(nil), copied[index].Headers...)
	}
	sort.Slice(copied, func(left, right int) bool {
		return deployTargetKey(copied[left]) < deployTargetKey(copied[right])
	})
	return copied
}

type DownloadIndexInput struct {
	Downloads   []Download
	Manifest    Manifest
	GeneratedAt core.UnixNanoTime
}

func (i DownloadIndexInput) Validate() error {
	index, err := downloadIndexFromInput(i)
	if err != nil {
		return err
	}
	return index.Validate()
}

func BuildDownloadIndex(input DownloadIndexInput) (DownloadIndex, error) {
	index, err := downloadIndexFromInput(input)
	if err != nil {
		return DownloadIndex{}, err
	}
	if err := index.VerifyManifest(input.Manifest); err != nil {
		return DownloadIndex{}, err
	}
	return index, nil
}

func downloadIndexFromInput(input DownloadIndexInput) (DownloadIndex, error) {
	if err := input.Manifest.Validate(); err != nil {
		return DownloadIndex{}, wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	count, err := core.DeriveCollectionCount(len(input.Downloads), 1, core.CollectionMaximumDefault)
	if err != nil {
		return DownloadIndex{}, wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	index := DownloadIndex{
		Version:       input.Manifest.Version,
		ReleaseID:     input.Manifest.ReleaseID,
		Commit:        input.Manifest.Commit,
		Downloads:     append([]Download(nil), input.Downloads...),
		GeneratedAt:   input.GeneratedAt,
		DownloadCount: count,
		Schema:        core.SchemaReleaseDownloadIndex,
		Product:       input.Manifest.Product,
	}
	sort.Slice(index.Downloads, func(left, right int) bool {
		return downloadSetKey(index.Downloads[left]) < downloadSetKey(index.Downloads[right])
	})
	if err := validateManifestDownloads(input.Manifest, index.Downloads); err != nil {
		return DownloadIndex{}, err
	}
	return index, nil
}

func validateManifestDownloads(manifest Manifest, downloads []Download) error {
	downloadableCount := 0
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind.RequiresPlatform() {
			downloadableCount++
		}
	}
	if downloadableCount == 0 || len(downloads) < downloadableCount || len(downloads)%downloadableCount != 0 {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	for _, download := range downloads {
		if !manifestHasDownload(manifest, download) {
			return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
		}
	}
	for _, download := range downloads {
		if !providerDownloadsEveryManifestArtifact(downloads, download.Provider, manifest) {
			return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
		}
	}
	return nil
}

func providerDownloadsEveryManifestArtifact(downloads []Download, provider core.StorageProvider, manifest Manifest) bool {
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind.RequiresPlatform() && !hasProviderDownload(downloads, provider, artifact.Name) {
			return false
		}
	}
	return true
}

func hasProviderDownload(downloads []Download, provider core.StorageProvider, artifact ArtifactName) bool {
	for _, download := range downloads {
		if download.Provider == provider && download.Artifact == artifact {
			return true
		}
	}
	return false
}

func manifestHasDownload(manifest Manifest, download Download) bool {
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind.RequiresPlatform() && artifact.Name == download.Artifact &&
			artifact.Platform == download.Platform && artifact.SHA256 == download.SHA256 &&
			artifact.Size == download.Size {
			return true
		}
	}
	return false
}
