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
}

var _ = coreContractInventory{}
