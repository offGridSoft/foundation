package license

import (
	"fmt"
	"time"

	"encoding/json"
	"github.com/offGridSoft/foundation/v2026/core"
)

type Body interface {
	Validate() error
	Canonical(dst []byte) ([]byte, error)
	ExpiresAt() core.UnixNanoTime
	WriteGraceUntil() core.UnixNanoTime
	CheckInAfter() core.UnixNanoTime
	CheckInBy() core.UnixNanoTime
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type SeatLeaseBody struct {
	DeveloperKeyID     DeveloperKeyID           `json:"developer_key_id"`
	DeviceFingerprint  core.DeviceFingerprint   `json:"device_fingerprint"`
	IssuedAt           core.UnixNanoTime        `json:"issued_at"`
	PaidUntil          core.UnixNanoTime        `json:"paid_until"`
	TokenExpiresAt     core.UnixNanoTime        `json:"lease_not_after"`
	CheckInAfterAt     core.UnixNanoTime        `json:"check_in_after"`
	CheckInByAt        core.UnixNanoTime        `json:"check_in_by"`
	WriteGraceDuration core.NanosecondsDuration `json:"write_grace_ns"`
	Schema             core.SchemaID            `json:"schema"`
	Plan               SeatPlan                 `json:"plan"`
	BillingPeriod      BillingPeriod            `json:"billing_period"`
	PrepaidYears       uint8                    `json:"prepaid_years"`
}

func (b SeatLeaseBody) Validate() error {
	if b.Schema != core.SchemaBugSeatLease {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.Plan.Validate(); err != nil {
		return err
	}
	if err := b.BillingPeriod.Validate(); err != nil {
		return err
	}
	if err := b.DeveloperKeyID.Validate(); err != nil {
		return err
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if err := validateCommonLeaseWindow(b); err != nil {
		return err
	}
	return validateLeaseBillingTerm(b.BillingPeriod, b.PrepaidYears)
}

func (b SeatLeaseBody) Canonical(dst []byte) ([]byte, error) {
	return core.AppendCanonicalJSON(dst, b)
}

func (b SeatLeaseBody) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendSeatLeaseBodyJSON(nil, b)
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

type leaseSchedule interface {
	ExpiresAt() core.UnixNanoTime
	CheckInAfter() core.UnixNanoTime
	CheckInBy() core.UnixNanoTime
}

func validCheckInWindow[B leaseSchedule](b B) bool {
	if b.CheckInAfter().IsZero() || b.CheckInBy().IsZero() {
		return false
	}
	if b.CheckInBy().Before(b.CheckInAfter()) {
		return false
	}
	return !b.CheckInBy().After(b.ExpiresAt())
}

type leaseWindowBody interface {
	Issued() core.UnixNanoTime
	PaidThrough() core.UnixNanoTime
	ExpiresAt() core.UnixNanoTime
	CheckInAfter() core.UnixNanoTime
	CheckInBy() core.UnixNanoTime
	WriteGrace() core.NanosecondsDuration
}

func validateCommonLeaseWindow[B leaseWindowBody](b B) error {
	if b.Issued().IsZero() || b.PaidThrough().IsZero() || b.ExpiresAt().IsZero() {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	if !b.ExpiresAt().After(b.Issued()) || !b.PaidThrough().After(b.Issued()) {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	if b.CheckInAfter().Before(b.PaidThrough()) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if b.CheckInAfter().Before(b.Issued()) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if !validCheckInWindow(b) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if b.WriteGrace().Duration() < 0 {
		return fmt.Errorf(ErrFmtLeaseWriteGrace, core.ErrLicenseContract)
	}
	return nil
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type SubscriptionLeaseBody struct {
	DeviceFingerprint  core.DeviceFingerprint   `json:"device_fingerprint"`
	IssuedAt           core.UnixNanoTime        `json:"issued_at"`
	PaidUntil          core.UnixNanoTime        `json:"paid_until"`
	TokenExpiresAt     core.UnixNanoTime        `json:"lease_not_after"`
	CheckInAfterAt     core.UnixNanoTime        `json:"check_in_after"`
	CheckInByAt        core.UnixNanoTime        `json:"check_in_by"`
	WriteGraceDuration core.NanosecondsDuration `json:"write_grace_ns"`
	Schema             core.SchemaID            `json:"schema"`
	Plan               SubscriptionPlan         `json:"plan"`
	BillingPeriod      BillingPeriod            `json:"billing_period"`
	PrepaidYears       uint8                    `json:"prepaid_years"`
}

func (b SubscriptionLeaseBody) Validate() error {
	if b.Schema != core.SchemaWitnessSubscription {
		return fmt.Errorf(ErrFmtSchema, core.ErrLicenseContract)
	}
	if err := b.Plan.Validate(); err != nil {
		return err
	}
	if err := b.BillingPeriod.Validate(); err != nil {
		return err
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if err := validateCommonLeaseWindow(b); err != nil {
		return err
	}
	return validateLeaseBillingTerm(b.BillingPeriod, b.PrepaidYears)
}

func (b SubscriptionLeaseBody) Canonical(dst []byte) ([]byte, error) {
	return core.AppendCanonicalJSON(dst, b)
}

func (b SubscriptionLeaseBody) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendSubscriptionLeaseBodyJSON(nil, b)
}

func appendSeatLeaseBodyJSON(dst []byte, b SeatLeaseBody) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeveloperKeyID, b.DeveloperKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPaidUntil, b.PaidUntil)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseNotAfter, b.TokenExpiresAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckInAfter, b.CheckInAfterAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckInBy, b.CheckInByAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriteGraceNS, b.WriteGraceDuration)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlan, b.Plan)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldBillingPeriod, b.BillingPeriod)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPrepaidYears, b.PrepaidYears)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func appendSubscriptionLeaseBodyJSON(dst []byte, b SubscriptionLeaseBody) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPaidUntil, b.PaidUntil)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseNotAfter, b.TokenExpiresAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckInAfter, b.CheckInAfterAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCheckInBy, b.CheckInByAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriteGraceNS, b.WriteGraceDuration)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlan, b.Plan)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldBillingPeriod, b.BillingPeriod)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPrepaidYears, b.PrepaidYears)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b SubscriptionLeaseBody) ExpiresAt() core.UnixNanoTime {
	return b.TokenExpiresAt
}

func (b SubscriptionLeaseBody) WriteGraceUntil() core.UnixNanoTime {
	return b.TokenExpiresAt.Add(b.WriteGraceDuration.Duration())
}

func (b SubscriptionLeaseBody) CheckInAfter() core.UnixNanoTime {
	return b.CheckInAfterAt
}

func (b SubscriptionLeaseBody) CheckInBy() core.UnixNanoTime {
	return b.CheckInByAt
}

func (b SeatLeaseBody) Issued() core.UnixNanoTime {
	return b.IssuedAt
}

func (b SeatLeaseBody) PaidThrough() core.UnixNanoTime {
	return b.PaidUntil
}

func (b SeatLeaseBody) WriteGrace() core.NanosecondsDuration {
	return b.WriteGraceDuration
}

func (b SubscriptionLeaseBody) Issued() core.UnixNanoTime {
	return b.IssuedAt
}

func (b SubscriptionLeaseBody) PaidThrough() core.UnixNanoTime {
	return b.PaidUntil
}

func (b SubscriptionLeaseBody) WriteGrace() core.NanosecondsDuration {
	return b.WriteGraceDuration
}

func validateLeaseBillingTerm(period BillingPeriod, prepaidYears uint8) error {
	switch period {
	case BillingPeriodFourWeeks:
		if prepaidYears != 0 {
			return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
		}
		return nil
	case BillingPeriodPrepaidYears:
		if prepaidYears == 0 {
			return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
		}
		return nil
	default:
		return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
	}
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

func (s LeaseState) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
	}
	return nil
}

func (s LeaseState) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

func ParseLeaseState(token string) (LeaseState, error) {
	for state := LeaseValid; int(state) < len(leaseStateNames); state++ {
		if leaseStateNames[state] == token {
			return state, nil
		}
	}
	return leaseStateInvalid, fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
}

func (s *LeaseState) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateState, core.ErrLicenseContract)
	}
	parsed, err := ParseLeaseState(token)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func (s LeaseState) Writable() bool {
	return s == LeaseValid || s == LeaseWarning || s == LeaseGrace
}

func Status[B Body](body B, now core.UnixNanoTime, warnWindow time.Duration) LeaseState {
	if warnWindow < 0 {
		warnWindow = 0
	}
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
