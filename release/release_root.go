package release

import (
	"fmt"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.CanonicalBody = ReleaseRootLayout{}

const (
	ReleaseRootDirName      = "releases"
	ReleasePlatformsDirName = "platforms"
	ReleaseReceiptsDirName  = "receipts"
	ReleaseManifestsDirName = "manifests"
	ReleaseDogfoodDirName   = "dogfood"
)

type ReleaseRootInput struct {
	Version   core.ProductVersion
	Date      ReleaseDate
	ReleaseID ReleaseID
	Product   core.Product
}

func (i ReleaseRootInput) Validate() error {
	if err := i.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	if err := i.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	if err := i.Date.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	if err := i.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	return nil
}

type ReleaseRootLayout struct {
	Version   core.ProductVersion `json:"version"`
	Date      ReleaseDate         `json:"date"`
	ReleaseID ReleaseID           `json:"release_id"`
	Root      BuildOutputPath     `json:"root"`
	Private   BuildOutputPath     `json:"private"`
	Public    BuildOutputPath     `json:"public"`
	Platforms BuildOutputPath     `json:"platforms"`
	Receipts  BuildOutputPath     `json:"receipts"`
	Manifests BuildOutputPath     `json:"manifests"`
	Dogfood   BuildOutputPath     `json:"dogfood"`
	Schema    core.SchemaID       `json:"schema"`
	Product   core.Product        `json:"product"`
}

func BuildReleaseRootLayout(input ReleaseRootInput) (ReleaseRootLayout, error) {
	if err := input.Validate(); err != nil {
		return ReleaseRootLayout{}, err
	}
	layout, err := releaseRootLayoutFromInput(input)
	if err != nil {
		return ReleaseRootLayout{}, wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	return layout, nil
}

func (l ReleaseRootLayout) Validate() error {
	if l.Schema != core.SchemaReleaseRootLayout {
		return fmt.Errorf(ErrFmtReleaseRoot, core.ErrReleaseContract)
	}
	input := ReleaseRootInput{
		Product:   l.Product,
		Version:   l.Version,
		Date:      l.Date,
		ReleaseID: l.ReleaseID,
	}
	if err := input.Validate(); err != nil {
		return err
	}
	expected, err := releaseRootLayoutFromInput(input)
	if err != nil {
		return wrapReleaseContract(ErrFmtReleaseRoot, err)
	}
	return validateReleaseRootPaths(l, expected)
}

func (l ReleaseRootLayout) Canonical(dst []byte) ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return appendReleaseRootLayoutJSON(dst, l)
}

func (l ReleaseRootLayout) SigningSchema() core.SchemaID {
	return l.Schema
}

func (l ReleaseRootLayout) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return appendReleaseRootLayoutJSON(nil, l)
}

func appendReleaseRootLayoutJSON(dst []byte, l ReleaseRootLayout) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, l.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, l.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, l.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDate, l.Date)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, l.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRoot, l.Root)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPrivate, l.Private)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPublic, l.Public)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatforms, l.Platforms)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReceipts, l.Receipts)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifests, l.Manifests)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDogfood, l.Dogfood)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func releaseRootLayoutFromInput(input ReleaseRootInput) (ReleaseRootLayout, error) {
	root, err := releaseRootPath(releaseRootSegments(input)...)
	if err != nil {
		return ReleaseRootLayout{}, err
	}
	layout, err := releaseRootChildren(root)
	if err != nil {
		return ReleaseRootLayout{}, err
	}
	layout.Product = input.Product
	layout.Schema = core.SchemaReleaseRootLayout
	layout.Version = input.Version
	layout.Date = input.Date
	layout.ReleaseID = input.ReleaseID
	return layout, nil
}

func releaseRootSegments(input ReleaseRootInput) []string {
	date := input.Date.String()
	return []string{
		DistDirName,
		ReleaseRootDirName,
		input.Product.String(),
		date[:4],
		date[5:7],
		date[8:10],
		input.ReleaseID.String(),
	}
}

func releaseRootChildren(root BuildOutputPath) (ReleaseRootLayout, error) {
	children, err := releaseRootChildPaths(root.String())
	if err != nil {
		return ReleaseRootLayout{}, err
	}
	return ReleaseRootLayout{
		Root:      root,
		Private:   children[0],
		Public:    children[1],
		Platforms: children[2],
		Receipts:  children[3],
		Manifests: children[4],
		Dogfood:   children[5],
	}, nil
}

func releaseRootChildPaths(root string) ([6]BuildOutputPath, error) {
	names := [...]string{
		ObjectSegmentPrivate,
		ObjectSegmentPublic,
		ReleasePlatformsDirName,
		ReleaseReceiptsDirName,
		ReleaseManifestsDirName,
		ReleaseDogfoodDirName,
	}
	var paths [6]BuildOutputPath
	for idx, name := range names {
		path, err := releaseRootPath(root, name)
		if err != nil {
			return paths, err
		}
		paths[idx] = path
	}
	return paths, nil
}

func releaseRootPath(segments ...string) (BuildOutputPath, error) {
	return ParseBuildOutputPath(strings.Join(segments, "/"))
}

func validateReleaseRootPaths(got, want ReleaseRootLayout) error {
	if !releaseRootPathsEqual(got, want) {
		return fmt.Errorf(ErrFmtReleaseRoot, core.ErrReleaseContract)
	}
	return nil
}

func releaseRootPathsEqual(got, want ReleaseRootLayout) bool {
	return got.Root.String() == want.Root.String() &&
		got.Private.String() == want.Private.String() &&
		got.Public.String() == want.Public.String() &&
		got.Platforms.String() == want.Platforms.String() &&
		got.Receipts.String() == want.Receipts.String() &&
		got.Manifests.String() == want.Manifests.String() &&
		got.Dogfood.String() == want.Dogfood.String()
}
