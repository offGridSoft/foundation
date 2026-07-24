package temporal

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestFirestoreInstantRoundTripPreservesNanosecondsTable(t *testing.T) {
	t.Parallel()

	for _, nanos := range persistenceInstantExtremes() {
		t.Run(persistenceCaseName(nanos), func(t *testing.T) {
			t.Parallel()

			instant, err := InstantFromNanoseconds(nanos)
			if err != nil {
				t.Fatal(err)
			}
			wantQuery := time.Unix(0, nanos).UTC().Truncate(time.Microsecond)
			firestore, err := instant.Firestore()
			if err != nil || firestore.Nanoseconds != nanos || !firestore.QueryTimestamp.Equal(wantQuery) {
				t.Fatalf("Firestore() = (%+v, %v), want nanos=%d query=%v", firestore, err, nanos, wantQuery)
			}
			fromFirestore, err := InstantFromFirestore(firestore)
			if err != nil || fromFirestore != instant {
				t.Fatalf("InstantFromFirestore() = (%+v, %v), want (%+v, nil)", fromFirestore, err, instant)
			}
		})
	}
}

func TestPostgreSQLInstantRoundTripPreservesNanosecondsTable(t *testing.T) {
	t.Parallel()

	for _, nanos := range persistenceInstantExtremes() {
		t.Run(persistenceCaseName(nanos), func(t *testing.T) {
			t.Parallel()

			instant, err := InstantFromNanoseconds(nanos)
			if err != nil {
				t.Fatal(err)
			}
			wantQuery := time.Unix(0, nanos).UTC().Truncate(time.Microsecond)
			postgresql, err := instant.PostgreSQL()
			if err != nil || postgresql.Nanoseconds != nanos || !postgresql.QueryTimestamp.Equal(wantQuery) {
				t.Fatalf("PostgreSQL() = (%+v, %v), want nanos=%d query=%v", postgresql, err, nanos, wantQuery)
			}
			fromPostgreSQL, err := InstantFromPostgreSQL(postgresql)
			if err != nil || fromPostgreSQL != instant {
				t.Fatalf("InstantFromPostgreSQL() = (%+v, %v), want (%+v, nil)", fromPostgreSQL, err, instant)
			}
		})
	}
}

func TestInstantPersistenceRejectsContradictoryOrMalformedProjectionTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		query   time.Time
		wantErr error
		name    string
		nanos   int64
	}{
		{name: "query carries forbidden nanosecond precision", nanos: 1, query: time.Unix(0, 1).UTC(), wantErr: core.ErrTemporalContract},
		{name: "query contradicts exact nanoseconds", nanos: 1_001, query: time.Unix(0, 2_000).UTC(), wantErr: core.ErrTemporalContract},
		{name: "query is zero time", nanos: 1, wantErr: core.ErrTemporalContract},
		{name: "query is not UTC", nanos: 1_001, query: time.Unix(0, 1_000).In(time.FixedZone("hostile", 3600)), wantErr: core.ErrTemporalContract},
		{name: "zero offset location clone is not canonical UTC", nanos: 1_001, query: time.Unix(0, 1_000).In(time.FixedZone("UTC clone", 0)), wantErr: core.ErrTemporalContract},
		{name: "exact nanoseconds are negative", nanos: -1, query: time.Unix(-1, 999_999_000).UTC(), wantErr: core.ErrTemporalContract},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, firestoreErr := InstantFromFirestore(FirestoreInstant{Nanoseconds: test.nanos, QueryTimestamp: test.query})
			if !errors.Is(firestoreErr, test.wantErr) {
				t.Fatalf("InstantFromFirestore() error = %v, want %v", firestoreErr, test.wantErr)
			}
			_, postgreSQLErr := InstantFromPostgreSQL(PostgreSQLInstant{Nanoseconds: test.nanos, QueryTimestamp: test.query})
			if !errors.Is(postgreSQLErr, test.wantErr) {
				t.Fatalf("InstantFromPostgreSQL() error = %v, want %v", postgreSQLErr, test.wantErr)
			}
		})
	}
}

func TestDurationPersistenceRoundTripTable(t *testing.T) {
	t.Parallel()

	for _, nanos := range []int64{0, 1, 9_007_199_254_740_993, math.MaxInt64} {
		duration, err := DurationFromNanoseconds(nanos)
		if err != nil {
			t.Fatal(err)
		}
		firestore, firestoreErr := duration.Firestore()
		fromFirestore, firestoreDecodeErr := DurationFromFirestore(firestore)
		postgresql, postgresqlErr := duration.PostgreSQL()
		fromPostgreSQL, postgresqlDecodeErr := DurationFromPostgreSQL(postgresql)
		if firestoreErr != nil || firestoreDecodeErr != nil || fromFirestore != duration {
			t.Fatalf("Firestore duration %d = (%+v, %v, %v)", nanos, fromFirestore, firestoreErr, firestoreDecodeErr)
		}
		if postgresqlErr != nil || postgresqlDecodeErr != nil || fromPostgreSQL != duration {
			t.Fatalf("PostgreSQL duration %d = (%+v, %v, %v)", nanos, fromPostgreSQL, postgresqlErr, postgresqlDecodeErr)
		}
	}
}

func TestDurationPersistenceRejectsNegativeTable(t *testing.T) {
	t.Parallel()

	if _, err := DurationFromFirestore(FirestoreDuration{Nanoseconds: -1}); !errors.Is(err, core.ErrTemporalContract) {
		t.Fatalf("negative FirestoreDuration error = %v, want %v", err, core.ErrTemporalContract)
	}
	if _, err := DurationFromPostgreSQL(PostgreSQLDuration{Nanoseconds: -1}); !errors.Is(err, core.ErrTemporalContract) {
		t.Fatalf("negative PostgreSQLDuration error = %v, want %v", err, core.ErrTemporalContract)
	}
}

func TestFirestoreAggregateRoundTripTable(t *testing.T) {
	t.Parallel()

	for _, decimal := range []string{"0", "1", maxUint64Decimal, uint64CarryDecimal, maxUint128Decimal} {
		value, err := ParseAggregateDuration(decimal)
		if err != nil {
			t.Fatal(err)
		}
		firestore, firestoreErr := value.Firestore()
		fromFirestore, firestoreDecodeErr := AggregateDurationFromFirestore(firestore)
		if firestoreErr != nil || firestoreDecodeErr != nil || fromFirestore != value || firestore.Nanoseconds != decimal {
			t.Fatalf("Firestore aggregate %q = (%+v, %+v, %v, %v)", decimal, firestore, fromFirestore, firestoreErr, firestoreDecodeErr)
		}
	}
}

func TestPostgreSQLAggregateRoundTripTable(t *testing.T) {
	t.Parallel()

	for _, decimal := range []string{"0", "1", maxUint64Decimal, uint64CarryDecimal, maxUint128Decimal} {
		value, err := ParseAggregateDuration(decimal)
		if err != nil {
			t.Fatal(err)
		}
		postgresql, postgresqlErr := value.PostgreSQL()
		fromPostgreSQL, postgresqlDecodeErr := AggregateDurationFromPostgreSQL(postgresql)
		if postgresqlErr != nil || postgresqlDecodeErr != nil || fromPostgreSQL != value || postgresql.Nanoseconds != decimal {
			t.Fatalf("PostgreSQL aggregate %q = (%+v, %+v, %v, %v)", decimal, postgresql, fromPostgreSQL, postgresqlErr, postgresqlDecodeErr)
		}
	}
}

func TestAggregatePersistenceRejectsMalformedTable(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "00", "-1", "340282366920938463463374607431768211456"} {
		if _, err := AggregateDurationFromFirestore(FirestoreAggregateDuration{Nanoseconds: value}); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("FirestoreAggregateDuration(%q) error = %v, want %v", value, err, core.ErrTemporalContract)
		}
		if _, err := AggregateDurationFromPostgreSQL(PostgreSQLAggregateDuration{Nanoseconds: value}); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("PostgreSQLAggregateDuration(%q) error = %v, want %v", value, err, core.ErrTemporalContract)
		}
	}
}

func persistenceInstantExtremes() []int64 {
	return []int64{
		0,
		1,
		999,
		1_000,
		1_001,
		9_007_199_254_740_993,
		math.MaxInt64 - 999,
		math.MaxInt64,
	}
}

func persistenceCaseName(nanos int64) string {
	switch nanos {
	case 0:
		return "epoch"
	case 1:
		return "one nanosecond degrades query timestamp only"
	case 999:
		return "last sub microsecond remainder"
	case 1_000:
		return "exact microsecond"
	case 1_001:
		return "microsecond plus one"
	case math.MaxInt64:
		return "maximum"
	default:
		return "large exact integer"
	}
}
