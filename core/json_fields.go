package core

import (
	"encoding/json"
)

const (
	JSONFieldNameMaxRunes = 128
	JSONLiteralNull       = "null"

	JSONFieldChainHash  = "chain_hash"
	JSONFieldCustomerID = "customer_id"
	JSONFieldIssuedAt   = "issued_at"
	JSONFieldLedgerSeq  = "ledger_seq"
	JSONFieldObjects    = "objects"
	JSONFieldProvider   = "provider"
	JSONFieldRelease    = "release"
	JSONFieldRetention  = "retention"
	JSONFieldSchema     = "schema"
	JSONFieldSessionID  = "session_id"
)

func ValidateJSONFieldName(name string) error {
	return ValidateOpaqueToken(name, JSONFieldNameMaxRunes)
}

func AppendJSONField[T any](dst []byte, name string, value T) ([]byte, error) {
	if err := ValidateJSONFieldName(name); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	fieldName, err := json.Marshal(name)
	if err != nil {
		return nil, err
	}
	dst = append(dst, fieldName...)
	dst = append(dst, ':')
	dst = append(dst, encoded...)
	return dst, nil
}

func AppendJSONFieldAfterComma[T any](dst []byte, prior error, name string, value T) ([]byte, error) {
	if prior != nil {
		return nil, prior
	}
	dst = append(dst, ',')
	return AppendJSONField(dst, name, value)
}
