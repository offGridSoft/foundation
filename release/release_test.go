package release

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/core"
)

const (
	testVersionToken = "1.2.3"
	testReleaseToken = "2026-07-08-1.2.3"
	testDateToken    = "2026-07-08"
	testBucketToken  = "offgrid-release"
)

func TestEnumWireContractsHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run     func(*testing.T)
		name    string
		wantErr bool
	}{
		{name: "kind tool bundle wire token", run: requireKindJSON(KindToolBundle, "tool_bundle")},
		{name: "kind unknown rejects", run: enumMarshalFails(KindUnknown), wantErr: true},
		{name: "visibility public wire token", run: requireVisibilityJSON(VisibilityPublic, "public")},
		{name: "visibility unknown rejects", run: enumMarshalFails(VisibilityUnknown), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestObjectKeyContractHostileTable(t *testing.T) {
	t.Parallel()
	input := validObjectKeyInput(t)
	got, err := BuildObjectKey(input)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{core.ProductTokenWitness, testDateToken, testReleaseToken, ObjectSegmentPublic, ToolsArchiveName}, "/")
	if got.String() != want {
		t.Fatalf("BuildObjectKey() = %q, want %q", got.String(), want)
	}
	for _, tc := range []struct {
		mutate func(*ObjectKeyInput)
		name   string
	}{
		{name: "unknown product", mutate: func(i *ObjectKeyInput) { i.Product = core.ProductUnknown }},
		{name: "bad date", mutate: func(i *ObjectKeyInput) { i.Date = ReleaseDate{} }},
		{name: "bad release id", mutate: func(i *ObjectKeyInput) { i.ReleaseID = ReleaseID{} }},
		{name: "unknown visibility", mutate: func(i *ObjectKeyInput) { i.Visibility = VisibilityUnknown }},
		{name: "missing artifact", mutate: func(i *ObjectKeyInput) { i.Artifact = ArtifactName{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			next := input
			tc.mutate(&next)
			if _, err := BuildObjectKey(next); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("BuildObjectKey() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestArchiveLayoutHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*ArchiveLayout)
		name   string
	}{
		{name: "control valid", mutate: func(*ArchiveLayout) {}},
		{name: "bad archive name", mutate: func(l *ArchiveLayout) { l.Name = ArtifactName{} }},
		{name: "entry count mismatch", mutate: func(l *ArchiveLayout) { l.EntryCount++ }},
		{name: "duplicate entry name", mutate: func(l *ArchiveLayout) {
			l.Entries = append(l.Entries, l.Entries[0])
			l.EntryCount = uint32(len(l.Entries))
		}},
		{name: "attacker chosen mode", mutate: func(l *ArchiveLayout) { l.Entries[0].Mode = 0o777 }},
		{name: "entry limit above total limit", mutate: func(l *ArchiveLayout) {
			l.MaxEntryBytes = core.NewByteCount(2)
			l.MaxTotalBytes = core.NewByteCount(1)
		}},
		{name: "zero limits reject", mutate: func(l *ArchiveLayout) { l.MaxEntryBytes = core.ByteCount{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			layout := validArchiveLayout(t)
			tc.mutate(&layout)
			err := layout.Validate()
			if tc.name == "control valid" {
				if err != nil {
					t.Fatalf("ArchiveLayout.Validate() = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ArchiveLayout.Validate() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestManifestHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*Manifest)
		name   string
	}{
		{name: "control valid", mutate: func(*Manifest) {}},
		{name: "schema mismatch", mutate: func(m *Manifest) { m.Schema = core.SchemaReleaseDownloadIndex }},
		{name: "artifact count mismatch", mutate: func(m *Manifest) { m.ArtifactCount++ }},
		{name: "total bytes mismatch", mutate: func(m *Manifest) { m.TotalBytes = core.NewByteCount(99) }},
		{name: "duplicate artifact", mutate: func(m *Manifest) {
			m.Artifacts = append(m.Artifacts, m.Artifacts[0])
			m.ArtifactCount = uint32(len(m.Artifacts))
			m.TotalBytes = core.NewByteCount(24)
		}},
		{name: "binary missing platform", mutate: func(m *Manifest) { m.Artifacts[0].Platform = core.PlatformUnknown }},
		{name: "manifest artifact cannot carry platform", mutate: func(m *Manifest) {
			m.Artifacts[0].Kind = KindManifest
		}},
		{name: "artifact size overflow", mutate: func(m *Manifest) {
			m.Artifacts = []Artifact{
				validArtifactWithSize(t, "one.tar", math.MaxUint64),
				validArtifactWithSize(t, "two.tar", 2),
			}
			m.ArtifactCount = uint32(len(m.Artifacts))
			m.TotalBytes = core.NewByteCount(1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manifest := validManifest(t)
			tc.mutate(&manifest)
			err := manifest.Validate()
			if tc.name == "control valid" {
				if err != nil {
					t.Fatalf("Manifest.Validate() = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("Manifest.Validate() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestUploadAndDownloadHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "valid upload target", run: func() error { return validUploadTarget(t).Validate() }},
		{name: "unknown provider", run: func() error {
			target := validUploadTarget(t)
			target.Provider = core.StorageProviderUnknown
			return target.Validate()
		}},
		{name: "bad prefix", run: func() error {
			target := validUploadTarget(t)
			target.Prefix = ObjectKey{}
			return target.Validate()
		}},
		{name: "valid upload receipt", run: func() error { return validUploadReceipt(t).Validate() }},
		{name: "duplicate uploaded artifact", run: func() error {
			receipt := validUploadReceipt(t)
			receipt.Objects = append(receipt.Objects, receipt.Objects[0])
			receipt.ObjectCount = uint32(len(receipt.Objects))
			receipt.TotalBytes = core.NewByteCount(24)
			return receipt.Validate()
		}},
		{name: "valid download index", run: func() error { return validDownloadIndex(t).Validate() }},
		{name: "duplicate download platform", run: func() error {
			index := validDownloadIndex(t)
			index.Downloads = append(index.Downloads, index.Downloads[0])
			index.DownloadCount = uint32(len(index.Downloads))
			return index.Validate()
		}},
		{name: "download url query rejected", run: func() error {
			urlValue := validDownloadURL(t).String() + "?token=x"
			_, err := ParseDownloadURL(urlValue)
			return err
		}},
		{name: "download url missing path", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com")
			return err
		}},
		{name: "download url userinfo", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com@evil.example/witness/tools.tar.gz")
			return err
		}},
		{name: "download url control rune", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com/witness\nx")
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if strings.HasPrefix(tc.name, "valid ") {
				if err != nil {
					t.Fatalf("%s error = %v", tc.name, err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("%s error = %v, want release/foundation contract", tc.name, err)
			}
		})
	}
}

func TestReleasePlatformsPinned(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		platform core.Platform
		want     bool
	}{
		{name: "darwin amd64", platform: core.PlatformDarwinAMD64},
		{name: "darwin arm64", platform: core.PlatformDarwinARM64, want: true},
		{name: "linux amd64", platform: core.PlatformLinuxAMD64, want: true},
		{name: "linux arm64", platform: core.PlatformLinuxARM64, want: true},
		{name: "windows amd64", platform: core.PlatformWindowsAMD64, want: true},
		{name: "windows arm64", platform: core.PlatformWindowsARM64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.platform.IsReleaseTarget(); got != tc.want {
				t.Fatalf("IsReleaseTarget() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestVisibilityRejectsUnknownWireToken(t *testing.T) {
	t.Parallel()
	var got Visibility
	if err := got.UnmarshalJSON([]byte(`"customer"`)); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("Visibility.UnmarshalJSON() error = %v, want ErrReleaseContract", err)
	}
}

func TestReleaseScalarJSONRoundTrips(t *testing.T) {
	t.Parallel()
	roundTripBucket(t)
	roundTripObjectKey(t)
	roundTripDownloadURL(t)
}

func roundTripBucket(t *testing.T) {
	t.Helper()
	bucket := mustBucket(t)
	data, err := bucket.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Bucket
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != bucket || roundTrip.String() != testBucketToken {
		t.Fatalf("Bucket round trip = %s, want %s", roundTrip.String(), testBucketToken)
	}
}

func roundTripObjectKey(t *testing.T) {
	t.Helper()
	key, err := BuildObjectKey(validObjectKeyInput(t))
	if err != nil {
		t.Fatal(err)
	}
	data, err := key.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip ObjectKey
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != key {
		t.Fatalf("ObjectKey round trip = %s, want %s", roundTrip.String(), key.String())
	}
}

func roundTripDownloadURL(t *testing.T) {
	t.Helper()
	downloadURL := validDownloadURL(t)
	data, err := downloadURL.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip DownloadURL
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != downloadURL {
		t.Fatalf("DownloadURL round trip = %s, want %s", roundTrip.String(), downloadURL.String())
	}
}

func TestManifestCanonicalWireForm(t *testing.T) {
	t.Parallel()
	got, err := validManifest(t).Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"offgrid-release-manifest-v1","version":"1.2.3","release_id":"2026-07-08-1.2.3","date":"2026-07-08","commit":"` +
		strings.Repeat("a", 40) +
		`","artifacts":[{"name":"tools.tar.gz","sha256":"` +
		strings.Repeat("b", 64) +
		`","size_bytes":12,"kind":"tool_bundle","platform":"linux-amd64"}],"created_at":1782302400000000000,"total_bytes":12,"artifact_count":1,"product":"witness"}`
	if string(got) != want {
		t.Fatalf("Manifest.Canonical()\n got: %s\nwant: %s", got, want)
	}
}

func TestManifestCanonicalRoundTripTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		manifest Manifest
	}{
		{name: "manifest", manifest: validManifest(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			original, err := tc.manifest.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := core.DecodeStrictJSON[Manifest](original)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Fatalf("Manifest canonical round trip\n got: %s\nwant: %s", got, original)
			}
		})
	}
}

func requireKindJSON(value Kind, token string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		want := `"` + token + `"`
		if string(data) != want {
			t.Fatalf("MarshalJSON() = %s, want %s", data, want)
		}
		var decoded Kind
		if err := decoded.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("Kind.UnmarshalJSON() = %s, want %s", decoded, value)
		}
	}
}

func requireVisibilityJSON(value Visibility, token string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		want := `"` + token + `"`
		if string(data) != want {
			t.Fatalf("MarshalJSON() = %s, want %s", data, want)
		}
		var decoded Visibility
		if err := decoded.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("Visibility.UnmarshalJSON() = %s, want %s", decoded, value)
		}
	}
}

func enumMarshalFails[T json.Marshaler](value T) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		if _, err := value.MarshalJSON(); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("MarshalJSON() error = %v, want ErrReleaseContract", err)
		}
	}
}

func validObjectKeyInput(t *testing.T) ObjectKeyInput {
	t.Helper()
	return ObjectKeyInput{
		Product:    core.ProductWitness,
		Date:       mustReleaseDate(t),
		ReleaseID:  mustReleaseID(t),
		Visibility: VisibilityPublic,
		Artifact:   mustArtifactName(t, ToolsArchiveName),
	}
}

func validArchiveLayout(t *testing.T) ArchiveLayout {
	t.Helper()
	return ArchiveLayout{
		Name: mustArtifactName(t, ToolsArchiveName),
		Entries: []ArchiveEntry{
			{Name: mustArtifactName(t, ManifestFileName), Mode: ArchiveMetadataFileMode},
			{Name: mustArtifactName(t, "witness"), Mode: ArchiveToolFileMode},
		},
		EntryCount:    2,
		MaxEntryBytes: core.NewByteCount(ArchiveMaxEntryBytes),
		MaxTotalBytes: core.NewByteCount(ArchiveMaxTotalBytes),
	}
}

func validManifest(t *testing.T) Manifest {
	t.Helper()
	return Manifest{
		Schema:        core.SchemaReleaseManifest,
		Product:       core.ProductWitness,
		Version:       mustVersion(t),
		ReleaseID:     mustReleaseID(t),
		Date:          mustReleaseDate(t),
		Commit:        mustCommit(t),
		CreatedAt:     core.UnixNanoTimeFromInt64(1782302400000000000),
		Artifacts:     []Artifact{validArtifactWithSize(t, ToolsArchiveName, 12)},
		ArtifactCount: 1,
		TotalBytes:    core.NewByteCount(12),
	}
}

func validArtifactWithSize(t *testing.T, name string, size uint64) Artifact {
	t.Helper()
	return Artifact{
		Name:     mustArtifactName(t, name),
		Kind:     KindToolBundle,
		Platform: core.PlatformLinuxAMD64,
		SHA256:   mustSHA256(t, "b"),
		Size:     core.NewByteCount(size),
	}
}

func validUploadTarget(t *testing.T) UploadTarget {
	t.Helper()
	prefix, err := ParseObjectKey(strings.Join([]string{core.ProductTokenWitness, testDateToken, testReleaseToken}, "/"))
	if err != nil {
		t.Fatal(err)
	}
	return UploadTarget{
		Provider: core.StorageProviderGCS,
		Bucket:   mustBucket(t),
		Prefix:   prefix,
		Method:   core.UploadMethodSignedPUT,
	}
}

func validUploadReceipt(t *testing.T) UploadReceipt {
	t.Helper()
	object, err := BuildObjectKey(validObjectKeyInput(t))
	if err != nil {
		t.Fatal(err)
	}
	return UploadReceipt{
		Schema:     core.SchemaReleaseUploadReceipt,
		Product:    core.ProductWitness,
		Version:    mustVersion(t),
		ReleaseID:  mustReleaseID(t),
		UploadedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Objects: []UploadedArtifact{{
			Artifact: mustArtifactName(t, ToolsArchiveName),
			Object:   object,
			Provider: core.StorageProviderGCS,
			Bucket:   mustBucket(t),
			SHA256:   mustSHA256(t, "b"),
			Size:     core.NewByteCount(12),
		}},
		ObjectCount: 1,
		TotalBytes:  core.NewByteCount(12),
	}
}

func validDownloadIndex(t *testing.T) DownloadIndex {
	t.Helper()
	return DownloadIndex{
		Schema:      core.SchemaReleaseDownloadIndex,
		Product:     core.ProductWitness,
		Version:     mustVersion(t),
		ReleaseID:   mustReleaseID(t),
		GeneratedAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Downloads: []Download{{
			Platform: core.PlatformLinuxAMD64,
			Artifact: mustArtifactName(t, ToolsArchiveName),
			URL:      validDownloadURL(t),
			SHA256:   mustSHA256(t, "b"),
			Size:     core.NewByteCount(12),
		}},
		DownloadCount: 1,
	}
}

func validDownloadURL(t *testing.T) DownloadURL {
	t.Helper()
	u, err := ParseDownloadURL("https://downloads.offgridsoftware.com/witness/2026-07-08/2026-07-08-1.2.3/public/tools.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustVersion(t *testing.T) core.ProductVersion {
	t.Helper()
	version, err := core.ParseProductVersion(testVersionToken)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func mustReleaseID(t *testing.T) ReleaseID {
	t.Helper()
	id, err := ParseReleaseID(testReleaseToken)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustReleaseDate(t *testing.T) ReleaseDate {
	t.Helper()
	date, err := ParseReleaseDate(testDateToken)
	if err != nil {
		t.Fatal(err)
	}
	if got := NewReleaseDate(time.Date(2026, 7, 8, 12, 0, 0, 0, time.FixedZone("offset", -4*60*60))).String(); got != testDateToken {
		t.Fatalf("NewReleaseDate() = %s, want %s", got, testDateToken)
	}
	return date
}

func mustCommit(t *testing.T) core.BuildCommit {
	t.Helper()
	commit, err := core.ParseBuildCommit(strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	return commit
}

func mustArtifactName(t *testing.T, value string) ArtifactName {
	t.Helper()
	name, err := ParseArtifactName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func mustBucket(t *testing.T) Bucket {
	t.Helper()
	bucket, err := ParseBucket(testBucketToken)
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func mustSHA256(t *testing.T, digit string) core.SHA256Hex {
	t.Helper()
	sum, err := core.ParseSHA256Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
