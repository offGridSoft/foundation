package license

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	SeatPlanTokenStandard          = "standard"
	SeatPlanTokenEnterprise        = "enterprise"
	SeatPlanTokenEnterpriseOffline = "enterprise_offline"
	SeatPlanTokenOSS               = "oss"
	SubscriptionPlanTokenBronze    = "bronze"
	SubscriptionPlanTokenSilver    = "silver"
	SubscriptionPlanTokenGold      = "gold"
	BillingPeriodTokenFourWeeks    = "four_weeks"
	BillingPeriodTokenPrepaidYears = "prepaid_years"
)

type SeatPlan uint8

const (
	seatPlanInvalid SeatPlan = iota
	SeatPlanStandard
	SeatPlanEnterprise
	SeatPlanEnterpriseOffline
	SeatPlanOSS
)

func seatPlanNames() [SeatPlanOSS + 1]string {
	return [...]string{
		SeatPlanStandard:          SeatPlanTokenStandard,
		SeatPlanEnterprise:        SeatPlanTokenEnterprise,
		SeatPlanEnterpriseOffline: SeatPlanTokenEnterpriseOffline,
		SeatPlanOSS:               SeatPlanTokenOSS,
	}
}

func (p SeatPlan) String() string {
	if p.IsValid() {
		return seatPlanNames()[p]
	}
	return ""
}

func (p SeatPlan) IsValid() bool {
	return p > seatPlanInvalid && int(p) < len(seatPlanNames()) && seatPlanNames()[p] != ""
}

func (p SeatPlan) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
	}
	return nil
}

func ParseSeatPlan(token string) (SeatPlan, error) {
	for plan := SeatPlanStandard; int(plan) < len(seatPlanNames()); plan++ {
		if seatPlanNames()[plan] == token {
			return plan, nil
		}
	}
	return seatPlanInvalid, fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
}

func (p SeatPlan) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *SeatPlan) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
	}
	parsed, err := ParseSeatPlan(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type SubscriptionPlan uint8

const (
	subscriptionPlanInvalid SubscriptionPlan = iota
	SubscriptionPlanBronze
	SubscriptionPlanSilver
	SubscriptionPlanGold
)

func subscriptionPlanNames() [SubscriptionPlanGold + 1]string {
	return [...]string{
		SubscriptionPlanBronze: SubscriptionPlanTokenBronze,
		SubscriptionPlanSilver: SubscriptionPlanTokenSilver,
		SubscriptionPlanGold:   SubscriptionPlanTokenGold,
	}
}

func (p SubscriptionPlan) String() string {
	if p.IsValid() {
		return subscriptionPlanNames()[p]
	}
	return ""
}

func (p SubscriptionPlan) IsValid() bool {
	return p > subscriptionPlanInvalid && int(p) < len(subscriptionPlanNames()) && subscriptionPlanNames()[p] != ""
}

func (p SubscriptionPlan) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
	}
	return nil
}

func ParseSubscriptionPlan(token string) (SubscriptionPlan, error) {
	for plan := SubscriptionPlanBronze; int(plan) < len(subscriptionPlanNames()); plan++ {
		if subscriptionPlanNames()[plan] == token {
			return plan, nil
		}
	}
	return subscriptionPlanInvalid, fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
}

func (p SubscriptionPlan) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *SubscriptionPlan) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtPlan, core.ErrLicenseContract)
	}
	parsed, err := ParseSubscriptionPlan(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type BillingPeriod uint8

const (
	billingPeriodInvalid BillingPeriod = iota
	BillingPeriodFourWeeks
	BillingPeriodPrepaidYears
)

func billingPeriodNames() [BillingPeriodPrepaidYears + 1]string {
	return [...]string{
		BillingPeriodFourWeeks:    BillingPeriodTokenFourWeeks,
		BillingPeriodPrepaidYears: BillingPeriodTokenPrepaidYears,
	}
}

func (p BillingPeriod) String() string {
	if p.IsValid() {
		return billingPeriodNames()[p]
	}
	return ""
}

func (p BillingPeriod) IsValid() bool {
	return p > billingPeriodInvalid && int(p) < len(billingPeriodNames()) && billingPeriodNames()[p] != ""
}

func (p BillingPeriod) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
	}
	return nil
}

func ParseBillingPeriod(token string) (BillingPeriod, error) {
	for period := BillingPeriodFourWeeks; int(period) < len(billingPeriodNames()); period++ {
		if billingPeriodNames()[period] == token {
			return period, nil
		}
	}
	return billingPeriodInvalid, fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
}

func (p BillingPeriod) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *BillingPeriod) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtBillingPeriod, core.ErrLicenseContract)
	}
	parsed, err := ParseBillingPeriod(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
