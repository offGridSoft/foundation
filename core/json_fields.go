package core

import (
	"encoding/json"
	"fmt"
)

const (
	JSONFieldNameMaxRunes  = 128
	JSONFieldNameSeparator = '_'
	JSONLiteralNull        = "null"

	JSONFieldChainHash     = "chain_hash"
	JSONFieldAcceptedAt    = "accepted_at"
	JSONFieldAuthority     = "authority"
	JSONFieldBundleRoot    = "bundle_root"
	JSONFieldCustomerID    = "customer_id"
	JSONFieldIssuedAt      = "issued_at"
	JSONFieldImprintSHA256 = "imprint_sha256"
	JSONFieldLedgerSeq     = "ledger_seq"
	JSONFieldObjects       = "objects"
	JSONFieldProvider      = "provider"
	JSONFieldRelease       = "release"
	JSONFieldReceiptID     = "receipt_id"
	JSONFieldRepositoryID  = "repository_id"
	JSONFieldRetention     = "retention"
	JSONFieldSchema        = "schema"
	JSONFieldSessionID     = "session_id"
	JSONFieldTimestamp     = "timestamp"
	JSONFieldTimestampedAt = "timestamped_at"
	JSONFieldToken         = "token"
	JSONFieldResponse      = "response"
	JSONFieldValidUntil    = "valid_until"
)

func ValidateJSONFieldName(name string) error {
	if err := ValidateOpaqueToken(name, JSONFieldNameMaxRunes); err != nil {
		return fmt.Errorf(ErrFmtJSONFieldName, ErrJSONContract)
	}
	if !isCanonicalJSONFieldName(name) {
		return fmt.Errorf(ErrFmtJSONFieldName, ErrJSONContract)
	}
	return nil
}

func isCanonicalJSONFieldName(name string) bool {
	previousSeparator := false
	for index, character := range name {
		if index == 0 && !isLowercaseASCII(character) {
			return false
		}
		if character == JSONFieldNameSeparator {
			if previousSeparator {
				return false
			}
			previousSeparator = true
			continue
		}
		if !isLowercaseASCII(character) && (character < '0' || character > '9') {
			return false
		}
		previousSeparator = false
	}
	return !previousSeparator
}

func isLowercaseASCII(character rune) bool {
	return character >= 'a' && character <= 'z'
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
