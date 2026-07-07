package license

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"
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

type GateInput[B Body] struct {
	Now            core.UnixNanoTime
	ClockHighWater core.UnixNanoTime
	Lease          B
	WarnWindow     time.Duration
	Armed          bool
	LeaseTrusted   bool
	TeachingShown  bool
}

type GateDecision struct {
	Remaining time.Duration
	Outcome   GateOutcome
	Reason    GateReason
	State     LeaseState
}

func Gate[B Body](in GateInput[B]) GateDecision {
	if !in.Armed {
		return GateDecision{Outcome: GateAllow, Reason: GateReasonDisarmed}
	}
	if in.ClockHighWater.After(in.Now) {
		return GateDecision{Outcome: GateRefuse, Reason: GateReasonClockRollback}
	}
	if !in.LeaseTrusted {
		return untrustedLeaseDecision(in.TeachingShown)
	}
	return trustedLeaseDecision(in.Now, in.Lease, in.WarnWindow)
}

func untrustedLeaseDecision(teachingShown bool) GateDecision {
	if !teachingShown {
		return GateDecision{Outcome: GateTeach, Reason: GateReasonTeaching}
	}
	return GateDecision{Outcome: GateRefuse, Reason: GateReasonMissingTrustedLease}
}

func trustedLeaseDecision[B Body](
	now core.UnixNanoTime,
	lease B,
	warnWindow time.Duration,
) GateDecision {
	state := Status(lease, now, warnWindow)
	switch state {
	case LeaseValid:
		return GateDecision{Outcome: GateAllow, Reason: GateReasonLeaseValid, State: state}
	case LeaseWarning:
		return GateDecision{
			Remaining: lease.ExpiresAt().Sub(now),
			Outcome:   GateWarn,
			Reason:    GateReasonLeaseWarning,
			State:     state,
		}
	case LeaseGrace:
		return GateDecision{
			Remaining: lease.WriteGraceUntil().Sub(now),
			Outcome:   GateWarn,
			Reason:    GateReasonLeaseGrace,
			State:     state,
		}
	default:
		return GateDecision{Outcome: GateRefuse, Reason: GateReasonLeaseExpired, State: state}
	}
}

type GateReason uint8

const (
	gateReasonInvalid GateReason = iota
	GateReasonDisarmed
	GateReasonClockRollback
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
