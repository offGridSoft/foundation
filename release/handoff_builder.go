package release

import (
	"crypto/sha256"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.Validatable = DeployPlanInput{}
	_ core.Validatable = DownloadIndexInput{}
)

type DeployPlanInput struct {
	Layout   ReleaseRootLayout
	Targets  []UploadTarget
	Manifest Manifest
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
		Product:        input.Manifest.Product,
		Version:        input.Manifest.Version,
		ReleaseID:      input.Manifest.ReleaseID,
		ManifestSHA256: core.NewSHA256Hex(sha256.Sum256(canonical)),
		Layout:         input.Layout,
		Targets:        append([]UploadTarget(nil), input.Targets...),
		Manifest:       manifest,
		TargetCount:    count,
	}, nil
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
	if err := index.Validate(); err != nil {
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
		Downloads:     append([]Download(nil), input.Downloads...),
		GeneratedAt:   input.GeneratedAt,
		DownloadCount: count,
		Schema:        core.SchemaReleaseDownloadIndex,
		Product:       input.Manifest.Product,
	}
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
	if downloadableCount != len(downloads) {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	for _, download := range downloads {
		if !manifestHasDownload(manifest, download) {
			return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
		}
	}
	return nil
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
