package license

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/core"
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
	if err := leaseTrustInvalid.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("LeaseTrust invalid error = %v, want ErrLicenseContract", err)
	}
}

func gateLease(now core.UnixNanoTime) SeatLeaseBody {
	return SeatLeaseBody{
		IssuedAt:           now.Add(-24 * time.Hour),
		TokenExpiresAt:     now.Add(30 * 24 * time.Hour),
		CheckInAfterAt:     now,
		CheckInByAt:        now.Add(29 * 24 * time.Hour),
		Schema:             SchemaBugSeatLease,
		DeveloperKeyID:     mustDeveloperKeyID(),
		DeviceFingerprint:  mustDeviceFingerprint(),
		WriteGraceDuration: core.NewNanosecondsDuration(72 * time.Hour),
		Plan:               SeatPlanStandard,
	}
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
