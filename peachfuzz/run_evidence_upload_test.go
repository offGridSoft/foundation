package peachfuzz

import (
	"errors"
	"strings"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
)

func TestRunEvidenceUploadRequestDerivesExactDescriptorOGSBoundary(t *testing.T) {
	t.Parallel()

	request := validRunEvidenceUploadRequest(t)
	descriptor, err := request.Descriptor()
	if err != nil {
		t.Fatalf("RunEvidenceUploadRequest.Descriptor() error = %v, want nil", err)
	}
	want, err := NewRunEvidenceObjectName(request.Evidence.Body, descriptor.Digest)
	if err != nil {
		t.Fatalf("NewRunEvidenceObjectName() error = %v, want nil", err)
	}
	if descriptor.Object != want || descriptor.Size.Uint64() == 0 || descriptor.Size.Uint64() > foundationcore.PeachfuzzRunEvidenceMaxBytes {
		t.Fatalf("descriptor = %+v, want exact object and bounded nonzero size", descriptor)
	}
	encoded, err := foundationcore.EncodeValidatedJSON(request)
	if err != nil {
		t.Fatalf("EncodeValidatedJSON(request) error = %v, want nil", err)
	}
	decoded, err := foundationcore.DecodeStrictJSON[RunEvidenceUploadRequest](encoded)
	if err != nil || decoded != request {
		t.Fatalf("DecodeStrictJSON(request) = (%+v, %v), want exact request", decoded, err)
	}
}

func TestRunEvidenceObjectNameRejectsHostileShapeOGSBoundaryTable(t *testing.T) {
	t.Parallel()

	descriptor, err := validRunEvidenceUploadRequest(t).Descriptor()
	if err != nil {
		t.Fatalf("Descriptor() error = %v, want nil", err)
	}
	valid := descriptor.Object.String()
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty"},
		{name: "traversal", value: "../" + valid},
		{name: "wrong version", value: strings.Replace(valid, foundationcore.FoundationVersion2026, "2027.0.0", 1)},
		{name: "wrong segment", value: strings.Replace(valid, RunEvidenceArchiveSegment, "runs", 1)},
		{name: "wrong shard", value: strings.Replace(valid, "/"+descriptor.Digest.String()[:RunEvidenceDigestShardHexCharacters]+"/", "/ff/", 1)},
		{name: "missing extension", value: strings.TrimSuffix(valid, RunEvidenceObjectExtension)},
		{name: "extra separator", value: strings.Replace(valid, RunEvidenceObjectDigestSeparator, RunEvidenceObjectDigestSeparator+"extra"+RunEvidenceObjectDigestSeparator, 1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseRunEvidenceObjectName(tc.value); !errors.Is(err, ErrContract) {
				t.Fatalf("ParseRunEvidenceObjectName(%q) error = %v, want errors.Is(..., %v)", tc.value, err, ErrContract)
			}
		})
	}
}

func TestRunEvidenceDigestShardContractOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
		want  RunEvidenceDigestShard
	}{
		{name: "first", value: "00", want: RunEvidenceDigestShard(0)},
		{name: "lowercase boundary", value: "0f", want: RunEvidenceDigestShard(15)},
		{name: "last", value: "ff", want: RunEvidenceDigestShard(255)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRunEvidenceDigestShard(tc.value)
			if err != nil || got != tc.want || got.String() != tc.value {
				t.Fatalf("ParseRunEvidenceDigestShard(%q) = %s/%v, want %s/nil", tc.value, got.String(), err, tc.want.String())
			}
			wantPrefix := foundationcore.FoundationVersion2026 + RunEvidenceObjectPathSeparator + RunEvidenceArchiveSegment + RunEvidenceObjectPathSeparator + tc.value + RunEvidenceObjectPathSeparator
			if gotPrefix := RunEvidenceDigestShardPrefix(got); gotPrefix != wantPrefix {
				t.Fatalf("RunEvidenceDigestShardPrefix(%s) = %q, want %q", got.String(), gotPrefix, wantPrefix)
			}
		})
	}
}

func TestRunEvidenceDigestShardRejectsNonCanonicalInputOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "0", "000", "0F", "gg", "-1", "+1"} {
		if _, err := ParseRunEvidenceDigestShard(value); !errors.Is(err, ErrContract) {
			t.Errorf("ParseRunEvidenceDigestShard(%q) error = %v, want %v", value, err, ErrContract)
		}
	}
}

func TestRunEvidenceUploadGrantBindsRequestOGSBoundaryTable(t *testing.T) {
	t.Parallel()

	request := validRunEvidenceUploadRequest(t)
	descriptor, err := request.Descriptor()
	if err != nil {
		t.Fatalf("Descriptor() error = %v, want nil", err)
	}
	uploadURL, err := foundationcore.ParseSignedUploadURL("https://storage.googleapis.com/evidence/object?X-Goog-Signature=abc")
	if err != nil {
		t.Fatalf("ParseSignedUploadURL() error = %v, want nil", err)
	}
	grant := RunEvidenceUploadGrant{
		Descriptor: descriptor,
		URL:        uploadURL,
		Headers:    []foundationcore.UploadHeader{{Name: foundationcore.HTTPHeaderContentType, Value: RunEvidenceContentType}},
		ExpiresAt:  foundationcore.UnixNanoTimeFromInt64(10),
		Provider:   foundationcore.StorageProviderGCS,
		Method:     foundationcore.UploadMethodSignedPUT,
		Schema:     foundationcore.SchemaPeachfuzzRunEvidenceUploadGrant,
	}
	if err := grant.ValidateRequest(request); err != nil {
		t.Fatalf("RunEvidenceUploadGrant.ValidateRequest() error = %v, want nil", err)
	}

	mutated := request
	mutated.Evidence.Body.CPU = foundationcore.NanosecondsDurationFromInt64(mutated.Evidence.Body.CPU.Nanoseconds() + 1)
	if err := grant.ValidateRequest(mutated); !errors.Is(err, ErrContract) {
		t.Fatalf("ValidateRequest(mutated signed evidence) error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
	other := validRunEvidenceUploadRequest(t)
	other.Evidence.Body.RunID, _ = ParseRunID(strings.Repeat("d", RunIDTextBytes))
	if err := grant.ValidateRequest(other); !errors.Is(err, ErrContract) {
		t.Fatalf("ValidateRequest(other evidence) error = %v, want errors.Is(..., %v)", err, ErrContract)
	}
}

func TestRunEvidenceUploadResponseHasExactlyOneDispositionOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	request := validRunEvidenceUploadRequest(t)
	descriptor, err := request.Descriptor()
	if err != nil {
		t.Fatalf("Descriptor() error = %v, want nil", err)
	}
	grant := validRunEvidenceUploadGrant(t, descriptor)
	tests := []struct {
		name     string
		response RunEvidenceUploadResponse
		wantErr  bool
	}{
		{name: "upload required", response: RunEvidenceUploadResponse{Grant: &grant, Descriptor: descriptor, Disposition: RunEvidenceUploadDispositionRequired, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadResponse}},
		{name: "already present", response: RunEvidenceUploadResponse{Descriptor: descriptor, Disposition: RunEvidenceUploadDispositionPresent, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadResponse}},
		{name: "required without grant", response: RunEvidenceUploadResponse{Descriptor: descriptor, Disposition: RunEvidenceUploadDispositionRequired, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadResponse}, wantErr: true},
		{name: "present with grant", response: RunEvidenceUploadResponse{Grant: &grant, Descriptor: descriptor, Disposition: RunEvidenceUploadDispositionPresent, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadResponse}, wantErr: true},
		{name: "unknown disposition", response: RunEvidenceUploadResponse{Descriptor: descriptor, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadResponse}, wantErr: true},
		{name: "wrong schema", response: RunEvidenceUploadResponse{Descriptor: descriptor, Disposition: RunEvidenceUploadDispositionPresent, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadGrant}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.response.ValidateRequest(request)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ValidateRequest() error = %v, want errors.Is(..., %v)", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateRequest() error = %v, want nil", err)
			}
		})
	}
}

func validRunEvidenceUploadGrant(t *testing.T, descriptor RunEvidenceDescriptor) RunEvidenceUploadGrant {
	t.Helper()
	uploadURL, err := foundationcore.ParseSignedUploadURL("https://storage.googleapis.com/evidence/object?X-Goog-Signature=abc")
	if err != nil {
		t.Fatalf("ParseSignedUploadURL() error = %v, want nil", err)
	}
	return RunEvidenceUploadGrant{
		Descriptor: descriptor,
		URL:        uploadURL,
		Headers:    []foundationcore.UploadHeader{{Name: foundationcore.HTTPHeaderContentType, Value: RunEvidenceContentType}},
		ExpiresAt:  foundationcore.UnixNanoTimeFromInt64(10),
		Provider:   foundationcore.StorageProviderGCS,
		Method:     foundationcore.UploadMethodSignedPUT,
		Schema:     foundationcore.SchemaPeachfuzzRunEvidenceUploadGrant,
	}
}

func validRunEvidenceUploadRequest(t *testing.T) RunEvidenceUploadRequest {
	t.Helper()
	key, machine := signedRunEvidenceKey(t)
	body := validRunEvidence(t)
	body.Machine = machine
	signed, err := foundationcore.SignCanonical(key, body)
	if err != nil {
		t.Fatalf("SignCanonical() error = %v, want nil", err)
	}
	evidence, err := NewSignedRunEvidence(signed)
	if err != nil {
		t.Fatalf("NewSignedRunEvidence() error = %v, want nil", err)
	}
	return RunEvidenceUploadRequest{Evidence: evidence, Schema: foundationcore.SchemaPeachfuzzRunEvidenceUploadRequest}
}
