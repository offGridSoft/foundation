package temporal

import "time"

// FirestoreInstant preserves exact nanoseconds and supplies a microsecond
// native-timestamp projection for queries.
type FirestoreInstant struct {
	QueryTimestamp time.Time
	Nanoseconds    int64
}

// Validate proves the exact and derived Firestore fields agree.
func (f FirestoreInstant) Validate() error {
	return validatePersistenceInstant(f.Nanoseconds, f.QueryTimestamp)
}

// PostgreSQLInstant preserves exact nanoseconds and supplies a microsecond
// native-timestamp projection for queries.
type PostgreSQLInstant struct {
	QueryTimestamp time.Time
	Nanoseconds    int64
}

// Validate proves the exact and derived PostgreSQL fields agree.
func (p PostgreSQLInstant) Validate() error {
	return validatePersistenceInstant(p.Nanoseconds, p.QueryTimestamp)
}

// FirestoreDuration is the exact signed Firestore duration representation.
type FirestoreDuration struct {
	Nanoseconds int64
}

// Validate rejects a negative Firestore duration.
func (f FirestoreDuration) Validate() error {
	if f.Nanoseconds < 0 {
		return contractError(errFmtPersistence)
	}
	return nil
}

// PostgreSQLDuration is the exact signed PostgreSQL duration representation.
type PostgreSQLDuration struct {
	Nanoseconds int64
}

// Validate rejects a negative PostgreSQL duration.
func (p PostgreSQLDuration) Validate() error {
	if p.Nanoseconds < 0 {
		return contractError(errFmtPersistence)
	}
	return nil
}

// FirestoreAggregateDuration is the canonical decimal Firestore aggregate
// duration representation.
type FirestoreAggregateDuration struct {
	Nanoseconds string
}

// Validate enforces canonical unsigned 128-bit decimal form.
func (f FirestoreAggregateDuration) Validate() error {
	_, err := ParseAggregateDuration(f.Nanoseconds)
	return err
}

// PostgreSQLAggregateDuration is the canonical decimal PostgreSQL aggregate
// duration representation.
type PostgreSQLAggregateDuration struct {
	Nanoseconds string
}

// Validate enforces canonical unsigned 128-bit decimal form.
func (p PostgreSQLAggregateDuration) Validate() error {
	_, err := ParseAggregateDuration(p.Nanoseconds)
	return err
}

// Firestore projects an instant without losing nanosecond identity.
func (i Instant) Firestore() (FirestoreInstant, error) {
	if err := i.Validate(); err != nil {
		return FirestoreInstant{}, err
	}
	return FirestoreInstant{
		QueryTimestamp: queryTimestamp(i.nanoseconds),
		Nanoseconds:    i.nanoseconds,
	}, nil
}

// PostgreSQL projects an instant without losing nanosecond identity.
func (i Instant) PostgreSQL() (PostgreSQLInstant, error) {
	if err := i.Validate(); err != nil {
		return PostgreSQLInstant{}, err
	}
	return PostgreSQLInstant{
		QueryTimestamp: queryTimestamp(i.nanoseconds),
		Nanoseconds:    i.nanoseconds,
	}, nil
}

// Firestore projects a duration to exact signed nanoseconds.
func (d Duration) Firestore() (FirestoreDuration, error) {
	if err := d.Validate(); err != nil {
		return FirestoreDuration{}, err
	}
	return FirestoreDuration{Nanoseconds: d.nanoseconds}, nil
}

// PostgreSQL projects a duration to exact signed nanoseconds.
func (d Duration) PostgreSQL() (PostgreSQLDuration, error) {
	if err := d.Validate(); err != nil {
		return PostgreSQLDuration{}, err
	}
	return PostgreSQLDuration{Nanoseconds: d.nanoseconds}, nil
}

// Firestore projects a full-range aggregate duration to canonical decimal.
func (a AggregateDuration) Firestore() (FirestoreAggregateDuration, error) {
	return FirestoreAggregateDuration{Nanoseconds: a.Decimal()}, nil
}

// PostgreSQL projects a full-range aggregate duration to canonical decimal.
func (a AggregateDuration) PostgreSQL() (PostgreSQLAggregateDuration, error) {
	return PostgreSQLAggregateDuration{Nanoseconds: a.Decimal()}, nil
}

func validatePersistenceInstant(nanoseconds int64, timestamp time.Time) error {
	if nanoseconds < 0 || timestamp != queryTimestamp(nanoseconds) {
		return contractError(errFmtPersistence)
	}
	return nil
}

func queryTimestamp(nanoseconds int64) time.Time {
	return time.Unix(0, nanoseconds).UTC().Truncate(time.Microsecond)
}
