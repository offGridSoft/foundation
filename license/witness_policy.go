package license

import (
	"fmt"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const ErrFmtWitnessPlanPolicy = "license.WitnessPlanPolicy: %w"

type WitnessPlanPolicy struct {
	Retention    core.WitnessRetentionPolicy `json:"retention"`
	MachineLimit core.MachineLimit           `json:"machine_limit"`
	Plan         SubscriptionPlan            `json:"plan"`
}

func (p WitnessPlanPolicy) Validate() error {
	if err := p.Plan.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessPlanPolicy, err)
	}
	if err := p.MachineLimit.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessPlanPolicy, err)
	}
	if err := p.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessPlanPolicy, err)
	}
	want, err := WitnessPolicyForPlan(p.Plan)
	if err != nil {
		return err
	}
	if p != want {
		return fmt.Errorf(ErrFmtWitnessPlanPolicy, core.ErrWitnessPolicyContract)
	}
	return nil
}

func WitnessPolicyForPlan(plan SubscriptionPlan) (WitnessPlanPolicy, error) {
	switch plan {
	case SubscriptionPlanBronze:
		return newWitnessPlanPolicy(plan, core.WitnessBronzeMachineLimit, core.WitnessBronzeRetentionCap), nil
	case SubscriptionPlanSilver:
		return newWitnessPlanPolicy(plan, core.WitnessSilverMachineLimit, core.WitnessSilverRetentionCap), nil
	case SubscriptionPlanGold:
		return newWitnessPlanPolicy(plan, core.WitnessGoldMachineLimit, core.WitnessGoldRetentionCap), nil
	default:
		return WitnessPlanPolicy{}, fmt.Errorf(ErrFmtWitnessPlanPolicy, core.ErrWitnessPolicyContract)
	}
}

func newWitnessPlanPolicy(plan SubscriptionPlan, machineLimit core.MachineLimit, retentionCap time.Duration) WitnessPlanPolicy {
	return WitnessPlanPolicy{
		Plan:         plan,
		MachineLimit: machineLimit,
		Retention:    core.NewWitnessRetentionPolicy(retentionCap),
	}
}
