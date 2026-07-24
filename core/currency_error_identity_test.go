package core

import (
	"errors"
	"testing"
)

func TestCurrencyErrorIdentityHierarchyTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		identity error
		name     string
	}{
		{name: "currency root", identity: ErrCurrencyContract},
		{name: "currency mismatch", identity: ErrCurrencyMismatch},
		{name: "currency overflow", identity: ErrCurrencyOverflow},
		{name: "currency decimal", identity: ErrCurrencyDecimal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(test.identity, ErrCurrencyContract) {
				t.Fatalf("errors.Is(%v, ErrCurrencyContract) = false, want true", test.identity)
			}
			if !errors.Is(test.identity, ErrFoundationContract) {
				t.Fatalf("errors.Is(%v, ErrFoundationContract) = false, want true", test.identity)
			}
		})
	}

	specialized := []error{ErrCurrencyMismatch, ErrCurrencyOverflow, ErrCurrencyDecimal}
	for leftIndex, left := range specialized {
		for rightIndex, right := range specialized {
			if leftIndex != rightIndex && errors.Is(left, right) {
				t.Fatalf("currency identities alias: errors.Is(%v,%v) = true", left, right)
			}
		}
	}
}
