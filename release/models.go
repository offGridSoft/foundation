package release

import (
	"crypto/sha256"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.CanonicalBody = Manifest{}
	_ core.CanonicalBody = UploadReceipt{}
	_ core.CanonicalBody = DownloadIndex{}
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
	if l.MaxEntryBytes.Uint64() > core.ArtifactMaximumBytes ||
		l.MaxTotalBytes.Uint64() > core.ArtifactSetMaximumBytes ||
		l.MaxEntryBytes.Uint64() > l.MaxTotalBytes.Uint64() {
		return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
	}
	return nil
}

func validateArchiveEntries(l ArchiveLayout) error {
	if err := (core.CollectionCardinality{
		Length:          len(l.Entries),
		DeclaredCount:   l.EntryCount,
		Minimum:         1,
		Maximum:         core.CollectionMaximumDefault,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
	}
	for index, entry := range l.Entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		for _, prior := range l.Entries[:index] {
			if prior.Name == entry.Name {
				return fmt.Errorf(ErrFmtArchiveLayout, core.ErrReleaseContract)
			}
		}
	}
	return nil
}

// Field order is signature-load-bearing when nested inside Manifest.
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

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
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
	if err := ValidateReleaseIDIdentity(m.ReleaseID, m.Product, m.Version, m.Commit); err != nil {
		return wrapReleaseContract(ErrFmtManifest, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(m.CreatedAt); err != nil {
		return fmt.Errorf(ErrFmtManifest, core.ErrReleaseContract)
	}
	return nil
}

func (m Manifest) Canonical(dst []byte) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return appendManifestJSON(dst, m)
}

func (m Manifest) SigningSchema() core.SchemaID {
	return m.Schema
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
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, m.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, m.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDate, m.Date)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, m.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldArtifacts, m.Artifacts)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCreatedAt, m.CreatedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTotalBytes, m.TotalBytes)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldArtifactCount, m.ArtifactCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, m.Product)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type UploadTarget struct {
	Artifact  ArtifactName         `json:"artifact"`
	Object    ObjectKey            `json:"object"`
	Bucket    Bucket               `json:"bucket"`
	URL       core.SignedUploadURL `json:"url"`
	AttemptID UploadAttemptID      `json:"upload_attempt_id"`
	Binding   UploadBinding        `json:"upload_binding"`
	Headers   []core.UploadHeader  `json:"headers"`
	ExpiresAt core.UnixNanoTime    `json:"expires_at"`
	Provider  core.StorageProvider `json:"provider"`
	Method    core.UploadMethod    `json:"method"`
}

func (t UploadTarget) Validate() error {
	if err := validateUploadTargetLocation(t); err != nil {
		return err
	}
	if err := validateUploadTargetAuthorization(t); err != nil {
		return err
	}
	if err := core.ValidateRequiredUnixNanoTime(t.ExpiresAt); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	return nil
}

func validateUploadTargetLocation(t UploadTarget) error {
	if err := t.Artifact.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Object.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Provider.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Bucket.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	return nil
}

func validateUploadTargetAuthorization(t UploadTarget) error {
	if err := t.URL.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Method.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.AttemptID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := t.Binding.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	if err := validateUploadBindingHeaders(t); err != nil {
		return err
	}
	return nil
}

type UploadedArtifact struct {
	Artifact  ArtifactName         `json:"artifact"`
	Object    ObjectKey            `json:"object"`
	Bucket    Bucket               `json:"bucket"`
	SHA256    core.SHA256Hex       `json:"sha256"`
	AttemptID UploadAttemptID      `json:"upload_attempt_id"`
	Binding   UploadBinding        `json:"upload_binding"`
	Size      core.ByteCount       `json:"size_bytes"`
	Provider  core.StorageProvider `json:"provider"`
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
	if err := a.AttemptID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := a.Binding.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	return nil
}

func (a UploadedArtifact) ArtifactSetName() string {
	return a.Provider.String() + "/" + a.Artifact.String()
}

func (a UploadedArtifact) ArtifactSetSize() core.ByteCount {
	return a.Size
}

type UploadReceipt struct {
	Version        core.ProductVersion `json:"version"`
	ReleaseID      ReleaseID           `json:"release_id"`
	Commit         core.BuildCommit    `json:"commit"`
	ManifestSHA256 core.SHA256Hex      `json:"manifest_sha256"`
	AttemptID      UploadAttemptID     `json:"upload_attempt_id"`
	Objects        []UploadedArtifact  `json:"objects"`
	UploadedAt     core.UnixNanoTime   `json:"uploaded_at"`
	TotalBytes     core.ByteCount      `json:"total_bytes"`
	ObjectCount    uint32              `json:"object_count"`
	Schema         core.SchemaID       `json:"schema"`
	Product        core.Product        `json:"product"`
}

func (r UploadReceipt) Validate() error {
	if r.Schema != core.SchemaReleaseUploadReceipt {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	if err := validateReceiptIdentity(r); err != nil {
		return err
	}
	return validateUploadedSet(r)
}

func (r UploadReceipt) Canonical(dst []byte) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendUploadReceiptJSON(dst, r)
}

func (r UploadReceipt) SigningSchema() core.SchemaID {
	return r.Schema
}

func (r UploadReceipt) VerifyManifest(manifest Manifest) error {
	if err := r.Validate(); err != nil {
		return err
	}
	canonical, err := manifest.Canonical(nil)
	if err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if r.Product != manifest.Product || r.Version != manifest.Version || r.ReleaseID != manifest.ReleaseID ||
		r.Commit != manifest.Commit || r.ManifestSHA256 != core.NewSHA256Hex(sha256.Sum256(canonical)) {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	return validateUploadedArtifactsMatchManifest(manifest, r.Objects)
}

func (r UploadReceipt) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return appendUploadReceiptJSON(nil, r)
}

func appendUploadReceiptJSON(dst []byte, r UploadReceipt) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, r.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, r.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, r.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, r.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifestSHA256, r.ManifestSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldObjects, r.Objects)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUploadAttemptID, r.AttemptID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUploadedAt, r.UploadedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTotalBytes, r.TotalBytes)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldObjectCount, r.ObjectCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, r.Product)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
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
	if err := r.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := r.ManifestSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := r.AttemptID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := ValidateReleaseIDIdentity(r.ReleaseID, r.Product, r.Version, r.Commit); err != nil {
		return wrapReleaseContract(ErrFmtUploadReceipt, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(r.UploadedAt); err != nil {
		return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
	}
	return nil
}

type Download struct {
	Artifact ArtifactName         `json:"artifact"`
	URL      DownloadURL          `json:"url"`
	SHA256   core.SHA256Hex       `json:"sha256"`
	Size     core.ByteCount       `json:"size_bytes"`
	Platform core.Platform        `json:"platform"`
	Provider core.StorageProvider `json:"provider"`
}

func (d Download) Validate() error {
	if err := d.Provider.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := validateReleasePlatform(d.Platform, ErrFmtDownloadIndex); err != nil {
		return err
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
	Commit        core.BuildCommit    `json:"commit"`
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

func (i DownloadIndex) Canonical(dst []byte) ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return appendDownloadIndexJSON(dst, i)
}

func (i DownloadIndex) SigningSchema() core.SchemaID {
	return i.Schema
}

func (i DownloadIndex) VerifyManifest(manifest Manifest) error {
	if err := i.Validate(); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if i.Product != manifest.Product || i.Version != manifest.Version || i.ReleaseID != manifest.ReleaseID || i.Commit != manifest.Commit {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	return validateManifestDownloads(manifest, i.Downloads)
}

func (i DownloadIndex) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return appendDownloadIndexJSON(nil, i)
}

func appendDownloadIndexJSON(dst []byte, i DownloadIndex) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, i.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, i.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, i.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, i.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDownloads, i.Downloads)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldGeneratedAt, i.GeneratedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDownloadCount, i.DownloadCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, i.Product)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
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
	if err := i.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := ValidateReleaseIDIdentity(i.ReleaseID, i.Product, i.Version, i.Commit); err != nil {
		return wrapReleaseContract(ErrFmtDownloadIndex, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(i.GeneratedAt); err != nil {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	return nil
}

func validateUploadedSet(receipt UploadReceipt) error {
	if err := validateCoreArtifactSet(core.ArtifactSet[UploadedArtifact]{
		Items:      receipt.Objects,
		Count:      receipt.ObjectCount,
		TotalBytes: receipt.TotalBytes,
	}, ErrFmtUploadReceipt); err != nil {
		return err
	}
	for index, object := range receipt.Objects {
		if object.AttemptID != receipt.AttemptID {
			return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
		}
		if err := validateUploadBinding(UploadBindingInput{
			Product: receipt.Product, ReleaseID: receipt.ReleaseID, ManifestSHA256: receipt.ManifestSHA256,
			Artifact: object.Artifact, ArtifactSHA256: object.SHA256, ArtifactSize: object.Size,
			Provider: object.Provider, Bucket: object.Bucket, Object: object.Object, AttemptID: object.AttemptID,
		}, object.Binding, ErrFmtUploadReceipt); err != nil {
			return err
		}
		for _, prior := range receipt.Objects[:index] {
			if prior.Provider == object.Provider && prior.Object == object.Object {
				return fmt.Errorf(ErrFmtUploadReceipt, core.ErrReleaseContract)
			}
		}
	}
	return nil
}

func validateDownloadSet(downloads []Download, count uint32) error {
	if err := (core.CollectionCardinality{
		Length:          len(downloads),
		DeclaredCount:   count,
		Minimum:         1,
		Maximum:         core.CollectionMaximumDefault,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
	}
	for index, download := range downloads {
		if err := download.Validate(); err != nil {
			return err
		}
		if index > 0 && downloadSetKey(downloads[index-1]) >= downloadSetKey(download) {
			return fmt.Errorf(ErrFmtDownloadIndex, core.ErrReleaseContract)
		}
	}
	return nil
}

func downloadSetKey(download Download) string {
	return download.Provider.String() + "/" + download.Artifact.String()
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
