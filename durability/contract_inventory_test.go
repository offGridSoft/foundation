package durability

import "github.com/offGridSoft/foundation/v2026/core"

type durabilityProtocolFact[T any] struct{}
type durabilityInternalFlow[T any] struct{}
type durabilityCapability[T any] struct{}

type durabilityContractInventory struct {
	WriteRequest        durabilityProtocolFact[WriteRequest]
	WriteOutcome        durabilityInternalFlow[WriteOutcome]
	CommitResult        durabilityInternalFlow[CommitResult]
	DirectoryRequest    durabilityProtocolFact[DirectoryRequest]
	ReadRequest         durabilityProtocolFact[ReadRequest]
	AppendRequest       durabilityProtocolFact[AppendRequest]
	TreeRemovalRequest  durabilityProtocolFact[TreeRemovalRequest]
	FileRemovalRequest  durabilityProtocolFact[FileRemovalRequest]
	ContentStageRequest durabilityProtocolFact[ContentStageRequest]
	Stage               durabilityCapability[*Stage]
	AppendWriter        durabilityCapability[*AppendWriter]
	ContentStage        durabilityCapability[*ContentStage]
}

var (
	_ core.Validatable = WriteRequest{}
	_ core.Validatable = WriteOutcome{}
	_ core.Validatable = CommitResult{}
	_ core.Validatable = DirectoryRequest{}
	_ core.Validatable = ReadRequest{}
	_ core.Validatable = AppendRequest{}
	_ core.Validatable = TreeRemovalRequest{}
	_ core.Validatable = FileRemovalRequest{}
	_ core.Validatable = ContentStageRequest{}
	_                  = durabilityContractInventory{}
)
