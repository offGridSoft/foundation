package core

import (
	"errors"
	"testing"
)

func TestValidateControlFreeUTF8HostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{name: "empty is structurally safe"},
		{name: "ordinary text", value: "customer evidence"},
		{name: "unicode text", value: "café 事故"},
		{name: "space is not a control", value: " leading and trailing "},
		{name: "nul rejected", value: "before\x00after", wantErr: ErrFoundationContract},
		{name: "newline rejected", value: "before\nafter", wantErr: ErrFoundationContract},
		{name: "tab rejected", value: "before\tafter", wantErr: ErrFoundationContract},
		{name: "delete rejected", value: "before\x7fafter", wantErr: ErrFoundationContract},
		{name: "c1 control rejected", value: "before\u0085after", wantErr: ErrFoundationContract},
		{name: "invalid utf8 rejected", value: string([]byte{0xff, 0xfe}), wantErr: ErrFoundationContract},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateControlFreeUTF8(tc.value)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateControlFreeUTF8() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidateControlFreeUTF8() error = %v, want errors.Is(..., %v)", err, tc.wantErr)
			}
		})
	}
}

func FuzzValidateControlFreeUTF8NeverPanics(f *testing.F) {
	f.Add("safe text")
	f.Add("before\u0085after")
	f.Add(string([]byte{0xff}))

	f.Fuzz(func(t *testing.T, value string) {
		_ = ValidateControlFreeUTF8(value)
	})
}
