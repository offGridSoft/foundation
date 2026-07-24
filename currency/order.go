package currency

import (
	"encoding/json"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

// Order is the closed result of comparing two amounts.
type Order uint8

const (
	OrderUnknown Order = iota
	OrderLess
	OrderEqual
	OrderGreater
)

func orderTokens() [OrderGreater + 1]string {
	return [...]string{
		OrderLess:    "less",
		OrderEqual:   "equal",
		OrderGreater: "greater",
	}
}

// IsValid reports membership in the closed comparison domain.
func (o Order) IsValid() bool {
	return o > OrderUnknown && int(o) < len(orderTokens()) && orderTokens()[o] != ""
}

// Validate enforces the closed comparison domain.
func (o Order) Validate() error {
	if !o.IsValid() {
		return contractError(errLabelOrder)
	}
	return nil
}

// String returns the canonical token or an empty string when invalid.
func (o Order) String() string {
	if !o.IsValid() {
		return ""
	}
	return orderTokens()[o]
}

// MarshalJSON emits the canonical token.
func (o Order) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}

// UnmarshalJSON accepts only a canonical token and preserves the receiver on
// failure.
func (o *Order) UnmarshalJSON(data []byte) error {
	if o == nil {
		return contractError(errLabelOrder)
	}
	if len(data) > canonicalEnumJSONMaximumBytes {
		return contractError(errLabelOrder, core.ErrJSONContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return contractError(errLabelOrder, core.ErrJSONContract, err)
	}
	if string(data) != strconv.Quote(token) {
		return contractError(errLabelOrder, core.ErrJSONContract)
	}
	parsed, err := parseOrder(token)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

func parseOrder(token string) (Order, error) {
	for order := OrderLess; order <= OrderGreater; order++ {
		if orderTokens()[order] == token {
			return order, nil
		}
	}
	return OrderUnknown, contractError(errLabelOrder)
}
