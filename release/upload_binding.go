package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	UploadBindingSchema        = "foundation-release-upload-binding-" + core.ContractVersionToken
	GCSUploadAttemptHeaderName = "x-goog-meta-upload-attempt-id"
	GCSUploadBindingHeaderName = "x-goog-meta-upload-binding"
	GCSUploadCreateOnlyName    = "x-goog-if-generation-match"
	GCSUploadCreateOnlyValue   = "0"
	S3UploadAttemptHeaderName  = "x-amz-meta-upload-attempt-id"
	S3UploadBindingHeaderName  = "x-amz-meta-upload-binding"
	S3UploadCreateOnlyName     = "If-None-Match"
	S3UploadCreateOnlyValue    = "*"
)

type UploadAttemptID struct {
	value string
}

func NewUploadAttemptID(entropy [core.RandomIdentityEntropyBytes]byte) (UploadAttemptID, error) {
	if entropy == [core.RandomIdentityEntropyBytes]byte{} {
		return UploadAttemptID{}, fmt.Errorf(ErrFmtUploadAttemptID, core.ErrReleaseContract)
	}
	return ParseUploadAttemptID(hex.EncodeToString(entropy[:]))
}

func ParseUploadAttemptID(value string) (UploadAttemptID, error) {
	parsed, err := parseReleaseRandomHex(value)
	if err != nil {
		return UploadAttemptID{}, fmt.Errorf(ErrFmtUploadAttemptID, err)
	}
	return UploadAttemptID{value: parsed}, nil
}

func parseReleaseRandomHex(value string) (string, error) {
	digest, err := core.ParseSHA256Hex(value)
	if err != nil || strings.Trim(digest.String(), "0") == "" {
		return "", core.ErrReleaseContract
	}
	return digest.String(), nil
}

func (id UploadAttemptID) String() string { return id.value }

func (id UploadAttemptID) IsZero() bool { return id.value == "" }

func (id UploadAttemptID) Validate() error {
	_, err := ParseUploadAttemptID(id.value)
	return err
}

func (id UploadAttemptID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *UploadAttemptID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtUploadAttemptID, core.ErrReleaseContract)
	}
	parsed, err := ParseUploadAttemptID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type UploadBinding struct {
	digest core.SHA256Hex
}

func ParseUploadBinding(value string) (UploadBinding, error) {
	digest, err := core.ParseSHA256Hex(value)
	if err != nil || digest.IsZero() {
		return UploadBinding{}, fmt.Errorf(ErrFmtUploadBinding, core.ErrReleaseContract)
	}
	return UploadBinding{digest: digest}, nil
}

func (b UploadBinding) String() string { return b.digest.String() }

func (b UploadBinding) IsZero() bool { return b.digest.IsZero() }

func (b UploadBinding) Validate() error {
	if err := b.digest.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadBinding, core.ErrReleaseContract)
	}
	return nil
}

func (b UploadBinding) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(b.String())
}

func (b *UploadBinding) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtUploadBinding, core.ErrReleaseContract)
	}
	parsed, err := ParseUploadBinding(value)
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

type UploadBindingInput struct {
	ReleaseID      ReleaseID
	ManifestSHA256 core.SHA256Hex
	Artifact       ArtifactName
	ArtifactSHA256 core.SHA256Hex
	Bucket         Bucket
	Object         ObjectKey
	AttemptID      UploadAttemptID
	ArtifactSize   core.ByteCount
	Product        core.Product
	Provider       core.StorageProvider
}

func (i UploadBindingInput) Validate() error {
	for _, err := range []error{
		i.Product.Validate(), i.ReleaseID.Validate(), i.ManifestSHA256.Validate(),
		i.Artifact.Validate(), i.ArtifactSHA256.Validate(), i.ArtifactSize.Validate(),
		i.Provider.Validate(), i.Bucket.Validate(), i.Object.Validate(), i.AttemptID.Validate(),
	} {
		if err != nil {
			return fmt.Errorf(ErrFmtUploadBinding, core.ErrReleaseContract)
		}
	}
	return nil
}

func DeriveUploadBinding(input UploadBindingInput) (UploadBinding, error) {
	canonical, err := input.canonical(nil)
	if err != nil {
		return UploadBinding{}, err
	}
	return UploadBinding{digest: core.NewSHA256Hex(sha256.Sum256(canonical))}, nil
}

func (i UploadBindingInput) canonical(dst []byte) ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, UploadBindingSchema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, i.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, i.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifestSHA256, i.ManifestSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldArtifact, i.Artifact)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldArtifactSHA256, i.ArtifactSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldArtifactSize, i.ArtifactSize)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProvider, i.Provider)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldBucket, i.Bucket)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldObject, i.Object)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUploadAttemptID, i.AttemptID)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func UploadAttemptHeader(provider core.StorageProvider, attemptID UploadAttemptID) (core.UploadHeader, error) {
	name, err := uploadMetadataHeaderName(provider, true)
	if err != nil || attemptID.Validate() != nil {
		return core.UploadHeader{}, fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
	return core.UploadHeader{Name: name, Value: attemptID.String()}, nil
}

func UploadBindingHeader(provider core.StorageProvider, binding UploadBinding) (core.UploadHeader, error) {
	name, err := uploadMetadataHeaderName(provider, false)
	if err != nil || binding.Validate() != nil {
		return core.UploadHeader{}, fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
	return core.UploadHeader{Name: name, Value: binding.String()}, nil
}

func UploadCreateOnlyHeader(provider core.StorageProvider) (core.UploadHeader, error) {
	switch provider {
	case core.StorageProviderGCS:
		return core.UploadHeader{Name: GCSUploadCreateOnlyName, Value: GCSUploadCreateOnlyValue}, nil
	case core.StorageProviderS3:
		return core.UploadHeader{Name: S3UploadCreateOnlyName, Value: S3UploadCreateOnlyValue}, nil
	default:
		return core.UploadHeader{}, fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
}

func validateUploadBindingHeaders(target UploadTarget) error {
	if err := core.ValidateUploadHeaders(target.Headers); err != nil {
		return wrapReleaseContract(ErrFmtUploadTarget, err)
	}
	attempt, err := UploadAttemptHeader(target.Provider, target.AttemptID)
	if err != nil || !containsExactUploadHeader(target.Headers, attempt) {
		return fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
	binding, err := UploadBindingHeader(target.Provider, target.Binding)
	if err != nil || !containsExactUploadHeader(target.Headers, binding) {
		return fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
	createOnly, err := UploadCreateOnlyHeader(target.Provider)
	if err != nil || !containsExactUploadHeader(target.Headers, createOnly) {
		return fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
	return nil
}

func containsExactUploadHeader(headers []core.UploadHeader, required core.UploadHeader) bool {
	for _, header := range headers {
		if strings.EqualFold(header.Name, required.Name) && header.Value == required.Value {
			return true
		}
	}
	return false
}

func validateUploadBinding(input UploadBindingInput, binding UploadBinding, errFmt string) error {
	derived, err := DeriveUploadBinding(input)
	if err != nil || derived != binding {
		return fmt.Errorf(errFmt, core.ErrReleaseContract)
	}
	return nil
}

func uploadMetadataHeaderName(provider core.StorageProvider, attempt bool) (string, error) {
	switch provider {
	case core.StorageProviderGCS:
		if attempt {
			return GCSUploadAttemptHeaderName, nil
		}
		return GCSUploadBindingHeaderName, nil
	case core.StorageProviderS3:
		if attempt {
			return S3UploadAttemptHeaderName, nil
		}
		return S3UploadBindingHeaderName, nil
	default:
		return "", fmt.Errorf(ErrFmtUploadTarget, core.ErrReleaseContract)
	}
}
