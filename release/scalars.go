package release

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	DateLayout              = "2006-01-02"
	ReleaseIDMaxRunes       = 128
	BucketMaxRunes          = 128
	DownloadURLMaxRunes     = 2048
	ObjectSegmentPublic     = "public"
	ObjectSegmentPrivate    = "private"
	ToolsArchiveName        = "tools.tar.gz"
	ManifestFileName        = "manifest.json"
	UploadReceiptFileName   = "upload_receipt.json"
	DownloadIndexFileName   = "download_index.json"
	GarbleSeedFileName      = "garble.seed"
	ArchiveToolFileMode     = 0o755
	ArchiveMetadataFileMode = 0o644
	ArchiveMaxEntryBytes    = 256 << 20
	ArchiveMaxTotalBytes    = 512 << 20
)

type ReleaseID struct {
	value string
}

func ParseReleaseID(value string) (ReleaseID, error) {
	if err := core.ValidateFileNameToken(value, ReleaseIDMaxRunes); err != nil {
		return ReleaseID{}, fmt.Errorf(ErrFmtReleaseID, core.ErrReleaseContract)
	}
	return ReleaseID{value: value}, nil
}

func (id ReleaseID) String() string {
	return id.value
}

func (id ReleaseID) Validate() error {
	_, err := ParseReleaseID(id.value)
	return err
}

func (id ReleaseID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *ReleaseID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtReleaseID, core.ErrReleaseContract)
	}
	parsed, err := ParseReleaseID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type ReleaseDate struct {
	value string
}

func ParseReleaseDate(value string) (ReleaseDate, error) {
	parsed, err := time.Parse(DateLayout, value)
	if err != nil || parsed.Format(DateLayout) != value {
		return ReleaseDate{}, fmt.Errorf(ErrFmtReleaseDate, core.ErrReleaseContract)
	}
	return ReleaseDate{value: value}, nil
}

func NewReleaseDate(t time.Time) ReleaseDate {
	return ReleaseDate{value: t.UTC().Format(DateLayout)}
}

func (d ReleaseDate) String() string {
	return d.value
}

func (d ReleaseDate) Validate() error {
	_, err := ParseReleaseDate(d.value)
	return err
}

func (d ReleaseDate) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.value)
}

func (d *ReleaseDate) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtReleaseDate, core.ErrReleaseContract)
	}
	parsed, err := ParseReleaseDate(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

type ArtifactName struct {
	value string
}

func ParseArtifactName(value string) (ArtifactName, error) {
	if err := core.ValidateFileNameToken(value, core.FileNameTokenMaxRunes); err != nil {
		return ArtifactName{}, fmt.Errorf(ErrFmtArtifactName, core.ErrReleaseContract)
	}
	return ArtifactName{value: value}, nil
}

func (n ArtifactName) String() string {
	return n.value
}

func (n ArtifactName) Validate() error {
	_, err := ParseArtifactName(n.value)
	return err
}

func (n ArtifactName) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.value)
}

func (n *ArtifactName) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtArtifactName, core.ErrReleaseContract)
	}
	parsed, err := ParseArtifactName(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

type Bucket struct {
	value string
}

func ParseBucket(value string) (Bucket, error) {
	if err := core.ValidateFileNameToken(value, BucketMaxRunes); err != nil {
		return Bucket{}, fmt.Errorf(ErrFmtBucket, core.ErrReleaseContract)
	}
	return Bucket{value: value}, nil
}

func (b Bucket) String() string {
	return b.value
}

func (b Bucket) Validate() error {
	_, err := ParseBucket(b.value)
	return err
}

func (b Bucket) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.value)
}

func (b *Bucket) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBucket, core.ErrReleaseContract)
	}
	parsed, err := ParseBucket(value)
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

type ObjectKey struct {
	value string
}

func ParseObjectKey(value string) (ObjectKey, error) {
	if err := core.ValidatePathToken(value, core.PathTokenMaxRunes); err != nil {
		return ObjectKey{}, wrapReleaseContract(ErrFmtObjectKey, err)
	}
	return ObjectKey{value: value}, nil
}

func (k ObjectKey) String() string {
	return k.value
}

func (k ObjectKey) Validate() error {
	_, err := ParseObjectKey(k.value)
	return err
}

func (k ObjectKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.value)
}

func (k *ObjectKey) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtObjectKey, core.ErrReleaseContract)
	}
	parsed, err := ParseObjectKey(value)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

type DownloadURL struct {
	value string
}

func ParseDownloadURL(value string) (DownloadURL, error) {
	if err := core.ValidateHTTPSURL(value, core.HTTPSURLPolicy{
		MaxRunes:    DownloadURLMaxRunes,
		RequirePath: true,
	}); err != nil {
		return DownloadURL{}, wrapReleaseContract(ErrFmtDownloadURL, err)
	}
	return DownloadURL{value: value}, nil
}

func (u DownloadURL) String() string {
	return u.value
}

func (u DownloadURL) Validate() error {
	_, err := ParseDownloadURL(u.value)
	return err
}

func (u DownloadURL) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(u.value)
}

func (u *DownloadURL) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDownloadURL, core.ErrReleaseContract)
	}
	parsed, err := ParseDownloadURL(value)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}
