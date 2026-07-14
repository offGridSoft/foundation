package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAPIEndpointBoundaryHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "production endpoint", value: OffgridAPIBaseURL + "/v2026/bug/check_in"},
		{name: "loopback endpoint", value: "http://127.0.0.1:40123/v2026/bug/check_in"},
		{name: "localhost endpoint", value: "http://localhost:40123/v2026/bug/check_in"},
		{name: "plain http rejected", value: "http://api.offgridsoftware.ca/v2026/bug/check_in", wantErr: true},
		{name: "missing path rejected", value: OffgridAPIBaseURL, wantErr: true},
		{name: "root path rejected", value: OffgridAPIBaseURL + "/", wantErr: true},
		{name: "userinfo rejected", value: "https://user@api.offgridsoftware.ca/v2026/bug/check_in", wantErr: true},
		{name: "query rejected", value: OffgridAPIBaseURL + "/v2026/bug/check_in?x=1", wantErr: true},
		{name: "fragment rejected", value: OffgridAPIBaseURL + "/v2026/bug/check_in#x", wantErr: true},
		{name: "newline rejected", value: OffgridAPIBaseURL + "/v2026/bug/check_in\nshadow", wantErr: true},
		{name: "unicode hostname rejected", value: "https://例.example/v2026/bug/check_in", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoint, err := ParseAPIEndpoint(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseAPIEndpoint() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAPIEndpoint() error = %v", err)
			}
			encoded, err := json.Marshal(endpoint)
			if err != nil {
				t.Fatal(err)
			}
			var decoded APIEndpoint
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded != endpoint {
				t.Fatalf("round trip = %q, want %q", decoded, endpoint)
			}
		})
	}
}

func TestAPIEndpointForBaseURLHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		base    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "production", base: OffgridAPIBaseURL, path: "/v2026/bug/check_in", want: OffgridAPIBaseURL + "/v2026/bug/check_in"},
		{name: "trailing slash", base: OffgridAPIBaseURL + "/", path: "/v2026/bug/check_in", want: OffgridAPIBaseURL + "/v2026/bug/check_in"},
		{name: "loopback", base: "http://[::1]:40123", path: "/v2026/bug/check_in", want: "http://[::1]:40123/v2026/bug/check_in"},
		{name: "base path rejected", base: OffgridAPIBaseURL + "/api", path: "/v2026/bug/check_in", wantErr: true},
		{name: "relative path rejected", base: OffgridAPIBaseURL, path: "v2026/bug/check_in", wantErr: true},
		{name: "root path rejected", base: OffgridAPIBaseURL, path: "/", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoint, err := APIEndpointForBaseURL(tc.base, tc.path)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("APIEndpointForBaseURL() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil || endpoint.String() != tc.want {
				t.Fatalf("APIEndpointForBaseURL() = (%q, %v), want (%q, nil)", endpoint, err, tc.want)
			}
		})
	}
}
