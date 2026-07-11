package core

import (
	"errors"
	"testing"
)

func TestWitnessVerificationIsPermanentlyUngated(t *testing.T) {
	t.Parallel()

	contract := WitnessVerificationAccessContract()
	if err := contract.Validate(); err != nil {
		t.Fatalf("VerificationAccessContract.Validate() error = %v, want nil", err)
	}
	if contract.License != AccessRequirementNever || contract.Network != AccessRequirementNever {
		t.Fatalf("Witness verification access = %s/%s, want never/never", contract.License, contract.Network)
	}

	for _, tc := range []struct {
		mutate func(*VerificationAccessContract)
		name   string
	}{
		{name: "license standing introduced", mutate: func(c *VerificationAccessContract) { c.License = AccessRequirementActiveStanding }},
		{name: "network dependency introduced", mutate: func(c *VerificationAccessContract) { c.Network = AccessRequirementActiveStanding }},
		{name: "unknown license requirement", mutate: func(c *VerificationAccessContract) { c.License = accessRequirementInvalid }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := contract
			tc.mutate(&got)
			if err := got.Validate(); !errors.Is(err, ErrAccessContract) {
				t.Fatalf("VerificationAccessContract.Validate() error = %v, want ErrAccessContract", err)
			}
		})
	}
}
