package currency

import "math"

// Amount is exact signed minor units bound to one supported currency.
type Amount struct {
	minorUnits int64
	code       Code
}

// Validate rejects the invalid unknown-currency zero value.
func (a Amount) Validate() error {
	if err := a.code.Validate(); err != nil {
		return contractError(errLabelAmount, err)
	}
	return nil
}

// Code returns the amount's validated currency.
func (a Amount) Code() (Code, error) {
	if err := a.Validate(); err != nil {
		return CodeUnknown, err
	}
	return a.code, nil
}

// MinorUnits returns the exact signed minor-unit value.
func (a Amount) MinorUnits() (int64, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	return a.minorUnits, nil
}

// IsZero reports whether the validated amount is zero.
func (a Amount) IsZero() (bool, error) {
	if err := a.Validate(); err != nil {
		return false, err
	}
	return a.minorUnits == 0, nil
}

// IsPositive reports whether the validated amount is positive.
func (a Amount) IsPositive() (bool, error) {
	if err := a.Validate(); err != nil {
		return false, err
	}
	return a.minorUnits > 0, nil
}

// IsNegative reports whether the validated amount is negative.
func (a Amount) IsNegative() (bool, error) {
	if err := a.Validate(); err != nil {
		return false, err
	}
	return a.minorUnits < 0, nil
}

// Add returns an exact same-currency sum.
func (a Amount) Add(other Amount) (Amount, error) {
	if err := validatePair(a, other); err != nil {
		return Amount{}, err
	}
	if addOverflows(a.minorUnits, other.minorUnits) {
		return Amount{}, overflowError()
	}
	return Amount{code: a.code, minorUnits: a.minorUnits + other.minorUnits}, nil
}

// Subtract returns an exact same-currency difference.
func (a Amount) Subtract(other Amount) (Amount, error) {
	if err := validatePair(a, other); err != nil {
		return Amount{}, err
	}
	if subtractOverflows(a.minorUnits, other.minorUnits) {
		return Amount{}, overflowError()
	}
	return Amount{code: a.code, minorUnits: a.minorUnits - other.minorUnits}, nil
}

// Multiply scales an amount by an unsigned dimensionless quantity.
func (a Amount) Multiply(quantity uint64) (Amount, error) {
	if err := a.Validate(); err != nil {
		return Amount{}, err
	}
	product, err := multiplySigned(a.minorUnits, quantity)
	if err != nil {
		return Amount{}, err
	}
	return Amount{code: a.code, minorUnits: product}, nil
}

// Compare orders two amounts of the same currency.
func (a Amount) Compare(other Amount) (Order, error) {
	if err := validatePair(a, other); err != nil {
		return OrderUnknown, err
	}
	switch {
	case a.minorUnits < other.minorUnits:
		return OrderLess, nil
	case a.minorUnits > other.minorUnits:
		return OrderGreater, nil
	default:
		return OrderEqual, nil
	}
}

func validatePair(left, right Amount) error {
	if err := left.Validate(); err != nil {
		return err
	}
	if err := right.Validate(); err != nil {
		return err
	}
	if left.code != right.code {
		return mismatchError()
	}
	return nil
}

func addOverflows(left, right int64) bool {
	return right > 0 && left > math.MaxInt64-right ||
		right < 0 && left < math.MinInt64-right
}

func subtractOverflows(left, right int64) bool {
	return right > 0 && left < math.MinInt64+right ||
		right < 0 && left > math.MaxInt64+right
}

func multiplySigned(value int64, quantity uint64) (int64, error) {
	if value == 0 || quantity == 0 {
		return 0, nil
	}
	magnitude := signedMagnitude(value)
	limit := uint64(math.MaxInt64)
	if value < 0 {
		limit++
	}
	if magnitude > limit/quantity {
		return 0, overflowError()
	}
	product := magnitude * quantity
	return signedValue(value < 0, product), nil
}

func signedMagnitude(value int64) uint64 {
	if value >= 0 {
		return uint64(value)
	}
	// #nosec G115 -- adding one before negation maps the complete negative
	// int64 domain into 0..MaxInt64 before this checked widening.
	return uint64(-(value + 1)) + 1
}

func signedValue(negative bool, magnitude uint64) int64 {
	if !negative {
		// #nosec G115 -- multiplySigned proves magnitude is at most MaxInt64.
		return int64(magnitude)
	}
	if magnitude == uint64(math.MaxInt64)+1 {
		return math.MinInt64
	}
	// #nosec G115 -- the MinInt64 magnitude is handled above; the remainder is
	// at most MaxInt64 before conversion.
	return -int64(magnitude)
}
