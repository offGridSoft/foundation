package core

// PeachfuzzRunEvidenceMaxBytes is the shared hard limit for one signed,
// immutable Peachfuzz run-evidence structure at every producer, persistence,
// transport, and fold boundary.
const PeachfuzzRunEvidenceMaxBytes = 16 << 10

// PeachfuzzRunEvidenceProtocolName is the current clean-break run atom. The
// name changes when the closed shape changes; every SchemaID still uses the
// one calendar-generation authority required across Foundation.
const PeachfuzzRunEvidenceProtocolName = "fuzz-run-evidence"
