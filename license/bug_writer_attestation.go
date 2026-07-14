package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.CanonicalBody = BugWriterAttestationBody{}

// BugWriterAttestationBody binds one repository operation digest to the
// server-certified device writer that produced it. The seat lease certifies
// the public key; this body supplies immutable per-operation authorship.
type BugWriterAttestationBody struct {
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	WriterKeyID       core.SigningKeyID      `json:"writer_key_id"`
	OperationDigest   core.SHA256Hex         `json:"operation_sha256"`
	OccurredAt        core.UnixNanoTime      `json:"occurred_at"`
	Schema            core.SchemaID          `json:"schema"`
}

func (b BugWriterAttestationBody) Validate() error {
	if b.Schema != core.SchemaBugWriterAttestation {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return writerAttestationError(err)
	}
	if err := b.WriterKeyID.Validate(); err != nil {
		return writerAttestationError(err)
	}
	if err := b.OperationDigest.Validate(); err != nil {
		return writerAttestationError(err)
	}
	if err := b.OccurredAt.Validate(); err != nil || b.OccurredAt.IsZero() {
		return writerAttestationError(err)
	}
	return nil
}

func writerAttestationError(err error) error {
	if err == nil {
		err = core.ErrFoundationContract
	}
	return fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
}

func (b BugWriterAttestationBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriterKeyID, b.WriterKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOperationSHA256, b.OperationDigest)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOccurredAt, b.OccurredAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b BugWriterAttestationBody) SigningSchema() core.SchemaID { return b.Schema }

func (b BugWriterAttestationBody) MarshalJSON() ([]byte, error) { return b.Canonical(nil) }
