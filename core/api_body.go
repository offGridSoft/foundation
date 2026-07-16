package core

// APIBody is the Foundation-owned response plug accepted by typed HTTP
// transports. The witness method prevents primitives and loose maps from
// becoming protocol responses.
type APIBody interface {
	Validatable
	APIBody()
}
