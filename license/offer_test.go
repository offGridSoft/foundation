package license

import (
	"errors"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestOfferCatalogHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		offer       Offer
		wantPrice   uint64
		wantProduct core.Product
		wantPeriod  BillingPeriod
		wantCode    OfferCode
	}{
		{name: "bug standard is fifty dollars every four weeks", offer: mustSeatOfferForTest(t, SeatPlanStandard), wantProduct: core.ProductBug, wantPeriod: BillingPeriodFourWeeks, wantPrice: BugStandardPricePennies, wantCode: OfferBugStandard},
		{name: "bug enterprise is prepaid one to five years", offer: mustSeatOfferForTest(t, SeatPlanEnterprise), wantProduct: core.ProductBug, wantPeriod: BillingPeriodPrepaidYears, wantPrice: BugEnterpriseMonthlyPennies, wantCode: OfferBugEnterprise},
		{name: "bug oss keeps a separate zero-price identity", offer: mustSeatOfferForTest(t, SeatPlanOSS), wantProduct: core.ProductBug, wantPeriod: BillingPeriodFourWeeks, wantPrice: 0, wantCode: OfferBugOSS},
		{name: "witness bronze is every four weeks", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanBronze), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodFourWeeks, wantPrice: WitnessBronzePricePennies, wantCode: OfferWitnessBronze},
		{name: "witness silver is every four weeks", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanSilver), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodFourWeeks, wantPrice: WitnessSilverPricePennies, wantCode: OfferWitnessSilver},
		{name: "witness gold is every four weeks", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanGold), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodFourWeeks, wantPrice: WitnessGoldStartingPricePennies, wantCode: OfferWitnessGold},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.offer.Validate(); err != nil {
				t.Fatalf("Offer.Validate() = %v", err)
			}
			if tc.offer.Product != tc.wantProduct {
				t.Fatalf("Product = %s, want %s", tc.offer.Product, tc.wantProduct)
			}
			if tc.offer.BillingPeriod != tc.wantPeriod {
				t.Fatalf("BillingPeriod = %s, want %s", tc.offer.BillingPeriod, tc.wantPeriod)
			}
			if got := tc.offer.PricePennies.Uint64(); got != tc.wantPrice {
				t.Fatalf("PricePennies = %d, want %d", got, tc.wantPrice)
			}
			if tc.offer.Code != tc.wantCode {
				t.Fatalf("Code = %s, want %s", tc.offer.Code, tc.wantCode)
			}
		})
	}
}

func TestOfferRejectsPolicyDriftHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*Offer)
		name   string
	}{
		{name: "unknown offer code rejected", mutate: func(o *Offer) { o.Code = offerCodeInvalid }},
		{name: "wrong product rejected", mutate: func(o *Offer) { o.Product = core.ProductWitness }},
		{name: "monthly spelling has no parser home", mutate: func(o *Offer) { o.BillingPeriod = billingPeriodInvalid }},
		{name: "negative write grace rejected", mutate: func(o *Offer) { o.WriteGrace = core.NanosecondsDurationFromInt64(-1) }},
		{name: "check-in after check-in by rejected", mutate: func(o *Offer) { o.CheckInAfter = core.NewNanosecondsDuration(33 * 24 * time.Hour) }},
		{name: "check-in by after lease rejected", mutate: func(o *Offer) { o.CheckInBy = core.NewNanosecondsDuration(29 * 24 * time.Hour) }},
		{name: "four-week offer cannot carry prepaid years", mutate: func(o *Offer) { o.MaxPrepaidYears = BugEnterpriseMaxPrepaidYears }},
		{name: "prepaid offer requires one to five years", mutate: func(o *Offer) {
			*o = mustSeatOfferForTest(t, SeatPlanEnterprise)
			o.MaxPrepaidYears = 6
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			offer := mustSeatOfferForTest(t, SeatPlanStandard)
			tc.mutate(&offer)
			if err := offer.Validate(); !errors.Is(err, core.ErrLicenseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Offer.Validate() error = %v, want license/foundation contract", err)
			}
		})
	}
}

func TestBillingPeriodParserHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		token string
		name  string
		want  BillingPeriod
	}{
		{name: "four weeks accepted", token: BillingPeriodTokenFourWeeks, want: BillingPeriodFourWeeks},
		{name: "prepaid years accepted", token: BillingPeriodTokenPrepaidYears, want: BillingPeriodPrepaidYears},
		{name: "monthly rejected", token: "monthly"},
		{name: "blank rejected", token: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBillingPeriod(tc.token)
			if tc.want != billingPeriodInvalid {
				if err != nil {
					t.Fatalf("ParseBillingPeriod() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("ParseBillingPeriod() = %s, want %s", got, tc.want)
				}
				return
			}
			if !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("ParseBillingPeriod() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestSubscriptionPlanParserHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		token string
		name  string
		want  SubscriptionPlan
	}{
		{name: "bronze accepted", token: SubscriptionPlanTokenBronze, want: SubscriptionPlanBronze},
		{name: "silver accepted", token: SubscriptionPlanTokenSilver, want: SubscriptionPlanSilver},
		{name: "gold accepted", token: SubscriptionPlanTokenGold, want: SubscriptionPlanGold},
		{name: "seat token rejected", token: SeatPlanTokenEnterprise},
		{name: "blank rejected", token: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSubscriptionPlan(tc.token)
			if tc.want != subscriptionPlanInvalid {
				if err != nil {
					t.Fatalf("ParseSubscriptionPlan() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("ParseSubscriptionPlan() = %s, want %s", got, tc.want)
				}
				return
			}
			if !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("ParseSubscriptionPlan() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestOfferCodeJSONHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		name string
		want OfferCode
	}{
		{name: "bug standard accepted", raw: `"bug_standard"`, want: OfferBugStandard},
		{name: "bug oss accepted", raw: `"bug_oss"`, want: OfferBugOSS},
		{name: "witness bronze accepted", raw: `"witness_bronze"`, want: OfferWitnessBronze},
		{name: "number rejected", raw: `7`},
		{name: "blank rejected", raw: `""`},
		{name: "unknown rejected", raw: `"monthly"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got OfferCode
			err := got.UnmarshalJSON([]byte(tc.raw))
			if tc.want != offerCodeInvalid {
				if err != nil {
					t.Fatalf("OfferCode.UnmarshalJSON() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("OfferCode.UnmarshalJSON() = %s, want %s", got, tc.want)
				}
				return
			}
			if !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("OfferCode.UnmarshalJSON() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestBuildLeaseWindowWaitsForCollectionGraceHostileTable(t *testing.T) {
	t.Parallel()
	issued := core.UnixNanoTimeFromInt64(1782302400000000000)
	for _, tc := range []struct {
		name         string
		offer        Offer
		prepaidYears uint8
		wantPaid     core.UnixNanoTime
		wantAfter    core.UnixNanoTime
		wantBy       core.UnixNanoTime
	}{
		{
			name:      "bug standard waits four weeks plus collection grace",
			offer:     mustSeatOfferForTest(t, SeatPlanStandard),
			wantPaid:  issued.Add(BillingPeriodFourWeeksDuration),
			wantAfter: issued.Add(ConnectedCheckInAfterDuration),
			wantBy:    issued.Add(ConnectedCheckInByDuration),
		},
		{
			name:      "witness bronze waits four weeks plus collection grace",
			offer:     mustSubscriptionOfferForTest(t, SubscriptionPlanBronze),
			wantPaid:  issued.Add(BillingPeriodFourWeeksDuration),
			wantAfter: issued.Add(ConnectedCheckInAfterDuration),
			wantBy:    issued.Add(ConnectedCheckInByDuration),
		},
		{
			name:         "bug enterprise selected five year offline lease",
			offer:        mustSeatOfferForTest(t, SeatPlanEnterprise),
			prepaidYears: BugEnterpriseMaxPrepaidYears,
			wantPaid:     issued.Add(time.Duration(BugEnterpriseMaxPrepaidYears) * PrepaidYearDuration),
			wantAfter:    issued.Add(time.Duration(BugEnterpriseMaxPrepaidYears) * PrepaidYearDuration),
			wantBy:       issued.Add(time.Duration(BugEnterpriseMaxPrepaidYears) * PrepaidYearDuration),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			window, err := BuildLeaseWindow(issued, tc.offer, tc.prepaidYears)
			if err != nil {
				t.Fatalf("BuildLeaseWindow() error = %v", err)
			}
			if !window.PaidUntil.Equal(tc.wantPaid) || !window.CheckInAfterAt.Equal(tc.wantAfter) || !window.CheckInByAt.Equal(tc.wantBy) {
				t.Fatalf("BuildLeaseWindow() = paid %d after %d by %d, want paid %d after %d by %d",
					window.PaidUntil.UnixNano(), window.CheckInAfterAt.UnixNano(), window.CheckInByAt.UnixNano(),
					tc.wantPaid.UnixNano(), tc.wantAfter.UnixNano(), tc.wantBy.UnixNano())
			}
		})
	}
}

func TestBuildLeaseWindowRejectsBadTermsHostileTable(t *testing.T) {
	t.Parallel()
	issued := core.UnixNanoTimeFromInt64(1782302400000000000)
	for _, tc := range []struct {
		name         string
		offer        Offer
		prepaidYears uint8
	}{
		{name: "zero issued rejected", offer: mustSeatOfferForTest(t, SeatPlanStandard)},
		{name: "connected offer rejects prepaid years", offer: mustSeatOfferForTest(t, SeatPlanStandard), prepaidYears: 1},
		{name: "prepaid offer rejects zero years", offer: mustSeatOfferForTest(t, SeatPlanEnterprise)},
		{name: "prepaid offer rejects six years", offer: mustSeatOfferForTest(t, SeatPlanEnterprise), prepaidYears: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start := issued
			if tc.name == "zero issued rejected" {
				start = core.UnixNanoTime{}
			}
			if _, err := BuildLeaseWindow(start, tc.offer, tc.prepaidYears); !errors.Is(err, core.ErrLicenseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("BuildLeaseWindow() error = %v, want license/foundation contract", err)
			}
		})
	}
}

func mustSeatOfferForTest(t *testing.T, plan SeatPlan) Offer {
	t.Helper()
	offer, err := OfferForSeatPlan(plan)
	return mustOffer(t, offer, err)
}

func mustSubscriptionOfferForTest(t *testing.T, plan SubscriptionPlan) Offer {
	t.Helper()
	offer, err := OfferForSubscriptionPlan(plan)
	return mustOffer(t, offer, err)
}

func mustOffer(t *testing.T, offer Offer, err error) Offer {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return offer
}
