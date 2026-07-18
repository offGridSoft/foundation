package workloadidentity

import (
	"errors"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

const workloadIdentityTestToken = "header.payload.signature"

func TestTokenOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: workloadIdentityTestToken},
		{name: "empty", wantErr: true},
		{name: "two segments", value: "header.payload", wantErr: true},
		{name: "four segments", value: "a.b.c.d", wantErr: true},
		{name: "empty segment", value: "a..c", wantErr: true},
		{name: "invalid byte", value: "a.b+c.d", wantErr: true},
		{name: "over cap", value: strings.Repeat("a", TokenMaxBytes+1), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseToken(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParseToken() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParseToken() error = %v", err)
			}
		})
	}
}

func TestAuthorizationOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: core.HTTPAuthorizationBearerPrefix + workloadIdentityTestToken},
		{name: "empty", wantErr: true},
		{name: "missing scheme", value: workloadIdentityTestToken, wantErr: true},
		{name: "wrong scheme", value: "Basic " + workloadIdentityTestToken, wantErr: true},
		{name: "empty assertion", value: core.HTTPAuthorizationBearerPrefix, wantErr: true},
		{name: "trailing space", value: core.HTTPAuthorizationBearerPrefix + workloadIdentityTestToken + " ", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseAuthorization(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParseAuthorization() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ParseAuthorization() error = %v", err)
			}
		})
	}
}

func TestPrincipalOGSBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid", value: "release-signer@offgridsoftware.iam.gserviceaccount.com"},
		{name: "empty", wantErr: true},
		{name: "human email", value: "operator@offgridsoftware.ca", wantErr: true},
		{name: "uppercase local", value: "Release@offgridsoftware.iam.gserviceaccount.com", wantErr: true},
		{name: "empty local", value: "@offgridsoftware.iam.gserviceaccount.com", wantErr: true},
		{name: "empty project", value: "release-signer@.iam.gserviceaccount.com", wantErr: true},
		{name: "extra at", value: "release@signer@offgridsoftware.iam.gserviceaccount.com", wantErr: true},
		{name: "over cap", value: strings.Repeat("a", PrincipalMaxBytes) + "@p.iam.gserviceaccount.com", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			principal, err := ParsePrincipal(tc.value)
			if tc.wantErr && !errors.Is(err, ErrContract) {
				t.Fatalf("ParsePrincipal() error = %v, want %v", err, ErrContract)
			}
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("ParsePrincipal() error = %v", err)
				}
				if principal.String() != tc.value || principal.Validate() != nil || principal.IsZero() {
					t.Fatalf("principal round trip = %#v", principal)
				}
			}
		})
	}
}
