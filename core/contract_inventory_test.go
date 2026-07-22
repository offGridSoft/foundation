package core

type protocolFact[T any] struct{}
type internalFlow[T any] struct{}

// coreContractInventory keeps every struct introduced by this slice classified
// with a compiler-visible data-flow role.
type coreContractInventory struct {
	VerificationAccess protocolFact[VerificationAccessContract]
	RetentionPolicy    protocolFact[WitnessRetentionPolicy]
	RetentionWindow    protocolFact[WitnessRetentionWindow]
	RetentionDecision  internalFlow[WitnessRetentionDecisionInput]
	CoalescingDelivery internalFlow[CoalescingDelivery[deliveryFixture]]
	IdempotencyKey     protocolFact[HTTPIdempotencyKey]
	RequestSemantics   protocolFact[HTTPRequestSemantics]
	NoBody             protocolFact[HTTPNoBody]
	RouteSemantics     protocolFact[HTTPRouteSemantics]
	HTTPHeader         protocolFact[HTTPHeader]
	HTTPHeaders        protocolFact[HTTPHeaders]
	HTTPQueryParameter protocolFact[HTTPQueryParameter]
	HTTPQuery          protocolFact[HTTPQuery]
	HTTPMediaType      protocolFact[HTTPMediaType]
	RetryDirective     protocolFact[HTTPRetryDirective]
	RetryPolicy        protocolFact[HTTPRetryPolicy]
	UpdateNotice       protocolFact[UpdateNotice]
}

var _ = coreContractInventory{}
