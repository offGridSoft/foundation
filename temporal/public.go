package temporal

import "time"

// Public package functions are intentionally centralized here. Public value
// methods remain beside their owning types so their invariants are visible
// with their implementations.

// NewInstant constructs an exact UTC instant from a standard-library time.
func NewInstant(value time.Time) (Instant, error) {
	return instantFromTime(value)
}

// InstantFromNanoseconds constructs a set instant from non-negative Unix
// nanoseconds.
func InstantFromNanoseconds(nanoseconds int64) (Instant, error) {
	if nanoseconds < 0 {
		return Instant{}, contractError(errFmtInstant)
	}
	return Instant{nanoseconds: nanoseconds, set: true}, nil
}

// NewDuration constructs a non-negative duration.
func NewDuration(value time.Duration) (Duration, error) {
	return DurationFromNanoseconds(int64(value))
}

// DurationFromNanoseconds constructs a non-negative duration from exact
// nanoseconds.
func DurationFromNanoseconds(nanoseconds int64) (Duration, error) {
	if nanoseconds < 0 {
		return Duration{}, contractError(errFmtDuration)
	}
	return Duration{nanoseconds: nanoseconds}, nil
}

// ParseAggregateDuration parses canonical unsigned 128-bit nanoseconds.
func ParseAggregateDuration(decimal string) (AggregateDuration, error) {
	return parseAggregateDuration(decimal)
}

// AggregateDurationFromDuration widens an ordinary duration without loss.
func AggregateDurationFromDuration(duration Duration) AggregateDuration {
	// #nosec G115 -- Duration's private representation and constructor boundary
	// prove nanoseconds are non-negative before this lossless widening.
	return AggregateDuration{low: uint64(duration.nanoseconds)}
}

// InstantFromFirestore reconstructs exclusively from exact nanoseconds after
// validating the derived query timestamp.
func InstantFromFirestore(value FirestoreInstant) (Instant, error) {
	if err := value.Validate(); err != nil {
		return Instant{}, err
	}
	return InstantFromNanoseconds(value.Nanoseconds)
}

// InstantFromPostgreSQL reconstructs exclusively from exact nanoseconds after
// validating the derived query timestamp.
func InstantFromPostgreSQL(value PostgreSQLInstant) (Instant, error) {
	if err := value.Validate(); err != nil {
		return Instant{}, err
	}
	return InstantFromNanoseconds(value.Nanoseconds)
}

// DurationFromFirestore validates and reconstructs a Firestore duration.
func DurationFromFirestore(value FirestoreDuration) (Duration, error) {
	if err := value.Validate(); err != nil {
		return Duration{}, err
	}
	return DurationFromNanoseconds(value.Nanoseconds)
}

// DurationFromPostgreSQL validates and reconstructs a PostgreSQL duration.
func DurationFromPostgreSQL(value PostgreSQLDuration) (Duration, error) {
	if err := value.Validate(); err != nil {
		return Duration{}, err
	}
	return DurationFromNanoseconds(value.Nanoseconds)
}

// AggregateDurationFromFirestore validates and reconstructs an aggregate.
func AggregateDurationFromFirestore(value FirestoreAggregateDuration) (AggregateDuration, error) {
	if err := value.Validate(); err != nil {
		return AggregateDuration{}, err
	}
	return ParseAggregateDuration(value.Nanoseconds)
}

// AggregateDurationFromPostgreSQL validates and reconstructs an aggregate.
func AggregateDurationFromPostgreSQL(value PostgreSQLAggregateDuration) (AggregateDuration, error) {
	if err := value.Validate(); err != nil {
		return AggregateDuration{}, err
	}
	return ParseAggregateDuration(value.Nanoseconds)
}

// ParseOrder parses a canonical order token.
func ParseOrder(token string) (Order, error) {
	for order := OrderBefore; order <= OrderAfter; order++ {
		if orderTokens()[order] == token {
			return order, nil
		}
	}
	return OrderUnknown, contractError(errFmtOrder)
}

// ParseHumanUnit parses a canonical unit token.
func ParseHumanUnit(token string) (HumanUnit, error) {
	for unit := HumanUnitAutomatic; unit <= HumanUnitYears; unit++ {
		if unit.String() == token {
			return unit, nil
		}
	}
	return HumanUnitUnknown, contractError(errFmtHumanize)
}

// ParseHumanizeStyle parses a canonical style token.
func ParseHumanizeStyle(token string) (HumanizeStyle, error) {
	for style := HumanizeStyleLong; style <= HumanizeStyleCompact; style++ {
		if style.String() == token {
			return style, nil
		}
	}
	return HumanizeStyleUnknown, contractError(errFmtHumanize)
}
