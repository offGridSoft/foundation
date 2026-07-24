package currency

import (
	"errors"
	"math"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type arithmeticOperation uint8

const (
	arithmeticAdd arithmeticOperation = iota + 1
	arithmeticSubtract
	arithmeticMultiply
)

type amountStateCase struct {
	name         string
	value        int64
	wantZero     bool
	wantPositive bool
	wantNegative bool
}

func TestAmountAccessorsAndSignedStateTable(t *testing.T) {
	t.Parallel()

	tests := []amountStateCase{
		{name: "negative signed minimum", value: math.MinInt64, wantNegative: true},
		{name: "negative one", value: -1, wantNegative: true},
		{name: "zero balance", value: 0, wantZero: true},
		{name: "positive one", value: 1, wantPositive: true},
		{name: "positive signed maximum", value: math.MaxInt64, wantPositive: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proveAmountState(t, test)
		})
	}
}

func proveAmountState(t *testing.T, test amountStateCase) {
	t.Helper()

	amount, err := New(CodeCAD, test.value)
	if err != nil {
		t.Fatalf("New(CodeCAD, %d) error = %v, want nil", test.value, err)
	}
	proveAmountIdentity(t, amount, test.value)
	proveAmountSigns(t, amount, test)
}

func proveAmountIdentity(t *testing.T, amount Amount, wantMinor int64) {
	t.Helper()

	gotCode, codeErr := amount.Code()
	gotMinor, minorErr := amount.MinorUnits()
	if codeErr != nil || minorErr != nil {
		t.Fatalf("Amount identity accessor errors = (%v,%v), want both nil", codeErr, minorErr)
	}
	if gotCode != CodeCAD || gotMinor != wantMinor {
		t.Fatalf("Amount identity = (%s,%d), want (CAD,%d)", gotCode, gotMinor, wantMinor)
	}
}

func proveAmountSigns(t *testing.T, amount Amount, test amountStateCase) {
	t.Helper()

	gotZero, zeroErr := amount.IsZero()
	gotPositive, positiveErr := amount.IsPositive()
	gotNegative, negativeErr := amount.IsNegative()
	if zeroErr != nil || positiveErr != nil || negativeErr != nil {
		t.Fatalf("Amount sign accessor errors = (%v,%v,%v), want all nil", zeroErr, positiveErr, negativeErr)
	}
	if gotZero != test.wantZero || gotPositive != test.wantPositive || gotNegative != test.wantNegative {
		t.Fatalf("Amount signs = (zero=%t positive=%t negative=%t), want (%t,%t,%t)", gotZero, gotPositive, gotNegative, test.wantZero, test.wantPositive, test.wantNegative)
	}
}

func TestAmountInvalidZeroValueFailsEveryBoundaryTable(t *testing.T) {
	t.Parallel()

	amount := Amount{}
	tests := []struct {
		run  func() error
		name string
	}{
		{name: "validate rejects missing currency", run: amount.Validate},
		{name: "code accessor rejects missing currency", run: func() error { _, err := amount.Code(); return err }},
		{name: "minor accessor rejects missing currency", run: func() error { _, err := amount.MinorUnits(); return err }},
		{name: "zero classifier rejects missing currency", run: func() error { _, err := amount.IsZero(); return err }},
		{name: "positive classifier rejects missing currency", run: func() error { _, err := amount.IsPositive(); return err }},
		{name: "negative classifier rejects missing currency", run: func() error { _, err := amount.IsNegative(); return err }},
		{name: "decimal projection rejects missing currency", run: func() error { _, err := amount.Decimal(); return err }},
		{name: "firestore projection rejects missing currency", run: func() error { _, err := amount.Firestore(); return err }},
		{name: "postgresql projection rejects missing currency", run: func() error { _, err := amount.PostgreSQL(); return err }},
		{name: "multiply rejects missing currency", run: func() error { _, err := amount.Multiply(0); return err }},
		{name: "compare rejects missing currency", run: func() error { _, err := amount.Compare(amount); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.run(); !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("%s error = %v, want ErrCurrencyContract", test.name, err)
			}
		})
	}
}

func TestAmountArithmeticExtremeBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		left      int64
		right     int64
		quantity  uint64
		want      int64
		operation arithmeticOperation
		wantErr   bool
	}{
		{name: "add two ordinary positive values", operation: arithmeticAdd, left: 40, right: 2, want: 42},
		{name: "add positive and negative values", operation: arithmeticAdd, left: 40, right: -2, want: 38},
		{name: "add zero to signed maximum", operation: arithmeticAdd, left: math.MaxInt64, right: 0, want: math.MaxInt64},
		{name: "add negative one to signed minimum plus one", operation: arithmeticAdd, left: math.MinInt64 + 1, right: -1, want: math.MinInt64},
		{name: "add positive one to signed maximum overflows", operation: arithmeticAdd, left: math.MaxInt64, right: 1, wantErr: true},
		{name: "add negative one to signed minimum underflows", operation: arithmeticAdd, left: math.MinInt64, right: -1, wantErr: true},
		{name: "subtract ordinary positive values", operation: arithmeticSubtract, left: 40, right: 2, want: 38},
		{name: "subtract negative from positive", operation: arithmeticSubtract, left: 40, right: -2, want: 42},
		{name: "subtract zero from signed minimum", operation: arithmeticSubtract, left: math.MinInt64, right: 0, want: math.MinInt64},
		{name: "subtract one from signed minimum plus one", operation: arithmeticSubtract, left: math.MinInt64 + 1, right: 1, want: math.MinInt64},
		{name: "subtract negative one from signed maximum overflows", operation: arithmeticSubtract, left: math.MaxInt64, right: -1, wantErr: true},
		{name: "subtract one from signed minimum underflows", operation: arithmeticSubtract, left: math.MinInt64, right: 1, wantErr: true},
		{name: "multiply zero by maximum quantity", operation: arithmeticMultiply, left: 0, quantity: math.MaxUint64, want: 0},
		{name: "multiply positive by zero", operation: arithmeticMultiply, left: math.MaxInt64, quantity: 0, want: 0},
		{name: "multiply signed maximum by one", operation: arithmeticMultiply, left: math.MaxInt64, quantity: 1, want: math.MaxInt64},
		{name: "multiply signed minimum by one", operation: arithmeticMultiply, left: math.MinInt64, quantity: 1, want: math.MinInt64},
		{name: "multiply negative one by minimum magnitude", operation: arithmeticMultiply, left: -1, quantity: uint64(math.MaxInt64) + 1, want: math.MinInt64},
		{name: "multiply positive one by signed maximum", operation: arithmeticMultiply, left: 1, quantity: math.MaxInt64, want: math.MaxInt64},
		{name: "multiply signed maximum by two overflows", operation: arithmeticMultiply, left: math.MaxInt64, quantity: 2, wantErr: true},
		{name: "multiply signed minimum by two underflows", operation: arithmeticMultiply, left: math.MinInt64, quantity: 2, wantErr: true},
		{name: "multiply positive one by maximum uint overflows", operation: arithmeticMultiply, left: 1, quantity: math.MaxUint64, wantErr: true},
		{name: "multiply negative one by maximum uint underflows", operation: arithmeticMultiply, left: -1, quantity: math.MaxUint64, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := runArithmetic(test)
			if test.wantErr {
				if got != (Amount{}) {
					t.Fatalf("%s result = %+v, want zero Amount", test.name, got)
				}
				if !errors.Is(err, core.ErrCurrencyOverflow) || !errors.Is(err, core.ErrNumericOverflow) {
					t.Fatalf("%s error = %v, want overflow identities", test.name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s error = %v, want nil", test.name, err)
			}
			gotMinor, err := got.MinorUnits()
			if err != nil || gotMinor != test.want {
				t.Fatalf("%s minor units = (%d, %v), want (%d, nil)", test.name, gotMinor, err, test.want)
			}
		})
	}
}

func runArithmetic(test struct {
	name      string
	left      int64
	right     int64
	quantity  uint64
	want      int64
	operation arithmeticOperation
	wantErr   bool
}) (Amount, error) {
	left, err := New(CodeUSD, test.left)
	if err != nil {
		return Amount{}, err
	}
	switch test.operation {
	case arithmeticAdd:
		right, rightErr := New(CodeUSD, test.right)
		if rightErr != nil {
			return Amount{}, rightErr
		}
		return left.Add(right)
	case arithmeticSubtract:
		right, rightErr := New(CodeUSD, test.right)
		if rightErr != nil {
			return Amount{}, rightErr
		}
		return left.Subtract(right)
	case arithmeticMultiply:
		return left.Multiply(test.quantity)
	default:
		return Amount{}, core.ErrCurrencyContract
	}
}

func TestAmountCrossCurrencyOperationsFailClosedTable(t *testing.T) {
	t.Parallel()

	usd, err := New(CodeUSD, 100)
	if err != nil {
		t.Fatal(err)
	}
	cad, err := New(CodeCAD, 100)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		run  func() (Amount, Order, error)
		name string
	}{
		{name: "add refuses numerically equal different currencies", run: func() (Amount, Order, error) { got, runErr := usd.Add(cad); return got, OrderUnknown, runErr }},
		{name: "subtract refuses numerically equal different currencies", run: func() (Amount, Order, error) { got, runErr := usd.Subtract(cad); return got, OrderUnknown, runErr }},
		{name: "compare refuses numerically equal different currencies", run: func() (Amount, Order, error) { got, runErr := usd.Compare(cad); return Amount{}, got, runErr }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotAmount, gotOrder, err := test.run()
			if gotAmount != (Amount{}) || gotOrder != OrderUnknown {
				t.Fatalf("%s results = (%+v,%d), want zero Amount and OrderUnknown", test.name, gotAmount, gotOrder)
			}
			if !errors.Is(err, core.ErrCurrencyMismatch) || !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("%s error = %v, want mismatch identities", test.name, err)
			}
		})
	}
}

func TestAmountComparisonSignedOrderingTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  int64
		right int64
		want  Order
	}{
		{name: "signed minimum is less than signed maximum", left: math.MinInt64, right: math.MaxInt64, want: OrderLess},
		{name: "negative one is less than zero", left: -1, right: 0, want: OrderLess},
		{name: "zero equals zero", left: 0, right: 0, want: OrderEqual},
		{name: "positive one is greater than zero", left: 1, right: 0, want: OrderGreater},
		{name: "signed maximum is greater than signed minimum", left: math.MaxInt64, right: math.MinInt64, want: OrderGreater},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			left, leftErr := New(CodeBHD, test.left)
			right, rightErr := New(CodeBHD, test.right)
			if leftErr != nil || rightErr != nil {
				t.Fatalf("comparison setup errors = (%v,%v), want nil", leftErr, rightErr)
			}
			got, err := left.Compare(right)
			if err != nil || got != test.want {
				t.Fatalf("Compare(%d,%d) = (%d,%v), want (%d,nil)", test.left, test.right, got, err, test.want)
			}
		})
	}
}
