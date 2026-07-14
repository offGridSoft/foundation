package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func FuzzAPIEndpointJSONBoundary(f *testing.F) {
	for _, seed := range []string{
		OffgridAPIBaseURL + "/v2026/bug/check_in",
		"http://127.0.0.1:40123/v2026/bug/check_in",
		"http://[::1]:40123/v2026/bug/check_in",
		"",
		OffgridAPIBaseURL,
		OffgridAPIBaseURL + "/",
		"http://api.offgridsoftware.ca/v2026/bug/check_in",
		"https://user@api.offgridsoftware.ca/v2026/bug/check_in",
		OffgridAPIBaseURL + "/v2026/bug/check_in?x=1",
		OffgridAPIBaseURL + "/v2026/bug/check_in#x",
		OffgridAPIBaseURL + "/v2026/bug/check_in\nshadow",
		"://bad",
		string([]byte("https://\xe6/0")),
		string([]byte("https://0/\x96")),
	} {
		f.Add(seed)
	}
	known, err := ParseAPIEndpoint(OffgridAPIBaseURL + "/v2026/bug/check_in")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, value string) {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		decoded := known
		decodeErr := json.Unmarshal(data, &decoded)
		parsed, parseErr := ParseAPIEndpoint(value)
		if parseErr != nil {
			if !errors.Is(parseErr, ErrFoundationContract) || !errors.Is(decodeErr, ErrFoundationContract) {
				t.Fatalf("invalid endpoint errors = (%v, %v), want %v", parseErr, decodeErr, ErrFoundationContract)
			}
			if decoded != known {
				t.Fatalf("rejected endpoint mutated receiver: got %q, want %q", decoded, known)
			}
			return
		}
		if decodeErr != nil || decoded != parsed {
			t.Fatalf("valid endpoint round trip = (%q, %v), want (%q, nil)", decoded, decodeErr, parsed)
		}
	})
}
