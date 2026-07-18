package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.CanonicalBody = CheckInTimeCommitmentBody[BugCheckInGrant]{}
	_ core.CanonicalBody = CheckInTimeCommitmentBody[WitnessCheckInGrant]{}
)

// CheckInTimeCommitmentGrant is the compiler-owned binding between a product
// check-in grant and its signed server-time commitment. Each grant type pins
// the exact commitment schema its product signs under; because the signing
// domain is resolved from that schema, a commitment instantiated for one
// product can never validate — and therefore never verify — under another
// product's grant type.
type CheckInTimeCommitmentGrant interface {
	core.Validatable
	TimeCommitmentSchema() core.SchemaID
	LeaseIdentity() CheckInLeaseIdentity
}

// CheckInLeaseIdentity is the durable lease identity a time commitment binds
// to: the exact device, lease, and generation the server observed. It is a
// projection from a validated grant, never an independently trusted input.
type CheckInLeaseIdentity struct {
	DeviceFingerprint core.DeviceFingerprint
	LeaseID           core.LeaseID
	Generation        LeaseGeneration
}

// CheckInTimeCommitmentBody is the narrow server-time authority persisted by
// a product after a granted online check-in. Its independent signature binds
// the observed time to the exact device, lease progression, and response
// nonce. Local clocks remain availability inputs and are never promoted into
// this authority. The grant type parameter pins the product schema and the
// grant MatchesGrant binds against; wire bytes are identical across products
// except for the product-distinct schema value.
type CheckInTimeCommitmentBody[G CheckInTimeCommitmentGrant] struct {
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
	LeaseID           core.LeaseID           `json:"lease_id"`
	RequestNonce      CheckInNonce           `json:"request_nonce"`
	ServerObservedAt  core.UnixNanoTime      `json:"server_observed_at"`
	LeaseGeneration   LeaseGeneration        `json:"lease_generation"`
	Schema            core.SchemaID          `json:"schema"`
}

func (b CheckInTimeCommitmentBody[G]) Validate() error {
	var grant G
	if b.Schema != grant.TimeCommitmentSchema() {
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

func (b CheckInTimeCommitmentBody[G]) Canonical(dst []byte) ([]byte, error) {
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

func (b CheckInTimeCommitmentBody[G]) SigningSchema() core.SchemaID { return b.Schema }

func (b CheckInTimeCommitmentBody[G]) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

func (b CheckInTimeCommitmentBody[G]) MatchesGrant(grant G, nonce CheckInNonce) error {
	if err := b.matchesStoredGrant(grant); err != nil {
		return err
	}
	if err := nonce.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	if b.RequestNonce != nonce {
		return checkInTimeCommitmentError(core.ErrLicenseContract)
	}
	return nil
}

// MatchesStoredGrant verifies the durable grant bindings that remain
// independently meaningful after the initiating request nonce has been
// consumed. The signed commitment still retains that nonce as audit evidence;
// resolve-time validation deliberately does not pretend to compare it against
// an unavailable pending request.
func (b CheckInTimeCommitmentBody[G]) MatchesStoredGrant(grant G) error {
	return b.matchesStoredGrant(grant)
}

func (b CheckInTimeCommitmentBody[G]) matchesStoredGrant(grant G) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := grant.Validate(); err != nil {
		return checkInTimeCommitmentError(err)
	}
	lease := grant.LeaseIdentity()
	if b.DeviceFingerprint.String() != lease.DeviceFingerprint.String() ||
		b.LeaseID.String() != lease.LeaseID.String() ||
		b.LeaseGeneration != lease.Generation {
		return checkInTimeCommitmentError(core.ErrLicenseContract)
	}
	return nil
}

func checkInTimeCommitmentError(err error) error {
	return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
}
