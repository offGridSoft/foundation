package core

import (
	"encoding/json"
	"fmt"
)

const (
	AccessRequirementTokenNever          = "never"
	AccessRequirementTokenActiveStanding = "active_standing"
	ErrFmtAccessRequirement              = "core.AccessRequirement: %w"
)

type AccessRequirement uint8

const (
	accessRequirementInvalid AccessRequirement = iota
	AccessRequirementNever
	AccessRequirementActiveStanding
)

func accessRequirementNames() [AccessRequirementActiveStanding + 1]string {
	return [...]string{
		AccessRequirementNever:          AccessRequirementTokenNever,
		AccessRequirementActiveStanding: AccessRequirementTokenActiveStanding,
	}
}

func (r AccessRequirement) String() string {
	if r.IsValid() {
		return accessRequirementNames()[r]
	}
	return ""
}

func (r AccessRequirement) IsValid() bool {
	return r > accessRequirementInvalid && int(r) < len(accessRequirementNames()) && accessRequirementNames()[r] != ""
}

func (r AccessRequirement) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtAccessRequirement, ErrAccessContract)
	}
	return nil
}

func ParseAccessRequirement(token string) (AccessRequirement, error) {
	for requirement := AccessRequirementNever; int(requirement) < len(accessRequirementNames()); requirement++ {
		if accessRequirementNames()[requirement] == token {
			return requirement, nil
		}
	}
	return accessRequirementInvalid, fmt.Errorf(ErrFmtAccessRequirement, ErrAccessContract)
}

func (r AccessRequirement) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func (r *AccessRequirement) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtAccessRequirement, ErrAccessContract)
	}
	parsed, err := ParseAccessRequirement(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

type VerificationAccessContract struct {
	License AccessRequirement `json:"license"`
	Network AccessRequirement `json:"network"`
}

func WitnessVerificationAccessContract() VerificationAccessContract {
	return VerificationAccessContract{
		License: AccessRequirementNever,
		Network: AccessRequirementNever,
	}
}

func (c VerificationAccessContract) Validate() error {
	if err := c.License.Validate(); err != nil {
		return err
	}
	if err := c.Network.Validate(); err != nil {
		return err
	}
	if c != WitnessVerificationAccessContract() {
		return fmt.Errorf(ErrFmtAccessRequirement, ErrAccessContract)
	}
	return nil
}
