package license

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestWitnessPolicyForPlan(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		plan      SubscriptionPlan
		wantLimit core.MachineLimit
		wantCap   core.NanosecondsDuration
	}{
		{name: "bronze five machines ninety day cap", plan: SubscriptionPlanBronze, wantLimit: core.WitnessBronzeMachineLimit, wantCap: core.NewNanosecondsDuration(core.WitnessBronzeRetentionCap)},
		{name: "silver ten machines three year cap", plan: SubscriptionPlanSilver, wantLimit: core.WitnessSilverMachineLimit, wantCap: core.NewNanosecondsDuration(core.WitnessSilverRetentionCap)},
		{name: "gold ten machines ten year cap", plan: SubscriptionPlanGold, wantLimit: core.WitnessGoldMachineLimit, wantCap: core.NewNanosecondsDuration(core.WitnessGoldRetentionCap)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy, err := WitnessPolicyForPlan(tc.plan)
			if err != nil {
				t.Fatalf("WitnessPolicyForPlan() error = %v, want nil", err)
			}
			if err := policy.Validate(); err != nil {
				t.Fatalf("WitnessPlanPolicy.Validate() error = %v, want nil", err)
			}
			if policy.MachineLimit != tc.wantLimit || policy.Retention.RetentionCap != tc.wantCap {
				t.Fatalf("WitnessPolicyForPlan() = limit %d cap %s, want limit %d cap %s", policy.MachineLimit, policy.Retention.RetentionCap.Duration(), tc.wantLimit, tc.wantCap.Duration())
			}
		})
	}
}

func TestWitnessPlanPolicyRejectsDrift(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*WitnessPlanPolicy)
		name   string
	}{
		{name: "machine limit drift", mutate: func(p *WitnessPlanPolicy) { p.MachineLimit++ }},
		{name: "retention cap drift", mutate: func(p *WitnessPlanPolicy) {
			p.Retention.RetentionCap = core.NanosecondsDurationFromInt64(p.Retention.RetentionCap.Nanoseconds() + 1)
		}},
		{name: "unknown plan", mutate: func(p *WitnessPlanPolicy) { p.Plan = subscriptionPlanInvalid }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy, err := WitnessPolicyForPlan(SubscriptionPlanSilver)
			if err != nil {
				t.Fatal(err)
			}
			tc.mutate(&policy)
			if err := policy.Validate(); !errors.Is(err, core.ErrWitnessPolicyContract) && !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("WitnessPlanPolicy.Validate() error = %v, want typed policy contract", err)
			}
		})
	}
}
