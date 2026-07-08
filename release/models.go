package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/core"
)

type ArchiveEntry struct {
	Name ArtifactName `json:"name"`
	Mode uint32       `json:"mode"`
}

func (e ArchiveEntry) Validate() error {
	if err := e.Name.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArchiveLayout, err)
	}
	if e.Mode != ArchiveToolFileMode && e.Mode != ArchiveMetadataFileMode {
		return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
	}
	return nil
}

type ArchiveLayout struct {
	Name          ArtifactName   `json:"name"`
	Entries       []ArchiveEntry `json:"entries"`
	EntryCount    uint32         `json:"entry_count"`
	MaxEntryBytes core.ByteCount `json:"max_entry_bytes"`
	MaxTotalBytes core.ByteCount `json:"max_total_bytes"`
}

func (l ArchiveLayout) Validate() error {
	if err := l.Name.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArchiveLayout, err)
	}
	if err := validateArchiveLimits(l); err != nil {
		return err
	}
	return validateArchiveEntries(l)
}

func validateArchiveLimits(l ArchiveLayout) error {
	if err := l.MaxEntryBytes.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArchiveLayout, err)
	}
	if err := l.MaxTotalBytes.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArchiveLayout, err)
	}
	if l.MaxEntryBytes.Uint64() > l.MaxTotalBytes.Uint64() {
		return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
	}
	return nil
}

func validateArchiveEntries(l ArchiveLayout) error {
	if l.EntryCount == 0 || int(l.EntryCount) != len(l.Entries) {
		return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
	}
	names := core.NewUniqueStringSet(len(l.Entries))
	for _, entry := range l.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		if err := names.Add(entry.Name.String()); err != nil {
			return wrapReleaseContract(ErrFmtArchiveLayout, err)
		}
	}
	return nil
}

type Artifact struct {
	Name     ArtifactName   `json:"name"`
	SHA256   core.SHA256Hex `json:"sha256"`
	Size     core.ByteCount `json:"size_bytes"`
	Kind     Kind           `json:"kind"`
	Platform core.Platform  `json:"platform,omitempty"`
}

func (a Artifact) Validate() error {
	if err := a.Name.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArtifact, err)
	}
	if err := a.Kind.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArtifact, err)
	}
	if err := validateArtifactPlatform(a); err != nil {
		return err
	}
	if err := a.SHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArtifact, err)
	}
	if err := a.Size.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtArtifact, err)
	}
	return nil
}

func (a Artifact) ArtifactSetName() string {
	return a.Name.String()
}

func (a Artifact) ArtifactSetSize() core.ByteCount {
	return a.Size
}

func validateArtifactPlatform(a Artifact) error {
	if a.Kind.RequiresPlatform() {
		return validateReleasePlatform(a.Platform, ErrFmtArtifact)
	}
	if a.Platform != core.PlatformUnknown {
		return fmt.Errorf(ErrFmtArtifact, core.ErrReleaseContract)
	}
	return nil
}

type Manifest struct {
	Version       core.ProductVersion `json:"version"`
	ReleaseID     ReleaseID           `json:"release_id"`
	Date          ReleaseDate         `json:"date"`
	Commit        core.BuildCommit    `json:"commit"`
	Artifacts     []Artifact          `json:"artifacts"`
	CreatedAt     core.UnixNanoTime   `json:"created_at"`
	TotalBytes    core.ByteCount      `json:"total_bytes"`
	ArtifactCount uint32              `json:"artifact_count"`
	Schema        core.SchemaID       `json:"schema"`
	Product       core.Product        `json:"product"`
}

func (m Manifest) Validate() error {
	if m.Schema != core.SchemaReleaseManifest {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	if err := validateManifestIdentity(m); err != nil {
		return err
	}
	return validateCoreArtifactSet(core.ArtifactSet[Artifact]{
		Items:      m.Artifacts,
		Count:      m.ArtifactCount,
		TotalBytes: m.TotalBytes,
	}, ErrFmtManifest)
}

func validateManifestIdentity(m Manifest) error {
	if err := m.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := m.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := m.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := m.Date.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := m.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	return nil
}

func (m Manifest) Canonical(dst []byte) ([]byte, error) {
	return core.AppendCanonicalJSON(dst, m)
}

func (m Manifest) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return appendManifestJSON(nil, m)
}

func appendManifestJSON(dst []byte, m Manifest) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, m.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "version", m.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "release_id", m.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "date", m.Date)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "commit", m.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "artifacts", m.Artifacts)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "created_at", m.CreatedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "total_bytes", m.TotalBytes)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "artifact_count", m.ArtifactCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "product", m.Product)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type UploadTarget struct {
	Bucket   Bucket               `json:"bucket"`
	Prefix   ObjectKey            `json:"prefix"`
	Provider core.StorageProvider `json:"provider"`
	Method   core.UploadMethod    `json:"method"`
}

func (t UploadTarget) Validate() error {
	if err := t.Provider.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Bucket.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Prefix.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Method.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	return nil
}

type UploadedArtifact struct {
	Artifact ArtifactName         `json:"artifact"`
	Object   ObjectKey            `json:"object"`
	Bucket   Bucket               `json:"bucket"`
	SHA256   core.SHA256Hex       `json:"sha256"`
	Size     core.ByteCount       `json:"size_bytes"`
	Provider core.StorageProvider `json:"provider"`
}

func (a UploadedArtifact) Validate() error {
	if err := a.Artifact.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.Object.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.Provider.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.Bucket.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.SHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.Size.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	return nil
}

func (a UploadedArtifact) ArtifactSetName() string {
	return a.Artifact.String()
}

func (a UploadedArtifact) ArtifactSetSize() core.ByteCount {
	return a.Size
}

type UploadReceipt struct {
	Version     core.ProductVersion `json:"version"`
	ReleaseID   ReleaseID           `json:"release_id"`
	Objects     []UploadedArtifact  `json:"objects"`
	UploadedAt  core.UnixNanoTime   `json:"uploaded_at"`
	TotalBytes  core.ByteCount      `json:"total_bytes"`
	ObjectCount uint32              `json:"object_count"`
	Schema      core.SchemaID       `json:"schema"`
	Product     core.Product        `json:"product"`
}

func (r UploadReceipt) Validate() error {
	if r.Schema != core.SchemaReleaseUploadReceipt {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	if err := validateReceiptIdentity(r); err != nil {
		return err
	}
	return validateUploadedSet(r.Objects, r.ObjectCount, r.TotalBytes)
}

func validateReceiptIdentity(r UploadReceipt) error {
	if err := r.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := r.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := r.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if r.UploadedAt.IsZero() {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	return nil
}

type Download struct {
	Artifact ArtifactName   `json:"artifact"`
	URL      DownloadURL    `json:"url"`
	SHA256   core.SHA256Hex `json:"sha256"`
	Size     core.ByteCount `json:"size_bytes"`
	Platform core.Platform  `json:"platform"`
}

func (d Download) Validate() error {
	if err := d.Platform.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := d.Artifact.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := d.URL.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := d.SHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := d.Size.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	return nil
}

type DownloadIndex struct {
	Version       core.ProductVersion `json:"version"`
	ReleaseID     ReleaseID           `json:"release_id"`
	Downloads     []Download          `json:"downloads"`
	GeneratedAt   core.UnixNanoTime   `json:"generated_at"`
	DownloadCount uint32              `json:"download_count"`
	Schema        core.SchemaID       `json:"schema"`
	Product       core.Product        `json:"product"`
}

func (i DownloadIndex) Validate() error {
	if i.Schema != core.SchemaReleaseDownloadIndex {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	if err := validateDownloadIndexIdentity(i); err != nil {
		return err
	}
	return validateDownloadSet(i.Downloads, i.DownloadCount)
}

func validateDownloadIndexIdentity(i DownloadIndex) error {
	if err := i.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := i.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := i.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if i.GeneratedAt.IsZero() {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	return nil
}

func validateUploadedSet(objects []UploadedArtifact, count uint32, total core.ByteCount) error {
	if err := validateCoreArtifactSet(core.ArtifactSet[UploadedArtifact]{
		Items:      objects,
		Count:      count,
		TotalBytes: total,
	}, ErrFmtUploadReceipt); err != nil {
		return err
	}
	objectsSeen := core.NewUniqueStringSet(len(objects))
	for _, object := range objects {
		if err := objectsSeen.Add(object.Object.String()); err != nil {
			return wrapReleaseContract(ErrFmtUploadReceipt, err)
		}
	}
	return nil
}

func validateDownloadSet(downloads []Download, count uint32) error {
	if count == 0 || int(count) != len(downloads) {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	platforms := core.NewUniqueStringSet(len(downloads))
	for _, download := range downloads {
		if err := download.Validate(); err != nil {
			return err
		}
		if err := platforms.Add(download.Platform.String()); err != nil {
			return wrapReleaseContract(ErrFmtDownloadIndex, err)
		}
	}
	return nil
}

func validateReleasePlatform(platform core.Platform, errFmt string) error {
	if err := platform.Validate(); err != nil {
		return wrapReleaseContract(errFmt, err)
	}
	if platform.IsReleaseTarget() {
		return nil
	}
	return fmt.Errorf(errFmt, core.ErrReleaseContract)
}

func validateCoreArtifactSet[T core.ArtifactSetItem](set core.ArtifactSet[T], errFmt string) error {
	if err := core.ValidateArtifactSet(set); err != nil {
		return wrapReleaseContract(errFmt, err)
	}
	return nil
}
