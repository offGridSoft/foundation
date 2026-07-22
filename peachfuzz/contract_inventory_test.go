package peachfuzz

import (
	foundationcore "github.com/offGridSoft/foundation/v2026/core"
	foundationfuzz "github.com/offGridSoft/foundation/v2026/fuzz"
)

var (
	_ foundationcore.Validatable   = ProjectID{}
	_ foundationcore.Validatable   = PackageImportPath{}
	_ foundationcore.Validatable   = FuzzTargetName{}
	_ foundationcore.Validatable   = RunID{}
	_ foundationcore.Validatable   = MachineID{}
	_ foundationcore.Validatable   = CommitSHA{}
	_ foundationcore.Validatable   = RunOutcomeUnknown
	_ foundationcore.Validatable   = RunEvidence{}
	_ foundationcore.Validatable   = ExecutionObservation{}
	_ foundationcore.Validatable   = FuzzSidecarKindUnknown
	_ foundationcore.Validatable   = FuzzSidecarStateUnknown
	_ foundationcore.Validatable   = FuzzSidecarRef{}
	_ foundationcore.Validatable   = FuzzEvidence{}
	_ foundationcore.Validatable   = foundationfuzz.ArtifactKindUnknown
	_ foundationcore.Validatable   = FuzzArtifactIndexStateUnknown
	_ foundationcore.Validatable   = FuzzArtifact{}
	_ foundationcore.Validatable   = FuzzArtifactIndex{}
	_ foundationcore.CanonicalBody = RunEvidence{}
	_ foundationcore.Validatable   = SignedRunEvidence{}
	_ foundationcore.Validatable   = MachineEvidenceIdentity{}
	_ foundationcore.Validatable   = RunEvidenceDescriptor{}
	_ foundationcore.Validatable   = RunEvidenceUploadRequest{}
	_ foundationcore.Validatable   = RunEvidenceUploadGrant{}
	_ foundationcore.Validatable   = RunEvidenceUploadDispositionUnknown
	_ foundationcore.Validatable   = RunEvidenceUploadResponse{}
	_ foundationcore.Validatable   = EffortNanoseconds{}
	_ foundationcore.APIBody       = RunEvidenceUploadRequest{}
	_ foundationcore.APIBody       = RunEvidenceUploadGrant{}
	_ foundationcore.APIBody       = RunEvidenceUploadResponse{}
	_ foundationcore.APIBody       = ProjectSnapshot{}
)
