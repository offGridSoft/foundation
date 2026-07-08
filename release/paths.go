package release

import (
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

type ObjectKeyInput struct {
	Date       ReleaseDate
	ReleaseID  ReleaseID
	Artifact   ArtifactName
	Product    core.Product
	Visibility Visibility
}

func (i ObjectKeyInput) Validate() error {
	if err := i.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtObjectKey, err)
	}
	if err := i.Date.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtObjectKey, err)
	}
	if err := i.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtObjectKey, err)
	}
	if err := i.Visibility.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtObjectKey, err)
	}
	if err := i.Artifact.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtObjectKey, err)
	}
	return nil
}

func BuildObjectKey(input ObjectKeyInput) (ObjectKey, error) {
	if err := input.Validate(); err != nil {
		return ObjectKey{}, err
	}
	key := strings.Join([]string{
		input.Product.String(),
		input.Date.String(),
		input.ReleaseID.String(),
		input.Visibility.String(),
		input.Artifact.String(),
	}, "/")
	parsed, err := ParseObjectKey(key)
	if err != nil {
		return ObjectKey{}, wrapReleaseContract(ErrFmtObjectKey, err)
	}
	return parsed, nil
}
