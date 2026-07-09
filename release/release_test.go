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
		{name: "explicit random seed is random", value: GarbleSeedRandom, want: GarbleSeedRandom},
		{name: "exact base64 seed accepted", value: exactSeed, want: exactSeed, required: true},
		{name: "long seed normalized to eight bytes", value: longSeed, want: exactSeed, required: true},
		{name: "required rejects empty random", value: "", required: true, wantError: core.ErrReleaseContract},
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

func TestReleaseBuildSpecHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*ProductReleaseSpec)
		name   string
	}{
		{name: "bug one command valid", mutate: func(s *ProductReleaseSpec) {
			*s = validBugReleaseSpec(t)
		}},
		{name: "witness multiple commands valid", mutate: func(*ProductReleaseSpec) {}},
		{name: "unknown product", mutate: func(s *ProductReleaseSpec) { s.Product = core.ProductUnknown }},
		{name: "bad version", mutate: func(s *ProductReleaseSpec) { s.Version = core.ProductVersion{} }},
		{name: "missing command", mutate: func(s *ProductReleaseSpec) {
			s.Commands = nil
			s.CommandCount = 0
		}},
		{name: "command count mismatch", mutate: func(s *ProductReleaseSpec) { s.CommandCount++ }},
		{name: "duplicate command name", mutate: func(s *ProductReleaseSpec) {
			s.Commands[1].Name = s.Commands[0].Name
		}},
		{name: "duplicate import path", mutate: func(s *ProductReleaseSpec) {
			s.Commands[1].ImportPath = s.Commands[0].ImportPath
		}},
		{name: "unsupported release platform", mutate: func(s *ProductReleaseSpec) {
			s.Platforms[0] = core.PlatformDarwinAMD64
		}},
		{name: "duplicate platform", mutate: func(s *ProductReleaseSpec) {
			s.Platforms[1] = s.Platforms[0]
		}},
		{name: "platform count mismatch", mutate: func(s *ProductReleaseSpec) { s.PlatformCount++ }},
		{name: "non-stripped build rejected", mutate: func(s *ProductReleaseSpec) { s.Policy.Strip = false }},
		{name: "duplicate build tag", mutate: func(s *ProductReleaseSpec) {
			s.Policy.Tags = append(s.Policy.Tags, s.Policy.Tags[0])
			s.Policy.TagCount = uint32(len(s.Policy.Tags))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := validWitnessReleaseSpec(t)
			tc.mutate(&spec)
			err := spec.Validate()
			if strings.Contains(tc.name, " valid") {
				if err != nil {
					t.Fatalf("ProductReleaseSpec.Validate() = %v", err)
				}
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
		mutate    func(*ProductReleaseSpec, *GarbleSeed, *ArtifactName)
		name      string
		wantCount int
	}{
		{name: "bug matrix expands one command across release platforms", wantCount: len(BuildPlatforms()), mutate: func(s *ProductReleaseSpec, _ *GarbleSeed, _ *ArtifactName) {
			*s = validBugReleaseSpec(t)
		}},
		{name: "witness matrix expands all commands", wantCount: len(BuildPlatforms()) * 2, mutate: func(*ProductReleaseSpec, *GarbleSeed, *ArtifactName) {}},
		{name: "random seed rejected before output planning", mutate: func(_ *ProductReleaseSpec, seed *GarbleSeed, _ *ArtifactName) {
			*seed = GarbleSeed{value: GarbleSeedRandom}
		}},
		{name: "bad output root rejected", mutate: func(_ *ProductReleaseSpec, _ *GarbleSeed, root *ArtifactName) {
			*root = ArtifactName{value: ".."}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := validWitnessReleaseSpec(t)
			seed := mustGarbleSeed(t)
			root, err := DefaultOutputRoot()
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&spec, &seed, &root)
			got, err := spec.GarbleBuildRequests(seed, root)
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
			if got[len(got)-1].Output.String() != "dist/windows-amd64/witness-sign.exe" && spec.Product == core.ProductWitness {
				t.Fatalf("last witness output = %q, want windows executable suffix", got[len(got)-1].Output.String())
			}
		})
	}
}

func TestCommandRunHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*CommandRun)
		name   string
	}{
		{name: "release run valid", mutate: func(*CommandRun) {}},
		{name: "deploy run valid", mutate: func(r *CommandRun) { r.Kind = CommandKindDeploy }},
		{name: "dirty tree run valid", mutate: func(r *CommandRun) { r.TreeState = TreeStateDirty }},
		{name: "unknown kind", mutate: func(r *CommandRun) { r.Kind = CommandKindUnknown }},
		{name: "unknown status", mutate: func(r *CommandRun) { r.Status = CommandStatusUnknown }},
		{name: "unknown tree state", mutate: func(r *CommandRun) { r.TreeState = TreeStateUnknown }},
		{name: "unknown product", mutate: func(r *CommandRun) { r.Product = core.ProductUnknown }},
		{name: "bad git commit", mutate: func(r *CommandRun) { r.GitCommit = core.BuildCommit{} }},
		{name: "bad machine platform", mutate: func(r *CommandRun) { r.Machine.Platform = core.PlatformUnknown }},
		{name: "bad hostname hash", mutate: func(r *CommandRun) { r.Machine.HostnameSHA256 = core.SHA256Hex{} }},
		{name: "zero started timestamp", mutate: func(r *CommandRun) { r.StartedAt = core.UnixNanoTime{} }},
		{name: "finish before start", mutate: func(r *CommandRun) {
			r.StartedAt = core.UnixNanoTimeFromInt64(20)
			r.FinishedAt = core.UnixNanoTimeFromInt64(10)
		}},
		{name: "bad evidence ref", mutate: func(r *CommandRun) { r.EvidenceRef = ObjectKey{value: "/absolute"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			run := validCommandRun(t)
			tc.mutate(&run)
			err := run.Validate()
			if strings.Contains(tc.name, " valid") {
				if err != nil {
					t.Fatalf("CommandRun.Validate() = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("CommandRun.Validate() error = %v, want release/foundation contract", err)
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
		Kind:        CommandKindRelease,
		Status:      CommandStatusSucceeded,
		TreeState:   TreeStateClean,
		Product:     core.ProductWitness,
		Version:     mustVersion(t),
		GitCommit:   mustCommit(t),
		StartedAt:   core.UnixNanoTimeFromInt64(1782302400000000000),
		FinishedAt:  core.UnixNanoTimeFromInt64(1782302400000000100),
		EvidenceRef: ref,
		Machine: MachineIdentity{
			Platform:       core.PlatformDarwinARM64,
			HostnameSHA256: mustSHA256(t, "c"),
			UserSHA256:     mustSHA256(t, "d"),
		},
	}
}

func mustGarbleSeed(t *testing.T) GarbleSeed {
	t.Helper()
	seed, err := ParseRequiredGarbleSeed("AQIDBAUGBwg=")
	if err != nil {
		t.Fatal(err)
	}
	return seed
}

func mustReleaseCommand(t *testing.T, name string, importPath string) ReleaseCommand {
	t.Helper()
	command, err := NewReleaseCommand(name, importPath)
	if err != nil {
		t.Fatal(err)
	}
	return command
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
