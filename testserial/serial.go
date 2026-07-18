// Package testserial owns the compiler-visible declaration that a Go test must
// remain serial because it exercises process-global or deliberately shared
// state.
package testserial

import (
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

// Serial validates the closed Foundation reason. Witness-lint recognizes this
// exact package and function contract; comments and local lookalikes do not
// satisfy the serial-test boundary.
func Serial(tb testing.TB, reason core.TestSerialReason) {
	tb.Helper()
	if err := reason.Validate(); err != nil {
		tb.Fatal(err)
	}
}

var _ func(testing.TB, core.TestSerialReason) = Serial
