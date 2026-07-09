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
		wantOutcome   GateOutcome
		wantReason    GateReason
		wantState     LeaseState
		wantRemaining time.Duration
	}{
		{
			name: "disarmed allows without trusting lease",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Armed: false,
			},
			wantOutcome: GateAllow,
			wantReason:  GateReasonDisarmed,
		},
		{
			name: "clock rollback refuses before lease",
			input: GateInput[SeatLeaseBody]{
				Now:            now,
				ClockHighWater: now.Add(time.Nanosecond),
				Armed:          true,
				Trust:          LeaseTrustTrusted,
				Lease:          lease,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonClockRollback,
		},
		{
			name: "armed invalid trust fails closed",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Armed: true,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonInvalidTrust,
		},
		{
			name: "untrusted first refusal teaches",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Armed: true,
				Trust: LeaseTrustUntrusted,
			},
			wantOutcome: GateTeach,
			wantReason:  GateReasonTeaching,
		},
		{
			name: "untrusted after teaching refuses",
			input: GateInput[SeatLeaseBody]{
				Now:           now,
				Armed:         true,
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
				Armed:      true,
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
				Armed:      true,
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
				Armed:      true,
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
				Armed:      true,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonLeaseExpired,
			wantState:   LeaseExpired,
		},
		{
			name: "trusted negative warning window refuses expired lease",
			input: GateInput[SeatLeaseBody]{
				Now:        lease.WriteGraceUntil().Add(time.Hour),
				Lease:      lease,
				WarnWindow: -30 * 24 * time.Hour,
				Armed:      true,
				Trust:      LeaseTrustTrusted,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonLeaseExpired,
			wantState:   LeaseExpired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Gate(tc.input)
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
		IssuedAt:           window.IssuedAt,
		PaidUntil:          window.PaidUntil,
		TokenExpiresAt:     window.TokenExpiresAt,
		CheckInAfterAt:     window.CheckInAfterAt,
		CheckInByAt:        window.CheckInByAt,
		Schema:             core.SchemaBugSeatLease,
		DeveloperKeyID:     mustDeveloperKeyID(),
		DeviceFingerprint:  mustDeviceFingerprint(),
		WriteGraceDuration: window.WriteGraceDuration,
		Plan:               SeatPlanStandard,
		BillingPeriod:      BillingPeriodFourWeeks,
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
