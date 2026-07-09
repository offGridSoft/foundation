package core

import "math"

type MoneyPennies struct {
	value uint64
}

func NewMoneyPennies(value uint64) MoneyPennies {
	return MoneyPennies{value: value}
}

func (m MoneyPennies) Uint64() uint64 {
	return m.value
}

func (m MoneyPennies) IsZero() bool {
	return m.value == 0
}

func (m MoneyPennies) IsPositive() bool {
	return m.value > 0
}

func (m MoneyPennies) Validate() error {
	return nil
}

func (m MoneyPennies) Add(other MoneyPennies) (MoneyPennies, error) {
	if other.value > math.MaxUint64-m.value {
		return MoneyPennies{}, wrapFoundationContract(ErrFmtMoneyPennies)
	}
	return NewMoneyPennies(m.value + other.value), nil
}

func (m MoneyPennies) Sub(other MoneyPennies) (MoneyPennies, error) {
	if other.value > m.value {
		return MoneyPennies{}, wrapFoundationContract(ErrFmtMoneyPennies)
	}
	return NewMoneyPennies(m.value - other.value), nil
}

func (m MoneyPennies) MulQuantity(quantity uint64) (MoneyPennies, error) {
	if quantity != 0 && m.value > math.MaxUint64/quantity {
		return MoneyPennies{}, wrapFoundationContract(ErrFmtMoneyPennies)
	}
	return NewMoneyPennies(m.value * quantity), nil
}

func (m MoneyPennies) MarshalJSON() ([]byte, error) {
	return appendUint64JSON(m.value), nil
}

func (m *MoneyPennies) UnmarshalJSON(data []byte) error {
	value, err := parseStrictUint64JSON(data)
	if err != nil {
		return wrapFoundationContract(ErrFmtMoneyPennies)
	}
	*m = NewMoneyPennies(value)
	return m.Validate()
}
