package release

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func validSeedGrantAccessBody(t *testing.T) SeedGrantAccessBody {
	t.Helper()
	body, err := BuildSeedGrantAccessBody(validSeedGrant(t).Request(), core.NewUnixNanoTime(time.Unix(1_800_000_000, 1)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestSeedGrantAccessBodyHostileTable(t *testing.T) {
	t.Parallel()
	valid := validSeedGrantAccessBody(t)

	wrongSchema := valid
	wrongSchema.Schema = core.SchemaReleaseSeedGrant

	zeroSchema := valid
	zeroSchema.Schema = core.SchemaUnknown

	brokenIdentity := valid
	brokenIdentity.Request.Product = core.ProductWitness

	zeroIssued := valid
	zeroIssued.IssuedAt = core.UnixNanoTime{}
	zeroExpires := valid
	zeroExpires.ExpiresAt = core.UnixNanoTime{}
	wrongLifetime := valid
	wrongLifetime.ExpiresAt = wrongLifetime.ExpiresAt.Add(time.Nanosecond)

	cases := []struct {
		name      string
		body      SeedGrantAccessBody
		wantError bool
	}{
		{name: "valid access body", body: valid},
		{name: "grant schema is not the access schema", body: wrongSchema, wantError: true},
		{name: "zero schema", body: zeroSchema, wantError: true},
		{name: "release identity broken by product swap", body: brokenIdentity, wantError: true},
		{name: "zero issued at", body: zeroIssued, wantError: true},
		{name: "zero expires at", body: zeroExpires, wantError: true},
		{name: "non-canonical lifetime", body: wrongLifetime, wantError: true},
		{name: "zero body", body: SeedGrantAccessBody{}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.body.Validate()
			if tc.wantError {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("Validate() error = %v, want %v", err, core.ErrReleaseContract)
				}
				if _, canonicalErr := tc.body.Canonical(nil); canonicalErr == nil {
					t.Fatalf("Canonical(invalid body) error = %v, want errors.Is(..., %v)", canonicalErr, core.ErrReleaseContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestSeedGrantAccessBodyFreshnessHostileTable(t *testing.T) {
	t.Parallel()
	body := validSeedGrantAccessBody(t)
	cases := []struct {
		name    string
		now     core.UnixNanoTime
		wantErr bool
	}{
		{name: "issued instant accepted", now: body.IssuedAt},
		{name: "clock skew floor accepted", now: body.IssuedAt.Add(-SeedGrantAccessClockSkew)},
		{name: "expiry instant accepted", now: body.ExpiresAt},
		{name: "before clock skew rejected", now: body.IssuedAt.Add(-SeedGrantAccessClockSkew - time.Nanosecond), wantErr: true},
		{name: "after expiry rejected", now: body.ExpiresAt.Add(time.Nanosecond), wantErr: true},
		{name: "zero now rejected", now: core.UnixNanoTime{}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := body.ValidateAt(tc.now)
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateAt() error = %v", err)
			}
			if tc.wantErr && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ValidateAt() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestBuildSeedGrantAccessBodyWireStability(t *testing.T) {
	t.Parallel()
	body := validSeedGrantAccessBody(t)
	if body.SigningSchema() != core.SchemaReleaseSeedGrantAccess {
		t.Fatalf("SigningSchema() = %v, want %v", body.SigningSchema(), core.SchemaReleaseSeedGrantAccess)
	}
	wire, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	decoded, err := core.DecodeStrictJSON[SeedGrantAccessBody](wire)
	if err != nil {
		t.Fatalf("DecodeStrictJSON() error = %v", err)
	}
	if decoded != body {
		t.Fatalf("decode round trip = %+v, want %+v", decoded, body)
	}
	rewire, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("json.Marshal(decoded) error = %v", err)
	}
	if string(rewire) != string(wire) {
		t.Fatalf("re-encode differs: %s vs %s", rewire, wire)
	}
	if _, err := BuildSeedGrantAccessBody(validSeedGrant(t).Request(), core.UnixNanoTime{}); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("BuildSeedGrantAccessBody(zero time) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func TestSeedGrantAccessSignatureBinding(t *testing.T) {
	t.Parallel()
	releaseSigner := newDeployTestSigner(t, 1)
	foreignSigner := newDeployTestSigner(t, 2)
	signed := signDeployBody(t, releaseSigner, validSeedGrantAccessBody(t))
	if err := signed.Verify(releaseSigner.keyring); err != nil {
		t.Fatalf("Verify(release keyring) error = %v", err)
	}
	if err := signed.Verify(foreignSigner.keyring); !errors.Is(err, core.ErrFoundationContract) {
		t.Fatalf("Verify(foreign keyring) error = %v, want %v", err, core.ErrFoundationContract)
	}
	tampered := signed
	tampered.Body.IssuedAt = core.NewUnixNanoTime(time.Unix(1_800_000_000, 2))
	if err := tampered.Verify(releaseSigner.keyring); err == nil {
		t.Fatalf("Verify(tampered access body) error = %v, want non-nil", err)
	}
}

func seedGrantTestClient(t *testing.T, serverURL string, releaseSigner, serverSigner deployTestSigner) DeployClient {
	t.Helper()
	endpoints, err := BugDeployEndpointsForBaseURL(serverURL)
	if err != nil {
		t.Fatalf("BugDeployEndpointsForBaseURL() error = %v", err)
	}
	return DeployClient{
		HTTP: http.DefaultClient, Endpoints: endpoints,
		ReleaseKeys: releaseSigner.keyring, ServerKeys: serverSigner.keyring,
	}
}

func otherSeedGrant(t *testing.T) SeedGrantBody {
	t.Helper()
	grant := validSeedGrant(t)
	commit, err := core.ParseBuildCommit(strings.Repeat("d", 40))
	if err != nil {
		t.Fatal(err)
	}
	releaseID, err := BuildReleaseID(core.ProductBug, grant.Version, commit)
	if err != nil {
		t.Fatal(err)
	}
	body, err := BuildSeedGrantBody(
		SeedRequest{Product: core.ProductBug, Version: grant.Version, ReleaseID: releaseID, Commit: commit},
		grant.Seed, grant.IssuedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestDeployClientSeedGrantHostileTable(t *testing.T) {
	t.Parallel()
	releaseSigner := newDeployTestSigner(t, 1)
	serverSigner := newDeployTestSigner(t, 3)

	writeGrant := func(t *testing.T, w http.ResponseWriter, grant SeedGrantBody) {
		t.Helper()
		data, err := json.Marshal(signDeployBody(t, serverSigner, grant))
		if err != nil {
			t.Fatal(err)
		}
		w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		_, _ = w.Write(data)
	}

	cases := []struct {
		handler func(*testing.T, http.ResponseWriter, *http.Request)
		check   func(*testing.T, core.Signed[SeedGrantBody], error)
		name    string
	}{
		{
			name: "verified grant for the exact signed request",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != OffgridBugReleaseSeedPath {
					t.Errorf("request path = %q, want %q", r.URL.Path, OffgridBugReleaseSeedPath)
				}
				body := make([]byte, r.ContentLength)
				if _, err := r.Body.Read(body); err != nil && !errors.Is(err, io.EOF) {
					t.Errorf("read request body: %v", err)
				}
				signed, err := core.DecodeStrictJSON[core.Signed[SeedGrantAccessBody]](body)
				if err != nil {
					t.Errorf("server strict decode: %v", err)
				}
				if err := signed.Verify(releaseSigner.keyring); err != nil {
					t.Errorf("server verify: %v", err)
				}
				writeGrant(t, w, validSeedGrant(t))
			},
			check: func(t *testing.T, grant core.Signed[SeedGrantBody], err error) {
				if err != nil {
					t.Fatalf("SeedGrant() error = %v", err)
				}
				want := validSeedGrant(t).Request()
				if grant.Body.Request() != want {
					t.Fatalf("grant request = %+v, want %+v", grant.Body.Request(), want)
				}
			},
		},
		{
			name: "grant for a different release is rejected",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				writeGrant(t, w, otherSeedGrant(t))
			},
			check: func(t *testing.T, _ core.Signed[SeedGrantBody], err error) {
				if !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("SeedGrant() error = %v, want %v", err, core.ErrReleaseContract)
				}
			},
		},
		{
			name: "fault envelope maps to a typed API error",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"data":null,"error":{"message":"seed refused","code":"forbidden"},"request_id":"req-1"}`))
			},
			check: func(t *testing.T, _ core.Signed[SeedGrantBody], err error) {
				var apiErr DeployAPIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("SeedGrant() error = %v, want DeployAPIError", err)
				}
				if apiErr.StatusCode != http.StatusForbidden {
					t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusForbidden)
				}
			},
		},
		{
			name: "non-JSON response is a typed transport failure",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(core.HTTPHeaderContentType, "text/plain")
				_, _ = w.Write([]byte("grant"))
			},
			check: func(t *testing.T, _ core.Signed[SeedGrantBody], err error) {
				var httpErr DeployHTTPError
				if !errors.As(err, &httpErr) {
					t.Fatalf("SeedGrant() error = %v, want DeployHTTPError", err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.handler(t, w, r)
			}))
			defer server.Close()
			client := seedGrantTestClient(t, server.URL, releaseSigner, serverSigner)
			grant, err := client.SeedGrant(t.Context(), signDeployBody(t, releaseSigner, validSeedGrantAccessBody(t)))
			tc.check(t, grant, err)
		})
	}
}

func TestDeployClientSeedGrantRejectsLocalContractBreaks(t *testing.T) {
	t.Parallel()
	releaseSigner := newDeployTestSigner(t, 1)
	serverSigner := newDeployTestSigner(t, 3)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Errorf("server reached = %t, want false for a local contract break", true)
	}))
	defer server.Close()

	signed := signDeployBody(t, releaseSigner, validSeedGrantAccessBody(t))

	witnessEndpoints, err := WitnessDeployEndpointsForBaseURL(server.URL)
	if err != nil {
		t.Fatalf("WitnessDeployEndpointsForBaseURL() error = %v", err)
	}
	mismatched := DeployClient{
		HTTP: http.DefaultClient, Endpoints: witnessEndpoints,
		ReleaseKeys: releaseSigner.keyring, ServerKeys: serverSigner.keyring,
	}
	if _, err := mismatched.SeedGrant(t.Context(), signed); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("SeedGrant(product mismatch) error = %v, want %v", err, core.ErrReleaseContract)
	}

	client := seedGrantTestClient(t, server.URL, newDeployTestSigner(t, 2), serverSigner)
	if _, err := client.SeedGrant(t.Context(), signed); !errors.Is(err, core.ErrFoundationContract) {
		t.Fatalf("SeedGrant(foreign release key) error = %v, want %v", err, core.ErrFoundationContract)
	}

	var nilContext context.Context
	if _, err := client.SeedGrant(nilContext, signed); err == nil {
		t.Fatalf("SeedGrant(nil context) error = %v, want non-nil", err)
	}
}

func TestDeployEndpointsIncludeDistinctSeedPaths(t *testing.T) {
	t.Parallel()
	if err := validateDeployEndpointPaths(); err != nil {
		t.Fatalf("validateDeployEndpointPaths() error = %v", err)
	}
	bug, err := BugDeployEndpointsForBaseURL(core.OffgridAPIBaseURL)
	if err != nil {
		t.Fatalf("BugDeployEndpointsForBaseURL() error = %v", err)
	}
	witness, err := WitnessDeployEndpointsForBaseURL(core.OffgridAPIBaseURL)
	if err != nil {
		t.Fatalf("WitnessDeployEndpointsForBaseURL() error = %v", err)
	}
	if !strings.HasSuffix(bug.Seed.String(), OffgridBugReleaseSeedPath) {
		t.Fatalf("bug seed endpoint = %q, want suffix %q", bug.Seed.String(), OffgridBugReleaseSeedPath)
	}
	if !strings.HasSuffix(witness.Seed.String(), OffgridWitnessReleaseSeedPath) {
		t.Fatalf("witness seed endpoint = %q, want suffix %q", witness.Seed.String(), OffgridWitnessReleaseSeedPath)
	}
	if bug.Seed == witness.Seed {
		t.Fatalf("bug seed endpoint = %q, must differ from witness seed endpoint = %q", bug.Seed, witness.Seed)
	}
}
