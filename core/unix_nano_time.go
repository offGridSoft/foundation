package core

import (
	"fmt"
	"time"

	json "github.com/goccy/go-json"
)

type UnixNanoTime struct {
	value time.Time
}

func NewUnixNanoTime(t time.Time) UnixNanoTime {
	if t.IsZero() {
		return UnixNanoTime{}
	}
	return UnixNanoTime{value: t.UTC()}
}

func UnixNanoTimeFromInt64(nanos int64) UnixNanoTime {
	if nanos == 0 {
		return UnixNanoTime{}
	}
	return UnixNanoTime{value: time.Unix(0, nanos).UTC()}
}

func (t UnixNanoTime) Validate() error {
	return nil
}

func (t UnixNanoTime) IsZero() bool {
	return t.value.IsZero()
}

func (t UnixNanoTime) Time() time.Time {
	if t.value.IsZero() {
		return time.Time{}
	}
	return t.value.UTC()
}

func (t UnixNanoTime) UnixNano() int64 {
	if t.IsZero() {
		return 0
	}
	return t.value.UnixNano()
}

func (t UnixNanoTime) Add(d time.Duration) UnixNanoTime {
	if t.IsZero() {
		return UnixNanoTime{}
	}
	return NewUnixNanoTime(t.value.Add(d))
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
	return json.Marshal(t.UnixNano())
}

func (t *UnixNanoTime) UnmarshalJSON(data []byte) error {
	var nanos int64
	if err := json.Unmarshal(data, &nanos); err != nil {
		return fmt.Errorf(ErrFmtUnixNanoTime, ErrFoundationContract)
	}
	*t = UnixNanoTimeFromInt64(nanos)
	return t.Validate()
}
