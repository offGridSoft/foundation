package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.CanonicalBody = BugWriterCertificateBody{}

// BugWriterCertificateBody is the public, repository-safe identity that the
// license server certifies for one Bug writer. It contains no developer key,
// account, plan, billing, repository, source, path, or bug metadata.
type BugWriterCertificateBody struct {
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	Writer            BugWriterKey           `json:"writer"`
	IssuedAt          core.UnixNanoTime      `json:"issued_at"`
	ValidUntil        core.UnixNanoTime      `json:"valid_until"`
	Schema            core.SchemaID          `json:"schema"`
}

func (b BugWriterCertificateBody) Validate() error {
	if b.Schema != core.SchemaBugWriterCertificate {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return writerCertificateError(err)
	}
	if err := b.Writer.Validate(); err != nil {
		return writerCertificateError(err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.IssuedAt); err != nil {
		return writerCertificateError(err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.ValidUntil); err != nil {
		return writerCertificateError(err)
	}
	if !b.ValidUntil.After(b.IssuedAt) {
		return writerCertificateError(core.ErrLicenseContract)
	}
	return nil
}

func writerCertificateError(err error) error {
	return fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
}

func (b BugWriterCertificateBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriter, b.Writer)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldValidUntil, b.ValidUntil)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b BugWriterCertificateBody) SigningSchema() core.SchemaID { return b.Schema }

func (b BugWriterCertificateBody) MarshalJSON() ([]byte, error) { return b.Canonical(nil) }
