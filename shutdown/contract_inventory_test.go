package shutdown

import "github.com/offGridSoft/foundation/v2026/core"

type shutdownProtocolFact[T any] struct{}
type shutdownInternalFlow[T any] struct{}
type shutdownCapability[T any] struct{}
type shutdownFailure[T any] struct{}

type shutdownContractInventory struct {
	PlanPolicy      shutdownProtocolFact[PlanPolicy]
	Step            shutdownProtocolFact[Step]
	Plan            shutdownCapability[*Plan]
	StepResult      shutdownInternalFlow[StepResult]
	Report          shutdownInternalFlow[Report]
	SignalPolicy    shutdownProtocolFact[SignalPolicy]
	ForceRequest    shutdownProtocolFact[ForceRequest]
	ForceResult     shutdownInternalFlow[ForceResult]
	WatchRequest    shutdownProtocolFact[WatchRequest]
	NotifyRequest   shutdownProtocolFact[NotifyRequest]
	Controller      shutdownCapability[*Controller]
	PanicValue      shutdownInternalFlow[PanicValue]
	StepPanicError  shutdownFailure[StepPanicError]
	ForcePanicError shutdownFailure[ForcePanicError]
	SignalCause     shutdownFailure[SignalCause]
}

var (
	_ core.Validatable = PlanPolicy{}
	_ core.Validatable = Step{}
	_ core.Validatable = (*Plan)(nil)
	_ core.Validatable = StepResult{}
	_ core.Validatable = Report{}
	_ core.Validatable = SignalPolicy{}
	_ core.Validatable = ForceRequest{}
	_ core.Validatable = ForceResult{}
	_ core.Validatable = WatchRequest{}
	_ core.Validatable = NotifyRequest{}
	_ core.Validatable = PanicValue{}
	_ core.Validatable = StepPanicError{}
	_ core.Validatable = ForcePanicError{}
	_ core.Validatable = SignalCause{}
	_                  = shutdownContractInventory{}
)
