package peachfuzz

import foundationcore "github.com/offGridSoft/foundation/v2026/core"

var (
	_ foundationcore.Validatable   = ProjectID{}
	_ foundationcore.Validatable   = PackageImportPath{}
	_ foundationcore.Validatable   = FuzzTargetName{}
	_ foundationcore.Validatable   = RunID{}
	_ foundationcore.Validatable   = MachineID{}
	_ foundationcore.Validatable   = CommitSHA{}
	_ foundationcore.Validatable   = RunOutcomeUnknown
	_ foundationcore.Validatable   = RunEvidence{}
	_ foundationcore.CanonicalBody = RunEvidence{}
	_ foundationcore.Validatable   = MachineContribution{}
	_ foundationcore.Validatable   = ProjectContributionSet{}
	_ foundationcore.Validatable   = ContributionReceipt{}
	_ foundationcore.APIBody       = MachineContribution{}
	_ foundationcore.APIBody       = ProjectSnapshot{}
	_ foundationcore.APIBody       = ContributionReceipt{}
)
