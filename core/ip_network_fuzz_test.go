package core

import "testing"

func FuzzIPNetworkCanonicalBoundary(f *testing.F) {
	for _, seed := range []string{"203.0.113.7", "203.0.113.0/24", "2001:db8::1", "2001:db8::/48", "::ffff:203.0.113.7", "203.0.113.7/24", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		network, err := ParseIPNetwork(raw)
		if err != nil {
			return
		}
		if err := network.Validate(); err != nil {
			t.Fatalf("ParseIPNetwork(%q) produced invalid value: %v", raw, err)
		}
		canonical := network.String()
		reparsed, err := ParseIPNetwork(canonical)
		if err != nil {
			t.Fatalf("canonical %q rejected: %v", canonical, err)
		}
		if reparsed.String() != canonical {
			t.Fatalf("canonical round trip = %q, want %q", reparsed.String(), canonical)
		}
	})
}
