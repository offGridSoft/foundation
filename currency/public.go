package currency

// Public package functions are intentionally centralized here. Public value
// methods remain beside their owning types.

// New constructs an exact amount from signed minor units.
func New(code Code, minorUnits int64) (Amount, error) {
	if err := code.Validate(); err != nil {
		return Amount{}, err
	}
	return Amount{code: code, minorUnits: minorUnits}, nil
}

// Parse constructs an exact amount from a bounded decimal representation.
func Parse(code Code, decimal string) (Amount, error) {
	if err := code.Validate(); err != nil {
		return Amount{}, err
	}
	minorUnits, err := parseDecimal(code, decimal)
	if err != nil {
		return Amount{}, err
	}
	return Amount{code: code, minorUnits: minorUnits}, nil
}

// ParseCode accepts one canonical uppercase supported ISO token.
func ParseCode(token string) (Code, error) {
	for code := CodeUSD; code <= CodeCLF; code++ {
		if codeTokens()[code] == token {
			return code, nil
		}
	}
	return CodeUnknown, contractError(errLabelCode)
}

// ParseDisplayUnit accepts one canonical display-unit token.
func ParseDisplayUnit(token string) (DisplayUnit, error) {
	for unit := DisplayUnitAutomatic; unit <= DisplayUnitBillions; unit++ {
		if displayUnitTokens()[unit] == token {
			return unit, nil
		}
	}
	return DisplayUnitUnknown, contractError(errLabelHumanize)
}

// FromFirestore validates and closes a Firestore projection.
func FromFirestore(value FirestoreAmount) (Amount, error) {
	return value.close()
}

// FromPostgreSQL validates and closes a PostgreSQL projection.
func FromPostgreSQL(value PostgreSQLAmount) (Amount, error) {
	return value.close()
}
