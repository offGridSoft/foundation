// Package license defines the signed lease and check-in wire contracts shared
// by the bug and witness clients and by the Offgrid service that issues them.
//
// Crypto exception: this package calls stdlib crypto/ed25519 (verify only) and
// crypto/rand directly instead of routing through professor. That is deliberate
// and load-bearing. license ships inside released customer binaries, which
// cannot depend on the server-side professor. Clients only ever VERIFY, against
// pinned public keys held in a core.SigningKeyring; all lease SIGNING happens
// server-side in Offgrid via professor over this package's Canonical() bytes. No
// private key material exists anywhere in foundation.
//
// Product trust boundary: foundation verifies signed bytes and decides gate
// outcomes after the product has resolved trust. Product code owns identity and
// rollback checks such as bug's device binding and high-water clock state, then
// passes GateInput.Trust as a typed LeaseTrust. A zero LeaseTrust fails closed.
package license
