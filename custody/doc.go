// Package custody defines the typed plug between witness and Offgrid custody:
// session-open requests, signed storage upload instructions, finalize reports,
// and receipt bodies. It never carries artifact bytes. Artifact bytes move
// directly from the client to cloud storage using server-minted upload targets;
// Offgrid verifies storage state before signing a receipt.
package custody
