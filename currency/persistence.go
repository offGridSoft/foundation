package currency

// FirestoreAmount is the driver-neutral Firestore projection.
type FirestoreAmount struct {
	Currency   string `firestore:"currency"`
	MinorUnits int64  `firestore:"minor_units"`
}

// Validate enforces canonical persistence state.
func (a FirestoreAmount) Validate() error {
	_, err := a.close()
	return err
}

func (a FirestoreAmount) close() (Amount, error) {
	code, err := ParseCode(a.Currency)
	if err != nil {
		return Amount{}, contractError(errLabelPersistence, err)
	}
	return Amount{code: code, minorUnits: a.MinorUnits}, nil
}

// PostgreSQLAmount is the driver-neutral PostgreSQL projection.
type PostgreSQLAmount struct {
	Currency   string
	MinorUnits int64
}

// Validate enforces canonical persistence state.
func (a PostgreSQLAmount) Validate() error {
	_, err := a.close()
	return err
}

func (a PostgreSQLAmount) close() (Amount, error) {
	code, err := ParseCode(a.Currency)
	if err != nil {
		return Amount{}, contractError(errLabelPersistence, err)
	}
	return Amount{code: code, minorUnits: a.MinorUnits}, nil
}

// Firestore projects an amount without loss.
func (a Amount) Firestore() (FirestoreAmount, error) {
	if err := a.Validate(); err != nil {
		return FirestoreAmount{}, err
	}
	return FirestoreAmount{Currency: a.code.String(), MinorUnits: a.minorUnits}, nil
}

// PostgreSQL projects an amount without loss.
func (a Amount) PostgreSQL() (PostgreSQLAmount, error) {
	if err := a.Validate(); err != nil {
		return PostgreSQLAmount{}, err
	}
	return PostgreSQLAmount{Currency: a.code.String(), MinorUnits: a.minorUnits}, nil
}
