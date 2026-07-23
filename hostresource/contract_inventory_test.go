package hostresource

import "github.com/offGridSoft/foundation/v2026/core"

type hostResourceProtocolFact[T any] struct{}
type hostResourceObservation[T any] struct{}

type hostResourceContractInventory struct {
	DiskAssessmentRequest   hostResourceProtocolFact[DiskAssessmentRequest]
	DiskAssessment          hostResourceObservation[DiskAssessment]
	DiskCapacity            hostResourceObservation[DiskCapacity]
	DiskPressurePolicy      hostResourceProtocolFact[DiskPressurePolicy]
	MemoryAssessmentRequest hostResourceProtocolFact[MemoryAssessmentRequest]
	MemoryAssessment        hostResourceObservation[MemoryAssessment]
	MemorySnapshot          hostResourceObservation[MemorySnapshot]
	MemoryLimit             hostResourceProtocolFact[MemoryLimit]
	RuntimeOOMEvidence      hostResourceObservation[RuntimeOOMEvidence]
	TreeUsageRequest        hostResourceProtocolFact[TreeUsageRequest]
	TreeUsage               hostResourceObservation[TreeUsage]
}

var (
	_ core.Validatable = DiskAssessmentRequest{}
	_ core.Validatable = DiskAssessment{}
	_ core.Validatable = DiskCapacity{}
	_ core.Validatable = DiskPressurePolicy{}
	_ core.Validatable = MemoryAssessmentRequest{}
	_ core.Validatable = MemoryAssessment{}
	_ core.Validatable = MemorySnapshot{}
	_ core.Validatable = MemoryLimit{}
	_ core.Validatable = RuntimeOOMEvidence{}
	_ core.Validatable = TreeUsageRequest{}
	_ core.Validatable = TreeUsage{}
	_                  = hostResourceContractInventory{}
)
