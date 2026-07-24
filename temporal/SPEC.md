# Temporal Package Specification

Status: Reviewed implementation; consumer migration pending
Package: `github.com/offGridSoft/foundation/v2026/temporal`

## 1. Purpose

`temporal` owns Foundation's canonical time values and deterministic
projections. Every stored duration and instant uses exact nanoseconds.

The package owns:

- an absolute UTC instant;
- an ordinary non-negative duration bounded by signed 64-bit nanoseconds;
- a fleet-scale unsigned 128-bit aggregate duration;
- checked arithmetic and comparison;
- deterministic humanization;
- canonical JSON;
- reversible Firestore-safe and PostgreSQL-safe projections.

The package does not observe the clock, start timers, wait, schedule work,
localize implicitly, or own product-specific labels such as “core-hours.”

## 2. Dependency boundary

`temporal` imports `core` only for stable error identities and universal
validation contracts. `core` MUST NOT import `temporal`.

Any contract that requires temporal values belongs outside `core` in its
precise primitive or protocol package. This direction prevents an import cycle
and keeps `core` deterministic and domain-neutral.

Consumers define narrow clock or waiter interfaces locally. `temporal` exports
no clock, timer, ticker, scheduler, global service, or mutable package state.

## 3. Closed value model

`Instant`, `Duration`, and `AggregateDuration` have private representation.
Their constructors, parsers, strict JSON decoders, and persistence inverse
projections are the only creation boundaries.

Successfully created values are proof-carrying and need no repeated validation
while immutable. Failed decoding MUST NOT mutate the receiver.

`Duration` and `AggregateDuration` have no publicly reachable invalid
representation after construction. Their pure arithmetic and comparison
methods therefore operate directly on proof-carrying values and do not
revalidate. Their standard-library, persistence, and wire projection methods
validate at the required external-output boundary. `Instant` is different:
Go can always construct its invalid unset zero value, so every accessor,
operation, comparison, and projection validates each `Instant` operand.

The zero values mean:

- zero `Instant`: unset and invalid;
- zero `Duration`: valid zero elapsed time;
- zero `AggregateDuration`: valid zero aggregate effort.

`Instant.IsSet()` is the only errorless observation permitted on an `Instant`.
`Instant.Nanoseconds`, `Time`, arithmetic, comparison, persistence projection,
and marshaling all validate every `Instant` operand and fail on unset. They
never project unset to the Unix epoch. A failed operation returns the zero
result for its result type.

## 4. Instant

`Instant` stores a set, non-negative UTC Unix instant as signed 64-bit
nanoseconds.

Required construction:

- `NewInstant(time.Time) (Instant, error)` normalizes to UTC and rejects zero,
  pre-epoch, and values that cannot round-trip through signed Unix
  nanoseconds.
- `InstantFromNanoseconds(int64) (Instant, error)` rejects negative values.
- `InstantFromFirestore(FirestoreInstant) (Instant, error)`;
- `InstantFromPostgreSQL(PostgreSQLInstant) (Instant, error)`.

Required behavior:

- `Validate() error`;
- `IsSet() bool`;
- `Nanoseconds() (int64, error)`;
- `Time() (time.Time, error)`;
- `Add(Duration) (Instant, error)` with checked overflow;
- `Subtract(Duration) (Instant, error)` with checked underflow;
- `Since(Instant) (Duration, error)`, rejecting reversed order;
- `Compare(Instant) (Order, error)`;
- `Firestore() (FirestoreInstant, error)`;
- `PostgreSQL() (PostgreSQLInstant, error)`.

`Order` is a closed enum with the states unknown, before, equal, and after.
Unknown is the zero value and is invalid. `IsValid` reports membership and
`Validate` preserves `core.ErrTemporalContract` for unknown or out-of-range
states. `Compare` returns `OrderUnknown` with any error so a discarded error
cannot appear to yield a meaningful ordering. Its canonical JSON tokens are
`"before"`, `"equal"`, and `"after"`; unknown is never emitted or accepted.

Instant arithmetic never accepts a negative duration. Presentation formatting
is not an `Instant` method because it requires explicit locale, location, and
layout policy not yet admitted to this package.

## 5. Duration

`Duration` stores non-negative signed 64-bit nanoseconds. It represents elapsed
work, a timeout, interval, delay, or budget.

Required construction:

- `NewDuration(time.Duration) (Duration, error)` rejects a negative duration;
- `DurationFromNanoseconds(int64) (Duration, error)` rejects negative input;
- `DurationFromFirestore(FirestoreDuration) (Duration, error)` rejects a
  negative `Nanoseconds` field;
- `DurationFromPostgreSQL(PostgreSQLDuration) (Duration, error)` rejects a
  negative `Nanoseconds` field.

Required behavior:

- `Validate() error`;
- `Nanoseconds() int64`;
- `Stdlib() (time.Duration, error)`;
- `IsZero() bool`;
- `Add(Duration) (Duration, error)` with checked overflow;
- `Multiply(uint64) (Duration, error)` with checked overflow;
- `Compare(Duration) Order`;
- `Aggregate() AggregateDuration`;
- `Firestore() (FirestoreDuration, error)`;
- `PostgreSQL() (PostgreSQLDuration, error)`;
- `Humanize(HumanizePolicy) (Humanized, error)`.

`FirestoreDuration` is a validated struct containing one signed `Nanoseconds`
field. Negative values are invalid.
`PostgreSQLDuration` has the same logical field and validation contract.

## 6. Aggregate duration

`AggregateDuration` stores an unsigned 128-bit nanosecond total as two private
machine limbs. Limbs are implementation, never API or wire shape.

Required construction:

- `ParseAggregateDuration(string) (AggregateDuration, error)`;
- `AggregateDurationFromDuration(Duration) AggregateDuration`;
- `AggregateDurationFromFirestore(FirestoreAggregateDuration)
  (AggregateDuration, error)`;
- `AggregateDurationFromPostgreSQL(PostgreSQLAggregateDuration)
  (AggregateDuration, error)`.

The canonical decimal form:

- is unsigned base 10;
- uses `"0"` for zero;
- contains no sign, whitespace, fraction, exponent, or separator;
- contains no leading zero;
- has at most the digits required by the maximum unsigned 128-bit value;
- rejects overflow.

Required behavior:

- `Validate() error`;
- `IsZero() bool`;
- `Decimal() string`;
- `Add(AggregateDuration) (AggregateDuration, error)`;
- `AddDuration(Duration) (AggregateDuration, error)`;
- `Multiply(uint64) (AggregateDuration, error)` with checked overflow;
- `Compare(AggregateDuration) Order`;
- `Firestore() (FirestoreAggregateDuration, error)`;
- `PostgreSQL() (PostgreSQLAggregateDuration, error)`;
- `Humanize(HumanizePolicy) (Humanized, error)`.

`FirestoreAggregateDuration` contains one canonical decimal `Nanoseconds`
string. Firestore stores it as a string because Firestore has no unsigned
128-bit integer. The inverse projection validates canonical form and range.
`PostgreSQLAggregateDuration` contains the same canonical decimal string for a
`NUMERIC(39,0)` or text column. Foundation does not delegate unsigned 128-bit
range or canonical-form validation to a database driver.

No floating-point conversion is exported.

## 7. Canonical JSON

All three canonical values encode as JSON strings containing canonical decimal
nanoseconds:

```json
"0"
```

This avoids loss in JavaScript and every JSON implementation whose numeric
domain cannot exactly represent 64-bit or 128-bit integers.

Marshaling an unset `Instant` fails. Decoding `"0"` always constructs the set
Unix epoch; decode behavior never depends on the receiver's prior state.

Decoders reject:

- empty strings;
- leading or trailing whitespace inside the string;
- `+` or `-`;
- leading zeros other than `"0"`;
- fractions;
- exponents;
- non-ASCII digits;
- values outside the owning type's range;
- non-string JSON;
- `null`, arrays, and objects.

Marshal validates before output. Unmarshal validates a temporary before
assignment.

## 8. Humanization

Humanization is deterministic presentation, never stored truth.

`HumanizePolicy` contains:

- `Unit`: automatic or one explicit `HumanUnit`;
- `FractionDigits`: a bounded decimal-place count;
- `Style`: long or compact.

`HumanUnit` is a closed enum:

- automatic;
- nanoseconds;
- microseconds;
- milliseconds;
- seconds;
- minutes;
- hours;
- days;
- years.

Its canonical JSON tokens are `"automatic"`, `"nanoseconds"`,
`"microseconds"`, `"milliseconds"`, `"seconds"`, `"minutes"`, `"hours"`,
`"days"`, and `"years"`. `HumanizeStyle` uses `"long"` and `"compact"`.
Unknown and out-of-range enum states are never emitted or accepted. Enum
decoders validate a temporary before assignment.

A year is exactly 365.25 days for this mechanical conversion. This is not a
calendar-year contract.

Automatic selection chooses the largest unit whose divisor does not exceed the
value. Zero selects nanoseconds. Explicit selection never changes the stored
value.

Humanization truncates discarded fractional nanoseconds and never rounds up.
It uses integer arithmetic only. `FractionDigits` is an exact count in the
inclusive range zero through nine. Every requested digit is emitted, including
trailing zeros. A value above nine is rejected, never clamped. Nine is the
global bound because additional digits would invent precision finer than the
nanosecond source of truth.

`Humanized` has private representation and exposes:

- `Validate() error`;
- `Number() string`;
- `Unit() HumanUnit`;
- `Text() string`.

Long style always uses stable English plural unit tokens, including for the
numeric value one. Compact style uses the exact ASCII tokens `ns`, `us`, `ms`,
`s`, `m`, `h`, `d`, and `y`. The decimal separator is always `.`.
Long style joins number and token with one ASCII space; compact style
concatenates them with no separator. Localization is outside this package.

For any value representable as both `Duration` and `AggregateDuration`, the two
`Humanize` methods return structurally identical `Humanized` values under the
same policy.

## 9. Persistence projections

Firestore and PostgreSQL projections are typed boundary structures, not domain
truth. Exact nanoseconds are always authoritative.

| Domain type | Firestore projection | PostgreSQL projection |
| --- | --- | --- |
| `Instant` | `FirestoreInstant{Nanoseconds int64, QueryTimestamp time.Time}` | `PostgreSQLInstant{Nanoseconds int64, QueryTimestamp time.Time}` |
| `Duration` | `FirestoreDuration{Nanoseconds int64}` | `PostgreSQLDuration{Nanoseconds int64}` |
| `AggregateDuration` | `FirestoreAggregateDuration{Nanoseconds string}` | `PostgreSQLAggregateDuration{Nanoseconds string}` |

`Nanoseconds` is the sole reconstruction and equality source. Signed values
map to Firestore integer fields and PostgreSQL `BIGINT`. Aggregate values map
to a Firestore string and PostgreSQL `NUMERIC(39,0)` or text, transported as a
canonical decimal string so driver-specific numeric representations cannot
weaken the contract.

`QueryTimestamp` is a derived UTC convenience field for native timestamp
queries and display. It is deliberately truncated down to the start of the
microsecond because both Firestore timestamps and PostgreSQL timestamps have
microsecond resolution. Projection validation requires it to equal the
microsecond-truncated instant derived from `Nanoseconds`. Inverse construction
validates this relation but reconstructs exclusively from `Nanoseconds`.
Consumers MUST NOT use `QueryTimestamp` as identity or for sub-microsecond
ordering.

Equality is deliberately structural Go `time.Time` equality, not
`time.Time.Equal`: the location MUST be the canonical `time.UTC` location,
microsecond truncation MUST already have occurred, and monotonic-clock
readings MUST be absent. Database adapters own normalization of driver-returned
timestamps before constructing a projection:
`returned.UTC().Truncate(time.Microsecond)`. This normalization is an ingress
contract, not silent repair inside `temporal`; a projection carrying a
non-canonical location or a contradictory wall value is rejected.

Every projection validates before output and has one inverse constructor.
Round-trip equality is required at zero, one, every signed boundary, every
sub-microsecond remainder, the unsigned 64-bit carry boundary, and the
unsigned 128-bit maximum.

The package imports neither database SDK nor database driver. Adapter
conformance tests outside this pure package MUST store and read these exact
fields against live Firestore and PostgreSQL and prove the inverse constructor
preserves nanosecond truth.

## 10. Errors

All package failures preserve `core.ErrTemporalContract`, which wraps
`core.ErrFoundationContract`. Therefore every temporal failure also satisfies
`errors.Is(err, core.ErrFoundationContract)`.

Range escape and arithmetic overflow additionally preserve
`core.ErrNumericOverflow`. Malformed canonical decimal input additionally
preserves `core.ErrInvalidDecimal`.

Callers and tests use `errors.Is`; diagnostic strings are not identity.

## 11. Complexity

Every operation has O(1) memory and bounded runtime.

Decimal parsing and formatting scan at most the maximum unsigned 128-bit
decimal width. Humanization emits at most the policy-bounded number of
fractional digits. The package allocates only bounded presentation or wire
strings.

No operation loads external data, walks collections, starts goroutines, or
performs I/O.

## 12. Cross-platform behavior

The package has identical behavior on macOS, Linux, and Windows. It depends
only on Go integer arithmetic and `time.Time`.

No behavior depends on local time zone, locale, wall-clock state, word size, or
operating-system APIs.

## 13. Export budget

The public surface is limited to:

- the value and projection structs named in this specification;
- the `Order`, `HumanUnit`, and `HumanizeStyle` enums;
- `HumanizePolicy` and `Humanized`;
- the constructors and parsers named in this specification;
- methods required by this specification;
- enum parsers and JSON methods needed for compiler-owned wire contracts.

Arithmetic helpers, 128-bit limbs, divisors, decimal scanners, formatting
buffers, and persistence conversion helpers remain private.

Every additional export requires a reviewed specification amendment before
implementation.

## 14. Required proof

Hostile tables MUST cover:

- unset, epoch, one nanosecond, maximum signed instant, and pre-epoch rejection;
- duration zero, one, maximum, negative, add overflow, and reversed `Since`;
- instant subtract underflow and duration/aggregate scalar multiplication at
  zero, one, exact maximum, and one-past overflow;
- exhaustive valid instant, duration, and aggregate ordering; unset `Instant`
  operands additionally return `OrderUnknown` with
  `core.ErrTemporalContract`;
- aggregate zero, unsigned-64 carry, unsigned-128 maximum, and overflow;
- canonical decimal and JSON rejection without receiver mutation;
- exact JavaScript-unsafe integer boundaries represented losslessly as strings;
- Firestore and PostgreSQL round trips, real microsecond timestamp degradation,
  exact-field preservation, derived-field contradiction, and malformed
  projection rejection;
- every human unit at one-below, exact, and one-above thresholds;
- every permitted fractional precision;
- truncation immediately below a display rollover;
- identical ordinary/aggregate humanization across their overlap;
- exhaustive enum states and hostile enum JSON;
- fuzz parsing that never accepts non-canonical input or mutates on rejection.

Tests use structural comparisons, typed identities, and table-driven extremes.
No sleeps, live clock, database emulator, reflection-based assertions, or
floating-point comparisons are permitted.
