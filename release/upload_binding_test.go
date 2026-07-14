package release

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestUploadAttemptIDRejectsHostileTextWithoutMutationTable(t *testing.T) {
	t.Parallel()

	valid := strings.Repeat("a", sha256.Size*2)
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "empty", value: ""},
		{name: "one nibble short", value: valid[:len(valid)-1]},
		{name: "one nibble long", value: valid + "a"},
		{name: "uppercase is noncanonical", value: strings.ToUpper(valid)},
		{name: "non hex", value: strings.Repeat("z", sha256.Size*2)},
		{name: "all zero is not random", value: strings.Repeat("0", sha256.Size*2)},
		{name: "leading whitespace", value: " " + valid},
		{name: "trailing newline", value: valid + "\n"},
		{name: "embedded nul", value: valid[:1] + "\x00" + valid[2:]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			before := validUploadAttemptID(t)
			decoded := before
			data, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			err = decoded.UnmarshalJSON(data)
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UploadAttemptID.UnmarshalJSON() error = %v, want %v", err, core.ErrReleaseContract)
			}
			if decoded != before {
				t.Fatalf("UploadAttemptID.UnmarshalJSON() mutated receiver: got %q, want %q", decoded, before)
			}
		})
	}
}

func TestUploadBindingCoversEverySwapSensitiveFactTable(t *testing.T) {
	t.Parallel()

	base := validUploadBindingInput(t)
	want, err := DeriveUploadBinding(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mutate func(*UploadBindingInput)
		name   string
	}{
		{name: "product", mutate: func(i *UploadBindingInput) { i.Product = core.ProductBug }},
		{name: "release id", mutate: func(i *UploadBindingInput) { i.ReleaseID = mustOtherReleaseID(t) }},
		{name: "manifest digest", mutate: func(i *UploadBindingInput) { i.ManifestSHA256 = mustSHA256(t, "c") }},
		{name: "artifact name", mutate: func(i *UploadBindingInput) { i.Artifact = mustArtifactName(t, "other.tar.gz") }},
		{name: "artifact digest", mutate: func(i *UploadBindingInput) { i.ArtifactSHA256 = mustSHA256(t, "d") }},
		{name: "artifact size", mutate: func(i *UploadBindingInput) { i.ArtifactSize = core.NewByteCount(13) }},
		{name: "provider", mutate: func(i *UploadBindingInput) { i.Provider = core.StorageProviderS3 }},
		{name: "bucket", mutate: func(i *UploadBindingInput) { i.Bucket = mustOtherBucket(t) }},
		{name: "object", mutate: func(i *UploadBindingInput) { i.Object = mustOtherObjectKey(t) }},
		{name: "upload attempt", mutate: func(i *UploadBindingInput) { i.AttemptID = mustOtherUploadAttemptID(t) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := base
			tc.mutate(&input)
			got, err := DeriveUploadBinding(input)
			if err != nil {
				t.Fatalf("DeriveUploadBinding() error = %v", err)
			}
			if got == want {
				t.Fatalf("DeriveUploadBinding() ignored %s", tc.name)
			}
		})
	}
}

func TestUploadTargetRejectsMetadataHeaderDriftTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*UploadTarget)
		name   string
	}{
		{name: "attempt header absent", mutate: func(target *UploadTarget) { removeUploadHeader(target, GCSUploadAttemptHeaderName) }},
		{name: "binding header absent", mutate: func(target *UploadTarget) { removeUploadHeader(target, GCSUploadBindingHeaderName) }},
		{name: "create only header absent", mutate: func(target *UploadTarget) { removeUploadHeader(target, GCSUploadCreateOnlyName) }},
		{name: "attempt header value drift", mutate: func(target *UploadTarget) {
			setUploadHeaderValue(target, GCSUploadAttemptHeaderName, mustOtherUploadAttemptID(t).String())
		}},
		{name: "binding header value drift", mutate: func(target *UploadTarget) {
			setUploadHeaderValue(target, GCSUploadBindingHeaderName, mustSHA256(t, "e").String())
		}},
		{name: "create only precondition disabled", mutate: func(target *UploadTarget) {
			setUploadHeaderValue(target, GCSUploadCreateOnlyName, "1")
		}},
		{name: "cross provider attempt header", mutate: func(target *UploadTarget) {
			renameUploadHeader(target, GCSUploadAttemptHeaderName, S3UploadAttemptHeaderName)
		}},
		{name: "cross provider binding header", mutate: func(target *UploadTarget) {
			renameUploadHeader(target, GCSUploadBindingHeaderName, S3UploadBindingHeaderName)
		}},
		{name: "case folded duplicate", mutate: func(target *UploadTarget) {
			target.Headers = append(target.Headers, core.UploadHeader{
				Name: strings.ToUpper(GCSUploadBindingHeaderName), Value: target.Binding.String(),
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := validUploadTarget(t)
			tc.mutate(&target)
			if err := target.Validate(); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UploadTarget.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestDeployPlanRejectsLocallyConsistentTargetSwapTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*DeployPlan)
		name   string
	}{
		{name: "object swap", mutate: func(plan *DeployPlan) { plan.Targets[0].Object = mustOtherObjectKey(t) }},
		{name: "bucket swap", mutate: func(plan *DeployPlan) { plan.Targets[0].Bucket = mustOtherBucket(t) }},
		{name: "attempt replay", mutate: func(plan *DeployPlan) {
			plan.Targets[0].AttemptID = mustOtherUploadAttemptID(t)
			setUploadHeaderValue(&plan.Targets[0], GCSUploadAttemptHeaderName, plan.Targets[0].AttemptID.String())
		}},
		{name: "attacker supplied binding", mutate: func(plan *DeployPlan) {
			plan.Targets[0].Binding = mustOtherUploadBinding(t)
			setUploadHeaderValue(&plan.Targets[0], GCSUploadBindingHeaderName, plan.Targets[0].Binding.String())
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := validDeployPlan(t)
			tc.mutate(&plan)
			if err := plan.Targets[0].Validate(); err != nil {
				t.Fatalf("hostile target was not locally consistent: %v", err)
			}
			if err := plan.Validate(); !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("DeployPlan.Validate() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func FuzzUploadAttemptIDJSONBoundary(f *testing.F) {
	for _, seed := range []string{
		strings.Repeat("a", sha256.Size*2),
		strings.Repeat("0", sha256.Size*2),
		"", "not-hex", strings.Repeat("a", sha256.Size*2-1),
	} {
		f.Add(seed)
	}
	known, err := ParseUploadAttemptID(strings.Repeat("b", sha256.Size*2))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, value string) {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		decoded := known
		err = decoded.UnmarshalJSON(data)
		parsed, parseErr := ParseUploadAttemptID(value)
		if parseErr != nil {
			if !errors.Is(parseErr, core.ErrReleaseContract) || !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("invalid value errors = (%v, %v), want %v", parseErr, err, core.ErrReleaseContract)
			}
			if decoded != known {
				t.Fatalf("rejected value mutated receiver: got %q, want %q", decoded, known)
			}
			return
		}
		if err != nil || decoded != parsed {
			t.Fatalf("valid value round trip = (%q, %v), want (%q, nil)", decoded, err, parsed)
		}
	})
}

func FuzzUploadBindingCanonicalFacts(f *testing.F) {
	f.Add(uint8(0), uint8(0))
	f.Add(uint8(9), uint8(255))
	f.Fuzz(func(t *testing.T, selector uint8, salt uint8) {
		base, err := fuzzUploadBindingInput()
		if err != nil {
			t.Fatal(err)
		}
		want, err := DeriveUploadBinding(base)
		if err != nil {
			t.Fatal(err)
		}
		mutated, err := mutateFuzzUploadBindingInput(base, selector, salt)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DeriveUploadBinding(mutated)
		if err != nil {
			t.Fatal(err)
		}
		if got == want {
			t.Fatalf("binding ignored canonical fact %d", selector%10)
		}
	})
}

func validUploadBindingInput(t *testing.T) UploadBindingInput {
	t.Helper()
	manifest := validManifest(t)
	canonical, err := manifest.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	target := validUploadTarget(t)
	artifact := manifest.Artifacts[0]
	return UploadBindingInput{
		Product: manifest.Product, ReleaseID: manifest.ReleaseID,
		ManifestSHA256: core.NewSHA256Hex(sha256.Sum256(canonical)),
		Artifact:       artifact.Name, ArtifactSHA256: artifact.SHA256, ArtifactSize: artifact.Size,
		Provider: target.Provider, Bucket: target.Bucket, Object: target.Object, AttemptID: target.AttemptID,
	}
}

func mustOtherBucket(t *testing.T) Bucket {
	t.Helper()
	bucket, err := ParseBucket("other-release-bucket")
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func mustOtherObjectKey(t *testing.T) ObjectKey {
	t.Helper()
	key, err := ParseObjectKey("public/witness/2026/07/08/other/tools.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func mustOtherUploadAttemptID(t *testing.T) UploadAttemptID {
	t.Helper()
	id, err := ParseUploadAttemptID(strings.Repeat("2", sha256.Size*2))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustOtherUploadBinding(t *testing.T) UploadBinding {
	t.Helper()
	binding, err := ParseUploadBinding(strings.Repeat("3", sha256.Size*2))
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func removeUploadHeader(target *UploadTarget, name string) {
	for index, header := range target.Headers {
		if strings.EqualFold(header.Name, name) {
			target.Headers = append(target.Headers[:index], target.Headers[index+1:]...)
			return
		}
	}
}

func setUploadHeaderValue(target *UploadTarget, name, value string) {
	for index := range target.Headers {
		if strings.EqualFold(target.Headers[index].Name, name) {
			target.Headers[index].Value = value
			return
		}
	}
}

func renameUploadHeader(target *UploadTarget, oldName, newName string) {
	for index := range target.Headers {
		if strings.EqualFold(target.Headers[index].Name, oldName) {
			target.Headers[index].Name = newName
			return
		}
	}
}

func fuzzUploadBindingInput() (UploadBindingInput, error) {
	releaseID, err := ParseReleaseID("witness-2026-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		return UploadBindingInput{}, err
	}
	artifact, err := ParseArtifactName("tools.tar.gz")
	if err != nil {
		return UploadBindingInput{}, err
	}
	bucket, err := ParseBucket("release-bucket")
	if err != nil {
		return UploadBindingInput{}, err
	}
	object, err := ParseObjectKey("public/witness/2026/07/08/tools.tar.gz")
	if err != nil {
		return UploadBindingInput{}, err
	}
	attemptID, err := ParseUploadAttemptID(strings.Repeat("1", sha256.Size*2))
	if err != nil {
		return UploadBindingInput{}, err
	}
	manifestSHA, err := core.ParseSHA256Hex(strings.Repeat("a", sha256.Size*2))
	if err != nil {
		return UploadBindingInput{}, err
	}
	artifactSHA, err := core.ParseSHA256Hex(strings.Repeat("b", sha256.Size*2))
	if err != nil {
		return UploadBindingInput{}, err
	}
	return UploadBindingInput{
		Product: core.ProductWitness, ReleaseID: releaseID, ManifestSHA256: manifestSHA,
		Artifact: artifact, ArtifactSHA256: artifactSHA, ArtifactSize: core.NewByteCount(12),
		Provider: core.StorageProviderGCS, Bucket: bucket, Object: object, AttemptID: attemptID,
	}, nil
}

func mutateFuzzUploadBindingInput(input UploadBindingInput, selector, salt uint8) (UploadBindingInput, error) {
	digit := "c"
	if salt%2 == 1 {
		digit = "d"
	}
	var err error
	switch selector % 10 {
	case 0:
		input.Product = core.ProductBug
	case 1:
		input.ReleaseID, err = ParseReleaseID("bug-2026-" + strings.Repeat(digit, 40))
	case 2:
		input.ManifestSHA256, err = core.ParseSHA256Hex(strings.Repeat(digit, sha256.Size*2))
	case 3:
		input.Artifact, err = ParseArtifactName("artifact-" + digit)
	case 4:
		input.ArtifactSHA256, err = core.ParseSHA256Hex(strings.Repeat(digit, sha256.Size*2))
	case 5:
		input.ArtifactSize = core.NewByteCount(uint64(salt) + 13)
	case 6:
		input.Provider = core.StorageProviderS3
	case 7:
		input.Bucket, err = ParseBucket("bucket-" + digit)
	case 8:
		input.Object, err = ParseObjectKey("public/witness/2026/07/08/object-" + digit)
	case 9:
		input.AttemptID, err = ParseUploadAttemptID(strings.Repeat(digit, sha256.Size*2))
	}
	return input, err
}
