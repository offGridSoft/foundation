# Foundation Package Specifications

Status: Draft for review

[Foundation Specification](../../SPEC.md) is authoritative for architecture and
cross-package doctrine. These documents define package ownership and proof
obligations.

## Primitive packages

| Package | Code status | Spec status | Responsibility |
| --- | --- | --- | --- |
| [`core`](../../core/SPEC.md) | Exists; ownership cleanup pending | Draft reviewed; comments resolved | Canonical shared contracts and stable errors |
| [`temporal`](../../temporal/SPEC.md) | Permanent primitive implemented; consumer migration pending | Reviewed implementation | Nanosecond instants, durations, aggregates, and projections |
| [`currency`](../../currency/SPEC.md) | Permanent primitive implemented; live adapter conformance pending | Reviewed implementation | Currency codes, exact monetary amounts, and checked arithmetic |
| `contextstate` | Target rename of `contextcheck` | Not written | Context boundary and terminal state |
| `exchange` | Exists under permanent name | Not written | Typed HTTP transmission and reception |
| `objectstore` | Target; code currently scattered | Not written | Signed, verified, streaming object transfer |
| `durability` | Exists under permanent name | Not written | Durable filesystem operations |
| `hostresource` | Exists under permanent name | Not written | Disk and memory assessment |
| `shutdown` | Exists under permanent name | Not written | Graceful cleanup and signal escalation |
| `keygen` | Exists under permanent name | Not written | Typed CSPRNG key and secret generation |
| `garble` | Target; code currently scattered | Not written | Custody and deterministic Garble release inputs |
| `probe` | Required target for `v2026.0.0` | Root boundary approved; package spec not written | Explicit primitive conformance reports |

## Existing protocol-package disposition

These packages are migration inventory, not approved Foundation families.
Product-owned contracts move to their owning modules; only independently
product-neutral primitives may remain after review.

| Package | Code status | Disposition |
| --- | --- | --- |
| `custody` | Exists | Extract product contracts |
| `fuzz` | Exists | Audit for product-neutral primitive admission |
| `license` | Exists | Extract product contracts |
| `peachfuzz` | Exists | Extract entirely |
| `release` | Exists | Audit for product-neutral primitive admission |
| `workloadidentity` | Exists | Audit for product-neutral primitive admission |

## Support packages

| Package | Code status | Spec status |
| --- | --- | --- |
| `durabilitytest` | Exists | Not written |
| `testserial` | Exists | Not written |

Each package name becomes a link when its local `SPEC.md` exists. Status changes
only after the corresponding ledger evidence changes.
