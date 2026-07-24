package currency

import (
	"encoding/json"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

type amountJSON struct {
	Currency   Code                `json:"currency"`
	MinorUnits canonicalMinorUnits `json:"minor_units"`
}

type canonicalMinorUnits struct {
	value int64
	set   bool
}

const canonicalMinorUnitsJSONMaximumBytes = 22

func newCanonicalMinorUnits(value int64) canonicalMinorUnits {
	return canonicalMinorUnits{value: value, set: true}
}

func (u canonicalMinorUnits) Validate() error {
	if !u.set {
		return decimalError(errLabelJSON, "")
	}
	return nil
}

func (u canonicalMinorUnits) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(strconv.FormatInt(u.value, 10))
}

func (u *canonicalMinorUnits) UnmarshalJSON(data []byte) error {
	if u == nil {
		return decimalError(errLabelJSON, "")
	}
	if len(data) < 3 || len(data) > canonicalMinorUnitsJSONMaximumBytes ||
		data[0] != '"' || data[len(data)-1] != '"' {
		return contractError(errLabelAmount, core.ErrJSONContract, core.ErrCurrencyDecimal)
	}
	raw := string(data[1 : len(data)-1])
	value, err := parseCanonicalMinorUnits(raw)
	if err != nil {
		return err
	}
	*u = newCanonicalMinorUnits(value)
	return nil
}

func (a amountJSON) Validate() error {
	if err := a.Currency.Validate(); err != nil {
		return contractError(errLabelAmount, err)
	}
	if err := a.MinorUnits.Validate(); err != nil {
		return err
	}
	return nil
}

func (a amountJSON) close() Amount {
	return Amount{code: a.Currency, minorUnits: a.MinorUnits.value}
}

// MarshalJSON emits the closed canonical amount object.
func (a Amount) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return core.EncodeValidatedJSON(amountJSON{
		Currency:   a.code,
		MinorUnits: newCanonicalMinorUnits(a.minorUnits),
	})
}

// UnmarshalJSON accepts the closed canonical amount object and preserves the
// receiver on failure.
func (a *Amount) UnmarshalJSON(data []byte) error {
	if a == nil {
		return contractError(errLabelAmount)
	}
	wire, err := core.DecodeStrictJSON[amountJSON](data)
	if err != nil {
		return contractError(errLabelAmount, err)
	}
	*a = wire.close()
	return nil
}

func parseCanonicalMinorUnits(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || strconv.FormatInt(value, 10) != raw {
		return 0, decimalError(errLabelJSON, raw)
	}
	return value, nil
}
