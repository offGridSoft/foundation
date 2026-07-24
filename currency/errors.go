package currency

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	errLabelAmount      = "currency.Amount"
	errLabelCode        = "currency.Code"
	errLabelDecimal     = "currency.Decimal"
	errLabelHumanize    = "currency.Humanize"
	errLabelJSON        = "currency.JSON"
	errLabelOrder       = "currency.Order"
	errLabelPersistence = "currency.Persistence"
)

func contractError(label string, identities ...error) error {
	joined := make([]error, 0, len(identities)+1)
	joined = append(joined, core.ErrCurrencyContract)
	joined = append(joined, identities...)
	return fmt.Errorf("%s: %w", label, errors.Join(joined...))
}

func decimalError(label, raw string) error {
	return fmt.Errorf(
		"%s(decimal_bytes=%d): %w",
		label,
		len(raw),
		errors.Join(core.ErrCurrencyContract, core.ErrCurrencyDecimal, core.ErrInvalidDecimal),
	)
}

func mismatchError() error {
	return contractError(errLabelAmount, core.ErrCurrencyMismatch)
}

func overflowError() error {
	return contractError(errLabelAmount, core.ErrCurrencyOverflow, core.ErrNumericOverflow)
}
