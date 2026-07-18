package custody

import (
	"bytes"
	"context"
	"encoding/asn1"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const testTimestampReplyPath = "/tsr"

func fixedTimestampClock() time.Time {
	return time.Unix(0, 1782302399500000000)
}

func testTimestampReplyDER(t testFatalHelper, status int) []byte {
	t.Helper()
	statusDER, err := asn1.Marshal(struct{ Status int }{Status: status})
	if err != nil {
		t.Fatal(err)
	}
	contentType, err := asn1.Marshal(rfc3161SignedDataOID())
	if err != nil {
		t.Fatal(err)
	}
	tokenDER := testDERSequence(t, append(contentType, []byte{0xa0, 0x02, 0x30, 0x00}...))
	return testDERSequence(t, append(statusDER, tokenDER...))
}

func testTimestampClientForServer(t testFatalHelper, server *httptest.Server) TimestampClient {
	t.Helper()
	endpoint, err := core.ParseAPIEndpoint(server.URL + testTimestampReplyPath)
	if err != nil {
		t.Fatal(err)
	}
	return TimestampClient{
		HTTP: server.Client(), Now: fixedTimestampClock,
		Endpoint: endpoint, Authority: TimestampAuthorityFreeTSA,
	}
}

func TestTimestampAuthorityEndpointURLTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		wantURL   string
		authority TimestampAuthority
		wantErr   bool
	}{
		{name: "freetsa resolves fixed submission url", authority: TimestampAuthorityFreeTSA, wantURL: TimestampAuthorityFreeTSAURL},
		{name: "zero authority refused", authority: timestampAuthorityInvalid, wantErr: true},
		{name: "unknown future authority refused", authority: TimestampAuthority(99), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoint, err := tc.authority.EndpointURL()
			if tc.wantErr {
				if !errors.Is(err, core.ErrCustodyContract) {
					t.Fatalf("EndpointURL() error = %v, want %v", err, core.ErrCustodyContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("EndpointURL() error = %v", err)
			}
			if got := endpoint.String(); got != tc.wantURL {
				t.Fatalf("EndpointURL() = %q, want %q", got, tc.wantURL)
			}
		})
	}
}

func TestNewFreeTSATimestampClientFailClosedTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		httpClient *http.Client
		now        func() time.Time
		name       string
		wantErr    bool
	}{
		{name: "http client and clock accepted", httpClient: http.DefaultClient, now: fixedTimestampClock},
		{name: "nil http client refused", now: fixedTimestampClock, wantErr: true},
		{name: "nil clock refused", httpClient: http.DefaultClient, wantErr: true},
		{name: "nil http client and nil clock refused", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewFreeTSATimestampClient(tc.httpClient, tc.now)
			if tc.wantErr {
				if !errors.Is(err, core.ErrCustodyContract) {
					t.Fatalf("NewFreeTSATimestampClient() error = %v, want %v", err, core.ErrCustodyContract)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewFreeTSATimestampClient() error = %v", err)
			}
			if got := client.Endpoint.String(); got != TimestampAuthorityFreeTSAURL {
				t.Fatalf("client endpoint = %q, want %q", got, TimestampAuthorityFreeTSAURL)
			}
			if client.Authority != TimestampAuthorityFreeTSA {
				t.Fatalf("client authority = %v, want %v", client.Authority, TimestampAuthorityFreeTSA)
			}
		})
	}
}

func TestTimestampClientValidateHostileTable(t *testing.T) {
	t.Parallel()

	validEndpoint, err := core.ParseAPIEndpoint(TimestampAuthorityFreeTSAURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mutate func(*TimestampClient)
		name   string
		accept bool
	}{
		{name: "fully configured client accepted", mutate: func(*TimestampClient) {}, accept: true},
		{name: "nil http client refused", mutate: func(c *TimestampClient) { c.HTTP = nil }},
		{name: "nil clock refused", mutate: func(c *TimestampClient) { c.Now = nil }},
		{name: "zero endpoint refused", mutate: func(c *TimestampClient) { c.Endpoint = core.APIEndpoint{} }},
		{name: "zero authority refused", mutate: func(c *TimestampClient) { c.Authority = timestampAuthorityInvalid }},
		{name: "unknown future authority refused", mutate: func(c *TimestampClient) { c.Authority = TimestampAuthority(99) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := TimestampClient{
				HTTP: http.DefaultClient, Now: fixedTimestampClock,
				Endpoint: validEndpoint, Authority: TimestampAuthorityFreeTSA,
			}
			tc.mutate(&client)
			err := client.Validate()
			if tc.accept {
				if err != nil {
					t.Fatalf("TimestampClient.Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrCustodyContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("TimestampClient.Validate() error = %v, want custody/foundation contract", err)
			}
		})
	}
}

func TestTimestampClientHostileTransportTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		body        func(*testing.T) []byte
		name        string
		contentType string
		status      int
		accept      bool
		wantHTTP    bool
	}{
		{name: "granted status zero reply accepted", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, accept: true},
		{name: "granted with mods status one reply accepted", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 1) }, accept: true},
		{name: "json content type refused", status: core.HTTPStatusOK, contentType: core.HTTPContentTypeJSON,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, wantHTTP: true},
		{name: "text plain content type refused", status: core.HTTPStatusOK, contentType: "text/plain",
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, wantHTTP: true},
		{name: "query content type echoed back refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeQuery,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, wantHTTP: true},
		{name: "http 400 with granted body refused", status: 400, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, wantHTTP: true},
		{name: "http 500 with granted body refused", status: core.HTTPStatusInternalServerError, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 0) }, wantHTTP: true},
		{name: "empty body refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(*testing.T) []byte { return nil }, wantHTTP: true},
		{name: "body one over maximum refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(*testing.T) []byte { return bytes.Repeat([]byte{0x30}, RFC3161DERMaximumBytes+1) }, wantHTTP: true},
		{name: "garbage body at exact maximum passes transport but fails parse", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(*testing.T) []byte { return bytes.Repeat([]byte{0x30}, RFC3161DERMaximumBytes) }},
		{name: "garbage der refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(*testing.T) []byte { return []byte("not a timestamp reply") }},
		{name: "truncated reply refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { reply := testTimestampReplyDER(t, 0); return reply[:len(reply)-1] }},
		{name: "trailing byte after reply refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return append(testTimestampReplyDER(t, 0), 0x00) }},
		{name: "rejection status two reply refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte { return testTimestampReplyDER(t, 2) }},
		{name: "status only reply without token refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(t *testing.T) []byte {
				statusDER, err := asn1.Marshal(struct{ Status int }{Status: 0})
				if err != nil {
					t.Fatal(err)
				}
				return testDERSequence(t, statusDER)
			}},
		{name: "empty der sequence reply refused", status: core.HTTPStatusOK, contentType: RFC3161ContentTypeReply,
			body: func(*testing.T) []byte { return []byte{0x30, 0x00} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bundleRoot := mustBundleRoot(t)
			imprint, wantQuery := mustTimestampQueryFixture(t, bundleRoot)
			server := newTimestampReplyTestServer(t, wantQuery, timestampReplyFixture{
				reply: tc.body(t), contentType: tc.contentType, status: tc.status,
			})
			defer server.Close()
			client := testTimestampClientForServer(t, server)
			proof, err := client.TimestampWitnessCustody(context.Background(), bundleRoot)
			if tc.accept {
				if err != nil {
					t.Fatalf("TimestampWitnessCustody() error = %v", err)
				}
				requireTimestampProofBound(t, proof, bundleRoot, imprint)
				return
			}
			requireTimestampRejection(t, err, tc.wantHTTP)
		})
	}
}

type timestampReplyFixture struct {
	contentType string
	reply       []byte
	status      int
}

func mustTimestampQueryFixture(t *testing.T, bundleRoot core.BLAKE3Hex) (core.SHA256Hex, []byte) {
	t.Helper()
	imprint, err := DeriveTimestampImprint(bundleRoot)
	if err != nil {
		t.Fatal(err)
	}
	wantQuery, err := EncodeRFC3161TimestampQuery(imprint)
	if err != nil {
		t.Fatal(err)
	}
	return imprint, wantQuery
}

func newTimestampReplyTestServer(t *testing.T, wantQuery []byte, fixture timestampReplyFixture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != testTimestampReplyPath {
			t.Errorf("request = %s %s, want POST %s", request.Method, request.URL.Path, testTimestampReplyPath)
		}
		if got := request.Header.Get(core.HTTPHeaderContentType); got != RFC3161ContentTypeQuery {
			t.Errorf("request content type = %q, want %q", got, RFC3161ContentTypeQuery)
		}
		gotQuery, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Errorf("request body read error = %v", readErr)
		}
		if !bytes.Equal(gotQuery, wantQuery) {
			t.Errorf("request query DER = %x, want %x", gotQuery, wantQuery)
		}
		if fixture.contentType != "" {
			w.Header().Set(core.HTTPHeaderContentType, fixture.contentType)
		}
		w.WriteHeader(fixture.status)
		_, _ = w.Write(fixture.reply)
	}))
}

func requireTimestampRejection(t *testing.T, err error, wantHTTP bool) {
	t.Helper()
	if !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("TimestampWitnessCustody() error = %v, want %v", err, core.ErrCustodyContract)
	}
	var httpErr TimestampHTTPError
	if got := errors.As(err, &httpErr); got != wantHTTP {
		t.Fatalf("errors.As(TimestampHTTPError) = %v, want %v; error=%v", got, wantHTTP, err)
	}
}

func requireTimestampProofBound(t *testing.T, proof TimestampProof, bundleRoot core.BLAKE3Hex, imprint core.SHA256Hex) {
	t.Helper()
	if err := proof.Validate(); err != nil {
		t.Fatalf("proof.Validate() error = %v", err)
	}
	if proof.Authority != TimestampAuthorityFreeTSA {
		t.Fatalf("proof authority = %v, want %v", proof.Authority, TimestampAuthorityFreeTSA)
	}
	if proof.BundleRoot != bundleRoot {
		t.Fatalf("proof bundle root = %v, want %v", proof.BundleRoot, bundleRoot)
	}
	if proof.ImprintSHA256 != imprint {
		t.Fatalf("proof imprint = %v, want %v", proof.ImprintSHA256, imprint)
	}
	want := core.NewUnixNanoTime(fixedTimestampClock())
	if proof.TimestampedAt != want {
		t.Fatalf("proof timestamped at = %v, want %v", proof.TimestampedAt, want)
	}
	requireTimestampTokenEmbedded(t, proof)
}

func requireTimestampTokenEmbedded(t *testing.T, proof TimestampProof) {
	t.Helper()
	tokenDER, err := proof.Token.Bytes()
	if err != nil {
		t.Fatalf("proof token bytes error = %v", err)
	}
	replyDER, err := proof.Response.Bytes()
	if err != nil {
		t.Fatalf("proof response bytes error = %v", err)
	}
	embedded, err := embeddedRFC3161Token(replyDER)
	if err != nil {
		t.Fatalf("embedded token error = %v", err)
	}
	if !bytes.Equal(tokenDER, embedded) {
		t.Fatalf("proof token = %x, want reply-embedded token %x", tokenDER, embedded)
	}
}

func TestTimestampClientFailClosedBeforeTransportTable(t *testing.T) {
	t.Parallel()

	var nilContext context.Context
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		mutate     func(*TimestampClient)
		ctx        context.Context
		name       string
		bundleRoot string
		wantHit    bool
		wantHTTP   bool
	}{
		{name: "nil context refused before transport", mutate: func(*TimestampClient) {}, ctx: nilContext, bundleRoot: "f"},
		{name: "canceled context fails typed without reaching server", mutate: func(*TimestampClient) {}, ctx: canceledContext, bundleRoot: "f", wantHTTP: true},
		{name: "nil http client refused before transport", mutate: func(c *TimestampClient) { c.HTTP = nil }, ctx: context.Background(), bundleRoot: "f"},
		{name: "nil clock refused before transport", mutate: func(c *TimestampClient) { c.Now = nil }, ctx: context.Background(), bundleRoot: "f"},
		{name: "zero authority refused before transport", mutate: func(c *TimestampClient) { c.Authority = timestampAuthorityInvalid }, ctx: context.Background(), bundleRoot: "f"},
		{name: "zero bundle root refused before transport", mutate: func(*TimestampClient) {}, ctx: context.Background(), bundleRoot: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requestSeen := make(chan struct{}, 1)
			grantedReply := testTimestampReplyDER(t, 0)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				select {
				case requestSeen <- struct{}{}:
				default:
				}
				w.Header().Set(core.HTTPHeaderContentType, RFC3161ContentTypeReply)
				_, _ = w.Write(grantedReply)
			}))
			defer server.Close()
			client := testTimestampClientForServer(t, server)
			tc.mutate(&client)
			bundleRoot := core.BLAKE3Hex{}
			if tc.bundleRoot != "" {
				bundleRoot = mustBLAKE3(t, tc.bundleRoot)
			}
			proof, err := client.TimestampWitnessCustody(tc.ctx, bundleRoot)
			if !errors.Is(err, core.ErrCustodyContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("TimestampWitnessCustody() error = %v, want custody/foundation contract", err)
			}
			var httpErr TimestampHTTPError
			if got := errors.As(err, &httpErr); got != tc.wantHTTP {
				t.Fatalf("errors.As(TimestampHTTPError) = %v, want %v; error=%v", got, tc.wantHTTP, err)
			}
			if proof != (TimestampProof{}) {
				t.Fatalf("proof = %+v, want zero on rejection", proof)
			}
			gotHit := len(requestSeen) > 0
			if gotHit != tc.wantHit {
				t.Fatalf("transport request seen = %v, want %v", gotHit, tc.wantHit)
			}
		})
	}
}

func TestTimestampClientNilContextCarriesNilContextSentinel(t *testing.T) {
	t.Parallel()

	client, err := NewFreeTSATimestampClient(http.DefaultClient, fixedTimestampClock)
	if err != nil {
		t.Fatal(err)
	}
	var nilContext context.Context
	_, err = client.TimestampWitnessCustody(nilContext, mustBundleRoot(t))
	if !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("TimestampWitnessCustody(nil ctx) error = %v, want %v", err, core.ErrNilContext)
	}
	if !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("TimestampWitnessCustody(nil ctx) error = %v, want %v", err, core.ErrCustodyContract)
	}
}

func TestTimestampClientCanceledContextCarriesCancelCause(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		t.Errorf("transport request sent on canceled context: method = %s, path = %s", request.Method, request.URL.Path)
	}))
	server.Close()
	client := testTimestampClientForServer(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.TimestampWitnessCustody(ctx, mustBundleRoot(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("TimestampWitnessCustody(canceled) error = %v, want %v", err, context.Canceled)
	}
	if !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("TimestampWitnessCustody(canceled) error = %v, want %v", err, core.ErrCustodyContract)
	}
}

func TestTimestampClientUnreachableEndpointFailsTyped(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := testTimestampClientForServer(t, server)
	server.Close()
	_, err := client.TimestampWitnessCustody(context.Background(), mustBundleRoot(t))
	if !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("TimestampWitnessCustody(closed server) error = %v, want %v", err, core.ErrCustodyContract)
	}
	var httpErr TimestampHTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("TimestampWitnessCustody(closed server) error = %v, want TimestampHTTPError", err)
	}
}
