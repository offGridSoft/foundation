package core

import (
	"time"
)

type NanosecondsDuration struct {
	nanos int64
}

func NewNanosecondsDuration(value time.Duration) NanosecondsDuration {
	return NanosecondsDuration{nanos: value.Nanoseconds()}
}

func NanosecondsDurationFromInt64(nanos int64) NanosecondsDuration {
	return NanosecondsDuration{nanos: nanos}
}

func (d NanosecondsDuration) Duration() time.Duration {
	return time.Duration(d.nanos)
}

func (d NanosecondsDuration) Nanoseconds() int64 {
	return d.nanos
}

func (d NanosecondsDuration) IsZero() bool {
	return d.nanos == 0
}

func (d NanosecondsDuration) Validate() error {
	if d.nanos < 0 {
		return wrapFoundationContract(ErrFmtNanosecondsDuration)
	}
	return nil
}

func (d NanosecondsDuration) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return appendInt64JSON(d.nanos), nil
}

//validate:unmarshal_ignore reason="NanosecondsDuration validates a temporary before assignment so rejected input cannot mutate the receiver."
func (d *NanosecondsDuration) UnmarshalJSON(data []byte) error {
	nanos, err := parseStrictInt64JSON(data)
	if err != nil {
		return wrapFoundationContract(ErrFmtNanosecondsDuration)
	}
	decoded := NanosecondsDurationFromInt64(nanos)
	if err := decoded.Validate(); err != nil {
		return err
	}
	*d = decoded
	return nil
}
