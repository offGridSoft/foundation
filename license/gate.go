package license

import (
	"fmt"
	"time"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

type GateOutcome uint8

const (
	gateOutcomeInvalid GateOutcome = iota
	GateAllow
	GateWarn
	GateTeach
	GateRefuse
)

var gateOutcomeNames = [...]string{
	GateAllow:  "allow",
	GateWarn:   "warn",
	GateTeach:  "teach",
	GateRefuse: "refuse",
}

func (o GateOutcome) String() string {
	if o.IsValid() {
		return gateOutcomeNames[o]
	}
	return ""
}

func (o GateOutcome) IsValid() bool {
	return o > gateOutcomeInvalid && int(o) < len(gateOutcomeNames) && gateOutcomeNames[o] != ""
}

func (o GateOutcome) MarshalJSON() ([]byte, error) {
	if !o.IsValid() {
		return nil, fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
	}
	return json.Marshal(o.String())
}

func (o *GateOutcome) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
	}
	for outcome := GateAllow; int(outcome) < len(gateOutcomeNames); outcome++ {
		if gateOutcomeNames[outcome] == token {
			*o = outcome
			return nil
		}
	}
	return fmt.Errorf(ErrFmtGateOutcome, core.ErrLicenseContract)
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

var leaseTrustNames = [...]string{
	LeaseTrustUntrusted: "untrusted",
	LeaseTrustTrusted:   "trusted",
}

func (t LeaseTrust) String() string {
	if t.IsValid() {
		return leaseTrustNames[t]
	}
	return ""
}

func (t LeaseTrust) IsValid() bool {
	return t > leaseTrustInvalid && int(t) < len(leaseTrustNames) && leaseTrustNames[t] != ""
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

func (t *LeaseTrust) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtLeaseTrust, core.ErrLicenseContract)
	}
	for trust := LeaseTrustUntrusted; int(trust) < len(leaseTrustNames); trust++ {
		if leaseTrustNames[trust] == token {
			*t = trust
			return nil
		}
	}
	return fmt.Errorf(ErrFmtLeaseTrust, core.ErrLicenseContract)
}

type GateInput[B Body] struct {
	Lease          B
	Now            core.UnixNanoTime
	ClockHighWater core.UnixNanoTime
	WarnWindow     time.Duration
	Trust          LeaseTrust
	Armed          bool
	TeachingShown  bool
}

type GateDecision struct {
	Remaining time.Duration
	Reason    GateReason
	State     LeaseState
}

func Gate[B Body](in GateInput[B]) GateDecision {
	if !in.Armed {
		return GateDecision{Reason: GateReasonDisarmed}
	}
	if in.ClockHighWater.After(in.Now) {
		return GateDecision{Reason: GateReasonClockRollback}
	}
	if !in.Trust.IsValid() {
		return GateDecision{Reason: GateReasonInvalidTrust}
	}
	if !in.Trust.Trusted() {
		return untrustedLeaseDecision(in.TeachingShown)
	}
	return trustedLeaseDecision(in.Now, in.Lease, in.WarnWindow)
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
	state := Status(lease, now, warnWindow)
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
	GateReasonClockRollback
	GateReasonInvalidTrust
	GateReasonTeaching
	GateReasonMissingTrustedLease
	GateReasonLeaseValid
	GateReasonLeaseWarning
	GateReasonLeaseGrace
	GateReasonLeaseExpired
)

var gateReasonNames = [...]string{
	GateReasonDisarmed:            "disarmed",
	GateReasonClockRollback:       "clock_rollback",
	GateReasonInvalidTrust:        "invalid_trust_resolution",
	GateReasonTeaching:            "teaching",
	GateReasonMissingTrustedLease: "missing_trusted_lease",
	GateReasonLeaseValid:          "lease_valid",
	GateReasonLeaseWarning:        "lease_warning",
	GateReasonLeaseGrace:          "lease_grace",
	GateReasonLeaseExpired:        "lease_expired",
}

func (r GateReason) String() string {
	if r.IsValid() {
		return gateReasonNames[r]
	}
	return ""
}

func (r GateReason) IsValid() bool {
	return r > gateReasonInvalid && int(r) < len(gateReasonNames) && gateReasonNames[r] != ""
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
	case GateReasonClockRollback, GateReasonInvalidTrust, GateReasonMissingTrustedLease, GateReasonLeaseExpired:
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

func (r *GateReason) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtGateReason, core.ErrLicenseContract)
	}
	for reason := GateReasonDisarmed; int(reason) < len(gateReasonNames); reason++ {
		if gateReasonNames[reason] == token {
			*r = reason
			return nil
		}
	}
	return fmt.Errorf(ErrFmtGateReason, core.ErrLicenseContract)
}
