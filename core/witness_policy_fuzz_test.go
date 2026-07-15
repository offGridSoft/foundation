package core

import "testing"

func FuzzWitnessRetentionDecisionBoundary(f *testing.F) {
	const day = int64(WitnessDeletionRiskNoticeAfter)
	f.Add(int64(1), int64(1), int64(0), int64(0), int64(0), true, false)
	f.Add(int64(1)+day, int64(1), int64(2), int64(0), int64(0), true, false)
	f.Fuzz(func(t *testing.T, now, missedAt, paymentAt, riskAt, retrievalAt int64, missed, legalHold bool) {
		input := WitnessRetentionDecisionInput{
			Now:             UnixNanoTimeFromInt64(now),
			MissedPaymentAt: optionalFuzzUnixNanoTime(missedAt),
			NoticeHistory: WitnessCustodyNoticeHistory{
				PaymentWarningAt:      optionalFuzzUnixNanoTime(paymentAt),
				DeletionRiskWarningAt: optionalFuzzUnixNanoTime(riskAt),
				RetrievalOnlyAt:       optionalFuzzUnixNanoTime(retrievalAt),
			},
			MissedPayment: missed,
			LegalHold:     legalHold,
		}
		action, err := DecideWitnessRetention(input, NewWitnessRetentionPolicy(WitnessSilverRetentionCap))
		if err != nil {
			return
		}
		if err := action.Validate(); err != nil {
			t.Fatalf("accepted retention action validation = %v, want nil", err)
		}
	})
}

func optionalFuzzUnixNanoTime(value int64) UnixNanoTime {
	if value == 0 {
		return UnixNanoTime{}
	}
	return UnixNanoTimeFromInt64(value)
}
