package license

import (
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestCheckInNonceHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	valid := strings.Repeat("a1", CheckInNonceBytes)
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "exact lowercase nonce accepted", value: valid},
		{name: "exact uppercase nonce accepted", value: strings.ToUpper(valid)},
		{name: "empty rejected", wantErr: true},
		{name: "all zero rejected", value: strings.Repeat("0", CheckInNonceBytes*2), wantErr: true},
		{name: "one byte short rejected", value: valid[:len(valid)-2], wantErr: true},
		{name: "one byte long rejected", value: valid + "a1", wantErr: true},
		{name: "odd hex length rejected", value: valid[:len(valid)-1], wantErr: true},
		{name: "non hex rejected", value: strings.Repeat("z", CheckInNonceBytes*2), wantErr: true},
		{name: "leading space rejected", value: " " + valid, wantErr: true},
		{name: "trailing newline rejected", value: valid + "\n", wantErr: true},
		{name: "unicode digit rejected", value: "１" + valid[2:], wantErr: true},
		{name: "embedded separator rejected", value: valid[:32] + "-" + valid[33:], wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseCheckInNonce(tc.value)
			if tc.wantErr {
				if !errors.Is(err, core.ErrCheckInNonce) {
					t.Fatalf("ParseCheckInNonce() error = %v, want %v", err, core.ErrCheckInNonce)
				}
				return
			}
			if err != nil || got.String() != strings.ToLower(tc.value) {
				t.Fatalf("ParseCheckInNonce() = %q, error = %v", got.String(), err)
			}
		})
	}
}

func TestNewCheckInNonceProducesValidChallenge(t *testing.T) {
	t.Parallel()

	nonce, err := NewCheckInNonce()
	if err != nil {
		t.Fatalf("NewCheckInNonce() error = %v", err)
	}
	if err := nonce.Validate(); err != nil {
		t.Fatalf("NewCheckInNonce().Validate() error = %v", err)
	}
}
