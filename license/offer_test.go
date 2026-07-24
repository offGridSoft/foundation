package license

import (
	"errors"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/currency"
)

const testPaidOfferMinorUnits = int64(1)

func TestOfferCatalogHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		offer       Offer
		wantPrice   int64
		wantProduct core.Product
		wantPeriod  BillingPeriod
		wantCode    OfferCode
	}{
		{name: "bug standard is monthly", offer: mustSeatOfferForTest(t, SeatPlanStandard), wantProduct: core.ProductBug, wantPeriod: BillingPeriodMonthly, wantPrice: testPaidOfferMinorUnits, wantCode: OfferBugStandard},
		{name: "bug enterprise is monthly", offer: mustSeatOfferForTest(t, SeatPlanEnterprise), wantProduct: core.ProductBug, wantPeriod: BillingPeriodMonthly, wantPrice: testPaidOfferMinorUnits, wantCode: OfferBugEnterprise},
		{name: "bug enterprise offline is prepaid one to five years", offer: mustSeatOfferForTest(t, SeatPlanEnterpriseOffline), wantProduct: core.ProductBug, wantPeriod: BillingPeriodPrepaidYears, wantPrice: testPaidOfferMinorUnits, wantCode: OfferBugEnterpriseOffline},
		{name: "bug oss keeps a separate zero-price identity", offer: mustSeatOfferForTest(t, SeatPlanOSS), wantProduct: core.ProductBug, wantPeriod: BillingPeriodMonthly, wantPrice: 0, wantCode: OfferBugOSS},
		{name: "witness bronze is monthly", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanBronze), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodMonthly, wantPrice: testPaidOfferMinorUnits, wantCode: OfferWitnessBronze},
		{name: "witness silver is monthly", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanSilver), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodMonthly, wantPrice: testPaidOfferMinorUnits, wantCode: OfferWitnessSilver},
		{name: "witness gold is monthly", offer: mustSubscriptionOfferForTest(t, SubscriptionPlanGold), wantProduct: core.ProductWitness, wantPeriod: BillingPeriodMonthly, wantPrice: testPaidOfferMinorUnits, wantCode: OfferWitnessGold},
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
			gotPrice, priceErr := tc.offer.Price.MinorUnits()
			gotCurrency, currencyErr := tc.offer.Price.Code()
			if priceErr != nil || currencyErr != nil || gotPrice != tc.wantPrice || gotCurrency != currency.CodeCAD {
				t.Fatalf("Price = (%s,%d,%v,%v), want (CAD,%d,nil,nil)", gotCurrency, gotPrice, currencyErr, priceErr, tc.wantPrice)
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
		mutate func(*testing.T, *Offer)
		name   string
	}{
		{name: "unknown offer code rejected", mutate: func(_ *testing.T, o *Offer) { o.Code = offerCodeInvalid }},
		{name: "wrong product rejected", mutate: func(_ *testing.T, o *Offer) { o.Product = core.ProductWitness }},
		{name: "paid offer rejects zero price", mutate: func(_ *testing.T, o *Offer) { o.Price = testOfferAmount(0) }},
		{name: "oss offer rejects nonzero price", mutate: func(t *testing.T, o *Offer) {
			*o = mustSeatOfferForTest(t, SeatPlanOSS)
			o.Price = testOfferAmount(testPaidOfferMinorUnits)
		}},
		{name: "offer rejects unknown currency", mutate: func(_ *testing.T, o *Offer) { o.Price = currency.Amount{} }},
		{name: "paid offer rejects negative price", mutate: func(_ *testing.T, o *Offer) { o.Price = testOfferAmount(-1) }},
		{name: "invalid billing period rejected", mutate: func(_ *testing.T, o *Offer) { o.BillingPeriod = billingPeriodInvalid }},
		{name: "negative write grace rejected", mutate: func(_ *testing.T, o *Offer) { o.WriteGrace = core.NanosecondsDurationFromInt64(-1) }},
		{name: "check-in after check-in by rejected", mutate: func(_ *testing.T, o *Offer) { o.CheckInAfter = core.NewNanosecondsDuration(36 * 24 * time.Hour) }},
		{name: "check-in by after lease rejected", mutate: func(_ *testing.T, o *Offer) { o.CheckInBy = core.NewNanosecondsDuration(36 * 24 * time.Hour) }},
		{name: "monthly offer cannot carry prepaid years", mutate: func(_ *testing.T, o *Offer) { o.MaxPrepaidYears = BugEnterpriseMaxPrepaidYears }},
		{name: "prepaid offer requires one to five years", mutate: func(t *testing.T, o *Offer) {
			*o = mustSeatOfferForTest(t, SeatPlanEnterpriseOffline)
			o.MaxPrepaidYears = 6
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			offer := mustSeatOfferForTest(t, SeatPlanStandard)
			tc.mutate(t, &offer)
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
		{name: "monthly accepted", token: BillingPeriodTokenMonthly, want: BillingPeriodMonthly},
		{name: "prepaid years accepted", token: BillingPeriodTokenPrepaidYears, want: BillingPeriodPrepaidYears},
		{name: "invalid ordinal token rejected", token: billingPeriodInvalid.String()},
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

func TestBuildLeaseWindowHostileTable(t *testing.T) {
	t.Parallel()
	issued := core.UnixNanoTimeFromInt64(1782302400000000000)
	for _, tc := range []struct {
		name         string
		offer        Offer
		prepaidYears uint8
		wantPaid     core.UnixNanoTime
		wantAfter    core.UnixNanoTime
		wantBy       core.UnixNanoTime
		wantExpires  core.UnixNanoTime
		wantGrace    core.NanosecondsDuration
	}{
		{
			name:        "bug standard waits one calendar month plus collection grace",
			offer:       mustSeatOfferForTest(t, SeatPlanStandard),
			wantPaid:    monthlyPaidUntil(issued),
			wantAfter:   monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration),
			wantBy:      monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration + CheckInCollectionWindowDuration),
			wantExpires: monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration + CheckInCollectionWindowDuration),
			wantGrace:   core.NewNanosecondsDuration(DefaultWriteGraceDuration),
		},
		{
			name:        "witness bronze waits one calendar month plus collection grace",
			offer:       mustSubscriptionOfferForTest(t, SubscriptionPlanBronze),
			wantPaid:    monthlyPaidUntil(issued),
			wantAfter:   monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration),
			wantBy:      monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration + CheckInCollectionWindowDuration),
			wantExpires: monthlyPaidUntil(issued).Add(PaymentCollectionGraceDuration + CheckInCollectionWindowDuration),
			wantGrace:   core.NewNanosecondsDuration(DefaultWriteGraceDuration),
		},
		{
			name:         "bug enterprise one year prepaid checks in after first offline year",
			offer:        mustSeatOfferForTest(t, SeatPlanEnterpriseOffline),
			prepaidYears: BugEnterpriseMinPrepaidYears,
			wantPaid:     issued.Add(time.Duration(BugEnterpriseMinPrepaidYears) * PrepaidYearDuration),
			wantAfter:    issued.Add(BugOfflineCheckInAfterDuration),
			wantBy:       issued.Add(BugOfflineCheckInByDuration),
			wantExpires:  issued.Add(BugOfflineCheckInByDuration),
			wantGrace:    core.NewNanosecondsDuration(DefaultWriteGraceDuration),
		},
		{
			name:         "bug enterprise five year prepaid still checks in after first offline year",
			offer:        mustSeatOfferForTest(t, SeatPlanEnterpriseOffline),
			prepaidYears: BugEnterpriseMaxPrepaidYears,
			wantPaid:     issued.Add(time.Duration(BugEnterpriseMaxPrepaidYears) * PrepaidYearDuration),
			wantAfter:    issued.Add(BugOfflineCheckInAfterDuration),
			wantBy:       issued.Add(BugOfflineCheckInByDuration),
			wantExpires:  issued.Add(BugOfflineCheckInByDuration),
			wantGrace:    core.NewNanosecondsDuration(DefaultWriteGraceDuration),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			window, err := BuildLeaseWindow(issued, tc.offer, tc.prepaidYears)
			if err != nil {
				t.Fatalf("BuildLeaseWindow() error = %v", err)
			}
			if !window.PaidUntil.Equal(tc.wantPaid) || !window.CheckInAfterAt.Equal(tc.wantAfter) ||
				!window.CheckInByAt.Equal(tc.wantBy) || !window.TokenExpiresAt.Equal(tc.wantExpires) ||
				window.WriteGraceDuration != tc.wantGrace {
				t.Fatalf("BuildLeaseWindow() = paid %d after %d by %d expires %d grace %s, want paid %d after %d by %d expires %d grace %s",
					window.PaidUntil.UnixNano(), window.CheckInAfterAt.UnixNano(), window.CheckInByAt.UnixNano(), window.TokenExpiresAt.UnixNano(), window.WriteGraceDuration.Duration(),
					tc.wantPaid.UnixNano(), tc.wantAfter.UnixNano(), tc.wantBy.UnixNano(), tc.wantExpires.UnixNano(), tc.wantGrace.Duration())
			}
		})
	}
}

func TestMonthlyPaidUntilClampsCalendarMonthEndsHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		issued time.Time
		want   time.Time
		name   string
	}{
		{name: "non leap January 31 clamps to February 28", issued: time.Date(2025, time.January, 31, 12, 30, 45, 123, time.UTC), want: time.Date(2025, time.February, 28, 12, 30, 45, 123, time.UTC)},
		{name: "leap January 31 clamps to February 29", issued: time.Date(2024, time.January, 31, 12, 30, 45, 123, time.UTC), want: time.Date(2024, time.February, 29, 12, 30, 45, 123, time.UTC)},
		{name: "March 31 clamps to April 30", issued: time.Date(2025, time.March, 31, 12, 30, 45, 123, time.UTC), want: time.Date(2025, time.April, 30, 12, 30, 45, 123, time.UTC)},
		{name: "April 30 remains May 30", issued: time.Date(2025, time.April, 30, 12, 30, 45, 123, time.UTC), want: time.Date(2025, time.May, 30, 12, 30, 45, 123, time.UTC)},
		{name: "leap day remains March 29", issued: time.Date(2024, time.February, 29, 12, 30, 45, 123, time.UTC), want: time.Date(2024, time.March, 29, 12, 30, 45, 123, time.UTC)},
		{name: "December 31 rolls into next year", issued: time.Date(2025, time.December, 31, 12, 30, 45, 123, time.UTC), want: time.Date(2026, time.January, 31, 12, 30, 45, 123, time.UTC)},
		{name: "ordinary day preserves day and clock", issued: time.Date(2025, time.June, 24, 12, 30, 45, 123, time.UTC), want: time.Date(2025, time.July, 24, 12, 30, 45, 123, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := monthlyPaidUntil(core.NewUnixNanoTime(tc.issued))
			if !got.Equal(core.NewUnixNanoTime(tc.want)) {
				t.Fatalf("monthlyPaidUntil(%s) = %s, want %s", tc.issued, got.Time(), tc.want)
			}
		})
	}
}

func TestPrepaidLeaseCheckInDueBeforeExpiryHostileTable(t *testing.T) {
	t.Parallel()
	issued := core.UnixNanoTimeFromInt64(1782302400000000000)
	offer := mustSeatOfferForTest(t, SeatPlanEnterpriseOffline)
	for _, tc := range []struct {
		name         string
		probe        leaseWindowProbe
		prepaidYears uint8
		wantDue      bool
	}{
		{name: "one year prepaid not due one nanosecond before offline check-in", probe: leaseWindowProbeBeforeCheckIn, prepaidYears: BugEnterpriseMinPrepaidYears},
		{name: "one year prepaid due at offline check-in boundary", probe: leaseWindowProbeAtCheckIn, prepaidYears: BugEnterpriseMinPrepaidYears, wantDue: true},
		{name: "five year prepaid due at first offline year", probe: leaseWindowProbeAtCheckIn, prepaidYears: BugEnterpriseMaxPrepaidYears, wantDue: true},
		{name: "five year prepaid due one nanosecond before token expiry", probe: leaseWindowProbeBeforeExpiry, prepaidYears: BugEnterpriseMaxPrepaidYears, wantDue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			window, err := BuildLeaseWindow(issued, offer, tc.prepaidYears)
			if err != nil {
				t.Fatalf("BuildLeaseWindow() error = %v", err)
			}
			body := SeatLeaseBody{
				LeaseID:            testLeaseID(t),
				Generation:         1,
				Schema:             core.SchemaBugSeatLease,
				DeveloperKeyID:     testDeveloperKeyID(t),
				DeviceFingerprint:  testDeviceFingerprint(t),
				Writer:             testBugWriterKey(t),
				IssuedAt:           window.IssuedAt,
				PaidUntil:          window.PaidUntil,
				TokenExpiresAt:     window.TokenExpiresAt,
				CheckInAfterAt:     window.CheckInAfterAt,
				CheckInByAt:        window.CheckInByAt,
				WriteGraceDuration: window.WriteGraceDuration,
				Plan:               SeatPlanEnterpriseOffline,
				BillingPeriod:      BillingPeriodPrepaidYears,
				PrepaidYears:       tc.prepaidYears,
			}
			if err := body.Validate(); err != nil {
				t.Fatalf("SeatLeaseBody.Validate() error = %v", err)
			}
			if got := CheckInDue(body, leaseWindowProbeTime(tc.probe, window)); got != tc.wantDue {
				t.Fatalf("CheckInDue(prepaid, %s) = %v, want %v", tc.name, got, tc.wantDue)
			}
		})
	}
}

type leaseWindowProbe uint8

const (
	leaseWindowProbeBeforeCheckIn leaseWindowProbe = iota + 1
	leaseWindowProbeAtCheckIn
	leaseWindowProbeBeforeExpiry
)

func leaseWindowProbeTime(probe leaseWindowProbe, window LeaseWindow) core.UnixNanoTime {
	switch probe {
	case leaseWindowProbeBeforeCheckIn:
		return window.CheckInAfterAt.Add(-time.Nanosecond)
	case leaseWindowProbeAtCheckIn:
		return window.CheckInAfterAt
	case leaseWindowProbeBeforeExpiry:
		return window.TokenExpiresAt.Add(-time.Nanosecond)
	default:
		return core.UnixNanoTime{}
	}
}

func TestBuildLeaseWindowRejectsBadTermsHostileTable(t *testing.T) {
	t.Parallel()
	issued := core.UnixNanoTimeFromInt64(1782302400000000000)
	for _, tc := range []struct {
		name         string
		offer        Offer
		prepaidYears uint8
		zeroIssued   bool
	}{
		{name: "zero issued rejected", offer: mustSeatOfferForTest(t, SeatPlanStandard), zeroIssued: true},
		{name: "connected offer rejects prepaid years", offer: mustSeatOfferForTest(t, SeatPlanStandard), prepaidYears: 1},
		{name: "prepaid offer rejects zero years", offer: mustSeatOfferForTest(t, SeatPlanEnterpriseOffline)},
		{name: "prepaid offer rejects six years", offer: mustSeatOfferForTest(t, SeatPlanEnterpriseOffline), prepaidYears: 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			start := issued
			if tc.zeroIssued {
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
	price := testOfferAmount(testPaidOfferMinorUnits)
	if plan == SeatPlanOSS {
		price = testOfferAmount(0)
	}
	offer, err := OfferForSeatPlan(plan, price)
	return mustOffer(t, offer, err)
}

func mustSubscriptionOfferForTest(t *testing.T, plan SubscriptionPlan) Offer {
	t.Helper()
	offer, err := OfferForSubscriptionPlan(plan, testOfferAmount(testPaidOfferMinorUnits))
	return mustOffer(t, offer, err)
}

func testOfferAmount(minorUnits int64) currency.Amount {
	amount, err := currency.New(currency.CodeCAD, minorUnits)
	if err != nil {
		panic(err)
	}
	return amount
}

func mustOffer(t *testing.T, offer Offer, err error) Offer {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return offer
}
