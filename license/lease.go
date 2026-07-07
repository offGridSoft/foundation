package license

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

type Body interface {
	Validate() error
	Canonical(dst []byte) ([]byte, error)
	ExpiresAt() core.UnixNanoTime
	WriteGraceUntil() core.UnixNanoTime
	CheckInAfter() core.UnixNanoTime
	CheckInBy() core.UnixNanoTime
}

type SeatLeaseBody struct {
	IssuedAt           core.UnixNanoTime        `json:"issued_at"`
	TokenExpiresAt     core.UnixNanoTime        `json:"expires_at"`
	CheckInAfterAt     core.UnixNanoTime        `json:"check_in_after"`
	CheckInByAt        core.UnixNanoTime        `json:"check_in_by"`
	Schema             string                   `json:"schema"`
	DeveloperKeyID     DeveloperKeyID           `json:"developer_key_id"`
	DeviceFingerprint  core.DeviceFingerprint   `json:"device_fingerprint"`
	WriteGraceDuration core.NanosecondsDuration `json:"write_grace_ns"`
	Plan               SeatPlan                 `json:"plan"`
}

func (b SeatLeaseBody) Validate() error {
	if b.Schema != SchemaBugSeatLease {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.Plan.Validate(); err != nil {
		return err
	}
	if err := b.DeveloperKeyID.Validate(); err != nil {
		return err
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if b.IssuedAt.IsZero() || !b.TokenExpiresAt.After(b.IssuedAt) {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	if b.CheckInAfterAt.Before(b.IssuedAt) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if !validCheckInWindow(b) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if b.WriteGraceDuration.Duration() < 0 {
		return fmt.Errorf(ErrFmtLeaseWriteGrace, core.ErrLicenseContract)
	}
	return nil
}

func (b SeatLeaseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return core.AppendCanonicalJSON(dst, b)
}

func (b SeatLeaseBody) ExpiresAt() core.UnixNanoTime {
	return b.TokenExpiresAt
}

func (b SeatLeaseBody) WriteGraceUntil() core.UnixNanoTime {
	return b.TokenExpiresAt.Add(b.WriteGraceDuration.Duration())
}

func (b SeatLeaseBody) CheckInAfter() core.UnixNanoTime {
	return b.CheckInAfterAt
}

func (b SeatLeaseBody) CheckInBy() core.UnixNanoTime {
	return b.CheckInByAt
}

func validCheckInWindow[B Body](b B) bool {
	if b.CheckInAfter().IsZero() || b.CheckInBy().IsZero() {
		return false
	}
	if b.CheckInBy().Before(b.CheckInAfter()) {
		return false
	}
	return !b.CheckInBy().After(b.ExpiresAt())
}

type SubscriptionLeaseBody struct {
	PaidUntil      core.UnixNanoTime `json:"paid_until"`
	TokenExpiresAt core.UnixNanoTime `json:"lease_not_after"`
	CheckInAfterAt core.UnixNanoTime `json:"check_in_after"`
	CheckInByAt    core.UnixNanoTime `json:"check_in_by"`
	Schema         string            `json:"schema"`
	Plan           SubscriptionPlan  `json:"plan"`
	BillingPeriod  BillingPeriod     `json:"billing_period"`
}

func (b SubscriptionLeaseBody) Validate() error {
	if b.Schema != SchemaWitnessSubscription {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.Plan.Validate(); err != nil {
		return err
	}
	if err := b.BillingPeriod.Validate(); err != nil {
		return err
	}
	if b.PaidUntil.IsZero() || b.TokenExpiresAt.IsZero() {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	if !validCheckInWindow(b) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	return nil
}

func (b SubscriptionLeaseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return core.AppendCanonicalJSON(dst, b)
}

func (b SubscriptionLeaseBody) ExpiresAt() core.UnixNanoTime {
	return b.TokenExpiresAt
}

func (b SubscriptionLeaseBody) WriteGraceUntil() core.UnixNanoTime {
	return b.TokenExpiresAt
}

func (b SubscriptionLeaseBody) CheckInAfter() core.UnixNanoTime {
	return b.CheckInAfterAt
}

func (b SubscriptionLeaseBody) CheckInBy() core.UnixNanoTime {
	return b.CheckInByAt
}

type LeaseState uint8

const (
	leaseStateInvalid LeaseState = iota
	LeaseValid
	LeaseWarning
	LeaseGrace
	LeaseExpired
)

var leaseStateNames = [...]string{
	LeaseValid:   "valid",
	LeaseWarning: "expiring",
	LeaseGrace:   "grace",
	LeaseExpired: "expired",
}

func (s LeaseState) String() string {
	if s.IsValid() {
		return leaseStateNames[s]
	}
	return ""
}

func (s LeaseState) IsValid() bool {
	return s > leaseStateInvalid && int(s) < len(leaseStateNames) && leaseStateNames[s] != ""
}

func (s LeaseState) MarshalJSON() ([]byte, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
	}
	return json.Marshal(s.String())
}

func (s *LeaseState) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
	}
	for state := LeaseValid; int(state) < len(leaseStateNames); state++ {
		if leaseStateNames[state] == token {
			*s = state
			return nil
		}
	}
	return fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
}

func (s LeaseState) Writable() bool {
	return s == LeaseValid || s == LeaseWarning || s == LeaseGrace
}

func Status[B Body](body B, now core.UnixNanoTime, warnWindow time.Duration) LeaseState {
	switch {
	case now.Before(body.ExpiresAt().Add(-warnWindow)):
		return LeaseValid
	case now.Before(body.ExpiresAt()):
		return LeaseWarning
	case now.Before(body.WriteGraceUntil()):
		return LeaseGrace
	default:
		return LeaseExpired
	}
}

func CheckInDue[B Body](body B, now core.UnixNanoTime) bool {
	return !now.Before(body.CheckInAfter())
}
