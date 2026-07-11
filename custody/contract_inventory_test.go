package custody

type custodyProtocolFact[T any] struct{}
type custodySignedProjection[T any] struct{}

// custodyContractInventory classifies every custody record touched by this
// protocol reset so later wiring changes remain visible to the compiler.
type custodyContractInventory struct {
	OpenRequest    custodyProtocolFact[SessionOpenRequest]
	OpenResponse   custodyProtocolFact[SessionOpenResponse]
	UploadGrant    custodyProtocolFact[SessionUploadGrant]
	UploadTarget   custodyProtocolFact[UploadTarget]
	Retention      custodyProtocolFact[RetentionPolicy]
	UploadedObject custodyProtocolFact[UploadedObject]
	Finalize       custodyProtocolFact[FinalizeRequest]
	Receipt        custodySignedProjection[ReceiptBody]
	ReceiptID      custodyProtocolFact[ReceiptID]
}

var _ = custodyContractInventory{}
