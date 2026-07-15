package core

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestWitnessPolicyConstants(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		limit MachineLimit
		want  uint16
	}{
		{name: "bronze allows five machines", limit: WitnessBronzeMachineLimit, want: 5},
		{name: "silver allows ten machines", limit: WitnessSilverMachineLimit, want: 10},
		{name: "gold allows ten machines", limit: WitnessGoldMachineLimit, want: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.limit.Validate(); err != nil {
				t.Fatalf("MachineLimit.Validate() error = %v, want nil", err)
			}
			if got := tc.limit.Uint16(); got != tc.want {
				t.Fatalf("MachineLimit.Uint16() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMachineLimitStrictJSONBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw     string
		name    string
		want    MachineLimit
		wantErr bool
	}{
		{name: "valid", raw: `5`, want: 5},
		{name: "zero", raw: `0`, wantErr: true},
		{name: "negative", raw: `-1`, wantErr: true},
		{name: "plus prefix", raw: `+5`, wantErr: true},
		{name: "leading zero", raw: `05`, wantErr: true},
		{name: "fraction", raw: `5.0`, wantErr: true},
		{name: "string", raw: `"5"`, wantErr: true},
		{name: "overflow", raw: `65536`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got MachineLimit
			err := got.UnmarshalJSON([]byte(tc.raw))
			if tc.wantErr {
				if !errors.Is(err, ErrWitnessPolicyContract) {
					t.Fatalf("MachineLimit.UnmarshalJSON() error = %v, want ErrWitnessPolicyContract", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("MachineLimit.UnmarshalJSON() = (%d, %v), want (%d, nil)", got, err, tc.want)
			}
		})
	}
}

func TestIsLowerHexRejectsEmptyToken(t *testing.T) {
	t.Parallel()
	if IsLowerHex("") {
		t.Fatal("IsLowerHex(empty) = true")
	}
}

func TestWitnessRetentionPolicyHostileTable(t *testing.T) {
	t.Parallel()

	valid := NewWitnessRetentionPolicy(WitnessSilverRetentionCap)
	for _, tc := range []struct {
		mutate func(*WitnessRetentionPolicy)
		name   string
	}{
		{name: "missing initial retention", mutate: func(p *WitnessRetentionPolicy) { p.InitialRetention = NanosecondsDuration{} }},
		{name: "missing monthly extension", mutate: func(p *WitnessRetentionPolicy) { p.PaymentExtension = NanosecondsDuration{} }},
		{name: "missing retention cap", mutate: func(p *WitnessRetentionPolicy) { p.RetentionCap = NanosecondsDuration{} }},
		{name: "initial retention exceeds cap", mutate: func(p *WitnessRetentionPolicy) { p.RetentionCap = NewNanosecondsDuration(time.Hour) }},
		{name: "missing deletion risk delay", mutate: func(p *WitnessRetentionPolicy) { p.DeletionRiskNoticeAfter = NanosecondsDuration{} }},
		{name: "missing retrieval-only delay", mutate: func(p *WitnessRetentionPolicy) { p.RetrievalOnlyAfter = NanosecondsDuration{} }},
		{name: "missing deletion eligibility delay", mutate: func(p *WitnessRetentionPolicy) { p.DeletionEligibleAfter = NanosecondsDuration{} }},
		{name: "deletion risk threshold drift", mutate: func(p *WitnessRetentionPolicy) { p.DeletionRiskNoticeAfter = NewNanosecondsDuration(time.Hour) }},
		{name: "retrieval threshold drift", mutate: func(p *WitnessRetentionPolicy) { p.RetrievalOnlyAfter = NewNanosecondsDuration(time.Hour) }},
		{name: "deletion threshold drift", mutate: func(p *WitnessRetentionPolicy) { p.DeletionEligibleAfter = NewNanosecondsDuration(time.Hour) }},
		{name: "immediate payment notice disabled", mutate: func(p *WitnessRetentionPolicy) { p.PaymentNoticeImmediate = false }},
		{name: "notice outbox requirement disabled", mutate: func(p *WitnessRetentionPolicy) { p.NoticeOutboxRequired = false }},
		{name: "deletion event requirement disabled", mutate: func(p *WitnessRetentionPolicy) { p.DeletionEventRequired = false }},
		{name: "legal hold protection disabled", mutate: func(p *WitnessRetentionPolicy) { p.LegalHoldBlocksDeletion = false }},
		{name: "retention expiry requirement disabled", mutate: func(p *WitnessRetentionPolicy) { p.RetentionExpiryRequired = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy := valid
			tc.mutate(&policy)
			if err := policy.Validate(); !errors.Is(err, ErrWitnessPolicyContract) {
				t.Fatalf("WitnessRetentionPolicy.Validate() error = %v, want ErrWitnessPolicyContract", err)
			}
		})
	}
}

func TestWitnessRetentionWindowCapsExtensions(t *testing.T) {
	t.Parallel()

	acceptedAt := NewUnixNanoTime(time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC))
	policy := NewWitnessRetentionPolicy(WitnessSilverRetentionCap)
	window, err := NewWitnessRetentionWindow(acceptedAt, policy)
	if err != nil {
		t.Fatalf("NewWitnessRetentionWindow() error = %v, want nil", err)
	}
	if got := window.RetainUntil.Sub(acceptedAt); got != WitnessInitialRetentionDuration {
		t.Fatalf("initial retention = %s, want %s", got, WitnessInitialRetentionDuration)
	}
	if got := window.MaximumRetainUntil.Sub(acceptedAt); got != WitnessSilverRetentionCap {
		t.Fatalf("retention cap = %s, want %s", got, WitnessSilverRetentionCap)
	}

	for window.RetainUntil.Before(window.MaximumRetainUntil) {
		window, err = ExtendWitnessRetention(window, policy)
		if err != nil {
			t.Fatalf("ExtendWitnessRetention() error = %v, want nil", err)
		}
	}
	if !window.RetainUntil.Equal(window.MaximumRetainUntil) {
		t.Fatalf("extended retention must stop at compiler-owned cap")
	}
	got, err := ExtendWitnessRetention(window, policy)
	if err != nil {
		t.Fatalf("ExtendWitnessRetention(at cap) error = %v, want nil", err)
	}
	if got != window {
		t.Fatalf("ExtendWitnessRetention(at cap) changed capped window")
	}
}

func TestDecideWitnessRetentionHostileTable(t *testing.T) {
	t.Parallel()

	missed := NewUnixNanoTime(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	paymentNotice := missed.Add(time.Hour)
	riskNotice := missed.Add(WitnessDeletionRiskNoticeAfter)
	retrievalNotice := missed.Add(WitnessRetrievalOnlyNoticeAfter)
	policy := NewWitnessRetentionPolicy(WitnessSilverRetentionCap)
	for _, tc := range []struct {
		name    string
		input   WitnessRetentionDecisionInput
		want    RetentionAction
		wantErr bool
	}{
		{name: "paid active bundle retained", input: WitnessRetentionDecisionInput{Now: missed}, want: RetentionActionRetain},
		{name: "missed payment immediately requests warning", input: WitnessRetentionDecisionInput{Now: missed, MissedPaymentAt: missed, MissedPayment: true}, want: RetentionActionPaymentWarning},
		{name: "sent payment warning retains before sixty days", input: WitnessRetentionDecisionInput{Now: missed.Add(WitnessDeletionRiskNoticeAfter - time.Nanosecond), MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice}}, want: RetentionActionRetain},
		{name: "sixty days requests deletion risk warning", input: WitnessRetentionDecisionInput{Now: riskNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice}}, want: RetentionActionDeletionRiskWarning},
		{name: "missing first warning cannot skip to deletion risk", input: WitnessRetentionDecisionInput{Now: riskNotice, MissedPaymentAt: missed, MissedPayment: true}, want: RetentionActionPaymentWarning},
		{name: "ninety days requests retrieval only notice", input: WitnessRetentionDecisionInput{Now: retrievalNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice}}, want: RetentionActionRetrievalOnly},
		{name: "ninety days cannot skip deletion risk notice", input: WitnessRetentionDecisionInput{Now: retrievalNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice}}, want: RetentionActionDeletionRiskWarning},
		{name: "six months with missing retrieval notice requests it first", input: WitnessRetentionDecisionInput{Now: missed.Add(WitnessDeletionEligibleAfter), MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice}}, want: RetentionActionRetrievalOnly},
		{name: "six months cannot skip deletion risk notice", input: WitnessRetentionDecisionInput{Now: missed.Add(WitnessDeletionEligibleAfter), MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice}}, want: RetentionActionDeletionRiskWarning},
		{name: "six months and complete notice history permits deletion", input: WitnessRetentionDecisionInput{Now: missed.Add(WitnessDeletionEligibleAfter), MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice, RetrievalOnlyAt: retrievalNotice}}, want: RetentionActionDeleteEligible},
		{name: "legal hold blocks otherwise eligible deletion", input: WitnessRetentionDecisionInput{Now: missed.Add(WitnessDeletionEligibleAfter), MissedPaymentAt: missed, MissedPayment: true, LegalHold: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice, RetrievalOnlyAt: retrievalNotice}}, want: RetentionActionLegalHold},
		{name: "future missed payment refused", input: WitnessRetentionDecisionInput{Now: missed, MissedPaymentAt: missed.Add(time.Nanosecond), MissedPayment: true}, wantErr: true},
		{name: "out of order notice history refused", input: WitnessRetentionDecisionInput{Now: retrievalNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: retrievalNotice, DeletionRiskWarningAt: riskNotice}}, wantErr: true},
		{name: "early deletion risk notice refused", input: WitnessRetentionDecisionInput{Now: riskNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice.Add(-time.Nanosecond)}}, wantErr: true},
		{name: "early retrieval-only notice refused", input: WitnessRetentionDecisionInput{Now: retrievalNotice, MissedPaymentAt: missed, MissedPayment: true, NoticeHistory: WitnessCustodyNoticeHistory{PaymentWarningAt: paymentNotice, DeletionRiskWarningAt: riskNotice, RetrievalOnlyAt: retrievalNotice.Add(-time.Nanosecond)}}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecideWitnessRetention(tc.input, policy)
			if tc.wantErr {
				if !errors.Is(err, ErrWitnessPolicyContract) {
					t.Fatalf("DecideWitnessRetention() error = %v, want %v", err, ErrWitnessPolicyContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecideWitnessRetention() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("DecideWitnessRetention() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestRetentionActionWireContract(t *testing.T) {
	t.Parallel()

	for action := RetentionActionRetain; action <= RetentionActionLegalHold; action++ {
		data, err := json.Marshal(action)
		if err != nil {
			t.Fatalf("json.Marshal(%s) error = %v", action, err)
		}
		var got RetentionAction
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
		}
		if got != action {
			t.Fatalf("RetentionAction round trip = %s, want %s", got, action)
		}
	}
	for _, token := range []string{"", "delete", "future"} {
		if _, err := ParseRetentionAction(token); !errors.Is(err, ErrWitnessPolicyContract) {
			t.Fatalf("ParseRetentionAction(%q) error = %v, want ErrWitnessPolicyContract", token, err)
		}
	}
}
