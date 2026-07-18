package workloadidentity

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type workloadIdentityRoundTrip func(*http.Request) (*http.Response, error)

const workloadIdentityTestPath = "/v2026/test/workload-identity"

func (f workloadIdentityRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestWorkloadIdentitySourceOGSMetadataBoundary(t *testing.T) {
	t.Parallel()
	audience, err := core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, workloadIdentityTestPath)
	if err != nil {
		t.Fatal(err)
	}
	transport := workloadIdentityRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != (Source{Audience: audience}).metadataURL() {
			t.Errorf("metadata URL = %q", request.URL.String())
		}
		if got := request.Header.Get(GoogleMetadataFlavorHeader); got != GoogleMetadataFlavorValue {
			t.Errorf("metadata flavor = %q, want %q", got, GoogleMetadataFlavorValue)
		}
		return &http.Response{
			StatusCode: core.HTTPStatusOK,
			Body:       io.NopCloser(strings.NewReader(workloadIdentityTestToken)),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})
	source := Source{HTTP: &http.Client{Transport: transport}, Audience: audience}
	token, err := source.Token(t.Context())
	if err != nil {
		t.Fatalf("WorkloadIdentitySource.Token() error = %v", err)
	}
	if err := token.Validate(); err != nil {
		t.Fatalf("WorkloadIdentityToken.Validate() error = %v", err)
	}
}

func TestWorkloadIdentitySourceOGSRejectsOversizedResponse(t *testing.T) {
	t.Parallel()
	audience, err := core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, workloadIdentityTestPath)
	if err != nil {
		t.Fatal(err)
	}
	transport := workloadIdentityRoundTrip(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: core.HTTPStatusOK,
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("a", TokenMaxBytes+1))),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})
	source := Source{HTTP: &http.Client{Transport: transport}, Audience: audience}
	if _, err := source.Token(t.Context()); err == nil {
		t.Fatal("WorkloadIdentitySource.Token() error = nil, want refusal")
	}
}

func TestWorkloadIdentitySourceOGSRefusesRedirect(t *testing.T) {
	t.Parallel()
	targetReached := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetReached = true
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirect.Close)
	audience, err := core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, workloadIdentityTestPath)
	if err != nil {
		t.Fatal(err)
	}
	source := Source{
		HTTP:     &http.Client{Transport: rewriteWorkloadIdentityURL{base: redirect.URL, transport: redirect.Client().Transport}},
		Audience: audience,
	}
	if _, err := source.Token(t.Context()); err == nil {
		t.Fatal("WorkloadIdentitySource.Token() error = nil, want redirect refusal")
	}
	if targetReached {
		t.Fatalf("redirect target reached = %t, want false", targetReached)
	}
}

type rewriteWorkloadIdentityURL struct {
	transport http.RoundTripper
	base      string
}

func (r rewriteWorkloadIdentityURL) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	parsed, err := url.Parse(r.base)
	if err != nil {
		return nil, err
	}
	copy.URL.Scheme = parsed.Scheme
	copy.URL.Host = parsed.Host
	return r.transport.RoundTrip(copy)
}
