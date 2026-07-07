package license

import (
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
				LeaseTrusted:   true,
				Lease:          lease,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonClockRollback,
		},
		{
			name: "untrusted first refusal teaches",
			input: GateInput[SeatLeaseBody]{
				Now:   now,
				Armed: true,
			},
			wantOutcome: GateTeach,
			wantReason:  GateReasonTeaching,
		},
		{
			name: "untrusted after teaching refuses",
			input: GateInput[SeatLeaseBody]{
				Now:           now,
				Armed:         true,
				TeachingShown: true,
			},
			wantOutcome: GateRefuse,
			wantReason:  GateReasonMissingTrustedLease,
		},
		{
			name: "trusted valid allows",
			input: GateInput[SeatLeaseBody]{
				Now:          now,
				Lease:        lease,
				WarnWindow:   7 * 24 * time.Hour,
				Armed:        true,
				LeaseTrusted: true,
			},
			wantOutcome: GateAllow,
			wantReason:  GateReasonLeaseValid,
			wantState:   LeaseValid,
		},
		{
			name: "trusted warning warns",
			input: GateInput[SeatLeaseBody]{
				Now:          lease.ExpiresAt().Add(-time.Hour),
				Lease:        lease,
				WarnWindow:   7 * 24 * time.Hour,
				Armed:        true,
				LeaseTrusted: true,
			},
			wantOutcome:   GateWarn,
			wantReason:    GateReasonLeaseWarning,
			wantState:     LeaseWarning,
			wantRemaining: time.Hour,
		},
		{
			name: "trusted grace warns",
			input: GateInput[SeatLeaseBody]{
				Now:          lease.ExpiresAt().Add(time.Hour),
				Lease:        lease,
				WarnWindow:   7 * 24 * time.Hour,
				Armed:        true,
				LeaseTrusted: true,
			},
			wantOutcome:   GateWarn,
			wantReason:    GateReasonLeaseGrace,
			wantState:     LeaseGrace,
			wantRemaining: 71 * time.Hour,
		},
		{
			name: "trusted expired refuses",
			input: GateInput[SeatLeaseBody]{
				Now:          lease.WriteGraceUntil(),
				Lease:        lease,
				WarnWindow:   7 * 24 * time.Hour,
				Armed:        true,
				LeaseTrusted: true,
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
