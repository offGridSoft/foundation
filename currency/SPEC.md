# Currency Package Specification

Status: Reviewed implementation; live adapter conformance pending
Package: `github.com/offGridSoft/foundation/v2026/currency`

## 1. Purpose

`currency` owns Foundation's exact monetary currency domain.

The package owns:

- a closed supported-currency domain;
- each supported currency's canonical ISO 4217 token and minor-unit exponent;
- exact signed minor-unit storage;
- checked monetary arithmetic and comparison;
- bounded decimal parsing and deterministic decimal formatting;
- deterministic minor/major/hundreds/thousands/millions/billions
  humanization;
- canonical JSON;
- reversible Firestore-safe and PostgreSQL-safe projections.

The package does not own prices, taxes, discounts, refunds, exchange rates,
rounding policy, locale, symbols, payment providers, products, plans, tenants,
or accounting rules. Consumers enforce those business requirements on a
validated `Amount`.

## 2. Dependency and ownership boundary

`currency` imports `core` only for universal error identities and strict JSON
closure. It imports no Foundation sibling and names no consumer.

`core.ByteCount` and `core.ByteLength` remain in `core`: exchange, durability,
host-resource, and object-storage capabilities all require those universal
contracts and MUST NOT import `currency` merely to obtain shared byte
vocabulary. A broad package named `units` would either violate that ownership
rule or become a miscellaneous drawer. The precise package name is therefore
`currency`.

The retired unsigned, currency-free core money type has no alias, forwarding
wrapper, compatibility constructor, or alternate spelling. Monetary truth has
one owner: `currency.Amount`.

## 3. Closed value model

`Code` is a closed `uint8` enum. Its zero value is unknown and invalid.
Each valid enum value owns one uppercase ISO token and one minor-unit exponent.
Parsing, validation, string conversion, JSON, decimal parsing, and persistence
projection use that single table.

The supported domain for `v2026.0.0` is:

| Currency | Exponent |
| --- | ---: |
| USD | 2 |
| EUR | 2 |
| GBP | 2 |
| CAD | 2 |
| AUD | 2 |
| JPY | 0 |
| CHF | 2 |
| NZD | 2 |
| SGD | 2 |
| HKD | 2 |
| BHD | 3 |
| CLF | 4 |

This is a supported subset, not a claim to implement every ISO 4217 currency.
Adding or removing a supported currency is a compiler-visible protocol change
requiring exhaustive enum, exponent, parser, JSON, and persistence tests.
BHD and CLF deliberately prevent an implementation from hard-coding the
common zero-or-two-decimal assumption.

`Amount` stores a private `Code` and private signed 64-bit minor-unit value.
Its zero value has an unknown currency and is invalid. A valid currency with
zero minor units is valid zero money.

Constructors, parsers, strict JSON decoding, and persistence inverse
projections are closure boundaries. Successfully created values are
proof-carrying while immutable. Failed decode or inverse projection MUST NOT
mutate an existing receiver.

## 4. Stable public intent

Package functions are centralized in `public.go`:

- `New(Code, int64) (Amount, error)`;
- `Parse(Code, string) (Amount, error)`;
- `ParseCode(string) (Code, error)`;
- `ParseDisplayUnit(string) (DisplayUnit, error)`;
- `FromFirestore(FirestoreAmount) (Amount, error)`;
- `FromPostgreSQL(PostgreSQLAmount) (Amount, error)`.

`Amount` provides:

- `Validate() error`;
- `Code() (Code, error)`;
- `MinorUnits() (int64, error)`;
- `IsZero() (bool, error)`;
- `IsPositive() (bool, error)`;
- `IsNegative() (bool, error)`;
- `Add(Amount) (Amount, error)`;
- `Subtract(Amount) (Amount, error)`;
- `Multiply(uint64) (Amount, error)`;
- `Compare(Amount) (Order, error)`;
- `Decimal() (string, error)`;
- `Humanize(HumanizePolicy) (Humanized, error)`;
- `Firestore() (FirestoreAmount, error)`;
- `PostgreSQL() (PostgreSQLAmount, error)`;
- canonical JSON marshal and unmarshal.

`Code` provides validation, membership, canonical token, exponent,
and canonical JSON behavior. `Order` is a package-local closed result with
unknown, less, equal, and greater states. Unknown is invalid and is returned
with every comparison error so a discarded error cannot resemble an answer.

The package exports no service object, registry, mutable global, provider
adapter, formatter interface, or generic arithmetic helper.

## 5. Arithmetic

Arithmetic operates on exact minor units.

- addition and subtraction require identical valid currencies;
- different currencies fail with `core.ErrCurrencyMismatch`;
- signed overflow or underflow fails with both `core.ErrCurrencyOverflow` and
  `core.ErrNumericOverflow`;
- multiplication accepts an unsigned dimensionless quantity and detects the
  complete positive and negative signed range, including `math.MinInt64`;
- multiplication by zero produces valid zero money in the receiver currency;
- no operation silently saturates, wraps, converts currency, or rounds.

Division, percentage arithmetic, allocation, tax, exchange, and rounding are
not admitted. They require consumer-owned policy and cannot be made correct by
a universal default.

## 6. Decimal parsing and formatting

`Parse` consumes a bounded ASCII decimal amount in the supplied currency and
constructs exact minor units without floating point or intermediate
allocation proportional to the numeric value.

Accepted grammar:

- an optional leading `-`;
- one or more ASCII whole digits;
- for a currency with a positive exponent, an optional decimal point followed
  by one through exactly that many ASCII fractional digits;
- leading whole-number zeroes, which are explicitly canonicalized away;
- omitted fractional digits, which are exactly zero;
- a short fraction, which is padded on the right to the currency exponent.

Rejected grammar:

- empty input or input above the package byte ceiling;
- `+`, whitespace, exponent notation, separators, multiple decimal points,
  non-ASCII digits, NUL, or any other byte;
- a missing whole part or a decimal point without following digits;
- any fraction for a zero-exponent currency;
- more fractional digits than the currency exponent;
- a value outside signed 64-bit minor units;
- negative zero in any spelling.

`Decimal` returns a numeric decimal string with exactly the currency's exponent
digits, including trailing zeros. It never includes a symbol, currency token,
grouping separator, locale rule, or scientific notation. The caller already
has the typed currency through `Code()`.

Parsing and formatting are exact inverses for every `int64` minor-unit value.

## 7. Humanization

Humanization is a typed presentation projection. It never changes arithmetic,
JSON, or persistence truth.

`HumanizePolicy` contains:

- `Unit`: automatic, minor, major, hundreds, thousands, millions, or billions;
- `FractionDigits`: an exact trailing-decimal count from zero through six.

`Humanized` has private representation and exposes its validated `Code`,
`Number`, and `Unit`. `Number` is signed ASCII decimal with exactly the
requested trailing digits. `Unit` is a closed enum. Consumers decide locale,
symbols, pluralization, and surrounding prose.

All named units except minor are powers of ten in major currency units:

- major = `10^currencyExponent` minor units;
- hundreds = `100` major units;
- thousands = `1,000` major units;
- millions = `1,000,000` major units;
- billions = `1,000,000,000` major units.

Automatic selection uses absolute magnitude:

- zero uses major;
- a non-zero value below one major unit uses minor;
- one through ninety-nine major units use major;
- one hundred through nine hundred ninety-nine use hundreds;
- one thousand through nine hundred ninety-nine thousand use thousands;
- one million through nine hundred ninety-nine million use millions;
- one billion or more uses billions.

Minor-unit output is always an integer and rejects a non-zero
`FractionDigits`. Other projections truncate toward zero at the requested
precision; they never round, use floating point, or silently saturate.
When a negative magnitude truncates below the first visible digit, the result
is canonical signless zero rather than negative zero. The sign is retained at
and above the first visible digit.
Formatting a signed minimum value is required to work without negation
overflow. The same policy produces identical structural results on every
platform.

## 8. Canonical JSON

`Amount` emits one closed object:

```json
{"currency":"CAD","minor_units":"1234"}
```

`minor_units` is a canonical signed base-10 JSON string, not a JSON number.
This preserves the entire signed 64-bit domain in JavaScript and other
IEEE-754 JSON consumers.

The decoder rejects unknown, missing, duplicated, case-variant, or additional
fields; non-string values; unknown or non-canonical currency tokens;
non-canonical minor-unit strings; trailing JSON; invalid UTF-8; `null`; arrays;
and input above `core.StrictJSONMaxBytes`.

Canonical minor-unit strings use `"0"` for zero, contain no `+`, whitespace,
fraction, exponent, escape-derived digit, or leading zero, and reject `"-0"`.
Object input field order is not semantic; successful decode always re-encodes
in the canonical order above.

`Code` JSON is its canonical uppercase token. `Order` JSON is its
canonical lowercase token. Failed unmarshal leaves the receiver unchanged.

## 9. Firestore and PostgreSQL projections

Foundation imports no database driver.

`FirestoreAmount` and `PostgreSQLAmount` are distinct open DTO structs with:

- `Currency string`;
- `MinorUnits int64`.

The currency field must be the exact canonical uppercase token. Minor units
are authoritative and use each database's signed 64-bit integer. There is no
floating point, decimal database rounding, duplicated exponent, or derived
display amount.

`Amount.Firestore` and `Amount.PostgreSQL` validate the source and emit a DTO.
`FromFirestore` and `FromPostgreSQL` validate the mutable DTO exactly once and
construct a closed `Amount`. Live adapter smoke tests remain consumer
obligations because Foundation does not import Firestore or PostgreSQL drivers.

## 10. Error identity

Every package contract failure preserves `core.ErrCurrencyContract`, which wraps
`core.ErrFoundationContract`.

Stable specializations are:

- `core.ErrCurrencyMismatch`;
- `core.ErrCurrencyOverflow`;
- `core.ErrCurrencyDecimal`.

Overflow also preserves `core.ErrNumericOverflow`. Decimal grammar or range
failure also preserves `core.ErrInvalidDecimal`. Diagnostics add local
operation and value context without becoming caller contracts.

Tests use `errors.Is`; no caller or test matches error strings.

## 11. Complexity and platform contract

All operations use O(1) memory and O(n) time only in the bounded input-string
length. Arithmetic, formatting, JSON projection, and persistence projection
have fixed storage. The package performs no I/O, filesystem access, network
access, clock observation, environment access, goroutine creation, or
platform syscall.

The implementation and public behavior are identical on macOS, Linux, and
Windows. Cross-build gates prove Linux and Windows compilation.

## 12. Proof obligations

Hostile tables MUST prove:

- every valid currency and every exponent mapping;
- unknown, gap, future, maximum-underlying, empty, lowercase, mixed-case,
  padded, Unicode-confusable, escaped, and non-string currency input;
- positive maximum, maximum plus one, negative minimum, minimum minus one,
  zero, negative zero, exponent zero/two/three/four, short fraction, exact
  fraction, overlong fraction, leading zeros, NUL, whitespace, exponent,
  separators, and non-ASCII digits;
- add, subtract, and multiply at zero, one, signed extrema, one inside, exact
  boundary, and one beyond both overflow boundaries;
- every cross-currency arithmetic and comparison rejection preserves the
  mismatch identity and returns an invalid/zero result;
- exact decimal round trips across representative and signed-extreme values;
- every humanization unit, automatic threshold at one below/exact/one above,
  every exponent class, signed extrema, zero, negative values, the first
  visible negative digit, negative values that truncate to signless zero,
  exact trailing digits, truncation, and invalid policy states;
- strict JSON positive, negative, and neutral shapes, duplicate fields,
  unknown fields, wrong types, malformed structure, trailing values,
  oversized input, canonical second encoding, and receiver non-mutation;
- Firestore and PostgreSQL projection round trips, hostile currency tokens,
  signed extrema, mutable DTO corruption, and source non-mutation;
- exact public API and production-import ratchets;
- no product, sibling, database-driver, reflection, floating-point, or
  compatibility dependency.

Fuzzing targets the decimal and JSON ingress boundaries. Every accepted input
must validate and round-trip canonically; every rejected input must preserve a
typed currency error and leave its receiver unchanged. “Did not panic” is not an
oracle.

## 13. Completion gate

The package is complete only when:

- this specification and the exact public API ratchet agree;
- the hostile proof obligations pass under ordinary and race execution;
- fuzz callbacks own semantic round-trip and error-identity oracles;
- `go fix`, build, vet, field alignment, cyclomatic complexity, constant,
  nilness, error, static, dead-code, vulnerability, security, and
  `witness-lint` gates are clean for the package and affected `core`;
- Linux and Windows cross-builds pass;
- the pending and completed ledgers state the exact evidence;
- the user reviews and approves the implementation before commit.
