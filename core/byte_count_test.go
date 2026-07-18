package core

import (
	"errors"
	"math"
	"testing"
)

func TestByteCountInt64OGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		value   uint64
		want    int64
		wantErr bool
	}{
		{name: "one", value: 1, want: 1},
		{name: "maximum", value: math.MaxInt64, want: math.MaxInt64},
		{name: "zero", wantErr: true},
		{name: "overflow", value: math.MaxInt64 + 1, wantErr: true},
		{name: "maximum unsigned", value: math.MaxUint64, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewByteCount(test.value).Int64()
			if test.wantErr && !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("ByteCount.Int64() error = %v, want %v", err, ErrFoundationContract)
			}
			if !test.wantErr && (err != nil || got != test.want) {
				t.Fatalf("ByteCount.Int64() = (%d, %v), want (%d, nil)", got, err, test.want)
			}
		})
	}
}
