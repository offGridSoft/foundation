package license

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	BugEnterpriseMinPrepaidYears    = 1
	BugEnterpriseMaxPrepaidYears    = 5
	BillingPeriodMonthlyMaxDuration = 31 * 24 * time.Hour
	PaymentCollectionGraceDuration  = 72 * time.Hour
	CheckInCollectionWindowDuration = 24 * time.Hour
	ConnectedCheckInAfterDuration   = BillingPeriodMonthlyMaxDuration + PaymentCollectionGraceDuration
	ConnectedCheckInByDuration      = ConnectedCheckInAfterDuration + CheckInCollectionWindowDuration
	PrepaidYearDuration             = 365 * 24 * time.Hour
	BugOfflineCheckInAfterDuration  = 365 * 24 * time.Hour
	BugOfflineCheckInByDuration     = 5 * 365 * 24 * time.Hour
	DefaultWriteGraceDuration       = 72 * time.Hour
)

type LeaseWindow struct {
	IssuedAt           core.UnixNanoTime
	PaidUntil          core.UnixNanoTime
	TokenExpiresAt     core.UnixNanoTime
	CheckInAfterAt     core.UnixNanoTime
	CheckInByAt        core.UnixNanoTime
	WriteGraceDuration core.NanosecondsDuration
}

func BuildLeaseWindow(issued core.UnixNanoTime, offer Offer, prepaidYears uint8) (LeaseWindow, error) {
	if err := core.ValidateRequiredUnixNanoTime(issued); err != nil {
		return LeaseWindow{}, fmt.Errorf(ErrFmtLeaseWindow, core.ErrLicenseContract)
	}
	if err := offer.Validate(); err != nil {
		return LeaseWindow{}, err
	}
	if err := validatePrepaidYearsForOffer(offer, prepaidYears); err != nil {
		return LeaseWindow{}, err
	}
	return buildLeaseWindow(issued, offer, prepaidYears), nil
}

func buildLeaseWindow(issued core.UnixNanoTime, offer Offer, prepaidYears uint8) LeaseWindow {
	if offer.BillingPeriod == BillingPeriodPrepaidYears {
		return buildPrepaidLeaseWindow(issued, offer, prepaidYears)
	}
	return buildConnectedLeaseWindow(issued, offer)
}

func buildConnectedLeaseWindow(issued core.UnixNanoTime, offer Offer) LeaseWindow {
	paidUntil := monthlyPaidUntil(issued)
	checkInAfter := paidUntil.Add(PaymentCollectionGraceDuration)
	checkInBy := checkInAfter.Add(CheckInCollectionWindowDuration)
	return LeaseWindow{
		IssuedAt:           issued,
		PaidUntil:          paidUntil,
		CheckInAfterAt:     checkInAfter,
		CheckInByAt:        checkInBy,
		TokenExpiresAt:     checkInBy,
		WriteGraceDuration: offer.WriteGrace,
	}
}

func monthlyPaidUntil(issued core.UnixNanoTime) core.UnixNanoTime {
	return core.NewUnixNanoTime(addCalendarMonthClamped(issued.Time()))
}

func addCalendarMonthClamped(value time.Time) time.Time {
	year, month, day := value.Date()
	hour, minute, second := value.Clock()
	targetMonth := time.Date(year, month+1, 1, hour, minute, second, value.Nanosecond(), value.Location())
	lastTargetDay := targetMonth.AddDate(0, 1, -1).Day()
	if day > lastTargetDay {
		day = lastTargetDay
	}
	return time.Date(targetMonth.Year(), targetMonth.Month(), day, hour, minute, second, value.Nanosecond(), value.Location())
}

func buildPrepaidLeaseWindow(issued core.UnixNanoTime, offer Offer, prepaidYears uint8) LeaseWindow {
	term := time.Duration(prepaidYears) * PrepaidYearDuration
	return LeaseWindow{
		IssuedAt:           issued,
		PaidUntil:          issued.Add(term),
		CheckInAfterAt:     issued.Add(offer.CheckInAfter.Duration()),
		CheckInByAt:        issued.Add(offer.CheckInBy.Duration()),
		TokenExpiresAt:     issued.Add(offer.LeaseDuration.Duration()),
		WriteGraceDuration: offer.WriteGrace,
	}
}

func validatePrepaidYearsForOffer(offer Offer, prepaidYears uint8) error {
	if offer.BillingPeriod == BillingPeriodPrepaidYears {
		if prepaidYears < offer.MinPrepaidYears || prepaidYears > offer.MaxPrepaidYears {
			return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
		}
		return nil
	}
	if prepaidYears != 0 {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	return nil
}

type OfferCode uint8

const (
	offerCodeInvalid OfferCode = iota
	OfferBugStandard
	OfferBugEnterprise
	OfferBugEnterpriseOffline
	OfferBugOSS
	OfferWitnessBronze
	OfferWitnessSilver
	OfferWitnessGold
)

const (
	OfferCodeTokenBugStandard          = "bug_standard"
	OfferCodeTokenBugEnterprise        = "bug_enterprise"
	OfferCodeTokenBugEnterpriseOffline = "bug_enterprise_offline"
	OfferCodeTokenBugOSS               = "bug_oss"
	OfferCodeTokenWitnessBronze        = "witness_bronze"
	OfferCodeTokenWitnessSilver        = "witness_silver"
	OfferCodeTokenWitnessGold          = "witness_gold"
)

func offerCodeNames() [OfferWitnessGold + 1]string {
	return [...]string{
		OfferBugStandard:          OfferCodeTokenBugStandard,
		OfferBugEnterprise:        OfferCodeTokenBugEnterprise,
		OfferBugEnterpriseOffline: OfferCodeTokenBugEnterpriseOffline,
		OfferBugOSS:               OfferCodeTokenBugOSS,
		OfferWitnessBronze:        OfferCodeTokenWitnessBronze,
		OfferWitnessSilver:        OfferCodeTokenWitnessSilver,
		OfferWitnessGold:          OfferCodeTokenWitnessGold,
	}
}

func (c OfferCode) String() string {
	if c.IsValid() {
		return offerCodeNames()[c]
	}
	return ""
}

func (c OfferCode) IsValid() bool {
	return c > offerCodeInvalid && int(c) < len(offerCodeNames()) && offerCodeNames()[c] != ""
}

func (c OfferCode) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	return nil
}

func ParseOfferCode(token string) (OfferCode, error) {
	for code := OfferBugStandard; int(code) < len(offerCodeNames()); code++ {
		if offerCodeNames()[code] == token {
			return code, nil
		}
	}
	return offerCodeInvalid, fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
}

func (c OfferCode) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func (c *OfferCode) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	parsed, err := ParseOfferCode(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

type Offer struct {
	PricePennies        core.MoneyPennies        `json:"price_pennies"`
	LeaseDuration       core.NanosecondsDuration `json:"lease_duration_ns"`
	CheckInAfter        core.NanosecondsDuration `json:"check_in_after_ns"`
	CheckInBy           core.NanosecondsDuration `json:"check_in_by_ns"`
	WriteGrace          core.NanosecondsDuration `json:"write_grace_ns"`
	Code                OfferCode                `json:"code"`
	Product             core.Product             `json:"product"`
	BillingPeriod       BillingPeriod            `json:"billing_period"`
	MinPrepaidYears     uint8                    `json:"min_prepaid_years"`
	MaxPrepaidYears     uint8                    `json:"max_prepaid_years"`
	OfflineLeaseAllowed bool                     `json:"offline_lease_allowed"`
}

func (o Offer) Validate() error {
	if err := validateOfferIdentity(o); err != nil {
		return err
	}
	if err := validateOfferDurations(o); err != nil {
		return err
	}
	return validateOfferPrepaidYears(o)
}

func OfferForSeatPlan(plan SeatPlan, price core.MoneyPennies) (Offer, error) {
	switch plan {
	case SeatPlanStandard:
		return bugStandardOffer(price), nil
	case SeatPlanEnterprise:
		return bugEnterpriseOffer(price), nil
	case SeatPlanEnterpriseOffline:
		return bugEnterpriseOfflineOffer(price), nil
	case SeatPlanOSS:
		return bugOSSOffer(price), nil
	default:
		return Offer{}, fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
}

func OfferForSubscriptionPlan(plan SubscriptionPlan, price core.MoneyPennies) (Offer, error) {
	switch plan {
	case SubscriptionPlanBronze:
		return witnessOffer(OfferWitnessBronze, price), nil
	case SubscriptionPlanSilver:
		return witnessOffer(OfferWitnessSilver, price), nil
	case SubscriptionPlanGold:
		return witnessOffer(OfferWitnessGold, price), nil
	default:
		return Offer{}, fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
}

func bugStandardOffer(price core.MoneyPennies) Offer {
	return connectedOffer(OfferBugStandard, core.ProductBug, price)
}

func bugEnterpriseOffer(price core.MoneyPennies) Offer {
	return connectedOffer(OfferBugEnterprise, core.ProductBug, price)
}

func bugEnterpriseOfflineOffer(price core.MoneyPennies) Offer {
	offer := connectedOffer(OfferBugEnterpriseOffline, core.ProductBug, price)
	offer.BillingPeriod = BillingPeriodPrepaidYears
	offer.LeaseDuration = core.NewNanosecondsDuration(BugOfflineCheckInByDuration)
	offer.CheckInAfter = core.NewNanosecondsDuration(BugOfflineCheckInAfterDuration)
	offer.CheckInBy = core.NewNanosecondsDuration(BugOfflineCheckInByDuration)
	offer.MinPrepaidYears = BugEnterpriseMinPrepaidYears
	offer.MaxPrepaidYears = BugEnterpriseMaxPrepaidYears
	offer.OfflineLeaseAllowed = true
	return offer
}

func bugOSSOffer(price core.MoneyPennies) Offer {
	return connectedOffer(OfferBugOSS, core.ProductBug, price)
}

func witnessOffer(code OfferCode, price core.MoneyPennies) Offer {
	return connectedOffer(code, core.ProductWitness, price)
}

func connectedOffer(code OfferCode, product core.Product, price core.MoneyPennies) Offer {
	return Offer{
		Code:          code,
		Product:       product,
		PricePennies:  price,
		BillingPeriod: BillingPeriodMonthly,
		LeaseDuration: core.NewNanosecondsDuration(ConnectedCheckInByDuration),
		CheckInAfter:  core.NewNanosecondsDuration(ConnectedCheckInAfterDuration),
		CheckInBy:     core.NewNanosecondsDuration(ConnectedCheckInByDuration),
		WriteGrace:    core.NewNanosecondsDuration(DefaultWriteGraceDuration),
	}
}

func validateOfferIdentity(o Offer) error {
	if err := o.Code.Validate(); err != nil {
		return err
	}
	if err := o.Product.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	if err := o.BillingPeriod.Validate(); err != nil {
		return err
	}
	if !offerProductMatches(o.Code, o.Product) {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	if o.Code == OfferBugOSS {
		if !o.PricePennies.IsZero() {
			return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
		}
	} else if !o.PricePennies.IsPositive() {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	return nil
}

func offerProductMatches(code OfferCode, product core.Product) bool {
	switch code {
	case OfferBugStandard, OfferBugEnterprise, OfferBugEnterpriseOffline, OfferBugOSS:
		return product == core.ProductBug
	case OfferWitnessBronze, OfferWitnessSilver, OfferWitnessGold:
		return product == core.ProductWitness
	default:
		return false
	}
}

func validateOfferDurations(o Offer) error {
	for _, duration := range []core.NanosecondsDuration{o.LeaseDuration, o.CheckInAfter, o.CheckInBy, o.WriteGrace} {
		if err := duration.Validate(); err != nil {
			return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
		}
	}
	if o.LeaseDuration.IsZero() || o.CheckInBy.IsZero() {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	if o.CheckInAfter.Duration() > o.CheckInBy.Duration() || o.CheckInBy.Duration() > o.LeaseDuration.Duration() {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	return nil
}

func validateOfferPrepaidYears(o Offer) error {
	if o.BillingPeriod == BillingPeriodPrepaidYears {
		if o.MinPrepaidYears != BugEnterpriseMinPrepaidYears || o.MaxPrepaidYears != BugEnterpriseMaxPrepaidYears {
			return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
		}
		return nil
	}
	if o.MinPrepaidYears != 0 || o.MaxPrepaidYears != 0 || o.OfflineLeaseAllowed {
		return fmt.Errorf(ErrFmtOffer, core.ErrLicenseContract)
	}
	return nil
}
