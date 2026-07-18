package peachfuzz

import foundationcore "github.com/offGridSoft/foundation/v2026/core"

var (
	_ foundationcore.Validatable = ProjectID{}
	_ foundationcore.Validatable = PackageImportPath{}
	_ foundationcore.Validatable = FuzzTargetName{}
	_ foundationcore.Validatable = RunID{}
	_ foundationcore.Validatable = MachineID{}
	_ foundationcore.Validatable = CommitSHA{}
	_ foundationcore.Validatable = RunOutcomeUnknown
	_ foundationcore.Validatable = RunStatsSchemaUnknown
	_ foundationcore.Validatable = RunStats{}
	_ foundationcore.APIBody     = RunStatsReceipt{}
	_ foundationcore.APIBody     = ProjectSnapshot{}
)
