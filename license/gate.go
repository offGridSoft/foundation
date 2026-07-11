package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

type GateOutcome uint8

const (
	gateOutcomeInvalid GateOutcome = iota
	GateAllow
	GateWarn
	GateTeach
	GateRefuse
)

const (
	GateOutcomeTokenAllow  = "allow"
	GateOutcomeTokenWarn   = "warn"
	GateOutcomeTokenTeach  = "teach"
	GateOutcomeTokenRefuse = "refuse"
)

func gateOutcomeNames() [GateRefuse + 1]string {
	return [...]string{
		GateAllow:  GateOutcomeTokenAllow,
		GateWarn:   GateOutcomeTokenWarn,
		GateTeach:  GateOutcomeTokenTeach,
		GateRefuse: GateOutcomeTokenRefuse,
	}
}

func (o GateOutcome) String() string {
	if o.IsValid() {
		return gateOutcomeNames()[o]
	}
	return ""
}

func (o GateOutcome) IsValid() bool {
	return o > gateOutcomeInvalid && int(o) < len(gateOutcomeNames()) && gateOutcomeNames()[o] != ""
}

func (o GateOutcome) Validate() error {
	if !o.IsValid() {
		return fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
	}
	return nil
}

func (o GateOutcome) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}

func ParseGateOutcome(token string) (GateOutcome, error) {
	for outcome := GateAllow; int(outcome) < len(gateOutcomeNames()); outcome++ {
		if gateOutcomeNames()[outcome] == token {
			return outcome, nil
		}
	}
	return gateOutcomeInvalid, fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
}

func (o *GateOutcome) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
	}
	parsed, err := ParseGateOutcome(token)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

func (o GateOutcome) Writable() bool {
	return o == GateAllow || o == GateWarn
}

type LeaseTrust uint8

const (
	leaseTrustInvalid LeaseTrust = iota
	LeaseTrustUntrusted
	LeaseTrustTrusted
)

const (
	LeaseTrustTokenUntrusted = "untrusted"
	LeaseTrustTokenTrusted   = "trusted"
)

func leaseTrustNames() [LeaseTrustTrusted + 1]string {
	return [...]string{
		LeaseTrustUntrusted: LeaseTrustTokenUntrusted,
		LeaseTrustTrusted:   LeaseTrustTokenTrusted,
	}
}

func (t LeaseTrust) String() string {
	if t.IsValid() {
		return leaseTrustNames()[t]
	}
	return ""
}

func (t LeaseTrust) IsValid() bool {
	return t > leaseTrustInvalid && int(t) < len(leaseTrustNames()) && leaseTrustNames()[t] != ""
}

func (t LeaseTrust) Validate() error {
	if !t.IsValid() {
		return fmt.Errorf(ErrFmtLeaseTrust, core.ErrLicenseContract)
	}
	return nil
}

func (t LeaseTrust) Trusted() bool {
	return t == LeaseTrustTrusted
}

func (t LeaseTrust) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t.String())
}

func ParseLeaseTrust(token string) (LeaseTrust, error) {
	for trust := LeaseTrustUntrusted; int(trust) < len(leaseTrustNames()); trust++ {
		if leaseTrustNames()[trust] == token {
			return trust, nil
		}
	}
	return leaseTrustInvalid, fmt.Errorf(ErrFmtLeaseTrust, core.ErrLicenseContract)
}

func (t *LeaseTrust) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtLeaseTrust, core.ErrLicenseContract)
	}
	parsed, err := ParseLeaseTrust(token)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

type GateInput[B Body] struct {
	Lease          B
	Now            core.UnixNanoTime
	ClockHighWater core.UnixNanoTime
	WarnWindow     time.Duration
	Trust          LeaseTrust
	Disarmed       bool
	TeachingShown  bool
}

type GateDecision struct {
	Remaining time.Duration
	Reason    GateReason
	State     LeaseState
}

type GateInputError struct {
	Reason GateReason
}

func (e GateInputError) Error() string {
	return ErrMsgGateInput
}

func (e GateInputError) Unwrap() error {
	return core.ErrLicenseContract
}

func (in GateInput[B]) Validate() error {
	reason := gateInputViolation(in)
	if reason == gateReasonInvalid {
		return nil
	}
	return GateInputError{Reason: reason}
}

func gateInputViolation[B Body](in GateInput[B]) GateReason {
	switch {
	case in.Disarmed:
		return gateReasonInvalid
	case in.Now.IsZero():
		return GateReasonInvalidClock
	case in.ClockHighWater.After(in.Now):
		return GateReasonClockRollback
	case in.WarnWindow < 0:
		return GateReasonInvalidWindow
	case !in.Trust.IsValid():
		return GateReasonInvalidTrust
	case in.Trust.Trusted():
		if err := in.Lease.Validate(); err != nil {
			return GateReasonInvalidLease
		}
	}
	return gateReasonInvalid
}

func (d GateDecision) Validate() error {
	if err := d.Reason.Validate(); err != nil {
		return fmt.Errorf(ErrFmtGateDecision, err)
	}
	switch d.Reason {
	case GateReasonLeaseValid:
		return validateGateDecisionState(d, LeaseValid, false)
	case GateReasonLeaseWarning:
		return validateGateDecisionState(d, LeaseWarning, true)
	case GateReasonLeaseGrace:
		return validateGateDecisionState(d, LeaseGrace, true)
	case GateReasonLeaseExpired:
		return validateGateDecisionState(d, LeaseExpired, false)
	case GateReasonDisarmed, GateReasonInvalidClock, GateReasonClockRollback, GateReasonInvalidTrust,
		GateReasonTeaching, GateReasonMissingTrustedLease, GateReasonInvalidWindow, GateReasonInvalidLease:
		if d.State != leaseStateInvalid || d.Remaining != 0 {
			return fmt.Errorf(ErrFmtGateDecision, core.ErrLicenseContract)
		}
		return nil
	default:
		return fmt.Errorf(ErrFmtGateDecision, core.ErrLicenseContract)
	}
}

func validateGateDecisionState(d GateDecision, state LeaseState, requireRemaining bool) error {
	if d.State != state {
		return fmt.Errorf(ErrFmtGateDecision, core.ErrLicenseContract)
	}
	if requireRemaining && d.Remaining <= 0 {
		return fmt.Errorf(ErrFmtGateDecision, core.ErrLicenseContract)
	}
	if !requireRemaining && d.Remaining != 0 {
		return fmt.Errorf(ErrFmtGateDecision, core.ErrLicenseContract)
	}
	return nil
}

func Gate[B Body](in GateInput[B]) GateDecision {
	if err := in.Validate(); err != nil {
		if inputErr, ok := errors.AsType[GateInputError](err); ok {
			return validatedGateDecision(GateDecision{Reason: inputErr.Reason})
		}
		return validatedGateDecision(GateDecision{Reason: GateReasonInvalidTrust})
	}
	if in.Disarmed {
		return validatedGateDecision(GateDecision{Reason: GateReasonDisarmed})
	}
	var decision GateDecision
	if !in.Trust.Trusted() {
		decision = untrustedLeaseDecision(in.TeachingShown)
	} else {
		decision = trustedLeaseDecision(in.Now, in.Lease, in.WarnWindow)
	}
	return validatedGateDecision(decision)
}

func validatedGateDecision(decision GateDecision) GateDecision {
	if err := decision.Validate(); err != nil {
		return GateDecision{Reason: GateReasonInvalidTrust}
	}
	return decision
}

func untrustedLeaseDecision(teachingShown bool) GateDecision {
	if !teachingShown {
		return GateDecision{Reason: GateReasonTeaching}
	}
	return GateDecision{Reason: GateReasonMissingTrustedLease}
}

func trustedLeaseDecision[B Body](
	now core.UnixNanoTime,
	lease B,
	warnWindow time.Duration,
) GateDecision {
	state := leaseStatus(lease, now, warnWindow)
	switch state {
	case LeaseValid:
		return GateDecision{Reason: GateReasonLeaseValid, State: state}
	case LeaseWarning:
		return GateDecision{
			Remaining: lease.ExpiresAt().Sub(now),
			Reason:    GateReasonLeaseWarning,
			State:     state,
		}
	case LeaseGrace:
		return GateDecision{
			Remaining: lease.WriteGraceUntil().Sub(now),
			Reason:    GateReasonLeaseGrace,
			State:     state,
		}
	default:
		return GateDecision{Reason: GateReasonLeaseExpired, State: state}
	}
}

type GateReason uint8

const (
	gateReasonInvalid GateReason = iota
	GateReasonDisarmed
	GateReasonInvalidClock
	GateReasonClockRollback
	GateReasonInvalidTrust
	GateReasonTeaching
	GateReasonMissingTrustedLease
	GateReasonLeaseValid
	GateReasonLeaseWarning
	GateReasonLeaseGrace
	GateReasonLeaseExpired
	GateReasonInvalidWindow
	GateReasonInvalidLease
)

const (
	GateReasonTokenDisarmed            = "disarmed"
	GateReasonTokenInvalidClock        = "invalid_clock"
	GateReasonTokenClockRollback       = "clock_rollback"
	GateReasonTokenInvalidTrust        = "invalid_trust_resolution"
	GateReasonTokenTeaching            = "teaching"
	GateReasonTokenMissingTrustedLease = "missing_trusted_lease"
	GateReasonTokenLeaseValid          = "lease_valid"
	GateReasonTokenLeaseWarning        = "lease_warning"
	GateReasonTokenLeaseGrace          = "lease_grace"
	GateReasonTokenLeaseExpired        = "lease_expired"
	GateReasonTokenInvalidWindow       = "invalid_warning_window"
	GateReasonTokenInvalidLease        = "invalid_lease"
)

func gateReasonNames() [GateReasonInvalidLease + 1]string {
	return [...]string{
		GateReasonDisarmed:            GateReasonTokenDisarmed,
		GateReasonInvalidClock:        GateReasonTokenInvalidClock,
		GateReasonClockRollback:       GateReasonTokenClockRollback,
		GateReasonInvalidTrust:        GateReasonTokenInvalidTrust,
		GateReasonTeaching:            GateReasonTokenTeaching,
		GateReasonMissingTrustedLease: GateReasonTokenMissingTrustedLease,
		GateReasonLeaseValid:          GateReasonTokenLeaseValid,
		GateReasonLeaseWarning:        GateReasonTokenLeaseWarning,
		GateReasonLeaseGrace:          GateReasonTokenLeaseGrace,
		GateReasonLeaseExpired:        GateReasonTokenLeaseExpired,
		GateReasonInvalidWindow:       GateReasonTokenInvalidWindow,
		GateReasonInvalidLease:        GateReasonTokenInvalidLease,
	}
}

func (r GateReason) String() string {
	if r.IsValid() {
		return gateReasonNames()[r]
	}
	return ""
}

func (r GateReason) IsValid() bool {
	return r > gateReasonInvalid && int(r) < len(gateReasonNames()) && gateReasonNames()[r] != ""
}

func (r GateReason) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtGateReason, core.ErrLicenseContract)
	}
	return nil
}

func (r GateReason) Outcome() GateOutcome {
	switch r {
	case GateReasonDisarmed, GateReasonLeaseValid:
		return GateAllow
	case GateReasonLeaseWarning, GateReasonLeaseGrace:
		return GateWarn
	case GateReasonTeaching:
		return GateTeach
	case GateReasonInvalidClock, GateReasonClockRollback, GateReasonInvalidTrust, GateReasonMissingTrustedLease, GateReasonLeaseExpired, GateReasonInvalidWindow, GateReasonInvalidLease:
		return GateRefuse
	default:
		return gateOutcomeInvalid
	}
}

func (r GateReason) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func ParseGateReason(token string) (GateReason, error) {
	for reason := GateReasonDisarmed; int(reason) < len(gateReasonNames()); reason++ {
		if gateReasonNames()[reason] == token {
			return reason, nil
		}
	}
	return gateReasonInvalid, fmt.Errorf(ErrFmtGateReason, core.ErrLicenseContract)
}

func (r *GateReason) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateReason, core.ErrLicenseContract)
	}
	parsed, err := ParseGateReason(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
