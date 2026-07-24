# Foundation Specification

Status: Draft for review
Target systems: macOS, Linux, and Windows
Normative terms: MUST, MUST NOT, SHOULD, and MAY

## 1. Purpose

Foundation is Off Grid Software's private library of compiler-owned,
product-neutral primitives.

Foundation solves difficult, reusable mechanics once. A consuming product owns
its business requirements and supplies typed requests. Foundation validates the
request, performs the bounded operation, and returns a typed result whose error
identity remains visible through `errors.Is` and `errors.As`.

Foundation is not:

- a product framework;
- a service locator;
- a global runtime;
- a repository for convenient helpers;
- a compatibility layer;
- an owner of any product, distribution, or control-plane policy;
- a home for product wire models merely because more than one process uses
  them.

## 2. Socket and plug model

Foundation packages are standardized sockets. Consumers provide typed plugs.

| Foundation owns | Consumer owns |
| --- | --- |
| I/O mechanics and operating-system differences | Product payloads and business rules |
| Validation of primitive invariants | Product-specific `Validate` rules |
| Bounded memory, time, retries, and cardinality | The selected budgets |
| Stable primitive error identities | Local context wrapped around those identities |
| Atomic durability and recovery state | What the persisted data means |
| HTTP transmission and reception mechanics | Routes, authorization, and domain actions |
| Signal observation and bounded cleanup execution | Cleanup steps and their ordering |

Foundation MUST NOT import a consumer to learn what a request means. The
consumer MUST NOT reproduce Foundation mechanics with copied constants,
private wrappers, or informal conventions.

### 2.1 Consumer invisibility

Dependencies point from a consumer to the capability it uses. They never point
from a reusable capability toward a caller.

`core` is the sole shared-contract marketplace. Every production Foundation
package may import `core`; sibling Foundation packages MUST NOT import one
another. When two packages need the same genuinely universal fact, that fact
moves to a focused `core` file as a typed value, enum, error identity, constant,
or validation rule. `core` MUST NOT become a dumping ground for package-local
mechanics or product meaning.

Capability composition occurs at the outer composition root through narrow
interfaces and explicit adapters. One sibling does not import another merely
because both participate in a workflow. A Foundation package MUST NOT:

- import a sibling Foundation package;
- import or name a consumer module;
- branch on caller, product, distribution, tenant, route, or deployment;
- own a consumer payload, policy, endpoint, schedule, or business error;
- accept a caller-kind enum or registration hook that recreates caller
  awareness indirectly;
- copy a consumer constant to avoid an import.

The same rule applies recursively above Foundation: a reusable framework does
not know its distributions, and a distribution does not teach lower layers
about its products. Every layer may use the socket below it but may not teach
that socket who is plugged in. “May use, must not know” is a
dependency-direction contract, not merely an import-cycle rule.

## 3. Primitive families

Foundation's target primitive families are:

| Family | Stable package name | Responsibility |
| --- | --- | --- |
| Shared contracts | `core` | Paths, HTTP semantics, identifiers, hashes, schemas, canonical encoding, and stable errors |
| Time | `temporal` | Instants, durations, aggregate nanoseconds, humanization, and persistence projections |
| Units | `units` | Byte extents, byte limits, money, and checked unit arithmetic |
| Context state | `contextstate` | Context ingress and closed cancellation/deadline classification |
| Transmission | `exchange` | Typed JSON, bounded-body, and streaming HTTP transmit/receive |
| Durability | `durability` | Rooted directories, bounded reads and writes, staging, atomic activation, synchronization, append, and removal |
| Object storage | `objectstore` | Provider-neutral signed upload/download, integrity, and O(1) object transfer |
| Host resources | `hostresource` | Disk, memory, tree-use, and runtime-OOM assessment |
| Graceful lifecycle | `shutdown` | Cleanup plans, signals, grace periods, escalation, panic containment, and reports |
| Key generation | `keygen` | CSPRNG-backed construction of typed key and secret contracts |
| Garble | `garble` | Custody seeds, deterministic release-seed derivation, and typed Garble inputs |
| Conformance | `probe` | Explicit bounded proof of selected primitive behavior |

The remaining admitted packages are narrowly scoped product-neutral support
packages. Product protocols live with the product that owns their meaning and
import Foundation primitives.

`temporal`, `units`, `objectstore`, `garble`, and `probe` are target packages.
Their current mechanics are present but scattered across existing packages.
They MUST be specified before code moves. Product-owned contracts encountered
during consolidation move out of Foundation rather than becoming new
Foundation packages. No temporary façade or alias package will bridge either
migration.

`contextstate` is the intended clean-cut replacement for the current
`contextcheck` name. The package owns both validation and classification, so a
state noun is more accurate than an action name. No alias package will be
retained.

The target names are intentional:

- an exchange transmits and receives;
- durability describes the promised outcome, not merely a write syscall;
- objectstore precisely names GCS/S3-style object transfer rather than generic
  application storage;
- host resources are observed without owning product policy;
- shutdown is the precise lifecycle operation; “graceful” is a quality of it;
- core is the universally shared deterministic vocabulary;
- temporal avoids colliding with Go's standard `time` package while naming the
  instant/duration domain;
- units groups non-temporal measured values without erasing their distinct
  types;
- keygen is the established, narrow Unix-style name for key generation;
- garble owns Garble-specific derivation rather than leaking it through generic
  key or release packages;
- probe is explicit conformance evidence, not monitoring.

## 4. Struct-first contract model

Important state MUST travel in typed structs. Multi-parameter operations SHOULD
accept one request struct and return one result struct.

A contract-owning struct MUST:

1. make every meaningful input compiler-visible;
2. represent closed states with typed enums;
3. own a `Validate` method for its local invariants;
4. reject unknown and contradictory states;
5. validate a temporary before mutating a receiver during decoding;
6. retain enough state to report partial success or exact recovery work;
7. avoid optional fields that combine unrelated operations into one request.

Sanitization MUST mean bounded structural rejection or an explicitly owned
canonicalization. Foundation MUST NOT silently repair hostile input or alter
evidence bytes.

Validation occurs once where untrusted, untyped, externally mutable, or
operating-system state becomes a closed typed value. Those closure boundaries
include ingress, decode, persistence input, execution input, and external
output only when that boundary receives a value whose validity has not already
been proven.

A value with private representation that was produced by its owning
constructor or strict validated decoder is proof-carrying. While it remains
immutable, a package receiving it MUST NOT revalidate it. Package crossing by
itself is not a reason to validate again.

An open request or DTO with exported mutable fields is not proof-carrying. Its
owner validates it exactly once immediately before projecting it into closed
values or executing the owned operation. It MUST NOT be passed through a chain
of packages that each defensively repeats the same validation.

Mutation, external decode, persistence round-trip, or a new operating-system
observation ends the prior proof and requires validation at the new closure
boundary. Repeated validation without one of those proof-ending events is a
design defect.

## 5. Interface discipline and injection

Foundation follows: accept interfaces, return concrete structs, and define
interfaces at the consumer.

A consumer MAY define the smallest interface it needs from a concrete
Foundation capability. Foundation MUST NOT publish a broad interface merely to
hide its implementation. A constructor MUST return a concrete validated type.

Nondeterministic dependencies—clock observation, waiting, entropy, network
transport, filesystem probes, and process signals—MUST enter through explicit
construction or request boundaries. Pure decision logic receives their
observations as typed data.

This permits deterministic fault injection without replacing product policy or
creating a global dependency container.

## 6. Stable intent surface

Foundation exists to stop mechanical implementation changes from propagating
through its consumers.

Each primitive package MUST expose the smallest stable vocabulary of
intent-oriented operations that preserves its real lifecycle contracts. The
public API describes what the consumer needs done. Package-private and
`internal` code owns how it is done.

A package MAY replace retry algorithms, buffer management, filesystem syscalls,
platform probes, serialization machinery, synchronization, or fault
classification without changing consumers when the public semantic contract
has not changed.

Public-surface rules:

- prefer a request struct and result struct over positional arguments;
- prefer a small number of precise verbs;
- return concrete validated capabilities;
- let consumers define narrow interfaces over those capabilities;
- keep implementation interfaces private;
- keep platform implementations private;
- do not export a helper solely because tests need it;
- do not expose intermediate machinery unless the caller must retain it to
  recover safely from a partial operation;
- do not collapse semantically different operations into an optional-field
  request or runtime type switch merely to reduce the exported symbol count;
- do not preserve an obsolete symbol with an alias or forwarding wrapper.

API size is an engineering budget. Every exported identifier requires a
cross-product use case, a stable semantic contract, hostile tests, and a reason
that the behavior cannot remain internal.

The intended review vocabulary is:

| Concern | Stable consumer intent |
| --- | --- |
| Time | Construct, validate, compare, humanize, and project typed nanosecond values |
| Units | Construct, validate, and perform checked arithmetic on typed measurements |
| Context | Validate ingress and classify terminal state |
| Transmission | Transmit and receive typed JSON, bounded bytes, or streams |
| Durability | Ensure, read, write, append, and remove through explicit lifecycle contracts |
| Object storage | Upload and download verified objects through bounded streams |
| Host resources | Assess disk or memory and measure bounded tree use |
| Shutdown | Watch signals and run a bounded cleanup plan |
| Key generation | Generate one explicitly requested typed secret or key |
| Garble | Derive typed deterministic build inputs from typed custody material |
| Probe | Prove a selected primitive through a bounded typed report |

The package-by-package specifications decide the exact exported names. No
renaming or façade is approved merely by this table.

## 7. Time model

Foundation separates an instant from a duration:

- `temporal.Instant` is a set, non-negative UTC Unix instant stored in
  nanoseconds.
- `temporal.Duration` is a non-negative elapsed or budget duration stored in
  nanoseconds.
- `temporal.AggregateDuration` is an unsigned 128-bit nanosecond accumulator
  for fleet-scale effort that cannot fit in `int64`.

They MUST NOT be interchanged or represented as an untyped integer.

`AggregateDuration` has private representation. Its canonical JSON form is a
JSON string containing unsigned base-10 nanoseconds with no sign, whitespace,
fraction, exponent, or leading zero except the single value `"0"`. It never
emits numeric high/low limbs. This preserves all 128 bits in Go, JavaScript, and
other JSON consumers without mutable representation leakage or IEEE-754 loss.

Wall-clock time is an observation, not an ordering authority across machines.
An outer shell observes the current time once and injects the resulting
`temporal.Instant`. Pure logic MUST NOT call `time.Now`.

Elapsed work MUST be measured using Go's monotonic clock while it remains
attached to `time.Time`, then converted to `temporal.Duration`. Serialized
`temporal.Instant` values do not retain monotonic-clock data and MUST NOT be
used to measure elapsed execution.

Persistence and wire representations use UTC Unix nanoseconds. Human time zones
and formatting belong at presentation boundaries.

Waiting MUST be context-cancellable and timer-backed. Tests MUST inject time
observations or waiting behavior; they MUST NOT prove behavior using arbitrary
sleep intervals.

### 7.1 Humanization

Humanization is a projection, never stored truth.

Both `temporal.Duration` and `temporal.AggregateDuration` own a `Humanize`
method. Each accepts the same typed precision and style policy and covers its
entire range using integer arithmetic. For every value representable by both
types, both methods MUST return structurally identical presentation values.
Neither method may use floating-point arithmetic.

Instant humanization requires an explicit location and presentation policy.
Foundation MUST NOT consult the machine's implicit local time zone.

Idiomatic Go uses the value as the capability:

```go
human, err := duration.Humanize(policy)
```

Foundation MUST NOT expose a mutable package-global `Temporal` service object.

### 7.2 Persistence projection

Firestore and PostgreSQL are external representation boundaries:

- an instant projects to a typed struct containing authoritative signed
  nanoseconds and a derived UTC timestamp truncated to the database's
  microsecond query resolution;
- an ordinary duration projects to a checked signed nanosecond value;
- an aggregate duration projects to a dedicated struct containing its
  canonical unsigned decimal nanosecond string, preserving all 128 bits
  without `uint64`, floating point, or lossy downcast.

Every projection has an inverse constructor and hostile round-trip tests.
Exact nanoseconds are the sole reconstruction and equality source. A native
timestamp is query/display convenience only. Firestore or PostgreSQL shape is
never the in-memory domain type, and database limitations MUST NOT weaken the
nanosecond source of truth.

## 8. Units model

Units are distinct types even when they share an integer representation.

- `units.ByteCount` is a required positive allocation or limit.
- `units.ByteLength` is an exact non-negative extent; zero is valid.
- `units.Money` binds exact minor units to a compiler-owned currency.

Unit arithmetic MUST detect overflow and underflow. Conversion to signed
standard-library sizes MUST be checked.

The current `core.MoneyPennies` does not identify a currency and is therefore a
migration source, not the final universal money contract. Floating-point values
MUST NOT represent stored or transmitted money.

`units` owns the closed `Currency` enum because the domain exists to define
`units.Money`. Each supported currency owns its ISO token and minor-unit
exponent. `Money` stores private currency and minor-unit state.

Arithmetic between different currencies MUST fail with a stable typed currency
mismatch identity. Construction, arithmetic, humanization, JSON, and
persistence projection use the currency-owned exponent; no caller supplies or
infers it. Adding a currency is a compiler-visible domain change with exhaustive
tests.

## 9. Transmission model

Foundation uses Go's standard HTTP stack. It does not replace TCP, TLS, HTTP, or
the Internet.

`exchange` adds typed structure around them:

- strict validated JSON requests and responses;
- bounded request and response bodies;
- O(1)-memory streaming upload and download;
- typed route semantics and replay safety;
- explicit attempt, body, retry, redirect, and backoff policies;
- context cancellation and attempt deadlines;
- retryable-status and `Retry-After` handling;
- typed attribution of request, response, status, cancellation, body-limit,
  and retry-exhaustion failures.

Retries MUST be permitted only by compiler-owned replay semantics. Retrying an
operation whose outcome is unknown requires an idempotency contract. Product
routes and payloads remain consumer-owned.

## 10. Object-storage model

`objectstore` is the provider-neutral O(1) transfer contract. Consumers own why
an object exists; `objectstore` owns the typed operation and integrity proof.
It consumes a narrow locally owned streaming transport interface. The outer
composition root adapts an `exchange` capability to that interface;
`objectstore` does not import `exchange`.

The package owns typed provider, object name, signed upload/download target,
method, required headers, exact content length, expected digest, write
semantics, and transfer result.

Upload and download stream through bounded buffers. They MUST NOT load an
object into memory. A successful result proves expected status, exact byte
count, and digest. Signed URL expiry, retry/replay rules, early rejection,
provider-specific headers, and cancellation remain internal mechanics built
behind the injected transport contract.

Consumers own grants, receipts, manifests, and authorization. They MUST NOT
implement another HTTP object-transfer client.

## 11. Durability model

Durability is more than calling `Write`.

A durable replacement consists of bounded streaming into a same-filesystem
stage, file synchronization, atomic activation, containing-directory
synchronization, and exact cleanup or recovery reporting.

Foundation owns:

- containment beneath validated roots where the primitive promises it;
- regular-file and directory checks;
- partial writes and short writes;
- file synchronization and platform full-sync behavior;
- atomic create/replace activation;
- directory synchronization;
- resumable stage lifecycle state;
- bounded append, read, removal, and crash-residue cleanup.

Distinct lifecycle contracts MUST remain distinct. A one-shot write, resumable
stage, content-addressed stage, append stream, and recursive removal are not one
operation with optional fields.

Residue sweeping requires exclusive directory ownership. The sweep request MUST
carry or be executed beneath a capability proving that no live stage can exist
in the swept directory. Staging and sweeping the same directory concurrently is
a contract violation. A sweep MUST NOT attempt to distinguish live stages from
residue through age, naming guesses, or race-prone inspection.

Streaming operations MUST use bounded buffers and O(1) working memory.

## 12. Host-resource model

Host-resource primitives observe and classify; product policy decides what to
do.

Disk and memory contracts MUST use typed byte units and explicit thresholds.
Tree measurement MUST stream over entries, remain context-cancellable, reject
invalid roots, and fail loudly when a subtree cannot be measured.

Tests MUST simulate resource exhaustion deterministically:

- disk write exhaustion uses typed ENOSPC fault injection;
- disk-pressure decisions use injected capacities and thresholds;
- memory-pressure decisions use injected snapshots and limits;
- runtime-OOM classification uses bounded captured evidence.

Tests MUST NOT fill a real disk or deliberately exhaust the test process.
Production-equivalent failure points that cannot currently be injected MUST be
recorded as contract gaps.

## 13. Graceful lifecycle model

Shutdown is a bounded state machine, not a list of deferred functions.

The consumer supplies typed, named cleanup steps. Foundation owns:

- LIFO execution;
- per-step and total budgets;
- continuation after an individual failure;
- panic containment with bounded diagnostics;
- typed results for success, failure, timeout, and panic;
- operating-system signal observation;
- first-signal graceful cancellation;
- second-signal and grace-expiry policies;
- restoration of operating-system defaults before forced action;
- an immutable final report.

The application remains responsible for stopping admission, cancelling owned
workers, joining every goroutine, then releasing persistence and process locks
in a provable order.

## 14. Complexity and ownership

Foundation follows Unix composition: one primitive does one job completely.

Operations over streams or filesystem trees MUST use O(1) working memory unless
the contract explicitly requires a bounded aggregate. Whole-world loading is
forbidden. Every collection, diagnostic, retry sequence, goroutine set, and
buffer has a compiler-visible bound.

Goroutines MUST have one owner, one cancellation path, and one join path.
Foundation MUST NOT start an unowned background goroutine.

Production functions MUST satisfy `gocyclo <= 10`.

## 15. Cross-platform contract

Supported systems are macOS, Linux, and Windows.

The public typed contract MUST remain the same on all three. Platform files MAY
implement different mechanisms:

- macOS durability prefers full filesystem synchronization and falls back only
  for explicitly unsupported operations;
- Linux uses the operating system's file and directory synchronization;
- Windows uses Windows directory handles and interrupt semantics;
- disk capacity uses the native volume API on each platform;
- signal sets expose only signals the platform can actually deliver.

Unsupported behavior MUST fail with a stable typed identity. It MUST NOT report
success merely because one operating system lacks another system's primitive.
Platform-specific tests and cross-build gates are required for every split.

## 16. Errors and observability

Stable error identity belongs in `core`. Packages wrap that identity with local
typed context. Callers and tests use `errors.Is` and `errors.As`; string
matching is forbidden.

Results SHOULD carry typed state needed for observability. Logs are diagnostic
output, not proof. Correlation, idempotency, byte counts, attempts, durations,
activation state, and cleanup state must remain typed wherever the operation
owns them.

No error may expose secrets or unbounded hostile input.

## 17. Testing contract

Every package follows [_docs/governance/testing_protocol.md](_docs/governance/testing_protocol.md).

Required proof includes:

- a red-proven behavioral failure before the production fix;
- hostile table tests at exact minimum, maximum, one-below, one-above, zero,
  nil, overflow, cancellation, and contradictory-state boundaries;
- positive, negative, and decision-smoke layer coverage;
- real production-path tests rather than decorative helper tests;
- exact typed error identity and structural result comparison;
- deterministic synchronization without timing sleeps;
- race tests for concurrent ownership;
- fuzz tests for parsers, canonical encoders, and state machines;
- platform-specific tests and macOS/Linux/Windows cross-builds;
- streaming and allocation proof where O(1) behavior is claimed.

Waivers are exceptional evidence documents. They are never an escape hatch.

## 18. Conformance probes

Foundation MUST provide the narrowly scoped `probe` package described here
before `v2026.0.0`. The package does not yet exist; this section approves its
purpose and makes its reviewed package specification and implementation a
release gate.

The probe package would consume explicitly injected narrow capabilities. The
outer composition root adapts the selected primitive implementations; `probe`
does not import sibling Foundation packages. It would not replace, duplicate,
or bypass them.

An exchange probe MUST:

- accept a consumer-owned typed endpoint and route contract;
- require an injected exchange capability for DNS, TCP, TLS, HTTP, retry, and
  response handling;
- validate correlation and the typed response;
- return bounded typed step results, attempts, bytes, duration, status, and
  stable error identities;
- avoid claiming that generic HTTP reachability proves a product workflow.

A durability probe MUST:

- require an explicit dedicated probe directory beneath a validated root;
- reject symlinks, broad roots, and directories containing unrelated data;
- perform bounded stage, stream-write, file-sync, atomic-activate,
  directory-sync, bounded-read, digest-verify, remove, and final-directory-sync
  steps;
- prove that its artifact and temporary state were removed;
- never fill a disk or modify consumer-owned content.

Probe execution MUST be explicit: an installation check, startup preflight,
`doctor` command, or smoke gate. The package MUST NOT own a daemon, global
health state, periodic goroutine, product route, or product policy.

Every report MUST be a validated struct with a closed probe kind, bounded step
set, UTC start instant, monotonic elapsed duration, final outcome, and typed
failure identity. Logs are supplemental diagnostics, not the report.

The package may be implemented only after its request and result structs, safe
root rules, and hostile failure matrix have been reviewed.

## 19. Release contract

Foundation is designed and completed in its own repository. Consumers do not
determine its API and are not compatibility constraints.

Tag `v2026.0.0` may be cut when:

1. the root and every package-local specification are approved;
2. every public export appears in its owning specification;
3. every pending clean-cut move is complete;
4. all primitive and protocol packages pass hostile and cross-platform gates;
5. no aliases, forwarding wrappers, compatibility packages, or generated
   artifacts remain;
6. `.ledger_completed.md` contains evidence for every release requirement.

After the tag, consumers adopt the stable contract and the compiler identifies
every stale use. Internal algorithms may change freely when the specified
behavior and public surface remain intact.

## 20. Evolution

Foundation uses clean upgrades. A contract change updates the real type and all
callers in one migration. No aliases, forwarding wrappers, copied constants, or
compatibility shims are permitted.

Historical tagged releases are the compatibility boundary for historical
artifacts. Current code represents one current contract.

A primitive enters Foundation only when:

1. at least two products need the same semantic contract, or a product and its
   verifier/issuer share it;
2. the invariant is independent of product policy;
3. the API is smaller than the duplicated implementations it replaces;
4. hostile tests prove the difficult failure paths;
5. macOS, Linux, and Windows behavior is explicit;
6. its memory, time, retry, and cardinality bounds are compiler-visible.

## 21. Package specifications

The package index is [_docs/specs/README.md](_docs/specs/README.md). Package
specifications refine this document. They cannot weaken it.
