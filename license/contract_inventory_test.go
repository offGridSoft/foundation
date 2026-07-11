package license

type licenseProtocolFact[T any] struct{}
type licenseTypedFailure[T any] struct{}

// licenseContractInventory classifies the slice's package-crossing records.
type licenseContractInventory struct {
	WitnessPlanPolicy licenseProtocolFact[WitnessPlanPolicy]
	WitnessUsage      licenseProtocolFact[WitnessUsage]
	CheckInResponse   licenseProtocolFact[CheckInResponse[SeatLeaseBody]]
	RefusalFailure    licenseTypedFailure[RefusalError]
}
