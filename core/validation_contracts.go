package core

// Validatable is the compiler-owned boundary contract for typed state.
type Validatable interface {
	Validate() error
}
