package exchange

import "github.com/offGridSoft/foundation/v2026/core"

type protocolFact[T any] struct{}
type internalFlow[T any] struct{}
type capabilityWrapper[T any] struct{}
type typedFailure[T any] struct{}

type exchangeContractInventory struct {
	ServerPolicy    protocolFact[ServerPolicy]
	Received        internalFlow[Received[receiveFixture]]
	ServerResponse  protocolFact[ServerResponse[responseFixture]]
	Client          capabilityWrapper[Client]
	ClientPolicy    protocolFact[ClientPolicy]
	Request         protocolFact[Request[receiveFixture]]
	Response        protocolFact[Response[responseFixture]]
	ResponseError   typedFailure[ResponseError]
	RetryExhausted  typedFailure[RetryExhaustedError]
	StatusError     typedFailure[StatusError]
	AttemptResult   internalFlow[attemptResult[responseFixture]]
	SendAttempt     internalFlow[sendAttemptInput[receiveFixture]]
	ResponseRead    internalFlow[responseReadInput]
	DecodedResponse internalFlow[decodedResponseInput[responseFixture]]
	BoundedPolicy   protocolFact[BoundedPolicy]
	BoundedRequest  protocolFact[BoundedRequest[core.APIEndpoint]]
	BoundedResponse internalFlow[BoundedResponse]
	HeaderSelection protocolFact[HeaderSelection]
	StreamPolicy    protocolFact[StreamPolicy]
	StreamUpload    protocolFact[StreamUploadRequest[core.APIEndpoint]]
	StreamDownload  protocolFact[StreamDownloadRequest[core.APIEndpoint]]
	StreamResponse  internalFlow[StreamResponse]
	StreamCounter   internalFlow[streamCounter]
	BoundedCopy     internalFlow[boundedCopyState]
	BoundedRead     internalFlow[boundedReadState]
	CommonRequest   internalFlow[commonRequestContract]
	UploadResponse  internalFlow[streamUploadResponseInput[core.APIEndpoint]]
	SystemClock     capabilityWrapper[systemClock]
	CryptoJitter    capabilityWrapper[cryptoJitter]
	TimerWaiter     capabilityWrapper[timerWaiter]
}

var _ core.Validatable = ServerPolicy{}
var _ core.Validatable = Received[receiveFixture]{}
var _ core.Validatable = ServerResponse[responseFixture]{}
var _ core.Validatable = Client{}
var _ core.Validatable = ClientPolicy{}
var _ core.Validatable = Request[receiveFixture]{}
var _ core.Validatable = Response[responseFixture]{}
var _ = exchangeContractInventory{}
