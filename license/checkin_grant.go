package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// CheckInGrant is the compiler-owned success payload carried by a check-in
// response. Each product owns an exact grant type; there is no optional field
// bag whose meaning changes by product.
type CheckInGrant interface {
	core.Validatable
	Verify(core.SigningKeyring) error
}

type BugCheckInGrant struct {
	Lease             core.Signed[SeatLeaseBody]            `json:"lease"`
	WriterCertificate core.Signed[BugWriterCertificateBody] `json:"writer_certificate"`
}

func (g BugCheckInGrant) Validate() error {
	if err := g.Lease.Validate(); err != nil {
		return checkInGrantError(err)
	}
	if err := g.WriterCertificate.Validate(); err != nil {
		return checkInGrantError(err)
	}
	lease := g.Lease.Body
	certificate := g.WriterCertificate.Body
	if lease.DeviceFingerprint != certificate.DeviceFingerprint || lease.Writer != certificate.Writer {
		return checkInGrantError(core.ErrLicenseContract)
	}
	if certificate.IssuedAt.After(lease.IssuedAt) || certificate.ValidUntil.Before(lease.WriteGraceUntil()) {
		return checkInGrantError(core.ErrLicenseContract)
	}
	return nil
}

func (g BugCheckInGrant) Verify(keyring core.SigningKeyring) error {
	if err := g.Validate(); err != nil {
		return err
	}
	if err := g.Lease.Verify(keyring); err != nil {
		return checkInGrantError(err)
	}
	if err := g.WriterCertificate.Verify(keyring); err != nil {
		return checkInGrantError(err)
	}
	return nil
}

func (BugCheckInGrant) TimeCommitmentSchema() core.SchemaID {
	return core.SchemaBugCheckInTimeCommitment
}

func (g BugCheckInGrant) LeaseIdentity() CheckInLeaseIdentity {
	lease := g.Lease.Body
	return CheckInLeaseIdentity{
		DeviceFingerprint: lease.DeviceFingerprint,
		LeaseID:           lease.LeaseID,
		Generation:        lease.Generation,
	}
}

type WitnessCheckInGrant struct {
	Lease core.Signed[SubscriptionLeaseBody] `json:"lease"`
}

func (g WitnessCheckInGrant) Validate() error {
	if err := g.Lease.Validate(); err != nil {
		return checkInGrantError(err)
	}
	return nil
}

func (g WitnessCheckInGrant) Verify(keyring core.SigningKeyring) error {
	if err := g.Validate(); err != nil {
		return err
	}
	if err := g.Lease.Verify(keyring); err != nil {
		return checkInGrantError(err)
	}
	return nil
}

func (WitnessCheckInGrant) TimeCommitmentSchema() core.SchemaID {
	return core.SchemaWitnessCheckInTimeCommitment
}

func (g WitnessCheckInGrant) LeaseIdentity() CheckInLeaseIdentity {
	lease := g.Lease.Body
	return CheckInLeaseIdentity{
		DeviceFingerprint: lease.DeviceFingerprint,
		LeaseID:           lease.LeaseID,
		Generation:        lease.Generation,
	}
}

func checkInGrantError(err error) error {
	return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
}
