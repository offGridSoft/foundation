# The Blink Go Doctrine: Spine of Steel

| Metadata      | Value                                            |
| ------------- | ------------------------------------------------ |
| **Status**    | Normative                                        |
| **Version**   | 1.0                                              |
| **Creator**   | Ase Deliri                                       |
| **Date**      | January 2026                                     |
| **Baselines** | Go 1.25; OS+Runtime; Stdlib+Approved             |
| **Target**    | 1,000,000 req/sec; p99 < 200ms; Zero Ghost State |

> **"What force am I aligning with here?"** **If the answer is "ourselves," the
> design is wrong.**

---

# 🏛️ Part I: Unbreakable Laws

**These laws govern how Blink Go is written, not what it does.** Code that
violates this section is rejected without discussion.

---

## 🏛️ 1.1 The Priority Stack

**Premise:** Priority order is fixed. Higher priorities override lower ones.
Faster wrong answers are strictly worse than slower correct ones.

**The Stack:**

| Rank | Priority               | Description                             |
| ---- | ---------------------- | --------------------------------------- |
| 1    | **Correctness**        | System obeys its contract for all cases |
| 2    | **Security**           | Hostile input cannot corrupt truth      |
| 3    | **Determinism**        | Same inputs produce same outputs        |
| 4    | **Boundedness**        | All resources have enforced limits      |
| 5    | **Tail Latency (p99)** | The tail defines user trust             |
| 6    | **Throughput**         | Sustained useful work per unit time     |
| 7    | **Convenience**        | Ease of writing, reading, modifying     |

**Axiom:** A performance change that weakens correctness is a defect.

---

## 🏎️ 1.2 Trust the Compiler

**Premise:** The Go compiler is the most informed participant in the system. It
understands escape analysis, inlining, SSA, register allocation, cache locality,
alias safety, and bounds check elimination.

**The Compiler Understands:**

- Escape analysis and stack allocation
- Function inlining and devirtualization
- Bounds check elimination
- SSA optimization passes
- Register allocation and spilling
- Dead code elimination

**If You Fight the Compiler:**

- Allocations escape to heap unexpectedly
- Inlining stops at boundaries
- Hot paths become opaque
- Performance becomes fragile and accidental
- Go upgrades make things slower

**If You Align:**

- Memory behavior becomes predictable
- Performance improves with new Go releases
- Profiles get simpler
- Optimizations compound automatically

**Axiom:** The compiler finishes the job you started.

---

## 🏛️ 1.3 Trust the Go Team

**Premise:** Every omission in Go is deliberate. The Go team encodes decades of
production failure into the language design.

**The Language Refuses:**

- Inheritance (composition instead)
- Implicit constructors (explicit initialization)
- Magic lifecycles (explicit resource management)
- Hidden async (goroutines are visible)
- Runtime polymorphism as default (interfaces are explicit)
- Exceptions (errors are values)
- Generics complexity (simple constraints only)

**If You Fight the Go Team:**

- You build fragile abstractions
- You reintroduce hidden control flow
- You confuse readers and tools
- Upgrades become painful
- Debugging becomes archaeology

**If You Align:**

- Code is boring and predictable
- New team members ramp up fast
- Refactors are mechanical
- Upgrades are gifts

**Axiom:** Boring is success. Clever is debt.

---

## 🏛️ 1.4 Trust the Operating System

**Premise:** Operating systems solved scheduling, memory management, networking,
file systems, timers, and synchronization decades ago. Go cooperates with the
OS; it does not replace it.

**The OS Provides:**

- Preemptive scheduling
- Virtual memory and paging
- File system caching
- Network stack optimization
- Timer precision
- Process isolation

**If You Fight the OS:**

- User-space schedulers break under load
- Memory management competes with GC
- Network hacks bypass kernel optimizations
- Latency becomes unpredictable
- Bugs appear only in production
- Security assumptions rot

**If You Align:**

- Kernel improvements help you automatically
- File system caches work in your favor
- Network stack handles edge cases
- Timers are precise enough

**Axiom:** Stand on accumulated engineering, not ego.

---

## 🏛️ 1.5 Trust the Network (To Be Hostile)

**Premise:** The network is unreliable by definition. Packets drop, reorder,
duplicate. Connections reset. Latency varies by orders of magnitude. This is
physics, not a defect.

**The Network Will:**

- Delay packets arbitrarily
- Drop packets silently
- Duplicate packets
- Reorder packets
- Reset connections mid-stream
- Partition completely
- Vary latency by 1000x

**If You Fight the Network:**

- Retries amplify outages
- Partial failures corrupt state
- Data lies silently
- Systems collapse under stress
- Race conditions hide in timing

**If You Align:**

- Idempotency makes retries safe
- Timeouts bound uncertainty
- Circuit breakers prevent cascade
- Proof survives partitions

**Axiom:** Truth survives hostile conditions through explicit failure handling.

---

## 🏛️ 1.6 Trust Types

**Premise:** Go types value honesty over expressiveness. They are memory
layouts, ownership declarations, contracts, and compile-time witnesses.

**Types Are:**

- Memory layouts (struct field ordering matters)
- Ownership declarations (pointer vs value)
- Contracts (method sets)
- Compile-time witnesses (interface satisfaction)
- Documentation that cannot lie

**If You Fight Types:**

- Invariants live in comments
- Errors move to runtime
- Refactors become dangerous
- Bugs survive reviews
- `interface{}` spreads like infection

**If You Align:**

- Compiler catches mistakes
- Refactors are mechanical
- Documentation stays current
- Tests focus on behavior, not types

**Axiom:** Clarity becomes structural when types are honest.

---

## 🏎️ 1.7 Trust the Margin

**Premise:** Marginal gains are the arbitrage of excellence. At 1,000,000
requests per second, 10ns per request equals 10 milliseconds of CPU per second.
Small costs compound into systemic drag.

**The Math of Scale:**

| Per-Request Cost | At 1M req/sec                |
| ---------------- | ---------------------------- |
| 1 allocation     | 1,000,000 allocations/sec GC |
| 1 microsecond    | 1 full CPU core consumed     |
| 1 cache miss     | Millions of memory stalls    |
| 1 KB response    | 1 GB/sec network egress      |
| 1 log line       | 1,000,000 log lines/sec      |

**If You Fight the Margin:**

- "Death by a thousand cuts" becomes acceptable
- Performance debt accumulates invisibly
- "Good enough" becomes the ceiling
- Tail latency grows without explanation

**If You Align:**

- Small improvements multiply
- Headroom accumulates
- Capacity grows without hardware

**Axiom:** We do it well, or we do not do it.

---

## 🏛️ 1.8 Trust Data Shapes

**Premise:** Structs with consistent shapes are contracts. The compiler and
runtime reward shape stability with optimization. Inconsistent shapes create
chaos.

**If You Fight Shapes:**

- `interface{}` everywhere
- Runtime type assertions
- Reflection in hot paths
- JSON as internal format
- Maps where structs belong

**If You Align:**

- Compile-time type checking
- Predictable memory layout
- Cache-friendly access patterns
- Clear ownership boundaries

**Axiom:** Stable shapes are the first optimization.

---

# 🏗️ Part II: Code Form Rules

**These rules govern how Go is written, not what it does.** Reviewers scan for
violations in ten seconds.

---

## 🛑 2.1 No Unchecked Errors

**Rule:** Every error returned must be checked or explicitly ignored with a
comment explaining why failure is safe.

**Forbidden:**

```go
// BAD: error ignored
doSomething()

// BAD: no explanation
_ = doSomething()
```

**Required:**

```go
// GOOD: error handled
if err := doSomething(); err != nil {
    return fmt.Errorf("doing something: %w", err)
}

// GOOD: explicit ignore with reason
_ = conn.Close() // Safe: already handling primary error
```

**Axiom:** Silent failure is a correctness defect.

---

## 🛑 2.2 No Global Mutable State

**Rule:** Package-level variables must be immutable after initialization.

**Forbidden:**

```go
// BAD: mutable at runtime
var Config = loadConfig()
var cache = make(map[string]Value)

func UpdateConfig(c Config) { Config = c } // BAD
```

**Required:**

```go
// GOOD: const for true constants
const maxRetries = 3

// GOOD: sync.Once for singleton initialization
var (
    clientOnce sync.Once
    client     *Client
)

func getClient() *Client {
    clientOnce.Do(func() {
        client = newClient()
    })
    return client
}

// GOOD: dependency injection
type Service struct {
    config Config
    cache  Cache
}
```

**Axiom:** Global mutation creates ghost state.

---

## 🏗️ 2.3 Flat Control Flow

**Rule:** Keep indentation shallow. Handle errors first. Happy path on the left
edge.

**Forbidden:**

```go
// BAD: deep nesting
func process(u *User) error {
    if u != nil {
        if u.Active {
            if u.HasPermission("write") {
                // Finally do something
            }
        }
    }
    return nil
}
```

**Required:**

```go
// GOOD: guard clauses
func process(u *User) error {
    if u == nil {
        return ErrNilUser
    }
    if !u.Active {
        return ErrInactiveUser
    }
    if !u.HasPermission("write") {
        return ErrNoPermission
    }

    // Happy path at left edge
    return doWork(u)
}
```

**Axiom:** If you cannot see the exit, you cannot prove correctness.

---

## 🏗️ 2.4 Explicit Dependencies

**Rule:** Functions and structs must ask for what they need. No reaching into
global state.

**Forbidden:**

```go
// BAD: reads global
func GetUser(id string) (*User, error) {
    return globalDB.Query(id) // Where did globalDB come from?
}

// BAD: init reaches out
func init() {
    db, _ = sql.Open("postgres", os.Getenv("DB_URL"))
}
```

**Required:**

```go
// GOOD: explicit dependency
type UserStore struct {
    db *sql.DB
}

func (s *UserStore) GetUser(ctx context.Context, id string) (*User, error) {
    return s.db.QueryRowContext(ctx, query, id)
}

// GOOD: constructor takes dependencies
func NewUserStore(db *sql.DB) *UserStore {
    return &UserStore{db: db}
}
```

**Axiom:** Hidden dependencies create hidden coupling.

---

## 🏗️ 2.5 Context First

**Rule:** If a function performs I/O or can block, `context.Context` is the
first argument. No exceptions.

**Forbidden:**

```go
// BAD: no context
func FetchUser(id string) (*User, error)

// BAD: context not first
func FetchUser(id string, ctx context.Context) (*User, error)

// BAD: context stored in struct
type Client struct {
    ctx context.Context // Never do this
}

// BAD: nil context
FetchUser(nil, id)
```

**Required:**

```go
// GOOD: context first
func FetchUser(ctx context.Context, id string) (*User, error) {
    if ctx == nil {
        return nil, errors.New("nil context")
    }
    // ...
}
```

**Axiom:** Context is the spine. You cannot add a spine later.

---

## 🛑 2.6 No Panics in Libraries

**Rule:** Library code must not panic for expected conditions.

**Forbidden:**

```go
// BAD: panic for expected error
func ParseConfig(data []byte) Config {
    var c Config
    if err := json.Unmarshal(data, &c); err != nil {
        panic(err) // User input could cause this
    }
    return c
}
```

**Allowed:**

```go
// OK: panic for programmer error at init time
func MustCompileRegex(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

var emailRegex = MustCompileRegex(`^[^@]+@[^@]+$`) // Init time only
```

**Required:**

```go
// GOOD: return error
func ParseConfig(data []byte) (Config, error) {
    var c Config
    if err := json.Unmarshal(data, &c); err != nil {
        return Config{}, fmt.Errorf("parse config: %w", err)
    }
    return c, nil
}
```

**Axiom:** A panic that escapes a boundary is a correctness failure.

---

## 🛑 2.7 No Magic Values

**Rule:** No naked numbers or strings in logic. Extract to named constants.

**Forbidden:**

```go
// BAD: magic numbers
if retries > 3 {
    return errors.New("too many retries")
}
time.Sleep(5 * time.Second)
if len(name) > 255 {
    return errors.New("name too long")
}
```

**Required:**

```go
// GOOD: named constants
const (
    maxRetries     = 3
    defaultTimeout = 5 * time.Second
    maxNameLength  = 255
)

if retries > maxRetries {
    return ErrTooManyRetries
}
time.Sleep(defaultTimeout)
if len(name) > maxNameLength {
    return ErrNameTooLong
}
```

**Axiom:** Magic values become magic bugs.

---

## 🛑 2.8 No init Magic

**Rule:** Avoid `init()` functions for logic. Use explicit initialization.

**Forbidden:**

```go
// BAD: side effects at import time
func init() {
    db, _ = sql.Open("postgres", os.Getenv("DB_URL"))
    http.HandleFunc("/", handler)
    go backgroundWorker()
}
```

**Required:**

```go
// GOOD: explicit initialization
func main() {
    cfg := loadConfig()
    db := mustConnectDB(cfg.DatabaseURL)
    defer db.Close()

    handler := newHandler(db)
    http.HandleFunc("/", handler.ServeHTTP)

    // ...
}
```

**Allowed in init:**

```go
// OK: registration that cannot fail
func init() {
    prometheus.MustRegister(requestCounter)
}

// OK: compile-time interface check
var _ io.Reader = (*MyReader)(nil)
```

**Axiom:** Explicit initialization is provable initialization.

---

## 🏗️ 2.9 Interface Discipline

**Rule:** Accept interfaces, return structs. Define interfaces at the consumer,
not the producer.

**Forbidden:**

```go
// BAD: returning interface
func NewUserStore() UserRepository {
    return &userStore{}
}

// BAD: large interface defined by producer
type UserRepository interface {
    Get(ctx context.Context, id string) (*User, error)
    List(ctx context.Context) ([]*User, error)
    Create(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id string) error
    // 20 more methods...
}
```

**Required:**

```go
// GOOD: return concrete type
func NewUserStore(db *sql.DB) *UserStore {
    return &UserStore{db: db}
}

// GOOD: small interface at consumer
// In service package:
type UserGetter interface {
    Get(ctx context.Context, id string) (*User, error)
}

func NewService(users UserGetter) *Service {
    return &Service{users: users}
}
```

**Axiom:** Interfaces describe what you need, not what you offer.

---

## 🏗️ 2.10 Defer Close Immediately

**Rule:** `defer Close()` must happen immediately after successful acquire.

**Forbidden:**

```go
// BAD: defer not immediately after acquire
f, err := os.Open(path)
if err != nil {
    return err
}
data, err := io.ReadAll(f)
if err != nil {
    return err // Leaked file handle!
}
defer f.Close()
```

**Required:**

```go
// GOOD: defer immediately after acquire
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()

data, err := io.ReadAll(f)
if err != nil {
    return err
}
```

**For multiple resources:**

```go
// GOOD: each defer follows its acquire
f1, err := os.Open(path1)
if err != nil {
    return err
}
defer f1.Close()

f2, err := os.Open(path2)
if err != nil {
    return err
}
defer f2.Close()
```

**Axiom:** Resource leaks are correctness failures.

---

## 🏎️ 2.11 No fmt in Hot Paths

**Rule:** Avoid `fmt.Sprintf` and related functions in high-throughput code.

**Forbidden in hot paths:**

```go
// BAD: allocates and is slow
func formatKey(prefix, id string) string {
    return fmt.Sprintf("%s:%s", prefix, id)
}
```

**Required:**

```go
// GOOD: direct concatenation (single allocation)
func formatKey(prefix, id string) string {
    return prefix + ":" + id
}

// GOOD: strings.Builder for complex cases
func formatKeys(prefix string, ids []string) string {
    var b strings.Builder
    b.Grow(len(prefix)*len(ids) + len(ids)*20) // Pre-size
    for i, id := range ids {
        if i > 0 {
            b.WriteByte(',')
        }
        b.WriteString(prefix)
        b.WriteByte(':')
        b.WriteString(id)
    }
    return b.String()
}

// GOOD: pre-computed strings
var statusStrings = map[Status]string{
    StatusPending: "pending",
    StatusActive:  "active",
    StatusDone:    "done",
}
```

**Axiom:** Formatting in hot paths widens the tail.

---

## 🏗️ 2.12 Named Returns Discipline

**Rule:** Named return values are allowed only for documentation or deferred
error handling. Never use bare returns in production code.

**Forbidden:**

```go
// BAD: bare return hides what's returned
func process(data []byte) (result []byte, err error) {
    // ... 50 lines of code ...
    return // What does this return?
}
```

**Allowed:**

```go
// OK: named for deferred error augmentation
func process(data []byte) (result []byte, err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("process: %w", err)
        }
    }()

    result = transform(data)
    return result, nil // Explicit return
}

// OK: named for documentation
func Split(s string) (head, tail string) {
    // Names document the meaning
    return s[:1], s[1:]
}
```

**Axiom:** Implicit flow hides mistakes.

---

## 🏗️ 2.13 Import Grouping

**Rule:** Imports are grouped and ordered: stdlib, external, internal.

**Required:**

```go
import (
    // Standard library
    "context"
    "errors"
    "fmt"

    // External packages
    "github.com/google/uuid"
    "google.golang.org/grpc"

    // Internal packages
    "blink/internal/domain"
    "blink/internal/repository"
)
```

**Axiom:** Consistent imports reveal dependencies at a glance.

---

# 🏗️ Part III: Function Doctrine

**Functions are where correctness is proven.** If a function is unclear, the
system is unclear.

---

## 🏗️ 3.1 Function Shape

**Premise:** A function exists to preserve truth. It transforms inputs to
outputs with explicit ownership semantics.

**Rules:**

1. Each function does **one thing**
2. Inputs in, outputs out
3. **No hidden side effects**
4. Ownership is explicit in the signature
5. Any function that retains input **MUST** copy or document sharing

**Function Categories:**

| Category | Side Effects | Returns     | Example            |
| -------- | ------------ | ----------- | ------------------ |
| Pure     | None         | Value       | `hash(data)`       |
| Query    | None         | Value+Error | `store.Get(id)`    |
| Command  | Yes          | Error       | `store.Save(item)` |
| Factory  | Allocation   | Value       | `NewService(deps)` |

**Axiom:** Hidden ownership creates ghost state.

---

## 🏗️ 3.2 Function Signatures

**Premise:** Signatures are contracts. They must not lie.

**Rules:**

1. `context.Context` is the **first parameter** when cancellation applies
2. Return `(T, error)` for fallible operations
3. Return `T` only when failure is impossible
4. Avoid variadic parameters except for logging/formatting
5. Max **3-4 positional arguments** for core logic
6. If more arguments needed, use a **Request Struct**
7. Do not return interface values from constructors

**Good Signatures:**

```go
// Clear: context first, returns value and error
func (s *Store) Get(ctx context.Context, id string) (*Item, error)

// Clear: no context for pure computation
func Hash(data []byte) [32]byte

// Clear: options struct for many parameters
func NewClient(cfg ClientConfig) (*Client, error)
```

**Bad Signatures:**

```go
// BAD: context not first
func Get(id string, ctx context.Context) (*Item, error)

// BAD: too many positional args
func Query(ctx context.Context, table, col, val, order string, limit int)

// BAD: returns interface
func NewStore() Store
```

**Axiom:** If the signature lies, the function lies.

---

## 🏗️ 3.3 Ownership Questions

**Premise:** Every function must answer four ownership questions.

| Question       | Answer Must Be Explicit           |
| -------------- | --------------------------------- |
| Who allocates? | Caller or callee                  |
| Who mutates?   | Owner only                        |
| Who owns?      | One owner at a time               |
| Who cleans up? | The allocator or explicit handoff |

**Ownership Patterns:**

```go
// Caller allocates, callee fills
func Read(buf []byte) (n int, err error)

// Callee allocates, caller owns
func ReadAll() ([]byte, error)

// Explicit ownership transfer
func (q *Queue) Push(item Item) // Queue now owns item

// Explicit sharing (rare, documented)
func (c *Cache) GetShared(key string) *Item // DO NOT MODIFY
```

**Axiom:** Ownership that is not documented is not owned.

---

## 🏗️ 3.4 Function Size

**Premise:** Functions that cannot fit in the mind cannot be proven correct.

**Rules:**

1. Proof-critical functions fit **one printed page** (~60 lines)
2. Split by concept, not aesthetics
3. Declare variables at **smallest possible scope**
4. Do not keep values alive "just in case"
5. `goto` is forbidden unless explicitly waived

**Size Guidelines:**

| Complexity     | Max Lines | Reason                |
| -------------- | --------- | --------------------- |
| Trivial        | 10        | Glance-verifiable     |
| Simple         | 25        | Single screen         |
| Complex        | 60        | One printed page      |
| Infrastructure | 100       | Requires extra review |

**Axiom:** If it cannot fit in the mind, it cannot be proven.

---

## 🏗️ 3.5 Error Wrapping

**Premise:** Error context is documentation for debugging.

**Rules:**

1. Wrap errors when crossing boundaries
2. Use `%w` to preserve the error chain
3. Add context that helps debugging
4. Do not wrap when re-returning unchanged

```go
// GOOD: wrap at boundary
func (s *Service) Process(ctx context.Context, id string) error {
    item, err := s.store.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("get item %s: %w", id, err)
    }
    // ...
}

// GOOD: no wrap when re-returning
func (s *Store) Get(ctx context.Context, id string) (*Item, error) {
    return s.db.Get(ctx, id) // Let caller wrap
}
```

**Axiom:** Errors without context are mysteries.

---

# 🏛️ Part IV: The Invariants

**Invariant:** A non-negotiable truth about correctness. If an invariant does
not hold, the system is incorrect.

---

## 🏛️ 4.1 Correctness is Top Priority

**Premise:** Correctness beats performance 100% of the time.

**For every piece of code, ask:**

1. 5 simple ways this can be gamed
2. 5 medium ways it can be gamed
3. 5 surprising and hard ways it can be gamed

**The Adversarial Checklist:**

- What if the input is empty?
- What if the input is maximum size?
- What if the input contains nulls?
- What if the operation times out?
- What if the operation succeeds but ack fails?
- What if this is called twice?
- What if this is called concurrently?

**Axiom:** Faster wrong answers are worse than slower correct ones.

---

## 🏛️ 4.2 Determinism is Mandatory in Core

**Premise:** Core is deterministic computation only. Same inputs produce same
outputs, always.

**Behavior MUST NOT vary with:**

- Map iteration order (undefined in Go)
- Goroutine scheduling (undefined)
- Select statement choice (pseudo-random)
- Time of day
- Random number sequence
- Environment variables (after startup)

**Making Code Deterministic:**

```go
// BAD: map iteration order is undefined
for k, v := range m {
    results = append(results, process(k, v))
}

// GOOD: sort keys first
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
slices.Sort(keys)
for _, k := range keys {
    results = append(results, process(k, m[k]))
}
```

**Axiom:** If it flakes, it is broken.

---

## 🏛️ 4.3 Boundaries Collapse Uncertainty

**Premise:** A boundary is where untrusted bytes become trusted, typed state and
collapse into one canonical form.

**The Boundary Procedure (Strictly Ordered):**

| Step         | Purpose                                          |
| ------------ | ------------------------------------------------ |
| 1. Sanitize  | Remove hostile structure before allocation       |
| 2. Validate  | Check rules: required fields, ranges, invariants |
| 3. Normalize | Collapse meaning: case, sorting, defaults, UTC   |
| 4. Bind      | Map into internal typed structures               |
| 5. Freeze    | Copy if retained; make immutable; one owner      |

**Why This Order:**

- Validation before Sanitization trusts hostile structure
- Normalization before Validation canonizes invalid data
- Binding before Normalization multiplies meaning variants
- Freezing before Binding freezes ambiguity

**Example:**

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 1. Sanitize: limit body size
    r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

    // 2. Decode
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    // 3. Validate
    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 4. Normalize
    req.Normalize()

    // 5. Bind to domain type
    user := domain.User{
        Name:  req.Name,
        Email: req.Email,
    }

    // 6. Process (core logic receives clean, typed data)
    if err := h.service.CreateUser(r.Context(), user); err != nil {
        // ...
    }
}
```

**Axiom:** After a boundary, ambiguity is forbidden.

---

## 🏛️ 4.4 No Ghost State

**Premise:** Ghost state is state you believe exists but cannot prove at
runtime. Ghost state is a correctness defect.

**Sources of Ghost State:**

| Source                    | Example                               |
| ------------------------- | ------------------------------------- |
| Partial failure           | DB write succeeds, cache update fails |
| Retries without proof     | Did the first attempt succeed?        |
| Logs as truth             | "Log says it worked"                  |
| Shared mutable state      | Two goroutines, one variable          |
| Optimistic UI without rec | "We showed success"                   |
| Fire-and-forget async     | "We sent the message"                 |

**Ghost State Elimination:**

1. Use transactions for related writes
2. Use idempotency keys for retries
3. Use proof (receipts) not logs
4. Use channels for ownership transfer
5. Use reconciliation for optimistic updates

**Axiom:** If you cannot prove what happened, you cannot ship what happened.

---

## 🏛️ 4.5 Everything is Bounded

**Premise:** Budgets are explicit, enforced limits. Unbounded systems fail
without warning.

**What Must Be Bounded:**

| Resource    | Must Declare        | Must Enforce         |
| ----------- | ------------------- | -------------------- |
| Time        | Timeout budget      | Context deadline     |
| Memory      | Max bytes/size      | Pre-allocation limit |
| Goroutines  | Max concurrent      | Semaphore/pool       |
| Queues      | Max depth           | Bounded channel      |
| Retries     | Max attempts + time | Counter + deadline   |
| Concurrency | Max parallel        | errgroup.SetLimit    |
| Cardinality | Max items           | Map/slice cap        |
| Connections | Max pool size       | Pool configuration   |

**Rules:**

- A budget not enforced is fiction
- Saturation is expected and must be observable
- Systems that hide saturation fail unpredictably

**Axiom:** A drop policy is declared behavior. Silence is not a policy.

---

## 🛑 4.6 No Allocation in Hot Paths Without Proof

**Premise:** Hot paths allocate zero by default. Any allocation requires
measurement and explicit waiver.

**Hot Path Defined:** Per-request execution inside the tail latency envelope.
The p99 path owns the budget.

**Allocation Sources to Audit:**

```go
// Each of these allocates:
make([]T, n)                    // Slice
make(map[K]V)                   // Map
new(T)                          // Pointer
&T{}                            // Pointer
append(slice, items...)         // When capacity exceeded
string([]byte)                  // String from bytes
[]byte(string)                  // Bytes from string
fmt.Sprintf(...)                // String formatting
errors.New(...)                 // Error creation
interface conversion            // Sometimes
closure capture                 // When escaping
```

**Zero-Allocation Patterns:**

```go
// Pre-allocate with known capacity
items := make([]Item, 0, expectedCount)

// Reuse buffers via sync.Pool
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func process(data []byte) {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()
    // ...
}
```

**Axiom:** Allocating in hot paths is a defect unless proven and waived.

---

## 🏛️ 4.7 Async Requires Freeze

**Premise:** Data passed to async routines must be frozen first. Sharing mutable
data across goroutines is forbidden.

**Freeze Means:**

1. **Copy:** Duplicate the bytes
2. **Canonicalize:** Resolve all variants
3. **Make Immutable:** No further mutation

**Forbidden:**

```go
// BAD: sharing mutable slice
items := getItems()
go func() {
    for _, item := range items { // Race condition
        process(item)
    }
}()
items = append(items, newItem) // Mutates shared slice
```

**Required:**

```go
// GOOD: copy before async
items := getItems()
itemsCopy := make([]Item, len(items))
copy(itemsCopy, items)
go func() {
    for _, item := range itemsCopy { // Safe
        process(item)
    }
}()
```

**Axiom:** Mutable sharing across async boundaries is forbidden.

---

## 🏛️ 4.8 Additive Change Only

**Premise:** Existing contracts cannot be removed or changed. Only additive
changes are safe.

**Safe Changes:**

- Add new optional field (with default)
- Add new endpoint
- Add new enum value (if consumer handles unknown)
- Widen accepted input range

**Unsafe Changes:**

- Remove field
- Rename field
- Change field type
- Narrow accepted input range
- Change behavior for existing inputs

**Axiom:** Breaking a contract breaks the system.

---

# 🏗️ Part V: Functional Core, Imperative Shell

**This is how systems remain correct under load, time, humans, and failure.**

---

## 🏗️ 5.1 The Two Worlds

**Premise:** Code is either Core or Shell. Never both.

**The Shell (Imperative, Hostile):**

- Performs I/O (network, disk, database)
- Observes time (`time.Now()`)
- Consumes entropy (`rand`)
- Handles signals
- Talks to the operating system
- Calls external dependencies
- Creates goroutines

**The Core (Functional, Deterministic):**

- No I/O
- No clocks
- No randomness
- No globals
- No goroutine creation
- No dependency calls
- Pure computation only

**Example:**

```go
// SHELL: handles I/O, time, entropy
func (s *Service) ProcessOrder(ctx context.Context, id string) error {
    // Shell: reads from database
    order, err := s.store.Get(ctx, id)
    if err != nil {
        return err
    }

    // Shell: gets current time
    now := time.Now()

    // CORE: pure computation (no I/O, no time observation)
    result := s.core.CalculateTotal(order, now)

    // Shell: writes to database
    return s.store.Save(ctx, result)
}

// CORE: pure function
func (c *Core) CalculateTotal(order Order, asOf time.Time) OrderResult {
    // No I/O, no time.Now(), no randomness
    // Just pure computation on inputs
}
```

**Axiom:** The shell absorbs uncertainty. The core refuses it.

---

## 🏗️ 5.2 Direction of Dependency

**Premise:** Dependency direction is strict and inward.

```
                    ┌─────────────┐
                    │    Shell    │  Touches the world
                    │  (HTTP, DB) │
                    └──────┬──────┘
                           │ depends on
                           ▼
                    ┌─────────────┐
                    │Orchestrator │  Coordinates flow
                    │  (Service)  │
                    └──────┬──────┘
                           │ depends on
                           ▼
                    ┌─────────────┐
                    │    Core     │  Pure logic
                    │  (Domain)   │
                    └─────────────┘
                           │ depends on
                           ▼
                       (nothing)
```

**Rules:**

1. **Core** depends on **nothing** (not even stdlib I/O)
2. **Orchestrators** depend on **core**
3. **Shells** depend on **orchestrators** and the **world**
4. Dependencies never point outward (core never imports shell)

**Axiom:** Dependencies point inward. Always.

---

## 🏗️ 5.3 Orchestrators

**Premise:** Orchestrators sit between shell and core. They coordinate sequence
but do not contain business logic.

**Orchestrators DO:**

- Call core functions
- Decide order of operations
- Handle retries and compensation
- Manage lifecycles and cancellation
- Aggregate results from multiple core calls
- Translate between shell and core types

**Orchestrators DO NOT:**

- Parse input (shell does this)
- Validate data (core does this)
- Normalize formats (boundary does this)
- Perform I/O directly (shell does this)
- Contain business rules (core does this)

**Example:**

```go
// Orchestrator: coordinates flow
func (s *OrderService) PlaceOrder(
    ctx context.Context,
    req PlaceOrderRequest,
) (*Order, error) {
    // Step 1: Get user (shell operation)
    user, err := s.users.Get(ctx, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }

    // Step 2: Validate order (core operation)
    if err := s.core.ValidateOrder(req, user); err != nil {
        return nil, fmt.Errorf("validate: %w", err)
    }

    // Step 3: Calculate pricing (core operation)
    pricing := s.core.CalculatePricing(req.Items, user.Tier)

    // Step 4: Create order (core operation)
    order := s.core.CreateOrder(req, user, pricing, time.Now())

    // Step 5: Persist (shell operation)
    if err := s.store.Save(ctx, order); err != nil {
        return nil, fmt.Errorf("save: %w", err)
    }

    return order, nil
}
```

**Axiom:** Orchestrators own sequence, not truth.

---

## 🏗️ 5.4 Testing Implications

**Premise:** The Core/Shell split enables perfect testability.

**Core Tests:**

- Pure unit tests
- No mocks required (no dependencies)
- No clocks (time is a parameter)
- No sleeps (no async)
- No network (no I/O)
- Deterministic (same input = same output)
- Fast (milliseconds)

**Shell Tests:**

- Integration tests
- Mocked or real dependencies
- Timeout and failure injection
- Boundary validation
- Fuzz testing for input handling

**Orchestrator Tests:**

- Mocked shell operations
- Verify correct sequencing
- Verify retry logic
- Verify cancellation propagation

```go
// Core test: pure, no mocks
func TestCalculatePricing(t *testing.T) {
    items := []Item{{Price: 100}, {Price: 200}}
    tier := TierGold

    result := core.CalculatePricing(items, tier)

    if result.Total != 270 { // 10% discount
        t.Errorf("got %d, want 270", result.Total)
    }
}

// No mocks. No setup. No teardown. Just input and output.
```

**Axiom:** Correctness lives in core tests. Integration tests verify wiring.

---

## 🏗️ 5.5 Where Boundaries Live

**Premise:** Boundaries transform data between worlds.

```
External World                 Internal World
     │                              │
     │    ┌──────────────────┐      │
     │    │                  │      │
     └───►│    BOUNDARY      │──────┘
          │                  │
          │  1. Sanitize     │
          │  2. Validate     │
          │  3. Normalize    │
          │  4. Bind         │
          │  5. Freeze       │
          │                  │
          └──────────────────┘
```

**Boundary Locations:**

| Boundary Type  | Location               | Example               |
| -------------- | ---------------------- | --------------------- |
| HTTP Ingress   | Handler function       | Request parsing       |
| Queue Consumer | Message receive loop   | Event deserialization |
| Database Read  | Repository function    | Row scanning          |
| External API   | Adapter function       | Response parsing      |
| Config Load    | Startup initialization | YAML/env parsing      |
| File Read      | File processing func   | Content parsing       |

**Axiom:** Every boundary executes the full procedure. No shortcuts.

---

# 🏎️ Part VI: Budgets, Saturation, and Drop Policy

**Boundedness is correctness.** Any system that can grow without limit will fail
without warning.

---

## 🏎️ 6.1 The Scale Math

**Premise:** At scale, intuition fails. Arithmetic does not.

**At 1,000,000 requests per second:**

| Per-Request Cost | Aggregate Cost                        |
| ---------------- | ------------------------------------- |
| 1 allocation     | 1,000,000 allocations/sec GC pressure |
| 1 microsecond    | 1 full CPU core consumed              |
| 1 cache miss     | Millions of memory stalls             |
| 1 KB memory      | 1 GB in-flight memory                 |
| 1 log line       | 1M log lines/sec to process           |

**Little's Law:**

```
Concurrency = Throughput × Latency

If target = 1M req/sec with 100ms latency:
Concurrency = 1,000,000 × 0.1 = 100,000 in-flight operations
```

**That is the shape of your machine. Design for it.**

**Axiom:** Scale multiplies small costs into systemic drag.

---

## 🏎️ 6.2 Saturation is Expected

**Premise:** Saturation is the state where demand exceeds budget. At scale,
saturation is normal, not exceptional.

**Hidden saturation creates ghost state:**

- Work appears accepted but never completes
- Latency grows without bound
- Retries amplify load (retry storm)
- Memory grows without limit (OOM)
- Queues back up silently

**Saturation Signals:**

| Signal                     | Meaning                         |
| -------------------------- | ------------------------------- |
| Queue depth increasing     | Producers faster than consumers |
| Latency percentiles rising | Processing falling behind       |
| Memory growing             | Work accumulating               |
| CPU near 100%              | Compute saturated               |
| Connection pool exhausted  | I/O saturated                   |

**Axiom:** If the system cannot do the work, it must say so clearly.

---

## 🏎️ 6.3 Drop Policy is Mandatory

**Premise:** Every budget must declare behavior under saturation. "Hope" is not
a policy.

**Common Drop Policies:**

| Policy            | When to Use                           |
| ----------------- | ------------------------------------- |
| Reject at ingress | Preserve capacity for accepted work   |
| Shed by priority  | Drop low-value work first             |
| Drop oldest       | Freshness matters more than order     |
| Degrade response  | Return partial result (if documented) |
| Circuit break     | Stop calling failing dependency       |

**Safe Priority Order for Dropping:**

1. Optional evidence (debug logs, verbose traces)
2. Best-effort reads (cache misses, fan-out lookups)
3. Background work (async pipelines, low-value tasks)
4. Reject new requests at ingress (preserve in-flight)

**Axiom:** Fast rejection preserves capacity. Slow acceptance destroys it.

---

## 🏎️ 6.4 Budget Declaration

**Premise:** Every hot path must declare and enforce budgets.

**Budget Declaration Table:**

| Budget Type        | Example Limit              | Enforcement         |
| ------------------ | -------------------------- | ------------------- |
| Allocations per op | 0 (proof required for any) | Benchmark assertion |
| Bytes per op       | < 1KB                      | Benchmark assertion |
| Goroutines per req | Max 4                      | errgroup.SetLimit   |
| Dependency calls   | Max 2                      | Counter + reject    |
| Lock hold time     | < 100 microseconds         | Deadline check      |
| Queue depth        | 1000 before drop           | Bounded channel     |
| Retry attempts     | Max 3                      | Counter             |
| Retry total time   | Max 5 seconds              | Context deadline    |
| Request timeout    | 10 seconds                 | Context deadline    |

**Enforcement in Code:**

```go
const (
    maxQueueDepth    = 1000
    maxGoroutines    = 8
    maxRetries       = 3
    requestTimeout   = 10 * time.Second
)

// Bounded queue
work := make(chan Task, maxQueueDepth)

// Bounded concurrency
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(maxGoroutines)

// Bounded retries
for attempt := 0; attempt < maxRetries; attempt++ {
    // ...
}

// Bounded time
ctx, cancel := context.WithTimeout(ctx, requestTimeout)
defer cancel()
```

**Axiom:** If a budget cannot be stated, the system is not understood.

---

## 🏎️ 6.5 Queue Discipline

**Premise:** Queues are the most dangerous abstraction. They hide latency and
create ghost state.

**Queue Rules:**

1. **Every queue is bounded** (no unbounded channels)
2. **Every queue has a drop policy** (what happens when full?)
3. **Every queue has observability** (depth metric)
4. **Every queue has a drain deadline** (shutdown behavior)

**Queue Implementation:**

```go
type BoundedQueue[T any] struct {
    ch      chan T
    drops   atomic.Int64
    onDrop  func(T)
}

func NewBoundedQueue[T any](size int, onDrop func(T)) *BoundedQueue[T] {
    return &BoundedQueue[T]{
        ch:     make(chan T, size),
        onDrop: onDrop,
    }
}

func (q *BoundedQueue[T]) Push(item T) bool {
    select {
    case q.ch <- item:
        return true
    default:
        q.drops.Add(1)
        if q.onDrop != nil {
            q.onDrop(item)
        }
        return false
    }
}

func (q *BoundedQueue[T]) Depth() int {
    return len(q.ch)
}

func (q *BoundedQueue[T]) Drops() int64 {
    return q.drops.Load()
}
```

**Axiom:** An unbounded queue is a memory leak with extra steps.

---

# 🏛️ Part VII: Proof, Evidence, and Ghost State

**Correctness is what you can prove happened.** If you cannot prove it, it did
not happen.

---

## 🏛️ 7.1 Evidence vs Proof

**Premise:** Evidence and proof are fundamentally different. Never confuse them.

**Evidence (Observational):**

- Logs
- Metrics
- Traces
- Telemetry

Evidence can be dropped, delayed, sampled, reordered, or lost. **Evidence
explains. Evidence does not decide.**

**Proof (Authoritative):**

- Committed database transactions
- Outbox records written atomically
- Durable idempotency keys with stored outcome
- Append-only decision logs
- Cryptographic receipts

Proof survives crashes, retries, and restarts. **Proof decides. Proof is
truth.**

**Example:**

```go
// Evidence: might be lost
log.Info("order processed", "id", orderID)

// Proof: survives crashes
receipt, err := store.SaveWithReceipt(ctx, order)
if err != nil {
    return err // No proof, no success claim
}
// Now we have proof: receipt.ID, receipt.Hash, receipt.Timestamp
```

**Axiom:** Logs explain. Proof decides.

---

## 🏛️ 7.2 Binary Outcomes Only

**Premise:** Every externally visible operation resolves to exactly one outcome:
Success or Failure. There is no third state.

| Outcome | Definition                                       |
| ------- | ------------------------------------------------ |
| Success | All effects occurred and are proven              |
| Failure | No effects occurred OR effects are proven failed |

**Non-Outcomes (Forbidden):**

- "Pending" is not an outcome
- "Best effort" is not an outcome
- "Probably succeeded" is not an outcome
- "Timeout but maybe it worked" is not an outcome
- "Unknown" is not an acceptable final state

**Handling Uncertainty:**

```go
func ProcessOrder(ctx context.Context, order Order) (Receipt, error) {
    // Attempt with timeout
    receipt, err := s.store.Save(ctx, order)
    if err != nil {
        // Check if we have proof despite error
        if existingReceipt, ok := s.checkExisting(ctx, order.IdempotencyKey); ok {
            return existingReceipt, nil // Proof exists, success
        }
        return Receipt{}, err // No proof, failure
    }
    return receipt, nil // Proof exists, success
}
```

**Axiom:** If the outcome is unknown, the system is incorrect.

---

## 🏛️ 7.3 Idempotency as Fallback

**Premise:** When durable proof cannot be recorded before the operation,
operations must be idempotent by design.

**Idempotency Strategies:**

| Strategy           | Mechanism                          | Use When               |
| ------------------ | ---------------------------------- | ---------------------- |
| Natural idempotent | Operation is inherently repeatable | SET x = 5              |
| Idempotency key    | Client-generated unique key        | Payments, orders       |
| Version/ETag       | Conditional update                 | Optimistic concurrency |
| Deduplication log  | Track processed IDs                | Message consumers      |

**Idempotency Key Pattern:**

```go
type IdempotencyStore interface {
    // Check returns existing result if key was processed
    Check(ctx context.Context, key string) (*Result, bool, error)
    // Store saves result atomically with key
    Store(ctx context.Context, key string, result *Result) error
}

func (s *Service) Process(ctx context.Context, key string, req Request) (*Result, error) {
    // Check if already processed
    existing, found, err := s.idem.Check(ctx, key)
    if err != nil {
        return nil, err
    }
    if found {
        return existing, nil // Idempotent return
    }

    // Process
    result, err := s.doProcess(ctx, req)
    if err != nil {
        return nil, err
    }

    // Store result with key (atomic)
    if err := s.idem.Store(ctx, key, result); err != nil {
        // Another request might have stored - check again
        if existing, found, _ := s.idem.Check(ctx, key); found {
            return existing, nil
        }
        return nil, err
    }

    return result, nil
}
```

**Axiom:** If an operation is neither provable nor idempotent, it must not
exist.

---

## 🏛️ 7.4 The Outbox Pattern

**Premise:** When an operation has multiple effects, use the outbox pattern to
ensure all-or-nothing semantics.

**Problem:** Database write + message publish must both succeed or both fail.

```go
// BAD: not atomic
err := db.Save(order)        // Succeeds
err = queue.Publish(event)   // Fails - ghost state!
```

**Solution: Outbox Pattern**

```go
// GOOD: atomic with outbox
func (s *Service) CreateOrder(ctx context.Context, order Order) error {
    return s.db.Transaction(ctx, func(tx *sql.Tx) error {
        // Save order
        if err := saveOrder(tx, order); err != nil {
            return err
        }

        // Save event to outbox (same transaction)
        event := OrderCreatedEvent{OrderID: order.ID, ...}
        if err := saveOutbox(tx, event); err != nil {
            return err
        }

        return nil // Both or neither
    })
}

// Separate process publishes from outbox
func (p *Publisher) Run(ctx context.Context) error {
    for {
        events, err := p.db.GetUnpublishedEvents(ctx, 100)
        if err != nil {
            return err
        }
        for _, e := range events {
            if err := p.queue.Publish(ctx, e); err != nil {
                continue // Will retry
            }
            p.db.MarkPublished(ctx, e.ID)
        }
        time.Sleep(100 * time.Millisecond)
    }
}
```

**Axiom:** Outbox turns distributed transactions into local transactions.

---

## 🏛️ 7.5 Receipt Pattern

**Premise:** A receipt is proof that an operation completed. Receipts are the
currency of correctness.

**Receipt Structure:**

```go
type Receipt struct {
    ID        string    // Unique receipt identifier
    Operation string    // What operation this proves
    Subject   string    // What entity was affected
    Outcome   Outcome   // Success or specific failure
    Timestamp time.Time // When proof was created
    Hash      [32]byte  // Integrity verification
}

type Outcome string

const (
    OutcomeSuccess      Outcome = "success"
    OutcomeDuplicate    Outcome = "duplicate"
    OutcomeNotFound     Outcome = "not_found"
    OutcomeConflict     Outcome = "conflict"
)

func (r Receipt) IsSuccess() bool {
    return r.Outcome == OutcomeSuccess || r.Outcome == OutcomeDuplicate
}
```

**Usage:**

```go
func (s *Service) Transfer(ctx context.Context, req TransferRequest) (Receipt, error) {
    // Execute transfer
    result, err := s.ledger.Transfer(ctx, req)
    if err != nil {
        return Receipt{}, err
    }

    // Create receipt (proof)
    receipt := Receipt{
        ID:        uuid.NewString(),
        Operation: "transfer",
        Subject:   req.TransferID,
        Outcome:   OutcomeSuccess,
        Timestamp: time.Now(),
    }
    receipt.Hash = hashReceipt(receipt)

    // Persist receipt
    if err := s.receipts.Save(ctx, receipt); err != nil {
        return Receipt{}, err
    }

    return receipt, nil
}
```

**Axiom:** If you have a receipt, you have proof.

---

# 🏛️ Part VIII: Determinism and Time

**Time is the most dangerous input a system can accept.**

---

## 🏛️ 8.1 Why Time is Hostile

**Premise:** Wall clock time is not stable. It is a hostile input that must be
treated with extreme care.

**Wall Clock Problems:**

- **Jumps forward:** NTP sync, manual adjustment
- **Jumps backward:** NTP sync, leap second
- **Skew:** Different machines disagree
- **Resolution:** Not precise enough for ordering
- **Zones:** Same instant, different representations

**Time Ambiguity:**

- Two timestamps that look equal can refer to different instants
- Two timestamps that look different can refer to the same instant
- Ordering by timestamp is unreliable

**Example Problem:**

```go
// BAD: race condition
t1 := time.Now() // Machine A: 12:00:00.000
t2 := time.Now() // Machine B: 12:00:00.001 (but actually earlier!)

if t1.Before(t2) { // Wrong conclusion!
    // ...
}
```

**Axiom:** Wall clock time is observation, not truth.

---

## 🏛️ 8.2 The Core Rule

**Premise:** Core logic must never observe time. Time must be injected as data.

**Forbidden in Core:**

```go
// BAD: Core observes time
func (c *Core) ProcessOrder(order Order) Result {
    now := time.Now() // Forbidden!
    if now.After(order.Deadline) {
        return Result{Expired: true}
    }
    // ...
}
```

**Required:**

```go
// GOOD: Time injected as parameter
func (c *Core) ProcessOrder(order Order, now time.Time) Result {
    if now.After(order.Deadline) {
        return Result{Expired: true}
    }
    // ...
}

// Shell provides time
func (s *Service) ProcessOrder(ctx context.Context, order Order) (Result, error) {
    now := time.Now() // Shell observes time
    result := s.core.ProcessOrder(order, now) // Core receives time
    // ...
}
```

**Axiom:** If core logic needs time, time must be injected as data.

---

## 🏛️ 8.3 UTC as Canonical Form

**Premise:** UTC is not a preference. UTC is a collapse of meaning. After
normalization to UTC, time has one representation.

**Why UTC:**

- No daylight saving shifts
- No regional interpretation
- No historical ambiguity
- No zone database dependency
- One instant, one representation

**After UTC Normalization:**

- Ordering is stable
- Comparison is deterministic
- Hashing is stable
- Storage is compact (no zone info)
- Retries do not invent new meaning

**Boundary Normalization:**

```go
// At boundary: normalize to UTC
func (r *CreateEventRequest) Normalize() {
    r.StartTime = r.StartTime.UTC()
    r.EndTime = r.EndTime.UTC()
}

// In core: all times are UTC
type Event struct {
    StartTime time.Time // Always UTC
    EndTime   time.Time // Always UTC
}
```

**Axiom:** UTC gives one instant one representation.

---

## 🏛️ 8.4 Presentation is Separate

**Premise:** Canonical time is not presentation time. Localization is a view
concern, not a storage concern.

**The Pattern:**

```
Storage          Core             Presentation
  │                │                   │
  │   UTC int64    │    time.Time     │   "Jan 2, 3:04 PM EST"
  │   ─────────────│───────────────── │──────────────────────
  │                │                   │
  │ 1706640000000  │  2024-01-30      │   "Tuesday, Jan 30"
  │                │  12:00:00 UTC    │   "7:00 AM EST"
  │                │                   │
```

**Rules:**

- Storage: UTC epoch milliseconds or RFC3339
- Core: `time.Time` in UTC
- Presentation: Localized string at the edge

```go
// Storage
type EventRow struct {
    StartTimeMs int64 `db:"start_time_ms"`
}

// Core (UTC)
type Event struct {
    StartTime time.Time
}

// Presentation (localized)
func FormatEventTime(t time.Time, loc *time.Location) string {
    return t.In(loc).Format("Monday, Jan 2 at 3:04 PM")
}
```

**Axiom:** Core stores instants. Shells translate for humans.

---

## 🏛️ 8.5 Monotonic Time for Measurement

**Premise:** Measuring duration requires monotonic time. Wall clock time can
jump, making duration calculations wrong.

**Two Clocks:**

| Clock      | Use                   | Go API                   |
| ---------- | --------------------- | ------------------------ |
| Wall Clock | Timestamps for humans | `time.Now()` (wall part) |
| Monotonic  | Duration measurement  | `time.Now()` (mono part) |

**Go's time.Time Has Both:**

```go
start := time.Now()  // Contains both wall and monotonic
// ... work ...
elapsed := time.Since(start) // Uses monotonic, safe

// But if you serialize and deserialize:
startStr := start.Format(time.RFC3339)
parsed, _ := time.Parse(time.RFC3339, startStr)
// parsed has NO monotonic component!
// time.Since(parsed) uses wall clock - unsafe
```

**For Explicit Measurement:**

```go
// When you need just monotonic
type Stopwatch struct {
    start time.Time
}

func StartStopwatch() Stopwatch {
    return Stopwatch{start: time.Now()}
}

func (s Stopwatch) Elapsed() time.Duration {
    return time.Since(s.start) // Uses monotonic
}
```

**Axiom:** Never measure duration with wall clock.

---

# 🏗️ Part IX: Concurrency and Structured Lifetimes

**Concurrency is not free performance.** **Concurrency is controlled complexity
with sharp edges.**

---

## 🏗️ 9.1 The Core Rule

**Premise:** Concurrency belongs to shells and orchestrators. Core logic is
single-threaded by design.

**Core Does NOT:**

- Create goroutines
- Manage locks
- Use channels
- Coordinate lifetimes
- Depend on scheduling

**Shell/Orchestrator DOES:**

- Create goroutines (with ownership)
- Manage synchronization
- Coordinate lifetimes
- Handle cancellation

**Why:**

- Core is deterministic (no scheduling dependence)
- Core is testable (no concurrency in tests)
- Core is provable (no race conditions)
- Concurrency complexity is contained

**Axiom:** If core depends on scheduling, correctness is accidental.

---

## 🏗️ 9.2 Structured Concurrency

**Premise:** Every goroutine must belong to a structure with clear ownership.

**Every Goroutine Must Have:**

1. An **owner** (who started it)
2. A **lifetime** (when it ends)
3. An **exit condition** (how it knows to stop)
4. A **cancellation path** (`ctx.Done`)

**Forbidden:**

```go
// BAD: fire-and-forget
go processAsync(data)

// BAD: no cancellation
go func() {
    for {
        // Runs forever, cannot be stopped
    }
}()
```

**Required:**

```go
// GOOD: owned goroutine with lifecycle
type Worker struct {
    cancel context.CancelFunc
    done   chan struct{}
}

func StartWorker(ctx context.Context) *Worker {
    ctx, cancel := context.WithCancel(ctx)
    w := &Worker{
        cancel: cancel,
        done:   make(chan struct{}),
    }
    go w.run(ctx)
    return w
}

func (w *Worker) run(ctx context.Context) {
    defer close(w.done)
    for {
        select {
        case <-ctx.Done():
            return // Exit condition
        default:
            // Work
        }
    }
}

func (w *Worker) Stop() {
    w.cancel()  // Signal stop
    <-w.done    // Wait for exit
}
```

**Axiom:** No fire-and-forget goroutines. Ever.

---

## 🏗️ 9.3 Goroutine Creation is Budgeted

**Premise:** Goroutine creation must be bounded. Unbounded goroutine creation is
a resource leak.

**Rules:**

1. Goroutine creation must be **bounded**
2. Fan-out requires an **explicit cap**
3. Caps are **enforced**, not documented
4. Goroutine count is observable

**Enforcement:**

```go
// Bounded worker pool
type Pool struct {
    work chan func()
    wg   sync.WaitGroup
}

func NewPool(size int) *Pool {
    p := &Pool{work: make(chan func(), size*2)}
    for i := 0; i < size; i++ {
        p.wg.Add(1)
        go p.worker()
    }
    return p
}

// Bounded fan-out with errgroup
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(8) // Maximum 8 concurrent goroutines

for _, item := range items {
    item := item
    g.Go(func() error {
        return process(ctx, item)
    })
}

return g.Wait()
```

**Axiom:** If you cannot state the maximum goroutine count, the design is wrong.

---

## 🏗️ 9.4 Parallelism Rules

**Premise:** Parallelism is allowed only for independent work that has been
measured to benefit from it.

**Rules:**

1. **Parallelize only independent effects** (no shared state)
2. **Measure first** (parallelism has overhead)
3. **Total time = slowest call**, not sum
4. **Sequential is the default**

**When Parallelism Helps:**

```go
// GOOD: independent I/O-bound operations
func fetchAll(ctx context.Context, ids []string) ([]*Item, error) {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10)

    results := make([]*Item, len(ids))
    for i, id := range ids {
        i, id := i, id
        g.Go(func() error {
            item, err := fetch(ctx, id)
            if err != nil {
                return err
            }
            results[i] = item // Safe: each goroutine writes different index
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

**When Parallelism Hurts:**

```go
// BAD: CPU-bound work with tiny items
for _, item := range items {
    go process(item) // Overhead exceeds benefit
}
```

**Axiom:** Parallelism without measurement widens the tail.

---

## 🏗️ 9.5 errgroup is the Default Shape

**Premise:** `errgroup` is the standard pattern for bounded parallel work.

**errgroup Features:**

- First error cancels siblings
- Parent waits for all children
- Concurrency limit via `SetLimit`
- Context propagation

**Standard Pattern:**

```go
func processAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(8) // Bound concurrency

    for _, item := range items {
        item := item // Capture (not needed in Go 1.22+)
        g.Go(func() error {
            return processItem(ctx, item)
        })
    }

    return g.Wait()
}
```

**With Results:**

```go
func fetchAll(ctx context.Context, ids []string) ([]Result, error) {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10)

    results := make([]Result, len(ids))
    for i, id := range ids {
        i, id := i, id
        g.Go(func() error {
            r, err := fetch(ctx, id)
            if err != nil {
                return err
            }
            results[i] = r
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

**Axiom:** Groups own lifetimes. Individual goroutines do not.

---

## 🏗️ 9.6 Channels vs Mutexes

**Premise:** Channels and mutexes serve different purposes. Choose based on the
pattern, not preference.

**Channels Are For:**

- Ownership transfer (sending data)
- Sequencing (coordinating order)
- Work queues (producer/consumer)
- Signaling (done channels)

**Mutexes Are For:**

- Shared mutable state (counters, maps)
- Short critical sections
- Protecting invariants

**Decision Table:**

| Pattern              | Use     | Example               |
| -------------------- | ------- | --------------------- |
| Transfer ownership   | Channel | `workChan <- task`    |
| Signal completion    | Channel | `close(done)`         |
| Protect shared state | Mutex   | `mu.Lock(); count++`  |
| Coordinate sequence  | Channel | Pipeline stages       |
| Pool of workers      | Channel | Buffered work channel |
| Read-heavy state     | RWMutex | Config lookups        |

**Rules:**

1. **Never** do I/O while holding a mutex
2. **Never** hold a mutex across `await` points
3. **Never** use channels as shared memory
4. **Never** mix atomics and mutexes on same field without care

**Axiom:** If ownership is unclear, the abstraction is wrong.

---

## 🏗️ 9.7 Scheduling is Not a Contract

**Premise:** The Go scheduler owes you nothing. Never depend on scheduling
behavior for correctness.

**The Scheduler Does Not Promise:**

- Goroutine execution order
- Fair time slicing
- Immediate preemption
- Select case preference

**Forbidden Dependencies:**

```go
// BAD: depends on ordering
go func() { step1() }()
go func() { step2() }() // Might run before step1!

// BAD: depends on select order
select {
case <-ch1:
    // Might not be chosen even if ready
case <-ch2:
    // Might be chosen instead
}

// BAD: depends on timing
go func() { data = compute() }()
time.Sleep(100 * time.Millisecond)
use(data) // Race condition!
```

**Required: Explicit Synchronization:**

```go
// GOOD: explicit ordering
done := make(chan struct{})
go func() {
    step1()
    close(done)
}()
<-done
step2()

// GOOD: explicit selection priority
select {
case <-ctx.Done():
    return ctx.Err()
default:
}
select {
case <-ctx.Done():
    return ctx.Err()
case item := <-ch:
    process(item)
}
```

**Axiom:** If correctness depends on scheduling, it is broken.

---

# 🏎️ Part X: Memory and Mechanical Sympathy

**Memory lives in space. Accessing it costs time. This is physics.**

---

## 🏎️ 10.1 The Memory Hierarchy

**Premise:** Memory is not uniform. Access speed varies by 1000x depending on
location.

| Level    | Latency | Size     | Notes           |
| -------- | ------- | -------- | --------------- |
| Register | <1ns    | 64 bytes | In CPU          |
| L1 Cache | ~1ns    | 64KB     | Per-core, tiny  |
| L2 Cache | ~3ns    | 256KB    | Per-core, small |
| L3 Cache | ~10ns   | 8MB      | Shared, medium  |
| RAM      | ~100ns  | GBs      | Main memory     |
| SSD      | ~100μs  | TBs      | Storage         |
| Network  | ~1ms    | ∞        | Remote          |

**The 100x Rule:**

- L1 → RAM = 100x slower
- RAM → SSD = 1000x slower
- SSD → Network = 10x slower

**Axiom:** Most performance problems are memory problems.

---

## 🏎️ 10.2 Cache Locality is Leverage

**Premise:** Code that respects cache locality gets faster as hardware improves.
Code that fights it gets slower.

**Spatial Locality:** Data used together should live together.

```go
// GOOD: contiguous memory, spatial locality
type Point struct {
    X, Y, Z float64
}
points := make([]Point, 1000)
// Points are contiguous in memory

// BAD: pointer chasing, no locality
type Point struct {
    X, Y, Z *float64
}
points := make([]*Point, 1000)
// Points scattered across heap
```

**Temporal Locality:** Data used repeatedly should stay hot.

```go
// GOOD: process array in order (prefetcher helps)
for i := range data {
    process(data[i])
}

// BAD: random access (cache thrashing)
for _, i := range randomIndices {
    process(data[i])
}
```

**Axiom:** Code that respects locality gets faster without code changes.

---

## 🏎️ 10.3 Struct Layout Matters

**Premise:** Struct field order affects memory layout, padding, and cache
behavior.

**Rules:**

1. Order fields **largest to smallest** (minimize padding)
2. Group **hot fields together** (same cache line)
3. Separate **hot writes** (avoid false sharing)
4. Avoid deep pointer graphs (pointer chasing)
5. Prefer contiguous memory (slices over linked structures)

**Padding Example:**

```go
// BAD: 24 bytes with padding
type Bad struct {
    a bool    // 1 byte
    // 7 bytes padding
    b int64   // 8 bytes
    c bool    // 1 byte
    // 7 bytes padding
}

// GOOD: 16 bytes, no padding
type Good struct {
    b int64   // 8 bytes
    a bool    // 1 byte
    c bool    // 1 byte
    // 6 bytes padding at end (unavoidable for alignment)
}
```

**False Sharing:**

```go
// BAD: counters on same cache line, false sharing
type Counters struct {
    read  int64
    write int64 // Same cache line, contention!
}

// GOOD: pad to separate cache lines
type Counters struct {
    read  int64
    _     [56]byte // Padding to 64-byte cache line
    write int64
}
```

**Axiom:** Cache misses are latency. Padding is wasted bandwidth.

---

## 🏎️ 10.4 Pointers vs Values

**Premise:** The copy vs pointer decision affects allocations, cache behavior,
and GC pressure.

**Prefer Values When:**

- Data is small (≤ 2 words / 16 bytes)
- Data is short-lived (function-local)
- You want to avoid nil checks
- You want deterministic memory layout

**Use Pointers When:**

- Data is large (copying is expensive)
- Data must be shared (explicit ownership)
- Data is optional (nil represents absence)
- Interface requires it

**Escape Analysis:**

```go
// Does NOT escape (stack allocated)
func sum(points []Point) float64 {
    var total float64
    for _, p := range points {
        total += p.X + p.Y + p.Z
    }
    return total
}

// DOES escape (heap allocated)
func getPoint() *Point {
    p := Point{X: 1, Y: 2, Z: 3}
    return &p // Escapes to heap
}
```

**Check Escapes:**

```bash
go build -gcflags="-m" ./...
```

**Axiom:** A copy that improves locality often beats a pointer that misses.

---

## 🏎️ 10.5 Garbage Collection Reality

**Premise:** GC is steady state, not an edge case. At scale, GC is always
running.

**GC Truths:**

- GC pauses are real (though short in Go)
- More allocations = more GC work
- Large heaps = longer mark phases
- Pointer-heavy structures = more work

**Reducing GC Pressure:**

```go
// BAD: allocates per iteration
for _, item := range items {
    result := process(item) // Allocates
    results = append(results, result)
}

// GOOD: pre-allocate
results := make([]Result, 0, len(items))
for _, item := range items {
    result := process(item)
    results = append(results, result) // No reallocation
}

// GOOD: sync.Pool for reusable buffers
var bufPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

func process(data []byte) {
    buf := bufPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufPool.Put(buf)
    }()
    // Use buf
}
```

**Axiom:** Mechanical sympathy respects the collector.

---

## 🛑 10.6 Forbidden in Hot Paths

**Premise:** These operations are banned in hot paths unless waived with
benchmark proof.

**Banned:**

| Operation             | Reason                     | Alternative           |
| --------------------- | -------------------------- | --------------------- |
| `fmt.Sprintf`         | Allocates, slow            | String concat/Builder |
| `reflect` package     | Slow, allocates            | Code generation       |
| `interface{}` / `any` | Type assertions, allocates | Concrete types        |
| Unbounded `append`    | Reallocation               | Pre-size slices       |
| Runtime sorting       | O(n log n), allocates      | Pre-sort, maintain    |
| Map iteration         | Non-deterministic          | Sorted keys           |
| `encoding/json`       | Reflection-based           | Code generation       |
| `regexp.Compile`      | Per-call compilation       | Pre-compile           |

**Axiom:** These widen the tail silently.

---

# 🏗️ Part XI: Context Doctrine

**Context is ownership made explicit.**

---

## 🏗️ 11.1 The Prime Rule

**Premise:** Context is mandatory for anything that can block. No exceptions.

**Context Is Required For:**

- Network calls
- Database queries
- File I/O
- Queue operations
- Any external dependency
- Any operation with a timeout

**Rules:**

1. **Never** pass nil context (fail fast at boundary)
2. Use `context.Background()` only at program root
3. Use `context.TODO()` only as temporary placeholder
4. Context is the **first parameter**

```go
// GOOD: context first, always
func (s *Store) Get(ctx context.Context, id string) (*Item, error) {
    if ctx == nil {
        panic("nil context") // Fail fast
    }
    // ...
}

// Start of request
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // Already has deadline from server
    // ...
}

// Program root
func main() {
    ctx := context.Background()
    // ...
}
```

**Axiom:** Context is the control plane from root to leaf.

---

## 🏗️ 11.2 Cancellation Propagation

**Premise:** Every blocking operation must observe `ctx.Done()` and exit
promptly when cancelled.

**Every Blocking Call Observes:**

```go
// Network call
resp, err := http.Do(req.WithContext(ctx))

// Database call
rows, err := db.QueryContext(ctx, query)

// Channel receive
select {
case item := <-ch:
    // Process
case <-ctx.Done():
    return ctx.Err()
}

// Custom blocking operation
func (w *Worker) Wait(ctx context.Context) error {
    select {
    case <-w.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Every Goroutine Has An Owner:**

1. Who started it (parent context)
2. Who cancels it (context cancellation)
3. When it must exit (deadline or cancellation)

**Axiom:** Ignoring `ctx.Done()` is resource theft.

---

## 🏗️ 11.3 Context is Not Storage

**Premise:** Context is for request-scoped values and cancellation, not
dependency injection or large data.

**Context Values Are For:**

- Request ID (for tracing)
- Trace ID (for distributed tracing)
- User ID (for authorization)
- Deadline (for timeouts)

**Context Values Are NOT For:**

- Dependencies (use struct fields)
- Configuration (use parameters)
- Large data (use parameters)
- Mutable state (never)

**NEVER Store Context in Struct:**

```go
// BAD: context stored
type Client struct {
    ctx context.Context // Never do this
}

// GOOD: context passed per-call
type Client struct {
    // No context field
}

func (c *Client) Get(ctx context.Context, id string) (*Item, error) {
    // Context is per-request
}
```

**Axiom:** If you need data, pass it explicitly.

---

## 🏗️ 11.4 Budget Subdivision

**Premise:** Child deadlines must be less than or equal to parent remaining
time. Never extend time.

**The Rule:**

```
Parent deadline: T
Child deadline:  t ≤ T - (time already spent)
```

**Implementation:**

```go
func childBudget(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
    // Check if parent has deadline
    if dl, ok := ctx.Deadline(); ok {
        remaining := time.Until(dl)
        if remaining <= 0 {
            // Already expired, return cancelled context
            ctx, cancel := context.WithCancel(ctx)
            cancel()
            return ctx, cancel
        }
        if d > remaining {
            d = remaining // Don't exceed parent
        }
    }
    return context.WithTimeout(ctx, d)
}

// Usage
func (s *Service) Process(ctx context.Context, req Request) error {
    // Parent has 10s deadline
    // Give database 3s max, but not more than remaining
    dbCtx, cancel := childBudget(ctx, 3*time.Second)
    defer cancel()

    return s.db.Query(dbCtx, req.Query)
}
```

**Axiom:** Never extend time. Subdivide or fail fast.

---

## 🏗️ 11.5 Error Taxonomy

**Premise:** Context errors have specific meanings. Handle them correctly.

**Context Errors:**

| Error                      | Meaning             | Response             |
| -------------------------- | ------------------- | -------------------- |
| `context.Canceled`         | Caller stopped work | Stop, clean up       |
| `context.DeadlineExceeded` | Budget exhausted    | Stop, return timeout |

**Handling:**

```go
func (s *Service) Process(ctx context.Context) error {
    result, err := s.dep.Call(ctx)
    if err != nil {
        if errors.Is(err, context.Canceled) {
            // Caller cancelled - not an error, just stop
            return err
        }
        if errors.Is(err, context.DeadlineExceeded) {
            // Budget exhausted - timeout error
            return fmt.Errorf("timeout calling dependency: %w", err)
        }
        // Actual error
        return fmt.Errorf("dependency error: %w", err)
    }
    // ...
}
```

**Axiom:** Cancellation is a control signal, not a generic error.

---

# 🏗️ Part XII: Error and Panic Doctrine

**Errors are part of truth. Panics signal broken reality.**

---

## 🏗️ 12.1 The Core Rule

**Premise:** Errors and panics serve different purposes. Never reverse them.

| Type  | Purpose          | Example                          | Response        |
| ----- | ---------------- | -------------------------------- | --------------- |
| Error | Expected outcome | User not found, timeout, invalid | Handle, return  |
| Panic | Programmer bug   | Nil deref, index bounds, broken  | Crash, fix code |

**Error = "This can happen in production, handle it"** **Panic = "This should
never happen, something is very wrong"**

**Axiom:** If it can happen in production, it is not a panic.

---

## 🏗️ 12.2 Errors are Values

**Premise:** Errors are values that can be examined, wrapped, and compared. Use
the type system, not string matching.

**Rules:**

1. Use typed errors, not string matching
2. Use `errors.Is` for comparison
3. Use `errors.As` for type extraction
4. Wrap errors only when adding meaning
5. Preserve the original cause with `%w`

**Sentinel Errors:**

```go
// Package-level sentinel errors
var (
    ErrNotFound   = errors.New("not found")
    ErrConflict   = errors.New("conflict")
    ErrBadRequest = errors.New("bad request")
)

// Checking
if errors.Is(err, ErrNotFound) {
    return http.StatusNotFound
}
```

**Typed Errors:**

```go
// Custom error type
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Checking
var ve *ValidationError
if errors.As(err, &ve) {
    return fmt.Sprintf("field %s: %s", ve.Field, ve.Message)
}
```

**Wrapping:**

```go
// GOOD: wrap with context
if err := s.store.Get(ctx, id); err != nil {
    return fmt.Errorf("get user %s: %w", id, err)
}

// BAD: wrap without adding value
if err := s.store.Get(ctx, id); err != nil {
    return fmt.Errorf("error: %w", err) // Adds nothing
}
```

**Error Message Format:**

- Lower case
- No trailing punctuation
- No newlines
- Stable (don't include timestamps or random values)

**Axiom:** If callers branch on error text, the design is broken.

---

## 🛑 12.3 Panic Rules

**Premise:** Panics are for programmer errors and impossible states. They must
never be used for expected runtime conditions.

**Panics Are Forbidden For:**

- User input validation
- Dependency failure
- Network errors
- Timeouts
- File not found
- Permission denied
- Any expected condition

**Panics Are For:**

- Nil pointer where contract guarantees non-nil
- Index out of bounds (bug in caller)
- Type assertion failure (bug in code)
- Unreachable code reached
- Invariant violation

**Example:**

```go
// OK: panic for programmer error
func MustParse(s string) Config {
    cfg, err := Parse(s)
    if err != nil {
        panic(fmt.Sprintf("invalid config literal: %v", err))
    }
    return cfg
}

// Use at init time only
var defaultConfig = MustParse(`{"timeout": "5s"}`)

// NOT OK: panic for user input
func ParseUserInput(s string) Config {
    cfg, err := Parse(s)
    if err != nil {
        panic(err) // BAD: user input can cause this
    }
    return cfg
}
```

**Axiom:** If it can happen in production, return an error.

---

## 🏗️ 12.4 Panic Containment

**Premise:** Panics must be contained at specific boundaries. They must never
leak to callers.

**Allowed Recovery Points:**

- HTTP handlers (per-request containment)
- gRPC handlers (per-request containment)
- Goroutine entry points (per-goroutine containment)
- Process supervisors (per-process containment)

**Recovery Must:**

1. Stop further side effects
2. Return deterministic failure response
3. Log the panic with stack trace
4. Emit metrics

**HTTP Handler Pattern:**

```go
func PanicRecovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if p := recover(); p != nil {
                // Log with stack trace
                stack := debug.Stack()
                log.Error("panic recovered",
                    "panic", p,
                    "stack", string(stack),
                    "path", r.URL.Path,
                )

                // Metrics
                panicCounter.Inc()

                // Deterministic response
                http.Error(w, "internal error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

**Axiom:** Swallowing panics silently is forbidden.

---

# 🏗️ Part XIII: Types and Data Doctrine

**Data is the system. Types decide correctness, latency, and cache behavior.**

---

## 🏗️ 13.1 Ownership and Mutability

**Premise:** Every piece of data has exactly one owner. Ownership must be
explicit in the type system.

**Rules:**

1. Prefer **value types** for immutable data
2. Use **pointers only** for optionality or mutation
3. Never share mutable memory across goroutines without sync
4. If a function retains input, it **MUST** copy

**Ownership Transfer:**

```go
// Ownership transfer via channel
func producer(out chan<- Item) {
    for item := range items {
        out <- item // Ownership transferred
        // producer no longer owns item
    }
}

func consumer(in <-chan Item) {
    for item := range in {
        // consumer now owns item
        process(item)
    }
}
```

**Copy Before Retain:**

```go
// BAD: retains reference to caller's data
func (c *Cache) Set(key string, value []byte) {
    c.data[key] = value // Caller might mutate!
}

// GOOD: copy before retain
func (c *Cache) Set(key string, value []byte) {
    copied := make([]byte, len(value))
    copy(copied, value)
    c.data[key] = copied
}
```

**Axiom:** Hidden sharing creates ghost state.

---

## 🏗️ 13.2 Strings and Bytes

**Premise:** Strings and bytes are different types with different guarantees.

**Rules:**

| Type     | Mutable | UTF-8 Guarantee | Use For     |
| -------- | ------- | --------------- | ----------- |
| `string` | No      | Not guaranteed  | Text        |
| `[]byte` | Yes     | Not guaranteed  | Binary data |

**Conversion Allocates:**

```go
s := string(b)  // Allocates new string
b := []byte(s)  // Allocates new slice
```

**Never Assume UTF-8:**

```go
// BAD: assumes UTF-8
for i, c := range s {
    // c is rune, but s might not be valid UTF-8
}

// GOOD: validate at boundary
if !utf8.ValidString(s) {
    return errors.New("invalid utf-8")
}
```

**Indexing:**

```go
s := "héllo"
len(s)       // 6 (bytes, not runes)
s[0]         // 'h' (byte)
s[1]         // 0xc3 (part of é, not the rune!)
```

**Axiom:** Meaning is validated at the boundary, not guessed in core.

---

## 🏗️ 13.3 Slices

**Premise:** Slices are views into arrays, not containers. Understanding this
prevents many bugs.

**Slice Internals:**

```go
// A slice is:
type slice struct {
    ptr *T    // Pointer to first element
    len int   // Number of elements
    cap int   // Capacity of underlying array
}
```

**Rules:**

1. A slice shares its backing array with other slices
2. **Copy** when you need ownership
3. **Preallocate** when size is known
4. Never retain a slice of a large buffer accidentally
5. Append growth in hot paths is a defect

**Sharing Pitfall:**

```go
// BAD: slices share backing array
original := []int{1, 2, 3, 4, 5}
subset := original[1:3] // {2, 3}
subset[0] = 99          // Modifies original too!
// original is now {1, 99, 3, 4, 5}

// GOOD: copy for independence
original := []int{1, 2, 3, 4, 5}
subset := make([]int, 2)
copy(subset, original[1:3])
subset[0] = 99 // original unchanged
```

**Memory Leak Pitfall:**

```go
// BAD: retains entire large buffer
func getHeader(buf []byte) []byte {
    return buf[:10] // Still references entire buf
}

// GOOD: copy to release large buffer
func getHeader(buf []byte) []byte {
    header := make([]byte, 10)
    copy(header, buf[:10])
    return header
}
```

**Axiom:** If you do not own the backing array, you do not own correctness.

---

## 🏗️ 13.4 Maps

**Premise:** Maps are lookup tools with specific semantics. They are not
general-purpose data models.

**Map Semantics:**

- **Iteration order is undefined** (randomized deliberately)
- **Not safe for concurrent access** (use sync.Map or mutex)
- **Cannot take address of value** (`&m[key]` is invalid)
- **Zero value is nil** (safe to read, panic on write)

**Rules:**

1. **Never rely** on iteration order
2. **Never use maps** for deterministic output
3. Convert to sorted slice when order matters
4. Maps do not belong in core logic (non-deterministic)

**Deterministic Iteration:**

```go
// BAD: order varies each run
for k, v := range m {
    fmt.Println(k, v) // Different order each time!
}

// GOOD: sorted iteration
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
slices.Sort(keys)
for _, k := range keys {
    fmt.Println(k, m[k]) // Deterministic order
}
```

**Axiom:** Maps are cache-hostile and non-deterministic. Use slices in core.

---

## 🏗️ 13.5 Canonical Data Shapes

**Premise:** Data should have one canonical representation. Multiple
representations create ambiguity.

**Rules:**

1. One meaning per type
2. One representation per meaning
3. Canonicalize at boundaries
4. Never canonicalize twice
5. Never defer canonicalization into core

**Canonicalization Examples:**

| Field Type | Variants          | Canonical Form        |
| ---------- | ----------------- | --------------------- |
| Email      | UPPER, lower, MiX | lowercase, trimmed    |
| Time       | local, UTC, epoch | UTC time.Time         |
| Phone      | formats vary      | E.164 (+1234567890)   |
| Currency   | float, string     | integer cents         |
| UUID       | upper, lower, hex | lowercase with dashes |

**Implementation:**

```go
type Email string

func NewEmail(s string) (Email, error) {
    s = strings.TrimSpace(s)
    s = strings.ToLower(s)
    if !isValidEmail(s) {
        return "", errors.New("invalid email")
    }
    return Email(s), nil
}
```

**Axiom:** Two equal meanings must be equal values.

---

## 🏗️ 13.6 Option Types

**Premise:** Optional values should be explicit, not implicit via nil or zero
values.

**Go Patterns for Optional:**

```go
// Pattern 1: Pointer (nil = absent)
type User struct {
    Name  string
    Email *string // nil means not provided
}

// Pattern 2: Comma-ok (for returns)
func (m *Map) Get(key string) (value Value, ok bool)

// Pattern 3: Explicit optional type
type Optional[T any] struct {
    Value T
    Valid bool
}

func Some[T any](v T) Optional[T] {
    return Optional[T]{Value: v, Valid: true}
}

func None[T any]() Optional[T] {
    return Optional[T]{}
}
```

**Axiom:** Explicit absence is better than magic zero values.

---

# 🏗️ Part XIV: Control Flow Doctrine

**Control flow is a correctness surface.** **When flow is complex, verification
collapses.**

---

## 🏗️ 14.1 Structured Flow Only

**Premise:** Control flow must be predictable, bounded, and auditable.

**Use:**

- `if` / `else`
- `switch` / `case`
- `for` / `range`
- `return`
- `break` / `continue`

**Avoid:**

- `goto` (unless waived)
- Deep nesting
- Complex conditionals
- Implicit fallthrough

**Axiom:** Each path should be obvious to a reviewer in seconds.

---

## 🏗️ 14.2 The Line of Sight

**Premise:** The happy path must never be indented. It resides on the left edge.

**Guard Clauses First:**

```go
func Purchase(ctx context.Context, u *User, order Order) error {
    // Guard 1: Rejection
    if u == nil {
        return ErrNilUser
    }

    // Guard 2: Validation
    if err := order.Validate(); err != nil {
        return fmt.Errorf("invalid order: %w", err)
    }

    // Guard 3: Authorization
    if !u.CanPurchase() {
        return ErrUnauthorized
    }

    // Guard 4: Cancellation
    if ctx.Err() != nil {
        return ctx.Err()
    }

    // Happy path: zero indentation
    return s.processPurchase(ctx, u, order)
}
```

**The Triad of Defeat:**

1. **Rejection:** Invalid inputs
2. **Saturation:** Resource limits
3. **Corruption:** Context cancellation

**Axiom:** Handle failure first. Success is the reward.

---

## 🏗️ 14.3 Switch Over If Chains

**Premise:** Switch statements make cases explicit and support exhaustive
handling.

**Forbidden:**

```go
// BAD: if-else chain
if status == "pending" {
    handlePending()
} else if status == "active" {
    handleActive()
} else if status == "done" {
    handleDone()
} else {
    handleUnknown()
}
```

**Required:**

```go
// GOOD: switch makes cases explicit
switch status {
case "pending":
    handlePending()
case "active":
    handleActive()
case "done":
    handleDone()
default:
    handleUnknown()
}

// GOOD: typed enum with exhaustive handling
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusDone
)

func handle(s Status) {
    switch s {
    case StatusPending:
        handlePending()
    case StatusActive:
        handleActive()
    case StatusDone:
        handleDone()
    // No default = compiler warns if new status added
    }
}
```

**Axiom:** Explicit cases prevent forgotten branches.

---

## 🛑 14.4 No Recursion in Proof-Critical Code

**Premise:** Recursion hides bounds. Iterative code has explicit bounds.

**Recursion Problems:**

- Stack overflow on deep input
- Hard to reason about termination
- Hard to bound memory usage
- Hard to add cancellation

**When Recursion Allowed:**

- Tree traversal with proven bounded depth
- Divide-and-conquer with logarithmic depth
- Must have explicit depth limit

```go
// If recursion is necessary, bound it
func traverse(node *Node, depth int) error {
    const maxDepth = 100
    if depth > maxDepth {
        return errors.New("max depth exceeded")
    }
    if node == nil {
        return nil
    }
    // Process node
    for _, child := range node.Children {
        if err := traverse(child, depth+1); err != nil {
            return err
        }
    }
    return nil
}
```

**Axiom:** If recursion is used, it requires explicit proof of a hard bound.

---

## 🏗️ 14.5 Loops Must Be Bounded

**Premise:** Every loop must have a provable termination condition.

**Valid Bounds:**

| Bound Type            | Example                              |
| --------------------- | ------------------------------------ |
| Fixed count           | `for i := 0; i < 100; i++`           |
| Collection size       | `for _, v := range slice`            |
| Budget + cancellation | `for ctx.Err() == nil && attempts<3` |
| Channel close         | `for v := range ch`                  |

**Forbidden:**

```go
// BAD: infinite loop without exit
for {
    doWork()
}

// BAD: condition might never be false
for !done {
    tryOnce()
}
```

**Required:**

```go
// GOOD: bounded by counter and context
const maxAttempts = 3
for attempt := 0; attempt < maxAttempts; attempt++ {
    if ctx.Err() != nil {
        return ctx.Err()
    }
    if err := tryOnce(); err == nil {
        return nil
    }
}
return ErrMaxAttemptsExceeded

// GOOD: bounded by channel close
for item := range ch { // Exits when ch closed
    process(item)
}

// GOOD: bounded by context
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case item := <-ch:
        process(item)
    }
}
```

**Axiom:** If the bound is not explicit, the loop is incorrect.

---

# 🏗️ Part XV: Compile Time, Startup, and Runtime

**What is proven early does not need to be defended later.**

---

## 🏗️ 15.1 The Phase Rule

**Premise:** Anything that can be decided earlier must be decided earlier. Later
decisions are more expensive.

| Phase        | Cost    | Examples                                 |
| ------------ | ------- | ---------------------------------------- |
| Compile Time | Zero    | Type checking, generated code, constants |
| Startup      | Once    | Config validation, connection pools      |
| Runtime      | Per-req | Request handling, business logic         |

**Axiom:** Runtime decisions are a tax on tail latency.

---

## 🏗️ 15.2 Compile Time Doctrine

**Premise:** Use compile time aggressively to eliminate runtime work.

**Compile Time Techniques:**

- **Generated code** over runtime reflection
- **Constants** over variables
- **Switch statements** over dynamic dispatch
- **Table-driven logic** over branching
- **Type assertions** at compile time

**Code Generation:**

```go
//go:generate go run gen.go

// Generated: switch statement for routes
func route(path string) Handler {
    switch path {
    case "/users":
        return usersHandler
    case "/orders":
        return ordersHandler
    // ... generated cases
    default:
        return notFoundHandler
    }
}
```

**Compile-Time Interface Check:**

```go
// Verify implementation at compile time
var _ io.Reader = (*MyReader)(nil)
var _ http.Handler = (*MyHandler)(nil)
```

**Axiom:** Code that does not run cannot fail.

---

## 🏗️ 15.3 Startup Time Doctrine

**Premise:** Startup runs once. Requests run forever. Do expensive work at
startup.

**What Belongs at Startup:**

1. Configuration loading and validation
2. Dependency initialization (DB pools, clients)
3. Schema validation
4. Route registration
5. Pre-computation (lookup tables, compiled regexes)

**Startup Pattern:**

```go
func main() {
    // 1. Load config (fail fast if invalid)
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("config:", err)
    }

    // 2. Validate config (fail fast if invalid)
    if err := cfg.Validate(); err != nil {
        log.Fatal("invalid config:", err)
    }

    // 3. Initialize dependencies
    db, err := sql.Open("postgres", cfg.DatabaseURL)
    if err != nil {
        log.Fatal("database:", err)
    }
    defer db.Close()

    // 4. Verify connectivity
    if err := db.Ping(); err != nil {
        log.Fatal("database ping:", err)
    }

    // 5. Build service graph
    repo := repository.New(db)
    svc := service.New(repo)
    handler := transport.NewHandler(svc)

    // 6. Start server
    // ...
}
```

**Axiom:** If the system cannot start safely, it must not start at all.

---

## 🏗️ 15.4 Runtime Doctrine

**Premise:** Runtime does only what cannot be done earlier.

**Runtime Does:**

- Accept requests
- Enforce budgets (timeouts, limits)
- Execute deterministic core logic
- Emit evidence (logs, metrics)
- Produce proof (receipts)

**Runtime Does NOT:**

- Parse configuration
- Discover schema
- Negotiate features
- Compile patterns

**Axiom:** Runtime should feel boring. Boring is fast.

---

# 👁️ Part XVI: Observability Discipline

**Observability explains what happened. It never decides what happened.**

---

## 👁️ 16.1 The Prime Rule

**Premise:** Observability must never participate in correctness.

**Forbidden:**

```go
// BAD: correctness depends on log
log.Info("order processed")
// If log fails, we don't know if order processed!

// BAD: retry depends on metric
if metrics.FailureRate() > 0.5 {
    return ErrCircuitOpen
}
// Metrics can be delayed, sampled, lost
```

**Required:**

```go
// GOOD: proof decides, observability explains
receipt, err := processOrder(ctx, order)
if err != nil {
    log.Error("order failed", "error", err) // Explains
    return err
}
log.Info("order processed", "receipt", receipt.ID) // Explains
return nil // Receipt is proof
```

**Axiom:** Truth must survive when observability is gone.

---

## 👁️ 16.2 Logging Doctrine

**Premise:** Logs are for debugging and audit, not for correctness.

**Rules:**

1. **One log line per request** (entry/exit pattern)
2. **Structured logging** (key-value pairs)
3. **Stable schema** (don't change field names)
4. **Redact secrets** before enqueue
5. **Best effort** (may drop under load)

**Structured Logging:**

```go
import "log/slog"

func (s *Service) Process(ctx context.Context, req Request) error {
    logger := slog.With(
        "request_id", req.ID,
        "user_id", req.UserID,
    )

    logger.Info("processing request")

    result, err := s.doProcess(ctx, req)
    if err != nil {
        logger.Error("request failed", "error", err)
        return err
    }

    logger.Info("request completed",
        "result_id", result.ID,
        "duration_ms", time.Since(start).Milliseconds(),
    )
    return nil
}
```

**Axiom:** A dropped log is acceptable. A dropped proof is not.

---

## 👁️ 16.3 Metrics Doctrine

**Premise:** Metrics show system health. They must not affect behavior.

**Required Metrics:**

| Metric Type | Examples                            |
| ----------- | ----------------------------------- |
| Counters    | requests_total, errors_total        |
| Histograms  | request_duration_seconds            |
| Gauges      | queue_depth, connections_active     |
| Saturation  | pool_exhausted_total, dropped_total |

**Rules:**

1. **Saturation counters mandatory** (know when you're full)
2. **Cardinality bounded** (no unbounded label values)
3. **No allocation** in hot path metrics
4. **Never gate behavior** on metrics

```go
var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "requests_total",
        },
        []string{"method", "status"}, // Bounded cardinality
    )

    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "request_duration_seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
)
```

**Axiom:** If metrics affect correctness, they have failed their purpose.

---

## 👁️ 16.4 Tracing Doctrine

**Premise:** Traces show request flow. They are sampled by definition.

**Rules:**

1. **Propagate trace context** across boundaries
2. **Sample traces** (not every request)
3. **Add spans** at significant boundaries
4. **Include relevant attributes** (IDs, durations)

```go
func (s *Service) Process(ctx context.Context, req Request) error {
    ctx, span := tracer.Start(ctx, "Service.Process")
    defer span.End()

    span.SetAttributes(
        attribute.String("request.id", req.ID),
        attribute.String("user.id", req.UserID),
    )

    // Child span for database
    result, err := s.queryDB(ctx, req)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetAttributes(attribute.String("result.id", result.ID))
    return nil
}
```

**Axiom:** Traces explain latency. They do not guarantee completeness.

---

## 👁️ 16.5 Drop Discipline

**Premise:** Under pressure, observability yields to truth.

**Drop Priority Order:**

1. Debug logs (drop first)
2. Verbose traces
3. Info logs
4. Metrics samples
5. Error logs (drop last)

**Rules:**

1. Drops are **counted** (metric for dropped logs)
2. Drops are **visible** (alert on drop rate)
3. Drops never **block requests**
4. Drops never **block proof creation**

```go
type LogBuffer struct {
    ch      chan LogEntry
    drops   atomic.Int64
}

func (b *LogBuffer) Log(entry LogEntry) {
    select {
    case b.ch <- entry:
        // Logged
    default:
        b.drops.Add(1) // Dropped, counted
    }
}
```

**Axiom:** When under pressure, evidence yields to truth.

---

# 🏛️ Part XVII: Security and Cryptography

**Security is correctness when someone is trying.**

---

## 🛑 17.1 Prime Rule

**Premise:** Never invent security. Never improvise cryptography.

**Rules:**

1. If a standard exists, use it
2. If a standard does not exist, the feature does not ship
3. Never modify standard algorithms
4. Never weaken defaults
5. Never skip validation steps

**Axiom:** Creativity is for products, not cryptography.

---

## 🏛️ 17.2 Threat Model

**Premise:** Assume adversaries exist and are funded.

**Assumptions:**

1. **Input is hostile** - every byte from outside
2. **Clients lie** - never trust client state
3. **Networks observe** - assume MITM exists
4. **Networks replay** - assume requests repeated
5. **Dependencies fail** - assume partial responses
6. **Humans misconfigure** - assume wrong settings

**Axiom:** Security begins by assuming curiosity is funded.

---

## 🏛️ 17.3 Cryptography Doctrine

**Premise:** Use only approved primitives from approved libraries.

**Rules:**

1. Use standard libraries (`crypto/*`) or authorized ecosystem
2. **Never** write custom crypto primitives
3. **Never** modify standard algorithms
4. **Never** weaken defaults
5. **Never** skip validation steps

**Axiom:** If you do not understand the math, you do not change the math.

---

## 🏛️ 17.4 Authorized Primitives

| Category   | Approved                                   |
| ---------- | ------------------------------------------ |
| Hashing    | SHA-256, SHA-3, BLAKE3, Argon2 (passwords) |
| Encryption | AES-GCM, ChaCha20-Poly1305                 |
| Signatures | Ed25519, ECDSA P-256                       |
| KDF        | HKDF, Argon2                               |
| Random     | `crypto/rand` only                         |
| Tokens     | JWT via `github.com/golang-jwt/jwt/v5`     |
| UUIDs      | `github.com/google/uuid`                   |

**Forbidden:**

- MD5, SHA-1 for security (OK for checksums)
- RC4, DES, 3DES
- ECB mode
- Custom anything

**Axiom:** Anything else requires written approval and review.

---

## 🏛️ 17.5 Key Management

**Premise:** Keys are the crown jewels. Protect them accordingly.

**Rules:**

1. Keys are **never logged**
2. Keys are **never committed** to source control
3. Keys are **rotated** on schedule
4. Key scope is **minimal** (least privilege)
5. Compromise is **assumed** and planned for

```go
// BAD: key in code
const apiKey = "sk_live_abc123"

// BAD: key in log
log.Info("using key", "key", apiKey)

// GOOD: key from environment/secret manager
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    log.Fatal("API_KEY required")
}

// GOOD: key reference in log (not value)
log.Info("using key", "key_id", keyID)
```

**Axiom:** Protect keys by limiting their power, not by hiding them.

---

## 🏛️ 17.6 Input Validation

**Premise:** All external input is hostile until proven safe.

**Validation at Boundary:**

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // 1. Size limit
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max

    // 2. Content type
    if r.Header.Get("Content-Type") != "application/json" {
        http.Error(w, "invalid content type", http.StatusBadRequest)
        return
    }

    // 3. Decode
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    // 4. Validate fields
    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 5. Normalize
    req.Email = strings.ToLower(strings.TrimSpace(req.Email))

    // Now safe to process
}
```

**Axiom:** Trust nothing from outside. Validate everything.

---

## 🏛️ 17.7 Authentication vs Authorization

**Premise:** Authentication and authorization are separate concerns.

| Concern        | Question         | Failure Response |
| -------------- | ---------------- | ---------------- |
| Authentication | Who are you?     | 401 Unauthorized |
| Authorization  | Can you do this? | 403 Forbidden    |

**Rules:**

1. Authenticate at the edge (middleware)
2. Authorize at the resource (handler/service)
3. Default deny (if unclear, deny)
4. Log all access decisions

**Axiom:** If access is ambiguous, the answer is no.

---

# 🏗️ Part XVIII: Testing and Verification

**Testing is asking: "How does this fail under pressure, confusion, and
malice?"**

---

## 🏗️ 18.1 Pure Go Testing Standard

**Premise:** We use `testing` directly. No assertion libraries.

**Rationale:**

- Assertion helpers hide values
- Assertion helpers hide control flow
- Assertion helpers add dependencies
- Standard library is sufficient

**Table Tests Are Mandatory:**

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid", "user@example.com", false},
        {"empty", "", true},
        {"no_at", "userexample.com", true},
        {"no_domain", "user@", true},
        {"multiple_at", "user@@example.com", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%q) error = %v, wantErr %v",
                    tt.email, err, tt.wantErr)
            }
        })
    }
}
```

**Axiom:** If a test cannot be expressed as a table, explain why.

---

## 🏗️ 18.2 Test Categories

| Type          | Purpose                                         |
| ------------- | ----------------------------------------------- |
| Unit          | Prove core logic, no I/O, no mocks, no clocks   |
| Determinism   | Same input produces identical output            |
| State Machine | Validate legal transitions, forbid illegal ones |
| Boundary      | Attack ingress with hostile input               |
| Benchmark     | Prove allocs/op, bytes/op, detect regressions   |
| Fuzz          | Automated hostile input generation              |
| Integration   | Test with real dependencies                     |
| Chaos         | Inject failures deliberately                    |

**Unit Test Rules:**

- No I/O
- No network
- No database
- No file system
- No time.Now()
- No random
- Deterministic

**Axiom:** If a unit test flakes, the code is broken.

---

## 🏗️ 18.3 Benchmark Discipline

**Premise:** Performance must be measured, not assumed.

**Required Benchmarks:**

```go
func BenchmarkProcess(b *testing.B) {
    item := createTestItem()
    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _ = Process(item)
    }
}
```

**Allocation Gates:**

```go
func BenchmarkProcess(b *testing.B) {
    // ...
    b.ReportAllocs()
    // CI asserts: allocs/op == 0
}
```

**Run Benchmarks:**

```bash
go test -bench=. -benchmem ./...
```

**Axiom:** What is not measured is not controlled.

---

## 🏗️ 18.4 Fuzz Testing

**Premise:** Fuzzing finds bugs that humans miss.

**When to Fuzz:**

- Parsers
- Validators
- Serializers
- Any code handling external input

**Fuzz Test Pattern:**

```go
func FuzzParseConfig(f *testing.F) {
    // Seed corpus
    f.Add([]byte(`{"timeout": "5s"}`))
    f.Add([]byte(`{}`))
    f.Add([]byte(`invalid`))

    f.Fuzz(func(t *testing.T, data []byte) {
        cfg, err := ParseConfig(data)
        if err != nil {
            return // Invalid input is expected
        }
        // If we got here, config should be usable
        if cfg.Timeout < 0 {
            t.Error("negative timeout")
        }
    })
}
```

**Run Fuzz:**

```bash
go test -fuzz=FuzzParseConfig -fuzztime=60s ./...
```

**Axiom:** Fuzzing reveals edge cases that imagination cannot.

---

## 🏗️ 18.5 Enforcement Checklist

**All code must pass:**

```bash
go test ./...                    # Unit tests
go test -race ./...              # Race detector
go test -count=1 ./...           # No caching
go vet ./...                     # Static analysis
staticcheck ./...                # Extended checks
govulncheck ./...                # Vulnerability scan
go test -bench=. -benchmem ./... # Benchmarks
```

**CI Gates:**

- All tests pass
- No race conditions
- No vet warnings
- No staticcheck warnings
- No known vulnerabilities
- Benchmark allocs within budget

**Axiom:** If it is not in CI, it does not exist.

---

# 🏗️ Part XIX: Process Signals and Humane Shutdown

**Before it is a service, it is a process managed by an operating system.**

---

## 🏗️ 19.1 The Prime Rule

**Premise:** SIGTERM is the normal shutdown request. It is not an error.

**Signal Meaning:**

| Signal  | Meaning                 | Response             |
| ------- | ----------------------- | -------------------- |
| SIGTERM | Please shut down        | Graceful shutdown    |
| SIGINT  | User interrupt (Ctrl+C) | Graceful shutdown    |
| SIGKILL | Die immediately         | Cannot be caught     |
| SIGHUP  | Terminal hangup         | Often: reload config |

**Axiom:** Systems that treat SIGTERM as exceptional will be terminated
exceptionally.

---

## 🏗️ 19.2 Shutdown Contract

**Premise:** Shutdown must be graceful and bounded.

**The Shutdown Sequence:**

1. **Stop accepting** new work
2. **Signal cancellation** to in-flight work
3. **Wait for drain** (with timeout)
4. **Flush telemetry** and close connections
5. **Exit cleanly**

```go
func main() {
    // Setup signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

    // Start server
    srv := &http.Server{Addr: ":8080", Handler: handler}
    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Error("server error", "error", err)
        }
    }()

    // Wait for signal
    <-sig
    log.Info("shutdown signal received")

    // Graceful shutdown with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
    defer shutdownCancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Error("shutdown error", "error", err)
    }

    log.Info("shutdown complete")
}
```

**Axiom:** Skipping steps leaves ghost state behind.

---

## 🏗️ 19.3 Bounded Shutdown

**Premise:** Shutdown must complete within a deadline.

**Rules:**

1. Define a **maximum drain time** (e.g., 30 seconds)
2. **Enforce strictly** (exit even if work remains)
3. **Refuse new work** first
4. **Escalate** when deadline approaches

```go
const shutdownTimeout = 30 * time.Second

func shutdown(ctx context.Context, components []Shutdowner) error {
    ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
    defer cancel()

    // Shutdown in reverse order of startup
    for i := len(components) - 1; i >= 0; i-- {
        if err := components[i].Shutdown(ctx); err != nil {
            log.Warn("component shutdown error",
                "component", components[i].Name(),
                "error", err,
            )
        }
    }

    return ctx.Err()
}
```

**Axiom:** A shutdown that can hang forever is a deadlock.

---

# 🏗️ Part XX: The Breathing Cycle

**Codebases naturally decay towards chaos.** **We prevent this with strict
expansion and contraction.**

---

## 🏗️ 20.1 The Exhale Rule

**Premise:** Refactoring is mandatory before sealing an iteration.

**The Cycle:**

| Phase             | Action                           |
| ----------------- | -------------------------------- |
| Build (Inhale)    | Write code to make the test pass |
| Verify            | Ensure correctness               |
| Refactor (Exhale) | Clean the mess immediately       |
| Seal              | Only clean code is committed     |

**During Exhale:**

- Shadowed vars? Fix
- Args > 3? Use struct
- No Context? Add it
- Magic values? Extract constants
- Deep nesting? Flatten
- Duplicate code? Extract function

**Axiom:** Debt does not roll over to the next iteration.

---

## 🏗️ 20.2 Refactoring Rules

**Premise:** Refactoring is behavior-preserving transformation.

**Rules:**

1. Tests pass before refactoring
2. Tests pass after refactoring
3. No behavior changes during refactoring
4. Commit refactoring separately from features

**Refactoring vs Features:**

| Activity            | Refactoring | Feature |
| ------------------- | ----------- | ------- |
| Rename variable     | Yes         | No      |
| Extract function    | Yes         | No      |
| Add new behavior    | No          | Yes     |
| Fix bug             | No          | Yes     |
| Improve performance | Maybe       | Maybe   |

**Axiom:** Clean code is not a destination. It is maintenance.

---

# 🛡️ Part XXI: Resilience Patterns

**Failure is not exceptional. Failure is steady state.** **Resilience is how
systems survive reality.**

---

## 🛡️ 21.1 The Resilience Stack

**Premise:** Resilience is layered defense. Each layer catches what the previous
layer missed.

| Layer               | Purpose                                    |
| ------------------- | ------------------------------------------ |
| **Timeouts**        | Bound waiting; prevent resource starvation |
| **Retries**         | Recover from transient failure             |
| **Backoff**         | Prevent retry storms; respect recovery     |
| **Circuit Breaker** | Stop calling failed dependencies           |
| **Bulkhead**        | Isolate failures; prevent cascade          |
| **Fallback**        | Degrade gracefully when all else fails     |

**Composition Order (Outside to Inside):**

```
Request
  └─► Timeout (outermost)
        └─► Bulkhead
              └─► Circuit Breaker
                    └─► Retry + Backoff
                          └─► Actual Call
```

**Axiom:** Resilience is an onion. Layers must be in the right order.

---

## 🛡️ 21.2 Timeout Doctrine

**Premise:** Every external call has a timeout. No exceptions.

**Rules:**

1. Timeout is the **first** thing you set
2. Timeouts are **budgeted**, not guessed
3. Child timeout ≤ parent remaining
4. Timeout includes retries

**Implementation:**

```go
func (c *Client) Call(ctx context.Context, req Request) (*Response, error) {
    // Ensure timeout exists
    if _, ok := ctx.Deadline(); !ok {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, c.defaultTimeout)
        defer cancel()
    }

    return c.doCall(ctx, req)
}
```

**Axiom:** A missing timeout is an infinite timeout.

---

## 🛡️ 21.3 Retry Doctrine

**Premise:** Retries are loops with side effects. They require idempotency.

**Rules:**

1. **Max attempts** declared (typically 2-3)
2. **Max total time** declared
3. Retry only **transient** failures
4. Never retry **validation** or **auth** failures
5. Retries require **idempotency**

**Transient vs Permanent:**

| Transient (Retry)       | Permanent (Fail Fast) |
| ----------------------- | --------------------- |
| Connection refused      | 400 Bad Request       |
| 503 Service Unavailable | 401 Unauthorized      |
| 429 Too Many Requests   | 403 Forbidden         |
| Timeout                 | 404 Not Found         |
| Connection reset        | 422 Unprocessable     |

**Implementation:**

```go
func withRetry(ctx context.Context, fn func() error) error {
    const maxAttempts = 3

    var lastErr error
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if ctx.Err() != nil {
            return ctx.Err()
        }

        err := fn()
        if err == nil {
            return nil
        }

        if !isTransient(err) {
            return err // Don't retry permanent errors
        }

        lastErr = err
        time.Sleep(backoff(attempt))
    }
    return lastErr
}
```

**Axiom:** Retrying permanent failures amplifies load.

---

## 🛡️ 21.4 Backoff Doctrine

**Premise:** Backoff prevents retry storms from killing recovering systems.

**Rules:**

1. Use **exponential backoff**
2. Add **jitter** (mandatory)
3. Cap **maximum delay**
4. Never use fixed delays

**Implementation:**

```go
func backoff(attempt int) time.Duration {
    const (
        base = 100 * time.Millisecond
        max  = 10 * time.Second
    )

    // Exponential: base * 2^attempt
    delay := base << attempt
    if delay > max {
        delay = max
    }

    // Jitter: 50-100% of delay
    jitter := delay/2 + time.Duration(rand.Int63n(int64(delay/2)))
    return jitter
}
```

**Axiom:** Without jitter, all clients retry in sync.

---

## 🛡️ 21.5 Circuit Breaker Doctrine

**Premise:** A circuit breaker stops calling a dependency that is failing.

**The Three States:**

| State         | Behavior                                   |
| ------------- | ------------------------------------------ |
| **Closed**    | Normal operation; requests flow through    |
| **Open**      | Fail fast; no requests sent                |
| **Half-Open** | Probe with limited requests; test recovery |

**Implementation Pattern:**

```go
type CircuitBreaker struct {
    mu          sync.Mutex
    state       State
    failures    int
    lastFailure time.Time
    threshold   int
    timeout     time.Duration
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen
            return true
        }
        return false
    case StateHalfOpen:
        return true // Allow probe
    }
    return false
}

func (cb *CircuitBreaker) Record(success bool) {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if success {
        cb.failures = 0
        cb.state = StateClosed
    } else {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.threshold {
            cb.state = StateOpen
        }
    }
}
```

**Axiom:** When a dependency is dying, stop hitting it.

---

## 🛡️ 21.6 Bulkhead Doctrine

**Premise:** A bulkhead isolates failures to prevent cascade.

**Bulkhead Types:**

| Type            | Mechanism                             |
| --------------- | ------------------------------------- |
| **Semaphore**   | Limit concurrent calls per dependency |
| **Worker Pool** | Dedicated workers per dependency      |
| **Queue**       | Bounded buffer per dependency         |

**Semaphore Bulkhead:**

```go
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(max int) *Bulkhead {
    return &Bulkhead{sem: make(chan struct{}, max)}
}

func (b *Bulkhead) Call(ctx context.Context, fn func() error) error {
    select {
    case b.sem <- struct{}{}:
        defer func() { <-b.sem }()
        return fn()
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Axiom:** When one dependency drowns, the bulkhead contains it.

---

## 🛡️ 21.7 Fallback Doctrine

**Premise:** Fallbacks provide degraded service when primary fails.

**Fallback Hierarchy:**

| Level   | Strategy                               |
| ------- | -------------------------------------- |
| Primary | Normal operation                       |
| Cache   | Return stale data (if contract allows) |
| Default | Return safe static default             |
| Degrade | Reduce functionality                   |
| Reject  | Fast failure with clear error          |

**Implementation:**

```go
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // Primary: database
    user, err := s.db.GetUser(ctx, id)
    if err == nil {
        return user, nil
    }

    // Fallback 1: cache
    if cached, ok := s.cache.Get(id); ok {
        s.metrics.Inc("user.fallback.cache")
        return cached, nil
    }

    // Fallback 2: default
    s.metrics.Inc("user.fallback.default")
    return s.defaultUser(), nil
}
```

**Axiom:** A fallback that is not tested will fail when needed.

---

# 📦 Part XXII: Package Layout

**Where code lives determines how code evolves.**

---

## 📦 22.1 The Standard Layout

**Premise:** Consistent layout enables navigation without thinking.

```
svc/
├── cmd/
│   └── svc/
│       └── main.go           # Entrypoint only
├── internal/
│   ├── domain/               # Pure business logic (no deps)
│   │   ├── user.go
│   │   └── user_test.go
│   ├── service/              # Application orchestration
│   │   ├── user_service.go
│   │   └── user_service_test.go
│   ├── repository/           # Data access interfaces
│   │   └── user_repo.go
│   ├── adapter/              # External implementations
│   │   ├── postgres/
│   │   │   └── user_repo.go
│   │   └── redis/
│   │       └── cache.go
│   ├── transport/            # HTTP/gRPC handlers
│   │   └── http/
│   │       ├── handler.go
│   │       └── handler_test.go
│   └── config/               # Configuration
│       └── config.go
├── pkg/                      # Public libraries (if any)
│   └── client/
│       └── client.go
├── go.mod
└── go.sum
```

**Axiom:** Good layout makes the right thing easy.

---

## 📦 22.2 Dependency Direction

**Premise:** Dependencies flow inward, never outward.

```
transport (shell)
    │
    ▼
service (orchestrator)
    │
    ▼
domain (core)
    │
    ▼
  (nothing)
```

**Rules:**

1. **domain** imports nothing from the project
2. **service** depends on domain and repository interfaces
3. **adapter** implements repository interfaces
4. **transport** depends on service
5. **cmd** wires everything

**Axiom:** The core must be extractable without touching the shell.

---

## 📦 22.3 Interface Placement

**Premise:** Interfaces belong with consumers, not producers.

```go
// internal/repository/user.go
package repository

// Interface defined by consumer
type UserRepository interface {
    Get(ctx context.Context, id string) (*domain.User, error)
    Save(ctx context.Context, user *domain.User) error
}

// internal/adapter/postgres/user.go
package postgres

// Implementation
type userRepo struct { db *sql.DB }

func NewUserRepository(db *sql.DB) repository.UserRepository {
    return &userRepo{db: db}
}
```

**Axiom:** The consumer defines the contract. The producer fulfills it.

---

## 📦 22.4 The internal/ Boundary

**Premise:** `internal/` is compiler-enforced privacy. Use it.

**Rules:**

- Code in `internal/` cannot be imported from outside the module
- Default to `internal/`
- Promote to `pkg/` only when external use is needed

**Axiom:** Default private. Public is a deliberate choice.

---

# 🏛️ Part XXIII: Foundations and Philosophy

**Foundation is what remains true regardless of preference.**

---

## 🏛️ 23.1 The Actors

**Premise:** Every running system involves actors with fixed contracts. Blink
philosophy refuses to fight any of them.

| Actor               | Contract                                         |
| ------------------- | ------------------------------------------------ |
| Physics & Time      | Speed of light is finite; time moves forward     |
| CPU & Cache         | Hierarchical; locality determines speed          |
| Memory & Storage    | Finite; access costs time                        |
| Operating System    | Syscalls cross boundaries; schedulers are unfair |
| Networks & Protocol | Packets drop, reorder, duplicate; latency varies |
| Go Language         | Spec defines behavior; undefined is forbidden    |
| Humans              | Make mistakes; forget; misconfigure              |

**Axiom:** Everything has a place. Nothing needs to be wrestled.

---

## 🏛️ 23.2 Blink Sympathy

**Premise:** When we align with actors rather than fight them, their
improvements become our improvements.

**Examples:**

- **Go Compiler:** Correct code gets faster with each release
- **TCP/IP:** Applications benefit from router upgrades
- **CPUs:** Cache-friendly code accelerates with new hardware
- **Runtime:** GC improvements help without code changes

**Axiom:** When we stop fighting reality, reality starts working for us.

---

## 🏛️ 23.3 The Litmus Tests

**If a Go upgrade breaks your system:** You were using undefined behavior.

**If new hardware does not make you faster:** You were fighting physics.

**If swapping a dependency causes cascading changes:** Your boundary leaked.

**Axiom:** This is how systems age without collapsing.

---

# 📚 Appendix A: The Authorized Ecosystem

---

## A.1 Preference Order

1. **Go Standard Library**
2. **Google Maintained Packages**
3. **Uber Maintained Packages**
4. **Well-Audited Ecosystem**

**Axiom:** Earlier is more predictable under load, time, and failure.

---

## A.2 Approved Defaults

| Capability      | Preferred                      |
| --------------- | ------------------------------ |
| Routing         | `net/http` (Go 1.22+)          |
| Logging         | `log/slog`                     |
| Testing         | `testing`                      |
| Test Comparison | `github.com/google/go-cmp`     |
| GCP Clients     | `cloud.google.com/go/...`      |
| UUIDs           | `github.com/google/uuid`       |
| JWT             | `github.com/golang-jwt/jwt/v5` |
| Backoff         | `github.com/cenkalti/backoff`  |
| Mocks           | `go.uber.org/mock`             |
| Context Groups  | `golang.org/x/sync/errgroup`   |

---

## A.3 Never Allowed

- Home-grown cryptography
- Manual JWT parsing
- Hidden retries
- Silent global state
- Runtime code generation in core

**Axiom:** These are correctness violations, not style issues.

---

# 📚 Appendix B: Scale Math Reference

---

## B.1 Cost Multiplication Table

| Per-Request Cost     | At 1M req/sec              |
| -------------------- | -------------------------- |
| 1 allocation         | 1M allocations/sec         |
| 1 microsecond stall  | 1 full CPU core            |
| 1 cache miss         | Millions of memory stalls  |
| 1 KB response        | 1 GB/sec network           |
| 100ms latency target | 100K concurrent operations |

---

## B.2 The Memory Hierarchy

| Level    | Latency | Size  |
| -------- | ------- | ----- |
| L1 Cache | ~1ns    | 64KB  |
| L2 Cache | ~3ns    | 256KB |
| L3 Cache | ~10ns   | 8MB   |
| RAM      | ~100ns  | GBs   |
| SSD      | ~100μs  | TBs   |
| Network  | ~1ms    | ∞     |

---

## B.3 The Law of Avoidance

- Work avoided cannot fail
- Work avoided cannot allocate
- Work avoided cannot block
- Work avoided cannot widen the tail

**Axiom:** The fastest code is no code.

---

# 📚 Appendix C: Reference Patterns

---

## C.1 Deterministic Map Iteration

```go
func sortedKeys[V any](m map[string]V) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    slices.Sort(keys)
    return keys
}
```

---

## C.2 Child Budget

```go
func childBudget(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
    if dl, ok := ctx.Deadline(); ok {
        rem := time.Until(dl)
        if rem <= 0 {
            ctx, cancel := context.WithCancel(ctx)
            cancel()
            return ctx, cancel
        }
        if d > rem {
            d = rem
        }
    }
    return context.WithTimeout(ctx, d)
}
```

---

## C.3 Bounded Concurrency

```go
func callAll(ctx context.Context, fns []func() error) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(8)
    for _, fn := range fns {
        fn := fn
        g.Go(func() error { return fn() })
    }
    return g.Wait()
}
```

---

## C.4 Clone Before Retain

```go
func cloneBytes(b []byte) []byte {
    if len(b) == 0 {
        return nil
    }
    out := make([]byte, len(b))
    copy(out, b)
    return out
}
```

---

## C.5 Graceful Shutdown

```go
func waitForShutdown(ctx context.Context, srv *http.Server) error {
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
    defer signal.Stop(sig)

    select {
    case <-sig:
    case <-ctx.Done():
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return srv.Shutdown(shutdownCtx)
}
```

---

## C.6 Request Validation

```go
type CreateRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateRequest) Validate() error {
    if r.Name == "" {
        return errors.New("name required")
    }
    if len(r.Name) > 255 {
        return errors.New("name too long")
    }
    if r.Email == "" {
        return errors.New("email required")
    }
    if !isValidEmail(r.Email) {
        return errors.New("invalid email")
    }
    return nil
}

func (r *CreateRequest) Normalize() {
    r.Name = strings.TrimSpace(r.Name)
    r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}
```

---

## C.7 Result Type

```go
type Result[T any] struct {
    Value T
    Err   error
}

func Ok[T any](v T) Result[T] {
    return Result[T]{Value: v}
}

func Err[T any](err error) Result[T] {
    return Result[T]{Err: err}
}

func (r Result[T]) IsOk() bool {
    return r.Err == nil
}
```

---

# 📚 Appendix D: Go Specification Anchors

---

## D.1 Map Semantics

- Iteration order is **not specified**
- Not stable across runs or within a run
- Removals during iteration may suppress keys

**Axiom:** If order matters, convert to sorted slice.

---

## D.2 Select Semantics

- If multiple cases ready, choice is **pseudo-random**
- All channel operands evaluated once at select entry

**Axiom:** Never rely on which case fires.

---

## D.3 Nil Channel Semantics

- Nil channel is never ready
- Send/receive on nil blocks forever
- Select with only nil channels blocks forever

**Axiom:** Accidental nil channel is a deadlock.

---

## D.4 String Semantics

- Strings are immutable byte sequences
- Indexing yields **bytes**, not runes
- `len()` returns byte count, not rune count

**Axiom:** UTF-8 must be validated at boundaries.

---

# 📚 Appendix E: Quick Reference

---

## E.1 The Stack

1. Correctness
2. Security
3. Determinism
4. Boundedness
5. Tail Latency
6. Throughput
7. Convenience

---

## E.2 The Bans

- Unchecked errors
- Global mutable state
- Panics for expected conditions
- Magic values
- `init()` with logic
- Nil context
- Unbounded loops
- Unbounded queues
- Reflection in hot paths
- `fmt` in hot paths

---

## E.3 The Musts

- Context first parameter
- Errors wrapped with context
- Boundaries validate everything
- Budgets enforced
- Timeouts on all external calls
- Graceful shutdown
- Tests pass with `-race`
- Benchmarks track allocations

---

> **"If you cannot prove it, it did not happen."**

> **"Boring is success. Clever is debt."**

> **"When we stop fighting reality, reality starts working for us."**

> **"Ghost state is forbidden."**

> **"Eye fatigue is the failure mode of documentation."**
