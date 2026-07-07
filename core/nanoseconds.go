package core

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"
)

type NanosecondsDuration struct {
	value time.Duration
}

func NewNanosecondsDuration(value time.Duration) NanosecondsDuration {
	return NanosecondsDuration{value: value}
}

func NanosecondsDurationFromInt64(nanos int64) NanosecondsDuration {
	return NanosecondsDuration{value: time.Duration(nanos)}
}

func (d NanosecondsDuration) Duration() time.Duration {
	return d.value
}

func (d NanosecondsDuration) Nanoseconds() int64 {
	return d.value.Nanoseconds()
}

func (d NanosecondsDuration) IsZero() bool {
	return d.value == 0
}

func (d NanosecondsDuration) Validate() error {
	return nil
}

func (d NanosecondsDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Nanoseconds())
}

func (d *NanosecondsDuration) UnmarshalJSON(data []byte) error {
	var nanos int64
	if err := json.Unmarshal(data, &nanos); err != nil {
		return fmt.Errorf(ErrFmtNanosecondsDuration, ErrFoundationContract)
	}
	*d = NanosecondsDurationFromInt64(nanos)
	return d.Validate()
}
