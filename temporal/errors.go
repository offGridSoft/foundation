package temporal

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	errFmtAggregate   = "temporal.AggregateDuration: %w"
	errFmtDuration    = "temporal.Duration: %w"
	errFmtHumanize    = "temporal.Humanize: %w"
	errFmtInstant     = "temporal.Instant: %w"
	errFmtOrder       = "temporal.Order: %w"
	errFmtPersistence = "temporal persistence: %w"
)

func contractError(format string, identities ...error) error {
	joined := append([]error{core.ErrTemporalContract}, identities...)
	return fmt.Errorf(format, errors.Join(joined...))
}
