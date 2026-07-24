package currency

import (
	"encoding/json"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

// Code is the closed supported ISO 4217 currency domain.
type Code uint8

const (
	CodeUnknown Code = iota
	CodeUSD
	CodeEUR
	CodeGBP
	CodeCAD
	CodeAUD
	CodeJPY
	CodeCHF
	CodeNZD
	CodeSGD
	CodeHKD
	CodeBHD
	CodeCLF
)

const canonicalEnumJSONMaximumBytes = 16

type minorUnitExponent uint8

const (
	minorUnitExponentZero  minorUnitExponent = 0
	minorUnitExponentTwo   minorUnitExponent = 2
	minorUnitExponentThree minorUnitExponent = 3
	minorUnitExponentFour  minorUnitExponent = 4
)

func codeTokens() [CodeCLF + 1]string {
	return [...]string{
		CodeUSD: "USD",
		CodeEUR: "EUR",
		CodeGBP: "GBP",
		CodeCAD: "CAD",
		CodeAUD: "AUD",
		CodeJPY: "JPY",
		CodeCHF: "CHF",
		CodeNZD: "NZD",
		CodeSGD: "SGD",
		CodeHKD: "HKD",
		CodeBHD: "BHD",
		CodeCLF: "CLF",
	}
}

func codeExponents() [CodeCLF + 1]minorUnitExponent {
	return [...]minorUnitExponent{
		CodeUSD: minorUnitExponentTwo,
		CodeEUR: minorUnitExponentTwo,
		CodeGBP: minorUnitExponentTwo,
		CodeCAD: minorUnitExponentTwo,
		CodeAUD: minorUnitExponentTwo,
		CodeJPY: minorUnitExponentZero,
		CodeCHF: minorUnitExponentTwo,
		CodeNZD: minorUnitExponentTwo,
		CodeSGD: minorUnitExponentTwo,
		CodeHKD: minorUnitExponentTwo,
		CodeBHD: minorUnitExponentThree,
		CodeCLF: minorUnitExponentFour,
	}
}

// IsValid reports membership in the supported currency domain.
func (c Code) IsValid() bool {
	return c > CodeUnknown && int(c) < len(codeTokens()) && codeTokens()[c] != ""
}

// Validate enforces the closed currency domain.
func (c Code) Validate() error {
	if !c.IsValid() {
		return contractError(errLabelCode)
	}
	return nil
}

// String returns the canonical uppercase token or an empty string when invalid.
func (c Code) String() string {
	if !c.IsValid() {
		return ""
	}
	return codeTokens()[c]
}

// FractionDigits returns the currency-owned minor-unit exponent.
func (c Code) FractionDigits() (uint8, error) {
	if err := c.Validate(); err != nil {
		return 0, err
	}
	return c.fractionDigits(), nil
}

func (c Code) fractionDigits() uint8 {
	return uint8(codeExponents()[c])
}

// MarshalJSON emits the canonical uppercase token.
func (c Code) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

// UnmarshalJSON accepts only the canonical uppercase token and preserves the
// receiver on failure.
func (c *Code) UnmarshalJSON(data []byte) error {
	if c == nil {
		return contractError(errLabelCode)
	}
	if len(data) > canonicalEnumJSONMaximumBytes {
		return contractError(errLabelCode, core.ErrJSONContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return contractError(errLabelCode, core.ErrJSONContract, err)
	}
	if string(data) != strconv.Quote(token) {
		return contractError(errLabelCode, core.ErrJSONContract)
	}
	parsed, err := ParseCode(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
