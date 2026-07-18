package custody

type custodyProtocolFact[T any] struct{}
type custodySignedProjection[T any] struct{}
type custodyCapabilityWrapper[T any] struct{}
type custodyTypedError[T any] struct{}
type custodyInternalFlow[T any] struct{}

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
	// RFC 3161 network slice: the message imprint is a wire fact DER-encoded
	// inside the TimeStampReq by EncodeRFC3161TimestampQuery; the client is a
	// transport capability whose reply bytes are only accepted through the
	// RFC3161Response/RFC3161Token constructors.
	TimestampQueryImprint custodyProtocolFact[rfc3161MessageImprint]
	TimestampClient       custodyCapabilityWrapper[TimestampClient]
	TimestampHTTPFailure  custodyTypedError[TimestampHTTPError]
	// Transfer slice: the client drives open/upload/finalize/download over
	// HTTP; upload input and the byte counter are internal flow on the
	// streaming path, never wire payloads.
	CustodyClient      custodyCapabilityWrapper[Client]
	CustodyEndpoints   custodyCapabilityWrapper[Endpoints]
	UploadInput        custodyInternalFlow[UploadArtifactInput]
	UploadByteCounter  custodyInternalFlow[countingReader]
	CustodyHTTPFailure custodyTypedError[CustodyHTTPError]
	CustodyAPIFailure  custodyTypedError[CustodyAPIError]
	// Retrieval slice: the download grant is the SSOT typed path shared by
	// CLI fetch and portal download.
	DownloadRequest custodyProtocolFact[DownloadRequest]
	DownloadTarget  custodyProtocolFact[DownloadTarget]
	DownloadGrant   custodySignedProjection[DownloadGrantBody]
}

var _ = custodyContractInventory{}
