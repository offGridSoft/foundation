package custody

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func mustDownloadURL(t testFatalHelper) DownloadURL {
	t.Helper()
	url, err := ParseDownloadURL("https://storage.googleapis.com/offgrid-custody/bundle.tar?sig=abc")
	if err != nil {
		t.Fatal(err)
	}
	return url
}

func validDownloadTarget(t testFatalHelper) DownloadTarget {
	t.Helper()
	return DownloadTarget{
		Artifact: mustArtifactNameValue(t, "bundle.tar"),
		Object:   mustObjectPath(t),
		URL:      mustDownloadURL(t),
		SHA256:   mustSHA256(t, "c"),
		BLAKE3:   mustBLAKE3(t, "d"),
		Size:     core.NewByteCount(12),
		Provider: core.StorageProviderGCS,
		Method:   DownloadMethodSignedGET,
	}
}

func validDownloadGrantBody(t testFatalHelper) DownloadGrantBody {
	t.Helper()
	return DownloadGrantBody{
		Schema:     core.SchemaCustodyDownloadGrant,
		Receipt:    mustReceiptID(t),
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Targets:    []DownloadTarget{validDownloadTarget(t)},
		IssuedAt:   core.UnixNanoTimeFromInt64(1782302400000000000),
		ExpiresAt:  core.UnixNanoTimeFromInt64(1782302700000000000),
	}
}

func validDownloadRequest(t testFatalHelper) DownloadRequest {
	t.Helper()
	return DownloadRequest{
		Schema:     core.SchemaCustodyDownloadRequest,
		Customer:   mustCustomerID(t),
		BundleRoot: mustBundleRoot(t),
		Lease:      mustOpenLeaseRef(t),
	}
}

func signedDownloadGrant(t testFatalHelper, body DownloadGrantBody) (core.Signed[DownloadGrantBody], core.SigningKeyring) {
	t.Helper()
	keyID, publicKey, privateKey := receiptSigningKey(t)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[DownloadGrantBody]{Body: body, KeyID: keyID, Signature: signature},
		core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
}

func TestDownloadURLHostileTable(t *testing.T) {
	t.Parallel()
	valid := []string{
		"https://storage.googleapis.com/offgrid-custody/bundle.tar?sig=abc",
		"https://storage.googleapis.com/offgrid-custody/bundle.tar",
		"https://127.0.0.1:8443/object?x=1",
	}
	for _, value := range valid {
		if _, err := ParseDownloadURL(value); err != nil {
			t.Fatalf("ParseDownloadURL(%q) error = %v, want nil", value, err)
		}
	}
	hostile := []string{
		"",
		"http://storage.googleapis.com/offgrid-custody/bundle.tar",
		"https://storage.googleapis.com",
		"https://storage.googleapis.com/object#fragment",
		"https://user:pass@storage.googleapis.com/object",
		"ftp://storage.googleapis.com/object",
		"https://storage.googleapis.com/" + strings.Repeat("a", core.HTTPSURLDefaultMaxRunes),
		"https://storage.googleapis.com/obj\x00ect",
	}
	for _, value := range hostile {
		if _, err := ParseDownloadURL(value); !errors.Is(err, core.ErrCustodyContract) {
			t.Fatalf("ParseDownloadURL(%q) error = %v, want %v", value, err, core.ErrCustodyContract)
		}
	}
	var zero DownloadURL
	if err := zero.Validate(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("zero DownloadURL Validate() error = %v, want %v", zero.Validate(), core.ErrCustodyContract)
	}
}

func TestDownloadMethodWireContract(t *testing.T) {
	t.Parallel()
	parsed, err := ParseDownloadMethod(DownloadMethodTokenSignedGET)
	if err != nil || parsed != DownloadMethodSignedGET {
		t.Fatalf("ParseDownloadMethod(%q) = %v, %v, want %v, nil", DownloadMethodTokenSignedGET, parsed, err, DownloadMethodSignedGET)
	}
	encoded, err := DownloadMethodSignedGET.MarshalJSON()
	if err != nil || string(encoded) != `"signed_get"` {
		t.Fatalf("DownloadMethodSignedGET.MarshalJSON() = %s, %v, want \"signed_get\", nil", encoded, err)
	}
	if _, err := downloadMethodInvalid.MarshalJSON(); !errors.Is(err, core.ErrCustodyContract) {
		t.Fatalf("invalid method MarshalJSON() error = %v, want %v", err, core.ErrCustodyContract)
	}
	for _, token := range []string{"", "SIGNED_GET", "signed-get", "signed_get ", "resumable_uri", "signed_put", "signed_get2"} {
		if _, err := ParseDownloadMethod(token); !errors.Is(err, core.ErrCustodyContract) {
			t.Fatalf("ParseDownloadMethod(%q) error = %v, want %v", token, err, core.ErrCustodyContract)
		}
	}
	method := DownloadMethodSignedGET
	if err := method.UnmarshalJSON([]byte(`"garbage"`)); !errors.Is(err, core.ErrCustodyContract) || method != DownloadMethodSignedGET {
		t.Fatalf("UnmarshalJSON rejection = %v method=%v, want contract violation with unmutated receiver", err, method)
	}
}

func TestDownloadRequestHostileTable(t *testing.T) {
	t.Parallel()
	if err := validDownloadRequest(t).Validate(); err != nil {
		t.Fatalf("valid DownloadRequest Validate() error = %v, want nil", err)
	}
	cases := []struct {
		mutate func(t *testing.T, r *DownloadRequest)
		name   string
	}{
		{name: "zero schema is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.Schema = core.SchemaUnknown }},
		{name: "grant schema on request is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.Schema = core.SchemaCustodyDownloadGrant }},
		{name: "open schema on request is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.Schema = core.SchemaCustodySessionOpenRequest }},
		{name: "zero customer is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.Customer = CustomerID{} }},
		{name: "zero bundle root is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.BundleRoot = core.BLAKE3Hex{} }},
		{name: "zero lease is rejected", mutate: func(_ *testing.T, r *DownloadRequest) { r.Lease = OpenLeaseRef{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := validDownloadRequest(t)
			tc.mutate(t, &request)
			if err := request.Validate(); !errors.Is(err, core.ErrCustodyContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Validate() error = %v, want contract violation", err)
			}
		})
	}
}

func TestDownloadGrantBodyHostileTable(t *testing.T) {
	t.Parallel()
	if err := validDownloadGrantBody(t).Validate(); err != nil {
		t.Fatalf("valid DownloadGrantBody Validate() error = %v, want nil", err)
	}
	cases := []struct {
		mutate func(t *testing.T, b *DownloadGrantBody)
		name   string
	}{
		{name: "request schema on grant is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Schema = core.SchemaCustodyDownloadRequest }},
		{name: "receipt schema on grant is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Schema = core.SchemaCustodyReceipt }},
		{name: "zero receipt id is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Receipt = ReceiptID{} }},
		{name: "zero customer is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Customer = CustomerID{} }},
		{name: "zero bundle root is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.BundleRoot = core.BLAKE3Hex{} }},
		{name: "empty target set is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets = nil }},
		{name: "duplicate artifact targets are rejected", mutate: func(t *testing.T, b *DownloadGrantBody) {
			second := validDownloadTarget(t)
			b.Targets = append(b.Targets, second)
		}},
		{name: "duplicate object paths are rejected", mutate: func(t *testing.T, b *DownloadGrantBody) {
			second := validDownloadTarget(t)
			second.Artifact = mustArtifactNameValue(t, "bundle2.tar")
			b.Targets = append(b.Targets, second)
		}},
		{name: "s3 provider is rejected on the gcs-only path", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].Provider = core.StorageProviderS3 }},
		{name: "foreign customer object identity is rejected", mutate: func(t *testing.T, b *DownloadGrantBody) {
			object, err := ParseObjectPath(core.WitnessCustodyPathRoot + "/01HZZZZZZZZZZZZZZZZZZZZZZ0/2026/07/" + mustBundleRoot(t).String() + "/bundle.tar")
			if err != nil {
				t.Fatal(err)
			}
			b.Targets[0].Object = object
		}},
		{name: "artifact name and object leaf drift is rejected", mutate: func(t *testing.T, b *DownloadGrantBody) {
			b.Targets[0].Artifact = mustArtifactNameValue(t, "bundle2.tar")
		}},
		{name: "zero issued time is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.IssuedAt = core.UnixNanoTime{} }},
		{name: "zero expiry is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.ExpiresAt = core.UnixNanoTime{} }},
		{name: "expiry equal to issue is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.ExpiresAt = b.IssuedAt }},
		{name: "expiry before issue is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.ExpiresAt = b.IssuedAt.Add(-time.Nanosecond) }},
		{name: "zero target url is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].URL = DownloadURL{} }},
		{name: "zero target method is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].Method = downloadMethodInvalid }},
		{name: "zero target sha256 is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].SHA256 = core.SHA256Hex{} }},
		{name: "zero target blake3 is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].BLAKE3 = core.BLAKE3Hex{} }},
		{name: "zero target size is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].Size = core.NewByteCount(0) }},
		{name: "int64-overflow target size is rejected", mutate: func(_ *testing.T, b *DownloadGrantBody) { b.Targets[0].Size = core.NewByteCount(math.MaxUint64) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := validDownloadGrantBody(t)
			tc.mutate(t, &body)
			if err := body.Validate(); !errors.Is(err, core.ErrCustodyContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("Validate() error = %v, want contract violation", err)
			}
			if _, err := body.Canonical(nil); err == nil {
				t.Fatal("Canonical() error = nil, want validation-gated refusal")
			}
		})
	}
}

func TestDownloadGrantValidateAtWindowBoundaries(t *testing.T) {
	t.Parallel()
	body := validDownloadGrantBody(t)
	cases := []struct {
		name    string
		now     core.UnixNanoTime
		wantErr bool
	}{
		{name: "at issue is accepted", now: body.IssuedAt},
		{name: "at expiry is accepted", now: body.ExpiresAt},
		{name: "one nanosecond after expiry is rejected", now: body.ExpiresAt.Add(time.Nanosecond), wantErr: true},
		{name: "at skew floor before issue is accepted", now: body.IssuedAt.Add(-DownloadGrantClockSkew)},
		{name: "one nanosecond below skew floor is rejected", now: body.IssuedAt.Add(-DownloadGrantClockSkew - time.Nanosecond), wantErr: true},
		{name: "zero clock is rejected", now: core.UnixNanoTime{}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := body.ValidateAt(tc.now)
			if tc.wantErr && !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("ValidateAt() error = %v, want %v", err, core.ErrCustodyContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateAt() error = %v, want nil", err)
			}
		})
	}
}

func TestDownloadGrantBindReceiptTable(t *testing.T) {
	t.Parallel()
	if err := validDownloadGrantBody(t).BindReceipt(validReceipt(t)); err != nil {
		t.Fatalf("BindReceipt() error = %v, want nil for matching object set", err)
	}
	cases := []struct {
		mutateGrant   func(t *testing.T, b *DownloadGrantBody)
		mutateReceipt func(t *testing.T, r *ReceiptBody)
		name          string
	}{
		{name: "receipt id drift is rejected", mutateGrant: func(t *testing.T, b *DownloadGrantBody) {
			receipt, err := ParseReceiptID("01J00000000000000000000009")
			if err != nil {
				t.Fatal(err)
			}
			b.Receipt = receipt
		}},
		{name: "target sha drift from receipt object is rejected", mutateGrant: func(t *testing.T, b *DownloadGrantBody) {
			b.Targets[0].SHA256 = mustSHA256(t, "9")
		}},
		{name: "target blake3 drift from receipt object is rejected", mutateGrant: func(t *testing.T, b *DownloadGrantBody) {
			b.Targets[0].BLAKE3 = mustBLAKE3(t, "9")
		}},
		{name: "target size drift from receipt object is rejected", mutateGrant: func(_ *testing.T, b *DownloadGrantBody) {
			b.Targets[0].Size = core.NewByteCount(13)
		}},
		{name: "receipt object without a grant target is rejected", mutateReceipt: func(t *testing.T, r *ReceiptBody) {
			second := mustUploadedObject(t)
			second.Artifact = mustArtifactNameValue(t, "bundle2.tar")
			second.Object = witnessObjectPathFor(t, "bundle2.tar")
			r.Objects = append(r.Objects, second)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			grant := validDownloadGrantBody(t)
			receipt := validReceipt(t)
			if tc.mutateGrant != nil {
				tc.mutateGrant(t, &grant)
			}
			if tc.mutateReceipt != nil {
				tc.mutateReceipt(t, &receipt)
			}
			if err := grant.BindReceipt(receipt); !errors.Is(err, core.ErrCustodyContract) {
				t.Fatalf("BindReceipt() error = %v, want %v", err, core.ErrCustodyContract)
			}
		})
	}
}

func TestDownloadGrantCanonicalRoundTripStability(t *testing.T) {
	t.Parallel()
	body := validDownloadGrantBody(t)
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatalf("Canonical() error = %v, want nil", err)
	}
	decoded, err := core.DecodeStrictJSON[DownloadGrantBody](canonical)
	if err != nil {
		t.Fatalf("DecodeStrictJSON(canonical) error = %v, want nil", err)
	}
	again, err := decoded.Canonical(nil)
	if err != nil || !bytes.Equal(canonical, again) {
		t.Fatalf("canonical instability: error = %v\n got %s\nwant %s", err, again, canonical)
	}
	marshaled, err := json.Marshal(body)
	if err != nil || !bytes.Equal(marshaled, canonical) {
		t.Fatalf("MarshalJSON = %s (err %v), want canonical %s", marshaled, err, canonical)
	}
}

func TestRequestDownloadReturnsVerifiedBoundGrant(t *testing.T) {
	t.Parallel()
	request := validDownloadRequest(t)
	signed, keyring := signedDownloadGrant(t, validDownloadGrantBody(t))
	wantBody, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != OffgridWitnessCustodyDownloadPath {
			t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, OffgridWitnessCustodyDownloadPath)
		}
		gotBody, readErr := io.ReadAll(r.Body)
		if readErr != nil || !bytes.Equal(gotBody, wantBody) {
			t.Errorf("request body = %s (err %v), want %s", gotBody, readErr, wantBody)
		}
		writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, signed))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	got, err := client.RequestDownload(context.Background(), request)
	if err != nil {
		t.Fatalf("RequestDownload() error = %v, want nil", err)
	}
	if got.Signature != signed.Signature || got.Body.Receipt != signed.Body.Receipt || len(got.Body.Targets) != 1 {
		t.Fatalf("RequestDownload() = %+v, want the signed grant returned intact", got)
	}
}

func TestRequestDownloadHostileTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		respond    func(t *testing.T) core.Signed[DownloadGrantBody]
		name       string
		wantAPIErr bool
	}{
		{
			name: "forged grant signature is rejected",
			respond: func(t *testing.T) core.Signed[DownloadGrantBody] {
				signed, _ := signedDownloadGrant(t, validDownloadGrantBody(t))
				forged, err := core.ParseEd25519SignatureHex(strings.Repeat("b", 128))
				if err != nil {
					t.Fatal(err)
				}
				signed.Signature = forged
				return signed
			},
		},
		{
			name: "tampered grant body breaks the signature",
			respond: func(t *testing.T) core.Signed[DownloadGrantBody] {
				signed, _ := signedDownloadGrant(t, validDownloadGrantBody(t))
				signed.Body.ExpiresAt = signed.Body.ExpiresAt.Add(time.Hour)
				return signed
			},
		},
		{
			name: "grant for a foreign customer is rejected",
			respond: func(t *testing.T) core.Signed[DownloadGrantBody] {
				body := validDownloadGrantBody(t)
				foreign, err := ParseCustomerID("01HZZZZZZZZZZZZZZZZZZZZZZ0")
				if err != nil {
					t.Fatal(err)
				}
				body.Customer = foreign
				object, err := ParseObjectPath(core.WitnessCustodyPathRoot + "/" + foreign.String() + "/2026/07/" + mustBundleRoot(t).String() + "/bundle.tar")
				if err != nil {
					t.Fatal(err)
				}
				body.Targets[0].Object = object
				signed, _ := signedDownloadGrant(t, body)
				return signed
			},
		},
		{
			name: "grant for a foreign bundle root is rejected",
			respond: func(t *testing.T) core.Signed[DownloadGrantBody] {
				body := validDownloadGrantBody(t)
				body.BundleRoot = mustBLAKE3(t, "0")
				object, err := ParseObjectPath(core.WitnessCustodyPathRoot + "/" + mustCustomerID(t).String() + "/2026/07/" + body.BundleRoot.String() + "/bundle.tar")
				if err != nil {
					t.Fatal(err)
				}
				body.Targets[0].Object = object
				signed, _ := signedDownloadGrant(t, body)
				return signed
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			signed := tc.respond(t)
			_, keyring := signedDownloadGrant(t, validDownloadGrantBody(t))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeCustodyJSON(w, http.StatusOK, custodyEnvelopeJSON(t, signed))
			}))
			defer server.Close()
			client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
			_, err := client.RequestDownload(context.Background(), validDownloadRequest(t))
			if !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("RequestDownload() error = %v, want %v", err, core.ErrFoundationContract)
			}
		})
	}
}

func TestRequestDownloadSurfacesAPIRefusal(t *testing.T) {
	t.Parallel()
	_, keyring := signedDownloadGrant(t, validDownloadGrantBody(t))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCustodyJSON(w, http.StatusForbidden, custodyFailureEnvelopeJSON[core.Signed[DownloadGrantBody]](t, core.APICodeForbidden, "retention expired"))
	}))
	defer server.Close()
	client := Client{HTTP: server.Client(), ServerKeys: keyring, Endpoints: custodyTestEndpoints(t, server.URL)}
	_, err := client.RequestDownload(context.Background(), validDownloadRequest(t))
	var apiErr CustodyAPIError
	if !errors.As(err, &apiErr) || apiErr.Body.Code != core.APICodeForbidden {
		t.Fatalf("RequestDownload() error = %v, want CustodyAPIError with %v", err, core.APICodeForbidden)
	}
	var nilContext context.Context
	if _, nilCtxErr := client.RequestDownload(nilContext, validDownloadRequest(t)); !errors.Is(nilCtxErr, core.ErrNilContext) {
		t.Fatalf("RequestDownload(nil ctx) error = %v, want %v", nilCtxErr, core.ErrNilContext)
	}
}

func downloadFixture(t testFatalHelper, payload []byte, serverURL string) DownloadTarget {
	t.Helper()
	url, err := ParseDownloadURL(serverURL + "/offgrid-custody/bundle.tar?sig=abc")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	target := validDownloadTarget(t)
	target.URL = url
	target.SHA256 = core.NewSHA256Hex(sum)
	target.Size = core.NewByteCount(uint64(len(payload)))
	return target
}

func TestDownloadArtifactStreamsAndVerifiesContentAddress(t *testing.T) {
	t.Parallel()
	payload := []byte("retrieved custody bundle payload")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("download method = %s, want %s", r.Method, http.MethodGet)
		}
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	client := uploadTestClient(t, server)
	var dst bytes.Buffer
	if err := client.DownloadArtifact(context.Background(), downloadFixture(t, payload, server.URL), &dst); err != nil {
		t.Fatalf("DownloadArtifact() error = %v, want nil", err)
	}
	if !bytes.Equal(dst.Bytes(), payload) {
		t.Fatalf("downloaded bytes = %q, want %q", dst.Bytes(), payload)
	}
}

func TestDownloadArtifactHostileTable(t *testing.T) {
	t.Parallel()
	payload := []byte("retrieved custody bundle payload")
	cases := []struct {
		handler     func(w http.ResponseWriter, r *http.Request)
		mutate      func(t *testing.T, target *DownloadTarget)
		makeContext func() (context.Context, context.CancelFunc)
		name        string
		nilWriter   bool
		wantStorage bool
	}{
		{
			name: "truncated stream is refused",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(payload[:len(payload)-4])
			},
			wantStorage: true,
		},
		{
			name: "oversized stream is refused",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(append(append([]byte{}, payload...), "extra"...))
			},
			wantStorage: true,
		},
		{
			name: "tampered bytes fail hash verification",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				tampered := append([]byte{}, payload...)
				tampered[0] ^= 0xff
				_, _ = w.Write(tampered)
			},
			wantStorage: true,
		},
		{
			name: "expired signed url status is surfaced",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
		},
		{
			name: "missing object status is surfaced",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		},
		{
			name:      "nil writer fails closed",
			nilWriter: true,
		},
		{
			name: "invalid target fails closed before transport",
			mutate: func(_ *testing.T, target *DownloadTarget) {
				target.SHA256 = core.SHA256Hex{}
			},
		},
		{
			name:        "nil context fails closed",
			makeContext: func() (context.Context, context.CancelFunc) { return nil, func() {} },
		},
		{
			name: "canceled context aborts the transfer",
			makeContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := tc.handler
			if handler == nil {
				handler = func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write(payload)
				}
			}
			server := httptest.NewTLSServer(http.HandlerFunc(handler))
			defer server.Close()
			client := uploadTestClient(t, server)
			target := downloadFixture(t, payload, server.URL)
			if tc.mutate != nil {
				tc.mutate(t, &target)
			}
			ctx := context.Context(context.Background())
			if tc.makeContext != nil {
				madeCtx, cancel := tc.makeContext()
				defer cancel()
				ctx = madeCtx
			}
			var dst bytes.Buffer
			writer := io.Writer(&dst)
			if tc.nilWriter {
				writer = nil
			}
			err := client.DownloadArtifact(ctx, target, writer)
			if !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DownloadArtifact() error = %v, want %v", err, core.ErrFoundationContract)
			}
			if tc.wantStorage && !errors.Is(err, core.ErrStorageVerification) {
				t.Fatalf("DownloadArtifact() error = %v, want %v", err, core.ErrStorageVerification)
			}
		})
	}
}
