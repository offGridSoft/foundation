# Core Package Specification

Status: Draft for review
Package: `github.com/offGridSoft/foundation/v2026/core`

## 1. Purpose

`core` owns Foundation's deterministic value vocabulary and invariants shared
across package boundaries. It contains typed units, closed enums, validated
identifiers, canonical representations, shared error identities, and protocol
facts that must have exactly one compiler-visible owner.

`core` is the value layer beneath Foundation's I/O primitives. It does not
observe the clock, access the filesystem, make network requests, read the
environment, start goroutines, or select product behavior.

## 2. Admission rule

A declaration belongs in `core` only when it is product-neutral and at least
two independent products need the same semantic fact. Use by two Foundation
packages is not sufficient. A fact serving one product and its control-plane
producer, verifier, or issuer belongs in that product's shared protocol package,
even when several Foundation packages consume it.

Import convenience never qualifies a declaration for `core`.

Convenient helpers, product orchestration, operating-system mechanisms, and
consumer policy MUST NOT enter `core`.

A shared fact MUST NOT be copied into another package. The caller imports
`core` and uses the owned type, enum, constant, validation rule, or error
identity directly.

## 3. Value-struct contract

Important scalar values SHOULD be structs with private representation and
explicit construction. This prevents raw integers and strings with different
meanings from being interchangeable.

A core value struct MUST:

- expose construction or parsing through a named function;
- own its `Validate` method;
- preserve the receiver when decoding hostile input fails;
- expose only conversions required at standard-library or external boundaries;
- reject arithmetic overflow and underflow;
- use canonical JSON where it crosses a wire or persistence boundary;
- keep its zero-value meaning explicit.

Closed domains use typed enums whose zero value is unknown or invalid unless
the domain explicitly defines zero as a meaningful state. Enum parsing,
validation, string conversion, and JSON behavior must agree exhaustively.

## 4. Ownership migrations

The current package contains several proven implementations whose final owner
is more precise:

- `UnixNanoTime` and `NanosecondsDuration` move to `temporal`;
- `ByteCount`, `ByteLength`, and `MoneyPennies` move or are replaced in
  `units`;
- Garble custody material moves to `garble`;
- storage provider and signed-transfer values move to `objectstore`;
- every product-specific contract moves out of Foundation to its owning
  product or control-plane module.

These are clean moves, not aliases. Their behavioral tests travel with their
new owners. Until migration, the current types are source implementations, not
approval for new core-owned contracts.

`core` MUST NOT export a global clock or perform clock, filesystem, network,
entropy, or environment observation.

## 5. Filesystem values

`AbsoluteDirectoryPath` and `AbsoluteFilePath` own lexical absolute-path
validation. Filesystem capability packages remain responsible for rooted
containment, symlink policy, object-kind checks, and race-resistant open
behavior.

A validated absolute path is not proof that a target remained beneath a root.
Callers MUST NOT infer containment from lexical validation alone.

File and directory mode constants belong in `core`; callers MUST NOT copy octal
permission literals.

## 6. HTTP values

HTTP behavior is represented with:

- `HTTPMethod`;
- `HTTPStatusCode`;
- `HTTPMediaType`;
- `HTTPReplaySafety`;
- `HTTPRequestSemantics`;
- `HTTPRouteSemantics`;
- `HTTPRetryPolicy`;
- `BackoffPolicy`;
- typed headers, queries, endpoints, redirects, and idempotency keys.

Method, body-bearing behavior, replay safety, retryability, expected status,
and idempotency are related compiler-owned facts. Callers MUST NOT reconstruct
their relationship using raw method strings or status integers.

`core` defines HTTP semantics; `exchange` performs HTTP I/O; consumers own
route catalogs and authorization. Neither lower package knows the consumer.

## 7. Strict JSON and canonical data

`Validatable` is the structural boundary shared by Foundation contracts.

`EncodeValidatedJSON` validates before output. `DecodeStrictJSON` and
`DecodeStrictJSONStructure` bound and strictly decode their input according to
their owned contract. Unknown fields, malformed scalar encodings, trailing
data, and invalid post-decode state are rejected.

Signed bodies implement compiler-owned canonical byte production. Canonical
encoding MUST have byte-exact tests. Map iteration, locale, wall clock,
whitespace, and reflection-dependent incidental ordering MUST NOT determine
signed bytes.

Foundation MUST NOT accept a loose `map[string]any` at a protocol boundary.

## 8. Integrity and identity

Digests, public keys, signatures, signing domains, schemas, platforms,
products, versions, and request IDs are typed values.

Parsing validates shape and canonical form. Cryptographic verification uses the
typed signing domain and authority contract. Raw key bytes and untyped schema
strings MUST NOT flow across package boundaries.

Entropy generation does not belong in deterministic `core`; `keygen` owns that
outer boundary.

## 9. Errors

Stable cross-package identities live in `core`. A package MAY wrap one with
local context but MUST preserve it through `%w`, `errors.Join`, or a typed
`Unwrap` method.

Error strings are diagnostics and may change. Callers and tests use
`errors.Is` and `errors.As`.

No error may contain a secret or unbounded hostile input.

## 10. Complexity

Core decisions MUST be deterministic. They MUST NOT depend on:

- wall-clock observation;
- environment variables;
- map iteration order;
- goroutine scheduling;
- random input not supplied as typed data;
- operating-system state.

Validation MUST be bounded. Collection contracts declare maximum cardinality.
Canonical encoding uses bounded output or caller-owned append buffers.

## 11. Cross-platform behavior

Core values have identical semantics on macOS, Linux, and Windows. Path
validation may recognize platform syntax through compiler-owned platform
contracts, but it MUST NOT silently reinterpret a value based on the machine
running the validator.

`Platform` and GOOS tokens are closed domains. Unsupported combinations fail
typed validation.

## 12. Required proof

Core changes require hostile tables covering:

- zero, unset, minimum, maximum, one-below, and one-above values;
- signed and unsigned overflow;
- invalid UTF-8 and control characters;
- malformed and non-canonical JSON;
- unknown enum values and exhaustive valid enum parity;
- receiver non-mutation after failed unmarshal;
- arithmetic boundary behavior;
- exact canonical bytes for signed bodies;
- structural equality after valid round trips;
- macOS, Linux, and Windows platform lattices where applicable.

Fuzz tests target parsers, JSON scalar decoders, canonical encoders, and closed
state machines. They must assert invariants and receiver safety, not merely
absence of panic.

## 13. Non-goals

`core` does not own:

- clock observation or timers;
- network clients or servers;
- filesystem access;
- process execution;
- signal handling;
- product routes;
- product business decisions;
- global configuration;
- compatibility aliases.
