package license

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestGateHostileTable(t *testing.T) {
	t.Parallel()
	now := testTime(1782302400000000000)
	lease := gateLease(now)
	for _, tc := range []struct {
		name          string
		input         GateInput[SeatLeaseBody]
		wantRemaining time.Duration
		wantOutcome   GateOutcome
		wantReason    GateReason
		wantState     LeaseState
		wantInputErr  bool
	}{
		{
			name:         "zero value fails closed before lease trust",
			input:        GateInput[SeatLeaseBody]{},
			wantOutcome:  GateRefuse,
			wantReason:   GateReasonInvalidClock,
			wantInputErr: true,
		},
		{
			name: "clock rollback refuses before lease",
			input: GateInput[SeatLeaseBody]{
				Now:            now,
				ClockHighWater: now.Add(time.Nanosecond),
				Trust:          LeaseTrustTrusted,
				Lease:          lease,
			},
			wantOutcome:  GateRefuse,
			wantReason:   GateReasonClockRollback,
			wantInputErr: true,
		},
		{
			name: "invalid trust fails closed",
			input: GateInput[SeatLeaseBody]{
				Now: now,
			},
			wantOutcome:  GateRefuse,
			wantReason:   GateReasonInvalidTrust,
			wantInputErr: true,
		},
		{
			name: "untrusted first refusal teaches",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Trust: LeaseTrustUntrusted,
			},
			wantOutcome: GateTeach,
			wantReason:  GateReasonTeaching,
		},
		{
			name: "untrusted after teaching refuses",
			input: GateInput[SeatLeaseBody]{
				Now:           now,
				Trust:         LeaseTrustUntrusted,
				TeachingShown: true,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonMissingTrustedLease,
		},
		{
			name: "trusted valid allows",
			input: GateInput[SeatLeaseBody]{
				Now:        now,
				Lease:      lease,
				WarnWindow: 7 * 24 * time.Hour,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome: GateAllow,
			wantReason:  GateReasonLeaseValid,
			wantState:   LeaseValid,
		},
		{
			name: "trusted warning warns",
			input: GateInput[SeatLeaseBody]{
				Now:        lease.ExpiresAt().Add(-time.Hour),
				Lease:      lease,
				WarnWindow: 7 * 24 * time.Hour,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome:   GateWarn,
			wantReason:    GateReasonLeaseWarning,
			wantState:     LeaseWarning,
			wantRemaining: time.Hour,
		},
		{
			name: "trusted grace warns",
			input: GateInput[SeatLeaseBody]{
				Now:        lease.ExpiresAt().Add(time.Hour),
				Lease:      lease,
				WarnWindow: 7 * 24 * time.Hour,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome:   GateWarn,
			wantReason:    GateReasonLeaseGrace,
			wantState:     LeaseGrace,
			wantRemaining: 71 * time.Hour,
		},
		{
			name: "trusted expired refuses",
			input: GateInput[SeatLeaseBody]{
				Now:        lease.WriteGraceUntil(),
				Lease:      lease,
				WarnWindow: 7 * 24 * time.Hour,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonLeaseExpired,
			wantState:   LeaseExpired,
		},
		{
			name: "trusted negative warning window fails closed",
			input: GateInput[SeatLeaseBody]{
				Now:        lease.WriteGraceUntil().Add(time.Hour),
				Lease:      lease,
				WarnWindow: -30 * 24 * time.Hour,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome:  GateRefuse,
			wantReason:   GateReasonInvalidWindow,
			wantInputErr: true,
		},
		{
			name: "malformed trusted lease fails closed",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Trust: LeaseTrustTrusted,
			},
			wantOutcome:  GateRefuse,
			wantReason:   GateReasonInvalidLease,
			wantInputErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inputErr := tc.input.Validate()
			if tc.wantInputErr {
				var typed GateInputError
				if !errors.As(inputErr, &typed) || typed.Reason != tc.wantReason || !errors.Is(inputErr, core.ErrLicenseContract) {
					t.Fatalf("GateInput.Validate() error = %v, want GateInputError reason=%s", inputErr, tc.wantReason)
				}
			} else if inputErr != nil {
				t.Fatalf("GateInput.Validate() error = %v", inputErr)
			}
			got := Gate(tc.input)
			if err := got.Validate(); err != nil {
				t.Fatalf("GateDecision.Validate() error = %v", err)
			}
			if got.Reason.Outcome() != tc.wantOutcome || got.Reason != tc.wantReason || got.State != tc.wantState {
				t.Fatalf("Gate = %+v, want outcome=%s reason=%s state=%s", got, tc.wantOutcome, tc.wantReason, tc.wantState)
			}
			if got.Remaining != tc.wantRemaining {
				t.Fatalf("Remaining = %s, want %s", got.Remaining, tc.wantRemaining)
			}
		})
	}
}

func TestLeaseTrustContract(t *testing.T) {
	t.Parallel()
	if !LeaseTrustTrusted.IsValid() || !LeaseTrustTrusted.Trusted() {
		t.Fatalf("LeaseTrustTrusted should be valid trusted")
	}
	if LeaseTrustUntrusted.Trusted() {
		t.Fatalf("LeaseTrustUntrusted should not be trusted")
	}
	if LeaseTrustTrusted.String() != "trusted" || LeaseTrustUntrusted.String() != "untrusted" {
		t.Fatalf("unexpected LeaseTrust strings")
	}
	if err := LeaseTrustTrusted.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := leaseTrustInvalid.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("LeaseTrust invalid error = %v, want ErrLicenseContract", err)
	}
}

func TestLeaseStatusRejectsInvalidExecutionInputs(t *testing.T) {
	t.Parallel()
	now := testTime(1782302400000000000)
	for _, tc := range []struct {
		name       string
		body       SeatLeaseBody
		now        core.UnixNanoTime
		warnWindow time.Duration
	}{
		{name: "zero clock", body: gateLease(now), warnWindow: WarnWindow},
		{name: "negative warning window", body: gateLease(now), now: now, warnWindow: -time.Nanosecond},
		{name: "invalid lease", now: now, warnWindow: WarnWindow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := Status(tc.body, tc.now, tc.warnWindow)
			if state != leaseStateInvalid || state.Writable() {
				t.Fatalf("Status() = %v, want invalid non-writable state", state)
			}
		})
	}
}

func TestLeaseTrustJSONRoundTrip(t *testing.T) {
	t.Parallel()
	data, err := LeaseTrustTrusted.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip LeaseTrust
	if err := roundTrip.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if roundTrip != LeaseTrustTrusted {
		t.Fatalf("LeaseTrust roundTrip = %s, want %s", roundTrip, LeaseTrustTrusted)
	}
}

func gateLease(now core.UnixNanoTime) SeatLeaseBody {
	window, err := BuildLeaseWindow(now.Add(-10*24*time.Hour), mustGateSeatOffer(), 0)
	if err != nil {
		panic(err)
	}
	return SeatLeaseBody{
		LeaseID:            mustLeaseIDNoT(),
		Generation:         1,
		IssuedAt:           window.IssuedAt,
		PaidUntil:          window.PaidUntil,
		TokenExpiresAt:     window.TokenExpiresAt,
		CheckInAfterAt:     window.CheckInAfterAt,
		CheckInByAt:        window.CheckInByAt,
		Schema:             core.SchemaBugSeatLease,
		DeveloperKeyID:     mustDeveloperKeyID(),
		DeviceFingerprint:  mustDeviceFingerprint(),
		Writer:             mustTestBugWriterKey(),
		WriteGraceDuration: window.WriteGraceDuration,
		Plan:               SeatPlanStandard,
		BillingPeriod:      BillingPeriodMonthly,
	}
}

func mustGateSeatOffer() Offer {
	offer, err := OfferForSeatPlan(SeatPlanStandard)
	if err != nil {
		panic(err)
	}
	return offer
}

func mustDeveloperKeyID() DeveloperKeyID {
	id, err := ParseDeveloperKeyID("OGS-DEV-alpha")
	if err != nil {
		panic(err)
	}
	return id
}

func mustDeviceFingerprint() core.DeviceFingerprint {
	id, err := core.ParseDeviceFingerprint(core.DeviceFingerprintPrefixSHA256 + strings.Repeat("a", 64))
	if err != nil {
		panic(err)
	}
	return id
}
