package core

// PeachfuzzRunEvidenceMaxBytes is the shared hard limit for one signed,
// immutable Peachfuzz run-evidence structure at every producer, persistence,
// transport, and fold boundary.
const PeachfuzzRunEvidenceMaxBytes = 16 << 10
