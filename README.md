# Foundation

`foundation` is private Off Grid Software infrastructure. It is not an SDK, not
an integration surface, and not licensed for third-party use, copying, reverse
engineering, redistribution, or reuse.

## Specifications

[SPEC.md](SPEC.md) defines Foundation's architecture, primitive model,
cross-platform contract, and admission rules. Focused package specifications
live in [_docs/specs](_docs/specs/README.md).

Work is tracked in [.ledger_pending.md](.ledger_pending.md) and verified
capabilities in [.ledger_completed.md](.ledger_completed.md).

## Doctrine

No package enters `foundation` unless it owns a cross-product, compiler-visible
contract used by at least two Off Grid Software products or by one product and
the Off Grid Software service that verifies or issues the same wire contract.

Every contract in this module must be single source of truth, compiler-owned,
and validated at its ownership boundary. Product-neutral, cross-domain
invariants and stable error identities belong in `core`; temporal, units,
exchange, durability, and other domain invariants belong in their precise
primitive package. Product packages must not import each other merely to share
strings, paths, errors, protocol values, field names, or validation rules.

Contracts must be typed structs, typed enums, typed errors, package constants,
and `Validate` methods. No loose maps, magic literals, duplicated protocol
strings, hidden naming conventions, or string-matched errors. Signed wire
contracts must have canonical byte-form tests. Current software may evolve
cleanly; retained historical releases are the compatibility boundary for old
artifacts.
