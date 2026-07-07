package core

import json "github.com/goccy/go-json"

type CanonicalJSONContract interface {
	Validate() error
}

// AppendCanonicalJSON appends the canonical JSON encoding of v to dst. It is the
// single signing-serialization path for foundation-owned closed structs: every
// signed wire body (leases, custody receipts) produces its signed bytes through
// here, so the marshaller and the field-order source of truth live in one place.
func AppendCanonicalJSON[T CanonicalJSONContract](dst []byte, v T) ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(dst, raw...), nil
}
