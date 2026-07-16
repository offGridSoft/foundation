package core

import (
	"errors"
	"testing"
)

func TestIPNetworkHostileTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		raw        string
		candidate  string
		wantString string
		wantMatch  bool
		wantErr    bool
	}{
		{name: "exact IPv4", raw: "203.0.113.7", candidate: "203.0.113.7", wantString: "203.0.113.7", wantMatch: true},
		{name: "IPv4 range", raw: "203.0.113.0/24", candidate: "203.0.113.255", wantString: "203.0.113.0/24", wantMatch: true},
		{name: "IPv4 outside", raw: "203.0.113.0/24", candidate: "203.0.114.1", wantString: "203.0.113.0/24"},
		{name: "exact IPv6", raw: "2001:db8::1", candidate: "2001:db8::1", wantString: "2001:db8::1", wantMatch: true},
		{name: "IPv6 range", raw: "2001:db8::/48", candidate: "2001:db8:0:ffff::1", wantString: "2001:db8::/48", wantMatch: true},
		{name: "host bits", raw: "203.0.113.7/24", wantErr: true},
		{name: "mapped IPv4 address", raw: "::ffff:203.0.113.7", wantErr: true},
		{name: "mapped IPv4 range", raw: "::ffff:203.0.113.0/120", wantErr: true},
		{name: "alternate IPv6 spelling", raw: "2001:0db8::1", wantErr: true},
		{name: "zone rejected", raw: "fe80::1%en0", wantErr: true},
		{name: "empty", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			network, err := ParseIPNetwork(tc.raw)
			if tc.wantErr {
				if !errors.Is(err, ErrIPNetworkContract) {
					t.Fatalf("ParseIPNetwork() error = %v, want ErrIPNetworkContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseIPNetwork() error = %v, want nil", err)
			}
			if got := network.String(); got != tc.wantString {
				t.Fatalf("String() = %q, want %q", got, tc.wantString)
			}
			got, err := network.Contains(tc.candidate)
			if err != nil || got != tc.wantMatch {
				t.Fatalf("Contains() = (%v, %v), want (%v, nil)", got, err, tc.wantMatch)
			}
		})
	}
}

func TestIPNetworkZeroFailsClosed(t *testing.T) {
	t.Parallel()
	var network IPNetwork
	if err := network.Validate(); !errors.Is(err, ErrIPNetworkContract) {
		t.Fatalf("Validate() error = %v, want ErrIPNetworkContract", err)
	}
	if _, err := network.Contains("203.0.113.7"); !errors.Is(err, ErrIPNetworkContract) {
		t.Fatalf("Contains() error = %v, want ErrIPNetworkContract", err)
	}
}
