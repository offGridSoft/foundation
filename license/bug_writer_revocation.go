package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// BugWriterRevocationBody is the server-signed cutoff for one certified Bug
// writer. Attestations before RevokedAt remain historical evidence; an
// attestation at or after RevokedAt must be refused.
type BugWriterRevocationBody struct {
	WriterKeyID core.SigningKeyID `json:"writer_key_id"`
	RevokedAt   core.UnixNanoTime `json:"revoked_at"`
	Schema      core.SchemaID     `json:"schema"`
}

func (b BugWriterRevocationBody) Validate() error {
	if b.Schema != core.SchemaBugWriterRevocation {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.WriterKeyID.Validate(); err != nil {
		return writerRevocationError(err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.RevokedAt); err != nil || b.RevokedAt.IsZero() {
		return writerRevocationError(err)
	}
	return nil
}

func writerRevocationError(err error) error {
	if err == nil {
		err = core.ErrFoundationContract
	}
	return fmt.Errorf(ErrFmtCheckInPayload, errors.Join(core.ErrLicenseContract, err))
}

func (b BugWriterRevocationBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriterKeyID, b.WriterKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRevokedAt, b.RevokedAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b BugWriterRevocationBody) SigningSchema() core.SchemaID { return b.Schema }

func (b BugWriterRevocationBody) MarshalJSON() ([]byte, error) { return b.Canonical(nil) }
