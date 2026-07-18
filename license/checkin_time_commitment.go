package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.CanonicalBody = BugCheckInTimeCommitmentBody{}

// BugCheckInTimeCommitmentBody is the narrow server-time authority persisted
// by Bug after a granted online check-in. Its independent signature binds the
// observed time to the exact device, lease progression, and response nonce.
// Local clocks remain availability inputs and are never promoted into this
// authority.
type BugCheckInTimeCommitmentBody struct {
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	LeaseID           core.LeaseID           `json:"lease_id"`
	RequestNonce      CheckInNonce           `json:"request_nonce"`
	ServerObservedAt  core.UnixNanoTime      `json:"server_observed_at"`
	LeaseGeneration   LeaseGeneration        `json:"lease_generation"`
	Schema            core.SchemaID          `json:"schema"`
}

func (b BugCheckInTimeCommitmentBody) Validate() error {
	if b.Schema != core.SchemaBugCheckInTimeCommitment {
		return checkInTimeCommitmentError(core.ErrLicenseContract)
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	if err := b.LeaseID.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	if err := b.LeaseGeneration.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	if err := b.RequestNonce.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.ServerObservedAt); err != nil {
		return checkInTimeCommitmentError(err)
	}
	return nil
}

func (b BugCheckInTimeCommitmentBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseID, b.LeaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseGeneration, b.LeaseGeneration)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestNonce, b.RequestNonce)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldServerObservedAt, b.ServerObservedAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b BugCheckInTimeCommitmentBody) SigningSchema() core.SchemaID { return b.Schema }

func (b BugCheckInTimeCommitmentBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

func (b BugCheckInTimeCommitmentBody) MatchesGrant(grant BugCheckInGrant, nonce CheckInNonce) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := grant.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	lease := grant.Lease.Body
	if b.DeviceFingerprint.String() != lease.DeviceFingerprint.String() ||
		b.LeaseID.String() != lease.LeaseID.String() ||
		b.LeaseGeneration != lease.Generation ||
		b.RequestNonce != nonce {
		return checkInTimeCommitmentError(core.ErrLicenseContract)
	}
	return nil
}

func checkInTimeCommitmentError(err error) error {
	return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
}
