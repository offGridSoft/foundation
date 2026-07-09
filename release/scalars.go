package release

import (
	"encoding/json"
	"fmt"
	"strings"
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
	EvidenceRefMaxRunes     = 512
	GarbleSeedRefMaxRunes   = 512
	ToolModuleMaxRunes      = 256
	ToolVersionMaxRunes     = 128
	GoSumHashMaxRunes       = 256
	GoSumHashPrefix         = "h1:"
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

type EvidenceRef struct {
	value string
}

func ParseEvidenceRef(value string) (EvidenceRef, error) {
	if err := core.ValidateOpaqueToken(value, EvidenceRefMaxRunes); err != nil {
		return EvidenceRef{}, wrapReleaseContract(ErrFmtEvidenceRef, err)
	}
	return EvidenceRef{value: value}, nil
}

func (r EvidenceRef) String() string {
	return r.value
}

func (r EvidenceRef) Validate() error {
	_, err := ParseEvidenceRef(r.value)
	return err
}

func (r EvidenceRef) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.value)
}

func (r *EvidenceRef) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtEvidenceRef, core.ErrReleaseContract)
	}
	parsed, err := ParseEvidenceRef(value)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

type GarbleSeedRef struct {
	value string
}

func ParseGarbleSeedRef(value string) (GarbleSeedRef, error) {
	if err := core.ValidateOpaqueToken(value, GarbleSeedRefMaxRunes); err != nil {
		return GarbleSeedRef{}, wrapReleaseContract(ErrFmtGarbleSeedRef, err)
	}
	if strings.ContainsAny(value, " \t") {
		return GarbleSeedRef{}, fmt.Errorf(ErrFmtGarbleSeedRef, core.ErrReleaseContract)
	}
	return GarbleSeedRef{value: value}, nil
}

func (r GarbleSeedRef) String() string {
	return r.value
}

func (r GarbleSeedRef) Validate() error {
	_, err := ParseGarbleSeedRef(r.value)
	return err
}

func (r GarbleSeedRef) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.value)
}

func (r *GarbleSeedRef) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGarbleSeedRef, core.ErrReleaseContract)
	}
	parsed, err := ParseGarbleSeedRef(value)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

type ToolVersion struct {
	value string
}

func ParseToolVersion(value string) (ToolVersion, error) {
	if err := core.ValidateOpaqueToken(value, ToolVersionMaxRunes); err != nil {
		return ToolVersion{}, wrapReleaseContract(ErrFmtToolVersion, err)
	}
	return ToolVersion{value: value}, nil
}

func (v ToolVersion) String() string {
	return v.value
}

func (v ToolVersion) Validate() error {
	_, err := ParseToolVersion(v.value)
	return err
}

func (v ToolVersion) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v.value)
}

func (v *ToolVersion) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtToolVersion, core.ErrReleaseContract)
	}
	parsed, err := ParseToolVersion(value)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

type ToolModule struct {
	value string
}

func ParseToolModule(value string) (ToolModule, error) {
	if err := core.ValidateOpaqueToken(value, ToolModuleMaxRunes); err != nil {
		return ToolModule{}, wrapReleaseContract(ErrFmtToolModule, err)
	}
	if strings.ContainsAny(value, " \t") {
		return ToolModule{}, fmt.Errorf(ErrFmtToolModule, core.ErrReleaseContract)
	}
	return ToolModule{value: value}, nil
}

func (m ToolModule) String() string {
	return m.value
}

func (m ToolModule) Validate() error {
	_, err := ParseToolModule(m.value)
	return err
}

func (m ToolModule) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.value)
}

func (m *ToolModule) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtToolModule, core.ErrReleaseContract)
	}
	parsed, err := ParseToolModule(value)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

type GoSumHash struct {
	value string
}

func ParseGoSumHash(value string) (GoSumHash, error) {
	if err := core.ValidateOpaqueToken(value, GoSumHashMaxRunes); err != nil {
		return GoSumHash{}, wrapReleaseContract(ErrFmtGoSumHash, err)
	}
	if !strings.HasPrefix(value, GoSumHashPrefix) || value == GoSumHashPrefix {
		return GoSumHash{}, fmt.Errorf(ErrFmtGoSumHash, core.ErrReleaseContract)
	}
	return GoSumHash{value: value}, nil
}

func (h GoSumHash) String() string {
	return h.value
}

func (h GoSumHash) Validate() error {
	_, err := ParseGoSumHash(h.value)
	return err
}

func (h GoSumHash) MarshalJSON() ([]byte, error) {
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(h.value)
}

func (h *GoSumHash) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGoSumHash, core.ErrReleaseContract)
	}
	parsed, err := ParseGoSumHash(value)
	if err != nil {
		return err
	}
	*h = parsed
	return nil
}
