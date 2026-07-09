package release

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	testVersionToken = core.FoundationVersion2026
	testReleaseToken = "2026-07-08-2026.0.0"
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
		run     func() error
		name    string
		wantErr bool
	}{
		{name: "valid upload target", run: func() error { return validUploadTarget(t).Validate() }},
		{name: "unknown provider", run: func() error {
			target := validUploadTarget(t)
			target.Provider = core.StorageProviderUnknown
			return target.Validate()
		}, wantErr: true},
		{name: "bad prefix", run: func() error {
			target := validUploadTarget(t)
			target.Prefix = ObjectKey{}
			return target.Validate()
		}, wantErr: true},
		{name: "valid upload receipt", run: func() error { return validUploadReceipt(t).Validate() }},
		{name: "duplicate uploaded artifact", run: func() error {
			receipt := validUploadReceipt(t)
			receipt.Objects = append(receipt.Objects, receipt.Objects[0])
			receipt.ObjectCount = uint32(len(receipt.Objects))
			receipt.TotalBytes = core.NewByteCount(24)
			return receipt.Validate()
		}, wantErr: true},
		{name: "valid download index", run: func() error { return validDownloadIndex(t).Validate() }},
		{name: "duplicate download platform", run: func() error {
			index := validDownloadIndex(t)
			index.Downloads = append(index.Downloads, index.Downloads[0])
			index.DownloadCount = uint32(len(index.Downloads))
			return index.Validate()
		}, wantErr: true},
		{name: "download url query rejected", run: func() error {
			urlValue := validDownloadURL(t).String() + "?token=x"
			_, err := ParseDownloadURL(urlValue)
			return err
		}, wantErr: true},
		{name: "download url missing path", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com")
			return err
		}, wantErr: true},
		{name: "download url userinfo", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com@evil.example/witness/tools.tar.gz")
			return err
		}, wantErr: true},
		{name: "download url control rune", run: func() error {
			_, err := ParseDownloadURL("https://downloads.offgridsoftware.com/witness\nx")
			return err
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if !tc.wantErr && err != nil {
				t.Fatalf("%s error = %v", tc.name, err)
				return
			}
			if !tc.wantErr {
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
	got := BuildPlatforms()
	want := []core.Platform{
		core.PlatformDarwinARM64,
		core.PlatformLinuxAMD64,
		core.PlatformLinuxARM64,
		core.PlatformWindowsAMD64,
	}
	if len(got) != len(want) {
		t.Fatalf("BuildPlatforms() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("BuildPlatforms()[%d] = %s, want %s", i, got[i], want[i])
		}
	}
	got[0] = core.PlatformUnknown
	if BuildPlatforms()[0] != core.PlatformDarwinARM64 {
		t.Fatalf("BuildPlatforms()[0] after caller mutation = %s, want %s", BuildPlatforms()[0], core.PlatformDarwinARM64)
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
	want := `{"schema":"offgrid-release-manifest-v1","version":"2026.0.0","release_id":"2026-07-08-2026.0.0","date":"2026-07-08","commit":"` +
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

func TestReleaseRootLayoutHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutateInput  func(*ReleaseRootInput)
		mutateLayout func(*testing.T, *ReleaseRootLayout)
		name         string
		wantError    bool
	}{
		{name: "valid release root"},
		{name: "unknown product", mutateInput: func(i *ReleaseRootInput) { i.Product = core.ProductUnknown }, wantError: true},
		{name: "bad version", mutateInput: func(i *ReleaseRootInput) { i.Version = core.ProductVersion{} }, wantError: true},
		{name: "bad date", mutateInput: func(i *ReleaseRootInput) { i.Date = ReleaseDate{value: "2026-7-8"} }, wantError: true},
		{name: "path escape release id", mutateInput: func(i *ReleaseRootInput) { i.ReleaseID = ReleaseID{value: ".."} }, wantError: true},
		{name: "tampered root path", mutateLayout: func(t *testing.T, l *ReleaseRootLayout) {
			l.Root = tamperedReleaseRootPath(t, "other")
		}, wantError: true},
		{name: "tampered private path", mutateLayout: func(t *testing.T, l *ReleaseRootLayout) {
			l.Private = tamperedReleaseRootPath(t, ObjectSegmentPrivate+"-copy")
		}, wantError: true},
		{name: "zero dogfood path", mutateLayout: func(_ *testing.T, l *ReleaseRootLayout) { l.Dogfood = BuildOutputPath{} }, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := validReleaseRootInput(t)
			if tc.mutateInput != nil {
				tc.mutateInput(&input)
			}
			layout, err := BuildReleaseRootLayout(input)
			if err == nil && tc.mutateLayout != nil {
				tc.mutateLayout(t, &layout)
				err = layout.Validate()
			}
			if !tc.wantError {
				if err != nil {
					t.Fatalf("ReleaseRootLayout valid case error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("ReleaseRootLayout hostile error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestReleaseRootLayoutCanonicalRoundTripTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		want  string
		input ReleaseRootInput
	}{
		{
			name:  "witness release root",
			input: validReleaseRootInput(t),
			want:  `{"product":"witness","version":"2026.0.0","date":"2026-07-08","release_id":"2026-07-08-2026.0.0","root":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0","private":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/private","public":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/public","platforms":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/platforms","receipts":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/receipts","manifests":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/manifests","dogfood":"dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/dogfood"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			layout, err := BuildReleaseRootLayout(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := layout.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("ReleaseRootLayout.Canonical()\n got: %s\nwant: %s", got, tc.want)
			}
			decoded, err := core.DecodeStrictJSON[ReleaseRootLayout](got)
			if err != nil {
				t.Fatal(err)
			}
			roundTrip, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(roundTrip) != string(got) {
				t.Fatalf("ReleaseRootLayout canonical round trip\n got: %s\nwant: %s", roundTrip, got)
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

func requireCommandKindJSON(value CommandKind, token string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		want := `"` + token + `"`
		if string(data) != want {
			t.Fatalf("CommandKind.MarshalJSON() = %s, want %s", data, want)
		}
		var decoded CommandKind
		if err := decoded.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("CommandKind.UnmarshalJSON() = %s, want %s", decoded, value)
		}
	}
}

func requireCommandStatusJSON(value CommandStatus, token string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		want := `"` + token + `"`
		if string(data) != want {
			t.Fatalf("CommandStatus.MarshalJSON() = %s, want %s", data, want)
		}
		var decoded CommandStatus
		if err := decoded.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("CommandStatus.UnmarshalJSON() = %s, want %s", decoded, value)
		}
	}
}

func requireTreeStateJSON(value TreeState, token string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		data, err := value.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		want := `"` + token + `"`
		if string(data) != want {
			t.Fatalf("TreeState.MarshalJSON() = %s, want %s", data, want)
		}
		var decoded TreeState
		if err := decoded.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if decoded != value {
			t.Fatalf("TreeState.UnmarshalJSON() = %s, want %s", decoded, value)
		}
	}
}

func commandKindJSONFails(data string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		var decoded CommandKind
		if err := decoded.UnmarshalJSON([]byte(data)); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("CommandKind.UnmarshalJSON() error = %v, want ErrReleaseContract", err)
		}
	}
}

func commandStatusJSONFails(data string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		var decoded CommandStatus
		if err := decoded.UnmarshalJSON([]byte(data)); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("CommandStatus.UnmarshalJSON() error = %v, want ErrReleaseContract", err)
		}
	}
}

func treeStateJSONFails(data string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		var decoded TreeState
		if err := decoded.UnmarshalJSON([]byte(data)); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("TreeState.UnmarshalJSON() error = %v, want ErrReleaseContract", err)
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

func TestReleaseSignerHostileTable(t *testing.T) {
	t.Parallel()

	validSignature, err := core.NewEd25519SignatureHex(make([]byte, ed25519.SignatureSize))
	if err != nil {
		t.Fatalf("NewEd25519SignatureHex(valid) error = %v", err)
	}
	signerErr := errors.New("release signer failed")
	var _ ReleaseSigner = testReleaseSigner{}

	for _, tc := range []struct {
		signer  ReleaseSigner
		wantErr error
		name    string
		payload []byte
	}{
		{name: "valid signer returns typed signature", signer: testReleaseSigner{signature: validSignature}, payload: []byte("manifest"), wantErr: nil},
		{name: "nil signer rejected", payload: []byte("manifest"), wantErr: core.ErrReleaseContract},
		{name: "empty payload rejected", signer: testReleaseSigner{signature: validSignature}, payload: nil, wantErr: core.ErrReleaseContract},
		{name: "signer error preserved", signer: testReleaseSigner{err: signerErr}, payload: []byte("manifest"), wantErr: signerErr},
		{name: "zero signature rejected", signer: testReleaseSigner{}, payload: []byte("manifest"), wantErr: core.ErrFoundationContract},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := SignReleasePayload(tc.signer, tc.payload)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("SignReleasePayload() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("SignReleasePayload() error = %v", err)
			}
			if got != validSignature {
				t.Fatalf("SignReleasePayload() = %q, want %q", got.String(), validSignature.String())
			}
		})
	}
}

type testReleaseSigner struct {
	err       error
	signature core.Ed25519SignatureHex
}

func (s testReleaseSigner) SignRelease([]byte) (core.Ed25519SignatureHex, error) {
	return s.signature, s.err
}

func TestGarbleSeedHostileTable(t *testing.T) {
	t.Parallel()
	const exactSeed = "AQIDBAUGBwg="
	const longSeed = "AQIDBAUGBwgJCg=="
	for _, tc := range []struct {
		wantError error
		name      string
		value     string
		want      string
		required  bool
	}{
		{name: "empty seed is random for development build", value: "", want: GarbleSeedRandom},
		{name: "whitespace seed is random for development build", value: " \t\n ", want: GarbleSeedRandom},
		{name: "explicit random seed is random", value: GarbleSeedRandom, want: GarbleSeedRandom},
		{name: "concrete seed with surrounding whitespace normalizes", value: " " + exactSeed + "\n", want: exactSeed, required: true},
		{name: "exact base64 seed accepted", value: exactSeed, want: exactSeed, required: true},
		{name: "long decoded seed rejected instead of truncated", value: longSeed, required: true, wantError: core.ErrReleaseContract},
		{name: "required rejects empty random", value: "", required: true, wantError: core.ErrReleaseContract},
		{name: "required rejects whitespace random", value: "\n\t", required: true, wantError: core.ErrReleaseContract},
		{name: "short decoded seed rejected", value: "AQIDBA==", wantError: core.ErrReleaseContract},
		{name: "malformed base64 rejected", value: "not-base64", wantError: core.ErrReleaseContract},
		{name: "oversize attacker seed rejected", value: strings.Repeat("A", GarbleSeedMaxInputBytes+1), wantError: core.ErrReleaseContract},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			parse := ParseGarbleSeed
			if tc.required {
				parse = ParseRequiredGarbleSeed
			}
			got, err := parse(tc.value)
			if tc.wantError != nil {
				if !errors.Is(err, tc.wantError) {
					t.Fatalf("parse seed error = %v, want %v", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse seed error = %v", err)
			}
			if got.String() != tc.want {
				t.Fatalf("parse seed = %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestGarbleSeedZeroValueHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "zero seed reports random so required gates reject it", run: func(t *testing.T) {
			t.Helper()
			seed := GarbleSeed{}
			if !seed.IsRandom() {
				t.Fatalf("GarbleSeed{}.IsRandom() = false, want true")
			}
			if err := validateRequiredReleaseSeed(seed, ErrFmtReleasePlan); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("validateRequiredReleaseSeed(GarbleSeed{}) error = %v, want ErrReleaseContract", err)
			}
		}},
		{name: "zero seed canonical JSON emits random token", run: func(t *testing.T) {
			t.Helper()
			data, err := (GarbleSeed{}).MarshalJSON()
			if err != nil {
				t.Fatalf("GarbleSeed{}.MarshalJSON() error = %v", err)
			}
			if string(data) != `"`+GarbleSeedRandom+`"` {
				t.Fatalf("GarbleSeed{}.MarshalJSON() = %s, want %q", data, GarbleSeedRandom)
			}
			var decoded GarbleSeed
			if err := decoded.UnmarshalJSON(data); err != nil {
				t.Fatalf("GarbleSeed.UnmarshalJSON(%s) error = %v", data, err)
			}
			if !decoded.IsRandom() || decoded.String() != GarbleSeedRandom {
				t.Fatalf("decoded zero seed = %q random=%v, want random token", decoded.String(), decoded.IsRandom())
			}
		}},
		{name: "zero seed build args cannot emit blank garble seed flag", run: func(t *testing.T) {
			t.Helper()
			request := validWitnessBuildRequest(t, mustGarbleSeed(t))
			request.Seed = GarbleSeed{}
			got, err := GarbleBuildArgs(request)
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleBuildArgs(zero seed) error = %v, want ErrReleaseContract", err)
			}
			if got != nil {
				t.Fatalf("GarbleBuildArgs(zero seed) = %#v, want nil args", got)
			}
		}},
		{name: "zero seed release matrix refuses before output planning", run: func(t *testing.T) {
			t.Helper()
			got, err := validWitnessReleaseSpec(t).GarbleBuildRequests(GarbleSeed{}, validWitnessReleaseRootLayout(t))
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleBuildRequests(zero seed) error = %v, want ErrReleaseContract", err)
			}
			if got != nil {
				t.Fatalf("GarbleBuildRequests(zero seed) = %#v, want nil requests", got)
			}
		}},
		{name: "zero seed preflight rejects release", run: func(t *testing.T) {
			t.Helper()
			input := validReleasePreflightInput(t)
			input.Seed = GarbleSeed{}
			if err := Preflight(input); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("Preflight(zero seed) error = %v, want ErrReleaseContract", err)
			}
		}},
		{name: "zero seed signed release plan rejects release", run: func(t *testing.T) {
			t.Helper()
			plan := validReleasePlan(t)
			plan.Seed = GarbleSeed{}
			if err := plan.Validate(); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ReleasePlan.Validate(zero seed) error = %v, want ErrReleaseContract", err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestReleasePlanningJSONHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "garble seed round trip", run: func(t *testing.T) {
			t.Helper()
			seed := mustGarbleSeed(t)
			data, err := seed.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var decoded GarbleSeed
			if err := decoded.UnmarshalJSON(data); err != nil {
				t.Fatal(err)
			}
			if decoded != seed {
				t.Fatalf("GarbleSeed JSON round trip = %q, want %q", decoded.String(), seed.String())
			}
		}},
		{name: "zero garble seed JSON normalizes to random", run: func(t *testing.T) {
			t.Helper()
			data, err := (GarbleSeed{}).MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != `"`+GarbleSeedRandom+`"` {
				t.Fatalf("GarbleSeed{} JSON = %s, want random token", data)
			}
			var decoded GarbleSeed
			if err := decoded.UnmarshalJSON(data); err != nil {
				t.Fatal(err)
			}
			roundTrip, err := decoded.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if string(roundTrip) != string(data) {
				t.Fatalf("GarbleSeed zero JSON round trip = %s, want %s", roundTrip, data)
			}
		}},
		{name: "garble seed wrong JSON shape", run: func(t *testing.T) {
			t.Helper()
			var decoded GarbleSeed
			if err := decoded.UnmarshalJSON([]byte(`7`)); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleSeed.UnmarshalJSON() error = %v, want ErrReleaseContract", err)
			}
		}},
		{name: "command kind round trip", run: requireCommandKindJSON(CommandKindRelease, CommandKindTokenRelease)},
		{name: "command kind unknown token", run: commandKindJSONFails(`"ship"`)},
		{name: "command status round trip", run: requireCommandStatusJSON(CommandStatusSucceeded, CommandStatusTokenSucceeded)},
		{name: "command status wrong shape", run: commandStatusJSONFails(`7`)},
		{name: "tree state round trip", run: requireTreeStateJSON(TreeStateClean, TreeStateTokenClean)},
		{name: "tree state unknown token", run: treeStateJSONFails(`"maybe"`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestReleaseContractJSONOwnershipTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "build import path marshals and unmarshals", run: func(t *testing.T) {
			requireJSONRoundTrip(t, mustBuildImportPath(t, "./cmd/witness"), new(BuildImportPath))
		}},
		{name: "build tag marshals and unmarshals", run: func(t *testing.T) {
			requireJSONRoundTrip(t, mustBuildTag(t, "netgo"), new(BuildTag))
		}},
		{name: "linker symbol marshals and unmarshals", run: func(t *testing.T) {
			requireJSONRoundTrip(t, mustLinkerSymbol(t, "github.com/offGridSoft/witness/internal/release.BuildCommit"), new(LinkerSymbol))
		}},
		{name: "commit stamp marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validWitnessBuildPolicy(t).CommitStamp)
		}},
		{name: "build policy marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validWitnessBuildPolicy(t))
		}},
		{name: "release command marshals", run: func(t *testing.T) {
			requireMarshalOK(t, mustReleaseCommand(t, core.ProductTokenWitness, "./cmd/witness"))
		}},
		{name: "garble build request marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validWitnessBuildRequest(t, mustGarbleSeed(t)))
		}},
		{name: "product release spec marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validWitnessReleaseSpec(t))
		}},
		{name: "release plan marshals and canonicalizes", run: func(t *testing.T) {
			plan := validReleasePlan(t)
			requireMarshalOK(t, plan)
			if _, err := plan.Canonical(nil); err != nil {
				t.Fatalf("ReleasePlan.Canonical() error = %v", err)
			}
		}},
		{name: "build toolchain marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validBuildToolchain(t))
		}},
		{name: "vuln db snapshot marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validVulnDBSnapshot(t))
		}},
		{name: "release gate evidence marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validReleaseGateEvidence(t))
		}},
		{name: "tool provenance marshals", run: func(t *testing.T) {
			requireMarshalOK(t, validToolProvenance(t)[0])
		}},
		{name: "evidence ref string marshals and unmarshals", run: func(t *testing.T) {
			ref := mustEvidenceRef(t, "witness://release/final/green")
			if ref.String() == "" {
				t.Fatalf("EvidenceRef.String() blank")
			}
			requireJSONRoundTrip(t, ref, new(EvidenceRef))
		}},
		{name: "seed ref string marshals and unmarshals", run: func(t *testing.T) {
			ref := mustGarbleSeedRef(t, "release/2026/07/08/seed")
			if ref.String() == "" {
				t.Fatalf("GarbleSeedRef.String() blank")
			}
			requireJSONRoundTrip(t, ref, new(GarbleSeedRef))
		}},
		{name: "tool version string marshals and unmarshals", run: func(t *testing.T) {
			version := mustToolVersion(t, "v2026.0.0")
			if version.String() == "" {
				t.Fatalf("ToolVersion.String() blank")
			}
			requireJSONRoundTrip(t, version, new(ToolVersion))
		}},
		{name: "tool module marshals and unmarshals", run: func(t *testing.T) {
			requireJSONRoundTrip(t, mustToolModule(t, "github.com/offGridSoft/deadcode"), new(ToolModule))
		}},
		{name: "go sum string marshals and unmarshals", run: func(t *testing.T) {
			sum := mustGoSumHash(t, "h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=")
			if sum.String() == "" {
				t.Fatalf("GoSumHash.String() blank")
			}
			requireJSONRoundTrip(t, sum, new(GoSumHash))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestBuildImportPathHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantError error
		name      string
		value     string
		want      string
	}{
		{name: "product command accepted", value: "./cmd/witness", want: "./cmd/witness"},
		{name: "release helper command accepted", value: "./cmd/witness-release", want: "./cmd/witness-release"},
		{name: "nested command package accepted", value: "./cmd/witness/internal", want: "./cmd/witness/internal"},
		{name: "missing command after prefix rejected", value: "./cmd/", wantError: core.ErrReleaseContract},
		{name: "traversal immediately after cmd rejected", value: "./cmd/../internal/steal", wantError: core.ErrReleaseContract},
		{name: "deep traversal after product rejected", value: "./cmd/witness/../../internal/steal", wantError: core.ErrReleaseContract},
		{name: "current directory segment rejected", value: "./cmd/./witness", wantError: core.ErrReleaseContract},
		{name: "double slash segment rejected", value: "./cmd//witness", wantError: core.ErrReleaseContract},
		{name: "trailing slash rejected", value: "./cmd/witness/", wantError: core.ErrReleaseContract},
		{name: "missing cmd prefix rejected", value: "./internal/witness", wantError: core.ErrReleaseContract},
		{name: "absolute path rejected", value: "/cmd/witness", wantError: core.ErrReleaseContract},
		{name: "backslash traversal rejected", value: `./cmd\..\internal\steal`, wantError: core.ErrReleaseContract},
		{name: "embedded space rejected", value: "./cmd/wit ness", wantError: core.ErrReleaseContract},
		{name: "newline rejected", value: "./cmd/witness\nshadow", wantError: core.ErrReleaseContract},
		{name: "oversized path rejected", value: "./cmd/" + strings.Repeat("a", BuildImportPathMaxRunes), wantError: core.ErrReleaseContract},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBuildImportPath(tc.value)
			if tc.wantError != nil {
				if !errors.Is(err, tc.wantError) {
					t.Fatalf("ParseBuildImportPath(%q) error = %v, want %v", tc.value, err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBuildImportPath(%q) error = %v", tc.value, err)
			}
			if got.String() != tc.want {
				t.Fatalf("ParseBuildImportPath(%q) = %q, want %q", tc.value, got.String(), tc.want)
			}
		})
	}
}

func TestBuildImportPathTraversalCannotReachBuildArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "constructor rejects traversal import path", run: func(t *testing.T) {
			t.Helper()
			if _, err := NewReleaseCommand("witness", "./cmd/../../internal/steal"); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("NewReleaseCommand(traversal) error = %v, want ErrReleaseContract", err)
			}
		}},
		{name: "forged traversal import path rejected before garble args", run: func(t *testing.T) {
			t.Helper()
			request := validWitnessBuildRequest(t, mustGarbleSeed(t))
			request.Command.ImportPath = BuildImportPath{value: "./cmd/../../internal/steal"}
			got, err := GarbleBuildArgs(request)
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleBuildArgs(traversal import path) error = %v, want ErrReleaseContract", err)
			}
			if got != nil {
				t.Fatalf("GarbleBuildArgs(traversal import path) = %#v, want nil args", got)
			}
		}},
		{name: "forged current directory import path rejected before garble args", run: func(t *testing.T) {
			t.Helper()
			request := validWitnessBuildRequest(t, mustGarbleSeed(t))
			request.Command.ImportPath = BuildImportPath{value: "./cmd/./witness"}
			got, err := GarbleBuildArgs(request)
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleBuildArgs(current-directory import path) error = %v, want ErrReleaseContract", err)
			}
			if got != nil {
				t.Fatalf("GarbleBuildArgs(current-directory import path) = %#v, want nil args", got)
			}
		}},
		{name: "forged double slash import path rejected before garble args", run: func(t *testing.T) {
			t.Helper()
			request := validWitnessBuildRequest(t, mustGarbleSeed(t))
			request.Command.ImportPath = BuildImportPath{value: "./cmd//witness"}
			got, err := GarbleBuildArgs(request)
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("GarbleBuildArgs(double-slash import path) error = %v, want ErrReleaseContract", err)
			}
			if got != nil {
				t.Fatalf("GarbleBuildArgs(double-slash import path) = %#v, want nil args", got)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func TestBuildTagRejectsSmugglingHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "comma cannot inject second tag", value: "netgo,debug"},
		{name: "interior space cannot split tag", value: "netgo debug"},
		{name: "tab cannot split tag", value: "netgo\tdebug"},
		{name: "newline cannot split tag", value: "netgo\ndebug"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseBuildTag(tc.value); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ParseBuildTag(%q) error = %v, want ErrReleaseContract", tc.value, err)
			}
		})
	}
}

func TestLinkerSymbolRejectsStampCorruptionHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "equals cannot corrupt ldflags assignment", value: "github.com/offGridSoft/witness/internal/release.BuildCommit=evil"},
		{name: "space cannot split ldflags assignment", value: "github.com/offGridSoft/witness/internal/release.Build Commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseLinkerSymbol(tc.value); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ParseLinkerSymbol(%q) error = %v, want ErrReleaseContract", tc.value, err)
			}
		})
	}
}

func TestReleaseBuildSpecHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*ProductReleaseSpec)
		name    string
		wantErr bool
	}{
		{name: "bug one command valid", mutate: func(s *ProductReleaseSpec) {
			*s = validBugReleaseSpec(t)
		}},
		{name: "witness multiple commands valid", mutate: func(*ProductReleaseSpec) {}},
		{name: "unknown product", wantErr: true, mutate: func(s *ProductReleaseSpec) { s.Product = core.ProductUnknown }},
		{name: "bad version", wantErr: true, mutate: func(s *ProductReleaseSpec) { s.Version = core.ProductVersion{} }},
		{name: "missing command", mutate: func(s *ProductReleaseSpec) {
			s.Commands = nil
			s.CommandCount = 0
		}, wantErr: true},
		{name: "command count mismatch", wantErr: true, mutate: func(s *ProductReleaseSpec) { s.CommandCount++ }},
		{name: "duplicate command name", mutate: func(s *ProductReleaseSpec) {
			s.Commands[1].Name = s.Commands[0].Name
		}, wantErr: true},
		{name: "duplicate import path", mutate: func(s *ProductReleaseSpec) {
			s.Commands[1].ImportPath = s.Commands[0].ImportPath
		}, wantErr: true},
		{name: "unsupported release platform", mutate: func(s *ProductReleaseSpec) {
			s.Platforms[0] = core.PlatformDarwinAMD64
		}, wantErr: true},
		{name: "duplicate platform", mutate: func(s *ProductReleaseSpec) {
			s.Platforms[1] = s.Platforms[0]
		}, wantErr: true},
		{name: "platform count mismatch", wantErr: true, mutate: func(s *ProductReleaseSpec) { s.PlatformCount++ }},
		{name: "non-stripped build rejected", wantErr: true, mutate: func(s *ProductReleaseSpec) { s.Policy.Strip = false }},
		{name: "duplicate build tag", mutate: func(s *ProductReleaseSpec) {
			s.Policy.Tags = append(s.Policy.Tags, s.Policy.Tags[0])
			s.Policy.TagCount = uint32(len(s.Policy.Tags))
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := validWitnessReleaseSpec(t)
			tc.mutate(&spec)
			err := spec.Validate()
			if !tc.wantErr && err != nil {
				t.Fatalf("ProductReleaseSpec.Validate() = %v", err)
				return
			}
			if !tc.wantErr {
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ProductReleaseSpec.Validate() error = %v, want ErrReleaseContract", err)
			}
		})
	}
}

func TestGarbleBuildArgsHostileTable(t *testing.T) {
	t.Parallel()
	seed := mustGarbleSeed(t)
	for _, tc := range []struct {
		mutate func(*ReleaseGarbleBuildRequest)
		name   string
		want   []string
	}{
		{name: "bug build args pinned", mutate: func(r *ReleaseGarbleBuildRequest) {
			*r = validBugBuildRequest(t, seed)
		}, want: []string{
			GarbleSeedFlagPrefix + seed.String(),
			GarbleArgLiterals,
			GarbleArgTiny,
			GoArgBuild,
			GoArgTrimPath,
			GoBuildLDFlagsPrefix + "-s -w",
			GoBuildOutputFlag,
			"dist/linux-amd64/bug",
			"./cmd/bug",
		}},
		{name: "witness build args pinned", mutate: func(*ReleaseGarbleBuildRequest) {}, want: []string{
			GarbleSeedFlagPrefix + seed.String(),
			GarbleArgLiterals,
			GarbleArgTiny,
			GoArgBuild,
			GoArgTrimPath,
			GoArgBuildVCS,
			GoBuildTagsPrefix + "osusergo,netgo,witness_production",
			GoBuildLDFlagsPrefix + "-s -w -buildid= -X github.com/offGridSoft/witness/internal/release.BuildCommit=" + strings.Repeat("a", 40),
			GoBuildOutputFlag,
			"dist/linux-amd64/witness",
			"./cmd/witness",
		}},
		{name: "absolute output path accepted for release temp roots", mutate: func(r *ReleaseGarbleBuildRequest) {
			r.Output = mustBuildOutputPath(t, "/private/tmp/witness-release/linux-amd64/witness")
		}, want: []string{
			GarbleSeedFlagPrefix + seed.String(),
			GarbleArgLiterals,
			GarbleArgTiny,
			GoArgBuild,
			GoArgTrimPath,
			GoArgBuildVCS,
			GoBuildTagsPrefix + "osusergo,netgo,witness_production",
			GoBuildLDFlagsPrefix + "-s -w -buildid= -X github.com/offGridSoft/witness/internal/release.BuildCommit=" + strings.Repeat("a", 40),
			GoBuildOutputFlag,
			"/private/tmp/witness-release/linux-amd64/witness",
			"./cmd/witness",
		}},
		{name: "random seed cannot build release", mutate: func(r *ReleaseGarbleBuildRequest) {
			r.Seed = GarbleSeed{value: GarbleSeedRandom}
		}},
		{name: "bad output path rejected", mutate: func(r *ReleaseGarbleBuildRequest) {
			r.Output = BuildOutputPath{value: "../dist/bug"}
		}},
		{name: "unsupported platform rejected", mutate: func(r *ReleaseGarbleBuildRequest) {
			r.Platform = core.PlatformWindowsARM64
		}},
		{name: "bad command import path rejected", mutate: func(r *ReleaseGarbleBuildRequest) {
			r.Command.ImportPath = BuildImportPath{value: "./internal/bug"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := validWitnessBuildRequest(t, seed)
			tc.mutate(&request)
			got, err := GarbleBuildArgs(request)
			if tc.want == nil {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("GarbleBuildArgs() error = %v, want ErrReleaseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GarbleBuildArgs() error = %v", err)
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Fatalf("GarbleBuildArgs()\n got: %#v\nwant: %#v", got, tc.want)
			}
		})
	}
}

func TestProductReleaseSpecBuildRequestsHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate    func(*ProductReleaseSpec, *GarbleSeed, *ReleaseRootLayout)
		name      string
		wantCount int
	}{
		{name: "bug matrix expands one command across release platforms", wantCount: len(BuildPlatforms()), mutate: func(s *ProductReleaseSpec, _ *GarbleSeed, layout *ReleaseRootLayout) {
			*s = validBugReleaseSpec(t)
			*layout = validBugReleaseRootLayout(t)
		}},
		{name: "witness matrix expands all commands", wantCount: len(BuildPlatforms()) * 2, mutate: func(*ProductReleaseSpec, *GarbleSeed, *ReleaseRootLayout) {}},
		{name: "random seed rejected before output planning", mutate: func(_ *ProductReleaseSpec, seed *GarbleSeed, _ *ReleaseRootLayout) {
			*seed = GarbleSeed{value: GarbleSeedRandom}
		}},
		{name: "tampered release root rejected", mutate: func(_ *ProductReleaseSpec, _ *GarbleSeed, layout *ReleaseRootLayout) {
			layout.Platforms = mustBuildOutputPath(t, "dist/wrong")
		}},
		{name: "layout product drift rejected", mutate: func(_ *ProductReleaseSpec, _ *GarbleSeed, layout *ReleaseRootLayout) {
			layout.Product = core.ProductBug
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := validWitnessReleaseSpec(t)
			seed := mustGarbleSeed(t)
			layout := validWitnessReleaseRootLayout(t)
			tc.mutate(&spec, &seed, &layout)
			got, err := spec.GarbleBuildRequests(seed, layout)
			if tc.wantCount == 0 {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("GarbleBuildRequests() error = %v, want ErrReleaseContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GarbleBuildRequests() error = %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("GarbleBuildRequests() len = %d, want %d", len(got), tc.wantCount)
			}
			wantSuffix := "dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/platforms/windows-amd64/witness-sign.exe"
			if got[len(got)-1].Output.String() != wantSuffix && spec.Product == core.ProductWitness {
				t.Fatalf("last witness output = %q, want windows executable suffix", got[len(got)-1].Output.String())
			}
		})
	}
}

func TestReleaseMetadataScalarsHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run     func() error
		name    string
		wantErr bool
	}{
		{name: "evidence ref accepts witness uri", run: func() error { _, err := ParseEvidenceRef("witness://release/final/gate"); return err }},
		{name: "evidence ref rejects newline", wantErr: true, run: func() error { _, err := ParseEvidenceRef("witness://release\nshadow"); return err }},
		{name: "seed ref rejects embedded space", wantErr: true, run: func() error { _, err := ParseGarbleSeedRef("release seed/path"); return err }},
		{name: "tool version rejects tab", wantErr: true, run: func() error { _, err := ParseToolVersion("go1.25\tshadow"); return err }},
		{name: "tool module rejects space", wantErr: true, run: func() error { _, err := ParseToolModule("github.com/off Grid/tool"); return err }},
		{name: "go sum rejects missing hash prefix", wantErr: true, run: func() error { _, err := ParseGoSumHash("sha256:abc"); return err }},
		{name: "go sum accepts module hash", run: func() error { _, err := ParseGoSumHash("h1:abcdefghijklmnopqrstuvwxyz0123456789+/="); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if !tc.wantErr && err != nil {
				t.Fatalf("%s error = %v", tc.name, err)
				return
			}
			if !tc.wantErr {
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("%s error = %v, want release/foundation contract", tc.name, err)
			}
		})
	}
}

func TestReleasePreflightHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*ReleasePreflightInput)
		name    string
		wantErr bool
	}{
		{name: "clean requested commit with release seed accepted", mutate: func(*ReleasePreflightInput) {}},
		{name: "dirty tree rejected before build", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.TreeState = TreeStateDirty }},
		{name: "head commit drift rejected", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.HeadCommit = mustOtherCommit(t) }},
		{name: "random seed rejected", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.Seed = GarbleSeed{value: GarbleSeedRandom} }},
		{name: "bad go version rejected", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.Toolchain.GoVersion = ToolVersion{value: "go1.25\nshadow"} }},
		{name: "zero vuln snapshot rejected", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.VulnDB.SnapshotAt = core.UnixNanoTime{} }},
		{name: "blank final gate ref rejected", wantErr: true, mutate: func(in *ReleasePreflightInput) { in.Evidence.FinalEvidenceRef = EvidenceRef{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := validReleasePreflightInput(t)
			tc.mutate(&input)
			err := Preflight(input)
			if !tc.wantErr && err != nil {
				t.Fatalf("Preflight() = %v", err)
				return
			}
			if !tc.wantErr {
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Preflight() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestReleasePlanHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*ReleasePlan)
		name    string
		wantErr bool
	}{
		{name: "valid witness plan accepted", mutate: func(*ReleasePlan) {}},
		{name: "layout product drift rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Layout.Product = core.ProductBug }},
		{name: "spec version drift rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Spec.Version = mustOtherVersion(t) }},
		{name: "commit stamp drift rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Spec.Policy.CommitStamp.Commit = mustOtherCommit(t) }},
		{name: "bad seed ref rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.SeedRef = GarbleSeedRef{value: "seed ref"} }},
		{name: "garble random seed rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Seed = GarbleSeed{value: GarbleSeedRandom} }},
		{name: "tool duplicate rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Tools[1].Module = p.Tools[0].Module }},
		{name: "tool unsorted rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Tools[0], p.Tools[1] = p.Tools[1], p.Tools[0] }},
		{name: "tool count mismatch rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.ToolCount++ }},
		{name: "fast gate ref newline rejected", wantErr: true, mutate: func(p *ReleasePlan) { p.Evidence.FastGateRef = EvidenceRef{value: "fast\nshadow"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := validReleasePlan(t)
			tc.mutate(&plan)
			err := plan.Validate()
			if !tc.wantErr && err != nil {
				t.Fatalf("ReleasePlan.Validate() = %v", err)
				return
			}
			if !tc.wantErr {
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("ReleasePlan.Validate() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestReleasePlanGarbleBuildRequestsUseReleaseRootHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		mutate     func(*ReleasePlan)
		wantOutput string
		wantCount  int
	}{
		{
			name:       "release root produces windows executable output",
			mutate:     func(*ReleasePlan) {},
			wantCount:  len(BuildPlatforms()) * 2,
			wantOutput: "dist/releases/witness/2026/07/08/2026-07-08-2026.0.0/platforms/windows-amd64/witness-sign.exe",
		},
		{name: "tampered platforms dir rejected", mutate: func(p *ReleasePlan) { p.Layout.Platforms = mustBuildOutputPath(t, "dist/platforms") }},
		{name: "unsupported platform rejected", mutate: func(p *ReleasePlan) { p.Spec.Platforms[0] = core.PlatformDarwinAMD64 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := validReleasePlan(t)
			tc.mutate(&plan)
			got, err := plan.GarbleBuildRequests()
			if tc.wantCount == 0 {
				if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
					t.Fatalf("GarbleBuildRequests() error = %v, want release/foundation contract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GarbleBuildRequests() error = %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("GarbleBuildRequests() len = %d, want %d", len(got), tc.wantCount)
			}
			if got[len(got)-1].Output.String() != tc.wantOutput {
				t.Fatalf("last output = %q, want %q", got[len(got)-1].Output.String(), tc.wantOutput)
			}
		})
	}
}

func TestDeployPlanHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*DeployPlan)
		name    string
		wantErr bool
	}{
		{name: "valid deploy plan accepted", mutate: func(*DeployPlan) {}},
		{name: "manifest product drift rejected", wantErr: true, mutate: func(p *DeployPlan) { p.Manifest.Product = core.ProductBug }},
		{name: "layout release id drift rejected", wantErr: true, mutate: func(p *DeployPlan) { p.Layout.ReleaseID = mustOtherReleaseID(t) }},
		{name: "manifest sha missing rejected", wantErr: true, mutate: func(p *DeployPlan) { p.ManifestSHA256 = core.SHA256Hex{} }},
		{name: "duplicate upload target rejected", mutate: func(p *DeployPlan) {
			p.Targets = append(p.Targets, p.Targets[0])
			p.TargetCount = uint32(len(p.Targets))
		}, wantErr: true},
		{name: "target count mismatch rejected", wantErr: true, mutate: func(p *DeployPlan) { p.TargetCount++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := validDeployPlan(t)
			tc.mutate(&plan)
			err := plan.Validate()
			if !tc.wantErr && err != nil {
				t.Fatalf("DeployPlan.Validate() = %v", err)
				return
			}
			if !tc.wantErr {
				if _, err := plan.MarshalJSON(); err != nil {
					t.Fatalf("DeployPlan.MarshalJSON() = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployPlan.Validate() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestCommandRunHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate  func(*CommandRun)
		name    string
		wantErr bool
	}{
		{name: "release run valid", mutate: func(*CommandRun) {}},
		{name: "deploy run valid", mutate: func(r *CommandRun) { r.Kind = CommandKindDeploy }},
		{name: "dirty tree run valid", mutate: func(r *CommandRun) { r.TreeState = TreeStateDirty }},
		{name: "unknown kind", wantErr: true, mutate: func(r *CommandRun) { r.Kind = CommandKindUnknown }},
		{name: "unknown status", wantErr: true, mutate: func(r *CommandRun) { r.Status = CommandStatusUnknown }},
		{name: "unknown tree state", wantErr: true, mutate: func(r *CommandRun) { r.TreeState = TreeStateUnknown }},
		{name: "unknown product", wantErr: true, mutate: func(r *CommandRun) { r.Product = core.ProductUnknown }},
		{name: "bad release id", wantErr: true, mutate: func(r *CommandRun) { r.ReleaseID = ReleaseID{} }},
		{name: "bad git commit", wantErr: true, mutate: func(r *CommandRun) { r.GitCommit = core.BuildCommit{} }},
		{name: "bad machine platform", wantErr: true, mutate: func(r *CommandRun) { r.Machine.Platform = core.PlatformUnknown }},
		{name: "bad hostname hash", wantErr: true, mutate: func(r *CommandRun) { r.Machine.HostnameSHA256 = core.SHA256Hex{} }},
		{name: "bad operator hash", wantErr: true, mutate: func(r *CommandRun) { r.OperatorSHA256 = core.SHA256Hex{} }},
		{name: "zero started timestamp", wantErr: true, mutate: func(r *CommandRun) { r.StartedAt = core.UnixNanoTime{} }},
		{name: "finish before start", mutate: func(r *CommandRun) {
			r.StartedAt = core.UnixNanoTimeFromInt64(20)
			r.FinishedAt = core.UnixNanoTimeFromInt64(10)
		}, wantErr: true},
		{name: "bad evidence ref", wantErr: true, mutate: func(r *CommandRun) { r.EvidenceRef = ObjectKey{value: "/absolute"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			run := validCommandRun(t)
			tc.mutate(&run)
			err := run.Validate()
			if !tc.wantErr && err != nil {
				t.Fatalf("CommandRun.Validate() = %v", err)
				return
			}
			if !tc.wantErr {
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("CommandRun.Validate() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestCommandRunCanonicalRoundTripTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		want string
		run  CommandRun
	}{
		{
			name: "release command run",
			run:  validCommandRun(t),
			want: `{"kind":"release","status":"succeeded","tree_state":"clean","product":"witness","version":"2026.0.0","release_id":"2026-07-08-2026.0.0","git_commit":"` +
				strings.Repeat("a", 40) +
				`","machine":{"platform":"darwin-arm64","hostname_sha256":"` +
				strings.Repeat("c", 64) +
				`","user_sha256":"` +
				strings.Repeat("d", 64) +
				`"},"operator_sha256":"` +
				strings.Repeat("e", 64) +
				`","started_at":1782302400000000000,"finished_at":1782302400000000100,"evidence_ref":"witness/2026-07-08/2026-07-08-2026.0.0/private/release_run.json"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.run.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("CommandRun.Canonical()\n got: %s\nwant: %s", got, tc.want)
			}
			decoded, err := core.DecodeStrictJSON[CommandRun](got)
			if err != nil {
				t.Fatal(err)
			}
			roundTrip, err := decoded.Canonical(nil)
			if err != nil {
				t.Fatal(err)
			}
			if string(roundTrip) != string(got) {
				t.Fatalf("CommandRun canonical round trip\n got: %s\nwant: %s", roundTrip, got)
			}
		})
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

func validBugReleaseSpec(t *testing.T) ProductReleaseSpec {
	t.Helper()
	return ProductReleaseSpec{
		Product:       core.ProductBug,
		Version:       mustVersion(t),
		Commands:      []ReleaseCommand{mustReleaseCommand(t, core.ProductTokenBug, "./cmd/bug")},
		CommandCount:  1,
		Platforms:     BuildPlatforms(),
		PlatformCount: uint32(len(BuildPlatforms())),
		Policy:        validBugBuildPolicy(t),
	}
}

func validWitnessReleaseSpec(t *testing.T) ProductReleaseSpec {
	t.Helper()
	commands := []ReleaseCommand{
		mustReleaseCommand(t, core.ProductTokenWitness, "./cmd/witness"),
		mustReleaseCommand(t, "witness-sign", "./cmd/witness-sign"),
	}
	return ProductReleaseSpec{
		Product:       core.ProductWitness,
		Version:       mustVersion(t),
		Commands:      commands,
		CommandCount:  uint32(len(commands)),
		Platforms:     BuildPlatforms(),
		PlatformCount: uint32(len(BuildPlatforms())),
		Policy:        validWitnessBuildPolicy(t),
	}
}

func validReleasePlan(t *testing.T) ReleasePlan {
	t.Helper()
	tools := validToolProvenance(t)
	return ReleasePlan{
		Product:   core.ProductWitness,
		Version:   mustVersion(t),
		ReleaseID: mustReleaseID(t),
		Date:      mustReleaseDate(t),
		Commit:    mustCommit(t),
		Seed:      mustGarbleSeed(t),
		SeedRef:   mustGarbleSeedRef(t, "release/2026/07/08/seed"),
		Layout:    validWitnessReleaseRootLayout(t),
		Spec:      validWitnessReleaseSpec(t),
		Toolchain: validBuildToolchain(t),
		Evidence:  validReleaseGateEvidence(t),
		VulnDB:    validVulnDBSnapshot(t),
		Tools:     tools,
		ToolCount: uint32(len(tools)),
	}
}

func validReleasePreflightInput(t *testing.T) ReleasePreflightInput {
	t.Helper()
	return ReleasePreflightInput{
		RequestedCommit: mustCommit(t),
		HeadCommit:      mustCommit(t),
		TreeState:       TreeStateClean,
		Seed:            mustGarbleSeed(t),
		Toolchain:       validBuildToolchain(t),
		VulnDB:          validVulnDBSnapshot(t),
		Evidence:        validReleaseGateEvidence(t),
	}
}

func validBuildToolchain(t *testing.T) BuildToolchain {
	t.Helper()
	return BuildToolchain{
		GoVersion:     mustToolVersion(t, "go1.25.0"),
		GarbleVersion: mustToolVersion(t, "v0.15.0"),
	}
}

func validVulnDBSnapshot(t *testing.T) VulnDBSnapshot {
	t.Helper()
	return VulnDBSnapshot{
		DBVersion:  mustToolVersion(t, "2026-07-08T00:00:00Z"),
		SnapshotAt: core.UnixNanoTimeFromInt64(1782302400000000000),
	}
}

func validReleaseGateEvidence(t *testing.T) ReleaseGateEvidence {
	t.Helper()
	return ReleaseGateEvidence{
		FastGateRef:      mustEvidenceRef(t, "witness://release/fast/green"),
		FinalEvidenceRef: mustEvidenceRef(t, "witness://release/final/green"),
	}
}

func validToolProvenance(t *testing.T) []ToolProvenance {
	t.Helper()
	return []ToolProvenance{
		{
			Module:  mustToolModule(t, "github.com/offGridSoft/deadcode"),
			Version: mustToolVersion(t, "v2026.0.0"),
			GoSum:   mustGoSumHash(t, "h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa="),
		},
		{
			Module:  mustToolModule(t, "honnef.co/go/tools"),
			Version: mustToolVersion(t, "v0.6.1"),
			GoSum:   mustGoSumHash(t, "h1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb="),
		},
	}
}

func validDeployPlan(t *testing.T) DeployPlan {
	t.Helper()
	return DeployPlan{
		Product:        core.ProductWitness,
		Version:        mustVersion(t),
		ReleaseID:      mustReleaseID(t),
		Manifest:       validManifest(t),
		ManifestSHA256: mustSHA256(t, "f"),
		Layout:         validWitnessReleaseRootLayout(t),
		Targets:        []UploadTarget{validUploadTarget(t)},
		TargetCount:    1,
	}
}

func validBugBuildRequest(t *testing.T, seed GarbleSeed) ReleaseGarbleBuildRequest {
	t.Helper()
	return ReleaseGarbleBuildRequest{
		Seed:     seed,
		Command:  mustReleaseCommand(t, core.ProductTokenBug, "./cmd/bug"),
		Output:   mustBuildOutputPath(t, "dist/linux-amd64/bug"),
		Platform: core.PlatformLinuxAMD64,
		Policy:   validBugBuildPolicy(t),
	}
}

func validWitnessBuildRequest(t *testing.T, seed GarbleSeed) ReleaseGarbleBuildRequest {
	t.Helper()
	return ReleaseGarbleBuildRequest{
		Seed:     seed,
		Command:  mustReleaseCommand(t, core.ProductTokenWitness, "./cmd/witness"),
		Output:   mustBuildOutputPath(t, "dist/linux-amd64/witness"),
		Platform: core.PlatformLinuxAMD64,
		Policy:   validWitnessBuildPolicy(t),
	}
}

func validBugBuildPolicy(t *testing.T) ReleaseBuildPolicy {
	t.Helper()
	return ReleaseBuildPolicy{Strip: true}
}

func validWitnessBuildPolicy(t *testing.T) ReleaseBuildPolicy {
	t.Helper()
	tags := []BuildTag{
		mustBuildTag(t, "osusergo"),
		mustBuildTag(t, "netgo"),
		mustBuildTag(t, "witness_production"),
	}
	return ReleaseBuildPolicy{
		BuildVCS:     true,
		ClearBuildID: true,
		Strip:        true,
		Tags:         tags,
		TagCount:     uint32(len(tags)),
		CommitStamp: BuildCommitStamp{
			Symbol: mustLinkerSymbol(t, "github.com/offGridSoft/witness/internal/release.BuildCommit"),
			Commit: mustCommit(t),
		},
	}
}

func validCommandRun(t *testing.T) CommandRun {
	t.Helper()
	ref, err := ParseObjectKey("witness/2026-07-08/2026-07-08-2026.0.0/private/release_run.json")
	if err != nil {
		t.Fatal(err)
	}
	return CommandRun{
		Kind:           CommandKindRelease,
		Status:         CommandStatusSucceeded,
		TreeState:      TreeStateClean,
		Product:        core.ProductWitness,
		Version:        mustVersion(t),
		ReleaseID:      mustReleaseID(t),
		GitCommit:      mustCommit(t),
		StartedAt:      core.UnixNanoTimeFromInt64(1782302400000000000),
		FinishedAt:     core.UnixNanoTimeFromInt64(1782302400000000100),
		EvidenceRef:    ref,
		OperatorSHA256: mustSHA256(t, "e"),
		Machine: MachineIdentity{
			Platform:       core.PlatformDarwinARM64,
			HostnameSHA256: mustSHA256(t, "c"),
			UserSHA256:     mustSHA256(t, "d"),
		},
	}
}

func validReleaseRootInput(t *testing.T) ReleaseRootInput {
	t.Helper()
	return ReleaseRootInput{
		Product:   core.ProductWitness,
		Version:   mustVersion(t),
		Date:      mustReleaseDate(t),
		ReleaseID: mustReleaseID(t),
	}
}

func validWitnessReleaseRootLayout(t *testing.T) ReleaseRootLayout {
	t.Helper()
	layout, err := BuildReleaseRootLayout(validReleaseRootInput(t))
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func validBugReleaseRootLayout(t *testing.T) ReleaseRootLayout {
	t.Helper()
	input := validReleaseRootInput(t)
	input.Product = core.ProductBug
	layout, err := BuildReleaseRootLayout(input)
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func tamperedReleaseRootPath(t *testing.T, finalSegment string) BuildOutputPath {
	t.Helper()
	segments := []string{
		DistDirName,
		ReleaseRootDirName,
		core.ProductTokenWitness,
		testDateToken[:4],
		testDateToken[5:7],
		testDateToken[8:10],
		finalSegment,
	}
	return mustBuildOutputPath(t, strings.Join(segments, "/"))
}

func mustGarbleSeed(t *testing.T) GarbleSeed {
	t.Helper()
	seed, err := ParseRequiredGarbleSeed("AQIDBAUGBwg=")
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

func mustEvidenceRef(t *testing.T, value string) EvidenceRef {
	t.Helper()
	ref, err := ParseEvidenceRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func mustGarbleSeedRef(t *testing.T, value string) GarbleSeedRef {
	t.Helper()
	ref, err := ParseGarbleSeedRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func mustToolVersion(t *testing.T, value string) ToolVersion {
	t.Helper()
	version, err := ParseToolVersion(value)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func mustToolModule(t *testing.T, value string) ToolModule {
	t.Helper()
	module, err := ParseToolModule(value)
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func mustGoSumHash(t *testing.T, value string) GoSumHash {
	t.Helper()
	hash, err := ParseGoSumHash(value)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func mustReleaseCommand(t *testing.T, name string, importPath string) ReleaseCommand {
	t.Helper()
	command, err := NewReleaseCommand(name, importPath)
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func mustBuildImportPath(t *testing.T, value string) BuildImportPath {
	t.Helper()
	path, err := ParseBuildImportPath(value)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func mustBuildOutputPath(t *testing.T, value string) BuildOutputPath {
	t.Helper()
	output, err := ParseBuildOutputPath(value)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func mustBuildTag(t *testing.T, value string) BuildTag {
	t.Helper()
	tag, err := ParseBuildTag(value)
	if err != nil {
		t.Fatal(err)
	}
	return tag
}

func mustLinkerSymbol(t *testing.T, value string) LinkerSymbol {
	t.Helper()
	symbol, err := ParseLinkerSymbol(value)
	if err != nil {
		t.Fatal(err)
	}
	return symbol
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
	u, err := ParseDownloadURL("https://downloads.offgridsoftware.com/witness/2026-07-08/2026-07-08-2026.0.0/public/tools.tar.gz")
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

func mustOtherVersion(t *testing.T) core.ProductVersion {
	t.Helper()
	version, err := core.ParseProductVersion("2027.0.0")
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

func mustOtherReleaseID(t *testing.T) ReleaseID {
	t.Helper()
	id, err := ParseReleaseID("2026-07-09-2026.0.0")
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

func mustOtherCommit(t *testing.T) core.BuildCommit {
	t.Helper()
	commit, err := core.ParseBuildCommit(strings.Repeat("b", 40))
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

func requireMarshalOK(t *testing.T, value any) {
	t.Helper()
	if _, err := json.Marshal(value); err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
}

func requireJSONRoundTrip(t *testing.T, value any, target any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("json.Unmarshal(%T) error = %v", target, err)
	}
}
