package license

import "testing"

func FuzzParseCheckInEndpointNeverPanics(f *testing.F) {
	f.Add(BugCheckInEndpoint().String())
	f.Add(WitnessCheckInEndpoint().String())
	f.Fuzz(func(t *testing.T, value string) {
		endpoint, err := ParseCheckInEndpoint(value)
		if err != nil {
			return
		}
		if err := endpoint.Validate(); err != nil {
			t.Fatalf("parsed endpoint validation = %v", err)
		}
		if endpoint.String() != value {
			t.Fatalf("parsed endpoint = %q, want %q", endpoint.String(), value)
		}
	})
}
