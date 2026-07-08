package license

import (
	"fmt"

	"encoding/json"
	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RefusalTokenNone             = "none"
	RefusalTokenKeyRevoked       = "key_revoked"
	RefusalTokenSeatLimit        = "seat_limit_exceeded" // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenPaymentRequired  = "payment_required"
	RefusalTokenUnsupportedBuild = "unsupported_build"     // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenDeviceLimit      = "device_limit_exceeded" // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenUnknownKey       = "unknown_key"
)

type Refusal uint8

const (
	refusalInvalid Refusal = iota
	RefusalNone
	RefusalKeyRevoked
	RefusalSeatLimit
	RefusalPaymentRequired
	RefusalUnsupportedBuild
	RefusalDeviceLimit
	RefusalUnknownKey
)

var refusalNames = [...]string{
	RefusalNone:             RefusalTokenNone,
	RefusalKeyRevoked:       RefusalTokenKeyRevoked,
	RefusalSeatLimit:        RefusalTokenSeatLimit,
	RefusalPaymentRequired:  RefusalTokenPaymentRequired,
	RefusalUnsupportedBuild: RefusalTokenUnsupportedBuild,
	RefusalDeviceLimit:      RefusalTokenDeviceLimit,
	RefusalUnknownKey:       RefusalTokenUnknownKey,
}

func (r Refusal) String() string {
	if r.IsValid() {
		return refusalNames[r]
	}
	return ""
}

func (r Refusal) IsValid() bool {
	return r > refusalInvalid && int(r) < len(refusalNames) && refusalNames[r] != ""
}

func (r Refusal) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	return nil
}

func ParseRefusal(token string) (Refusal, error) {
	for refusal := RefusalNone; int(refusal) < len(refusalNames); refusal++ {
		if refusalNames[refusal] == token {
			return refusal, nil
		}
	}
	return refusalInvalid, fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
}

func (r Refusal) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func (r *Refusal) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	parsed, err := ParseRefusal(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
