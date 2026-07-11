package core

import (
	"encoding/json"
	"testing"
)

type strictJSONFuzzContract struct {
	Value string `json:"value"`
}

func (c strictJSONFuzzContract) Validate() error {
	return ValidateOpaqueToken(c.Value, OpaqueTokenDefaultMaxRunes)
}

func FuzzDecodeStrictJSONNeverPanics(f *testing.F) {
	valid, err := json.Marshal(strictJSONFuzzContract{Value: FoundationVersion2026})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := DecodeStrictJSON[strictJSONFuzzContract](data)
		if err != nil {
			return
		}
		roundTrip, err := json.Marshal(decoded)
		if err != nil {
			t.Fatalf("json.Marshal(decoded) error = %v", err)
		}
		if len(roundTrip) > StrictJSONMaxBytes {
			t.Fatalf("round-trip bytes = %d, want <= %d", len(roundTrip), StrictJSONMaxBytes)
		}
	})
}
