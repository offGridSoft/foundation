package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.Validatable = ManifestInput{}

type ManifestInput struct {
	Version   core.ProductVersion
	ReleaseID ReleaseID
	Date      ReleaseDate
	Commit    core.BuildCommit
	Artifacts []Artifact
	CreatedAt core.UnixNanoTime
	Product   core.Product
}

func (i ManifestInput) Validate() error {
	_, err := validateManifestInput(i)
	return err
}

func validateManifestInput(i ManifestInput) (core.ArtifactSet[Artifact], error) {
	if err := i.Product.Validate(); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := i.Version.Validate(); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := i.ReleaseID.Validate(); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := i.Date.Validate(); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := i.Commit.Validate(); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := ValidateReleaseIDIdentity(i.ReleaseID, i.Product, i.Version, i.Commit); err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(i.CreatedAt); err != nil {
		return core.ArtifactSet[Artifact]{}, fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	set, err := core.BuildArtifactSet(i.Artifacts)
	if err != nil {
		return core.ArtifactSet[Artifact]{}, wrapReleaseContract(ErrFmtManifest, err)
	}
	return set, nil
}

func BuildManifest(input ManifestInput) (Manifest, error) {
	set, err := validateManifestInput(input)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		Version:       input.Version,
		ReleaseID:     input.ReleaseID,
		Date:          input.Date,
		Commit:        input.Commit,
		Artifacts:     set.Items,
		CreatedAt:     input.CreatedAt,
		TotalBytes:    set.TotalBytes,
		ArtifactCount: set.Count,
		Schema:        core.SchemaReleaseManifest,
		Product:       input.Product,
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}
