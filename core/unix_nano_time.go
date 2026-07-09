package core

import "time"

type UnixNanoTime struct {
	nanos int64
	set   bool
}

func NewUnixNanoTime(t time.Time) UnixNanoTime {
	if t.IsZero() {
		return UnixNanoTime{}
	}
	return UnixNanoTime{nanos: t.UTC().UnixNano(), set: true}
}

func UnixNanoTimeFromInt64(nanos int64) UnixNanoTime {
	return UnixNanoTime{nanos: nanos, set: true}
}

func (t UnixNanoTime) Validate() error {
	if !t.set {
		return wrapFoundationContract(ErrFmtUnixNanoTime)
	}
	if t.nanos < 0 {
		return wrapFoundationContract(ErrFmtUnixNanoTime)
	}
	return nil
}

func ValidateRequiredUnixNanoTime(t UnixNanoTime) error {
	return t.Validate()
}

func (t UnixNanoTime) IsZero() bool {
	return !t.set
}

func (t UnixNanoTime) Time() time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return time.Unix(0, t.nanos).UTC()
}

func (t UnixNanoTime) UnixNano() int64 {
	return t.nanos
}

func (t UnixNanoTime) Add(d time.Duration) UnixNanoTime {
	if t.IsZero() {
		return UnixNanoTime{}
	}
	return NewUnixNanoTime(t.Time().Add(d))
}

func (t UnixNanoTime) Sub(other UnixNanoTime) time.Duration {
	return t.Time().Sub(other.Time())
}

func (t UnixNanoTime) Before(other UnixNanoTime) bool {
	return t.Time().Before(other.Time())
}

func (t UnixNanoTime) After(other UnixNanoTime) bool {
	return t.Time().After(other.Time())
}

func (t UnixNanoTime) Equal(other UnixNanoTime) bool {
	return t.Time().Equal(other.Time())
}

func (t UnixNanoTime) Format(layout string) string {
	return t.Time().Format(layout)
}

func (t UnixNanoTime) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return appendInt64JSON(t.nanos), nil
}

//validate:unmarshal_ignore reason="UnixNanoTime validates a temporary before assignment so rejected input cannot mutate the receiver."
func (t *UnixNanoTime) UnmarshalJSON(data []byte) error {
	nanos, err := parseStrictInt64JSON(data)
	if err != nil {
		return wrapFoundationContract(ErrFmtUnixNanoTime)
	}
	decoded := UnixNanoTimeFromInt64(nanos)
	if err := decoded.Validate(); err != nil {
		return err
	}
	*t = decoded
	return nil
}
