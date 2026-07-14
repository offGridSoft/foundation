package license

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.CanonicalBody = SeatLeaseBody{}
	_ core.CanonicalBody = SubscriptionLeaseBody{}
)

type Body interface {
	core.CanonicalBody
	ExpiresAt() core.UnixNanoTime
	WriteGraceUntil() core.UnixNanoTime
	CheckInAfter() core.UnixNanoTime
	CheckInBy() core.UnixNanoTime
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type SeatLeaseBody struct {
	Writer             BugWriterKey             `json:"writer"`
	LeaseID            core.LeaseID             `json:"lease_id"`
	DeveloperKeyID     DeveloperKeyID           `json:"developer_key_id"`
	DeviceFingerprint  core.DeviceFingerprint   `json:"device_fingerprint"`
	PaidUntil          core.UnixNanoTime        `json:"paid_until"`
	IssuedAt           core.UnixNanoTime        `json:"issued_at"`
	TokenExpiresAt     core.UnixNanoTime        `json:"lease_not_after"`
	CheckInAfterAt     core.UnixNanoTime        `json:"check_in_after"`
	CheckInByAt        core.UnixNanoTime        `json:"check_in_by"`
	Generation         LeaseGeneration          `json:"lease_generation"`
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
	if err := b.validateIdentity(); err != nil {
		return err
	}
	return b.validateWindowAndBilling()
}

func (b SeatLeaseBody) validateIdentity() error {
	if err := b.LeaseID.Validate(); err != nil {
		return err
	}
	if err := b.Generation.Validate(); err != nil {
		return err
	}
	if err := b.DeveloperKeyID.Validate(); err != nil {
		return err
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	if err := b.Writer.Validate(); err != nil {
		return err
	}
	return nil
}

func (b SeatLeaseBody) validateWindowAndBilling() error {
	if err := validateCommonLeaseWindow(b); err != nil {
		return err
	}
	if err := validateCollectionLeaseWindow(b.BillingPeriod, b.CheckInAfterAt, b.PaidUntil); err != nil {
		return err
	}
	if err := validateLeaseBillingTerm(b.BillingPeriod, b.PrepaidYears); err != nil {
		return err
	}
	return validateSeatLeasePlanBilling(b.Plan, b.BillingPeriod)
}

func (b SeatLeaseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendSeatLeaseBodyJSON(dst, b)
}

func (b SeatLeaseBody) SigningSchema() core.SchemaID {
	return b.Schema
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
	if err := validateRequiredLeaseTimes(b); err != nil {
		return err
	}
	if !b.ExpiresAt().After(b.Issued()) || !b.PaidThrough().After(b.Issued()) {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
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
	if _, err := core.AddUnixNanoDuration(b.ExpiresAt(), b.WriteGrace().Duration()); err != nil {
		return fmt.Errorf(ErrFmtLeaseWriteGrace, core.ErrLicenseContract)
	}
	return nil
}

func validateRequiredLeaseTimes[B leaseWindowBody](b B) error {
	if err := validateRequiredLeaseBoundaryTime(b.Issued()); err != nil {
		return err
	}
	if err := validateRequiredLeaseBoundaryTime(b.PaidThrough()); err != nil {
		return err
	}
	if err := validateRequiredLeaseBoundaryTime(b.ExpiresAt()); err != nil {
		return err
	}
	return validateRequiredCheckInTimes(b.CheckInAfter(), b.CheckInBy())
}

func validateRequiredLeaseBoundaryTime(value core.UnixNanoTime) error {
	if err := core.ValidateRequiredUnixNanoTime(value); err != nil {
		return fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	return nil
}

func validateRequiredCheckInTimes(checkInAfter core.UnixNanoTime, checkInBy core.UnixNanoTime) error {
	if err := core.ValidateRequiredUnixNanoTime(checkInAfter); err != nil {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(checkInBy); err != nil {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	return nil
}

func validateCollectionLeaseWindow(period BillingPeriod, checkInAfter core.UnixNanoTime, paidUntil core.UnixNanoTime) error {
	if period == BillingPeriodMonthly && checkInAfter.Before(paidUntil) {
		return fmt.Errorf(ErrFmtLeaseCheckInWindow, core.ErrLicenseContract)
	}
	return nil
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type SubscriptionLeaseBody struct {
	DeviceFingerprint  core.DeviceFingerprint   `json:"device_fingerprint"`
	LeaseID            core.LeaseID             `json:"lease_id"`
	CheckInAfterAt     core.UnixNanoTime        `json:"check_in_after"`
	IssuedAt           core.UnixNanoTime        `json:"issued_at"`
	PaidUntil          core.UnixNanoTime        `json:"paid_until"`
	TokenExpiresAt     core.UnixNanoTime        `json:"lease_not_after"`
	CheckInByAt        core.UnixNanoTime        `json:"check_in_by"`
	Generation         LeaseGeneration          `json:"lease_generation"`
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
	if err := b.validateIdentity(); err != nil {
		return err
	}
	return b.validateWindowAndBilling()
}

func (b SubscriptionLeaseBody) validateIdentity() error {
	if err := b.LeaseID.Validate(); err != nil {
		return err
	}
	if err := b.Generation.Validate(); err != nil {
		return err
	}
	if err := b.DeviceFingerprint.Validate(); err != nil {
		return err
	}
	return nil
}

func (b SubscriptionLeaseBody) validateWindowAndBilling() error {
	if err := validateCommonLeaseWindow(b); err != nil {
		return err
	}
	if err := validateCollectionLeaseWindow(b.BillingPeriod, b.CheckInAfterAt, b.PaidUntil); err != nil {
		return err
	}
	if err := validateLeaseBillingTerm(b.BillingPeriod, b.PrepaidYears); err != nil {
		return err
	}
	return validateSubscriptionLeaseBilling(b.BillingPeriod)
}

func (b SubscriptionLeaseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendSubscriptionLeaseBodyJSON(dst, b)
}

func (b SubscriptionLeaseBody) SigningSchema() core.SchemaID {
	return b.Schema
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
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseID, b.LeaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseGeneration, b.Generation)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeveloperKeyID, b.DeveloperKeyID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDeviceFingerprint, b.DeviceFingerprint)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriter, b.Writer)
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
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseID, b.LeaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLeaseGeneration, b.Generation)
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
	case BillingPeriodMonthly:
		if prepaidYears != 0 {
			return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
		}
		return nil
	case BillingPeriodPrepaidYears:
		if prepaidYears < BugEnterpriseMinPrepaidYears || prepaidYears > BugEnterpriseMaxPrepaidYears {
			return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
		}
		return nil
	default:
		return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
	}
}

func validateSeatLeasePlanBilling(plan SeatPlan, period BillingPeriod) error {
	switch plan {
	case SeatPlanEnterpriseOffline:
		if period == BillingPeriodPrepaidYears {
			return nil
		}
	case SeatPlanStandard, SeatPlanEnterprise, SeatPlanOSS:
		if period == BillingPeriodMonthly {
			return nil
		}
	}
	return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
}

func validateSubscriptionLeaseBilling(period BillingPeriod) error {
	if period != BillingPeriodMonthly {
		return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
	}
	return nil
}

type LeaseState uint8

const (
	leaseStateInvalid LeaseState = iota
	LeaseValid
	LeaseWarning
	LeaseGrace
	LeaseExpired
)

const (
	LeaseStateTokenValid   = "valid"
	LeaseStateTokenWarning = "expiring"
	LeaseStateTokenGrace   = "grace"
	LeaseStateTokenExpired = "expired"
)

func leaseStateNames() [LeaseExpired + 1]string {
	return [...]string{
		LeaseValid:   LeaseStateTokenValid,
		LeaseWarning: LeaseStateTokenWarning,
		LeaseGrace:   LeaseStateTokenGrace,
		LeaseExpired: LeaseStateTokenExpired,
	}
}

func (s LeaseState) String() string {
	if s.IsValid() {
		return leaseStateNames()[s]
	}
	return ""
}

func (s LeaseState) IsValid() bool {
	return s > leaseStateInvalid && int(s) < len(leaseStateNames()) && leaseStateNames()[s] != ""
}

func (s LeaseState) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtLeaseState, core.ErrLicenseContract)
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
	for state := LeaseValid; int(state) < len(leaseStateNames()); state++ {
		if leaseStateNames()[state] == token {
			return state, nil
		}
	}
	return leaseStateInvalid, fmt.Errorf(ErrFmtLeaseState, core.ErrLicenseContract)
}

func (s *LeaseState) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtLeaseState, core.ErrLicenseContract)
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
	if now.IsZero() || warnWindow < 0 {
		return leaseStateInvalid
	}
	if err := body.Validate(); err != nil {
		return leaseStateInvalid
	}
	return leaseStatus(body, now, warnWindow)
}

func leaseStatus[B Body](body B, now core.UnixNanoTime, warnWindow time.Duration) LeaseState {
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
