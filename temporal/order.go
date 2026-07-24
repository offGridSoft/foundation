package temporal

import (
	"encoding/json"

	"github.com/offGridSoft/foundation/v2026/core"
)

// Order is the closed result of comparing two temporal values.
type Order uint8

const (
	OrderUnknown Order = iota
	OrderBefore
	OrderEqual
	OrderAfter
)

const (
	orderTokenBefore = "before"
	orderTokenEqual  = "equal"
	orderTokenAfter  = "after"
)

func orderTokens() [OrderAfter + 1]string {
	return [...]string{
		OrderBefore: orderTokenBefore,
		OrderEqual:  orderTokenEqual,
		OrderAfter:  orderTokenAfter,
	}
}

// IsValid reports whether the order is a known state.
func (o Order) IsValid() bool {
	return o > OrderUnknown && int(o) < len(orderTokens()) && orderTokens()[o] != ""
}

// Validate enforces the closed order domain.
func (o Order) Validate() error {
	if !o.IsValid() {
		return contractError(errFmtOrder)
	}
	return nil
}

// String returns the canonical token or an empty string for an invalid state.
func (o Order) String() string {
	if !o.IsValid() {
		return ""
	}
	return orderTokens()[o]
}

// MarshalJSON emits the canonical string token.
func (o Order) MarshalJSON() ([]byte, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(o.String())
}

// UnmarshalJSON accepts only a canonical string token and does not mutate on
// failure.
func (o *Order) UnmarshalJSON(data []byte) error {
	if o == nil {
		return contractError(errFmtOrder)
	}
	token, err := decodeJSONString(data)
	if err != nil {
		return orderJSONError(err)
	}
	parsed, err := ParseOrder(token)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

func compareInt64(left, right int64) Order {
	switch {
	case left < right:
		return OrderBefore
	case left > right:
		return OrderAfter
	default:
		return OrderEqual
	}
}

func orderJSONError(err error) error {
	return contractError(errFmtOrder, core.ErrJSONContract, err)
}
