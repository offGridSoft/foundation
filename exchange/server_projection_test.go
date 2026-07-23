package exchange

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	projectionHeaderTenant       = "X-Projection-Tenant"
	projectionTenantMaximumRunes = 16
	projectionNameMaximumRunes   = 24
	projectionBodyLimit          = 128
)

type projectedReceiveFixture struct {
	Name   string `json:"name"`
	Tenant string `json:"-"`
}

func (f *projectedReceiveFixture) Validate() error {
	if f == nil {
		return core.ErrFoundationContract
	}
	if err := validateProjectionText(f.Name, projectionNameMaximumRunes); err != nil {
		return err
	}
	return validateProjectionText(f.Tenant, projectionTenantMaximumRunes)
}

func validateProjectionText(value string, maximum int) error {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximum {
		return core.ErrFoundationContract
	}
	return nil
}

func projectReceiveTenant(_ context.Context, request *http.Request, body *projectedReceiveFixture) error {
	body.Tenant = request.Header.Get(projectionHeaderTenant)
	return nil
}

func rejectReceiveProjection(context.Context, *http.Request, *projectedReceiveFixture) error {
	return core.ErrAccessContract
}

func TestReceiveProjectedJSONHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	semantics := core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt}
	policy := ServerPolicy{RequestBodyLimit: core.NewByteCount(projectionBodyLimit)}
	cases := []struct {
		project JSONProjector[projectedReceiveFixture, *projectedReceiveFixture]
		setup   func(*testing.T) *http.Request
		wantErr error
		name    string
		want    projectedReceiveFixture
	}{
		{
			name:  "one rune floors survive structure projection",
			setup: projectedReceiveRequest(`{"name":"a"}`, "t"), project: projectReceiveTenant,
			want: projectedReceiveFixture{Name: "a", Tenant: "t"},
		},
		{
			name:  "exact rune ceilings survive structure projection",
			setup: projectedReceiveRequest(`{"name":"`+strings.Repeat("n", projectionNameMaximumRunes)+`"}`, strings.Repeat("t", projectionTenantMaximumRunes)), project: projectReceiveTenant,
			want: projectedReceiveFixture{Name: strings.Repeat("n", projectionNameMaximumRunes), Tenant: strings.Repeat("t", projectionTenantMaximumRunes)},
		},
		{name: "missing projected field fails completed body validation", setup: projectedReceiveRequest(`{"name":"valid"}`, ""), project: projectReceiveTenant, wantErr: core.ErrExchangeRequest},
		{name: "projected field one rune above ceiling fails completed body validation", setup: projectedReceiveRequest(`{"name":"valid"}`, strings.Repeat("t", projectionTenantMaximumRunes+1)), project: projectReceiveTenant, wantErr: core.ErrExchangeRequest},
		{name: "wire field one rune above ceiling fails completed body validation", setup: projectedReceiveRequest(`{"name":"`+strings.Repeat("n", projectionNameMaximumRunes+1)+`"}`, "tenant"), project: projectReceiveTenant, wantErr: core.ErrExchangeRequest},
		{name: "unknown wire projection field is rejected before projection", setup: projectedReceiveRequest(`{"name":"valid","tenant":"smuggled"}`, "tenant"), project: projectReceiveTenant, wantErr: core.ErrExchangeRequest},
		{name: "projection error preserves shared request and owned error identities", setup: projectedReceiveRequest(`{"name":"valid"}`, "tenant"), project: rejectReceiveProjection, wantErr: core.ErrAccessContract},
		{name: "nil projector fails closed", setup: projectedReceiveRequest(`{"name":"valid"}`, "tenant"), wantErr: core.ErrFoundationContract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := ReceiveProjectedJSON(tc.setup(t), semantics, policy, tc.project)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("ReceiveProjectedJSON() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr == nil && (got.Body == nil || *got.Body != tc.want) {
				t.Fatalf("ReceiveProjectedJSON() body = %+v, want %+v", got.Body, tc.want)
			}
			if tc.wantErr != nil && got.Body != nil {
				t.Fatalf("ReceiveProjectedJSON() rejected body = %+v, want nil", got.Body)
			}
		})
	}
}

func projectedReceiveRequest(body, tenant string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "https://api.offgridsoftware.ca/v1/projected", strings.NewReader(body))
		request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		if tenant != "" {
			request.Header.Set(projectionHeaderTenant, tenant)
		}
		return request
	}
}

func TestReceiveNoBodyHostileMethodAndMetadataTable(t *testing.T) {
	t.Parallel()

	type noBodyCase struct {
		mutate  func(*http.Request)
		wantErr error
		name    string
		method  core.HTTPMethod
		replay  core.HTTPReplaySafety
	}
	cases := []noBodyCase{
		{name: "get safe accepts exact empty request", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe},
		{name: "head safe accepts exact empty request", method: core.HTTPMethodHead, replay: core.HTTPReplaySafe},
		{name: "options safe accepts exact empty request", method: core.HTTPMethodOptions, replay: core.HTTPReplaySafe},
		{name: "delete idempotent accepts exact empty request", method: core.HTTPMethodDelete, replay: core.HTTPReplayIdempotent},
		{name: "post single attempt accepts explicitly empty request", method: core.HTTPMethodPost, replay: core.HTTPReplaySingleAttempt},
		{name: "put idempotent accepts explicitly empty request", method: core.HTTPMethodPut, replay: core.HTTPReplayIdempotent},
		{name: "patch single attempt accepts explicitly empty request", method: core.HTTPMethodPatch, replay: core.HTTPReplaySingleAttempt},
		{name: "one byte body rejects smuggling", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithBody, wantErr: core.ErrExchangeRequest},
		{name: "unknown length body rejects smuggling", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithUnknownLength, wantErr: core.ErrExchangeRequest},
		{name: "content type without body rejects implicit protocol", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithContentType, wantErr: core.ErrExchangeContentType},
		{name: "content encoding without body rejects transform", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithContentEncoding, wantErr: core.ErrExchangeContentType},
		{name: "transfer encoding without body rejects framing ambiguity", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithTransferEncoding, wantErr: core.ErrExchangeRequest},
		{name: "idempotency key on safe get rejects replay contradiction", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithIdempotencyKey, wantErr: core.ErrExchangeRequest},
		{name: "wire method mismatch rejects route confusion", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithMismatchedMethod, wantErr: core.ErrExchangeRequest},
		{name: "cancelled request rejects before application dispatch", method: core.HTTPMethodGet, replay: core.HTTPReplaySafe, mutate: noBodyWithCancelledContext, wantErr: core.ErrExchangeCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(tc.method.String(), "https://api.offgridsoftware.ca/v1/no-body", nil)
			if tc.mutate != nil {
				tc.mutate(request)
			}
			semantics := core.HTTPRouteSemantics{Method: tc.method, Replay: tc.replay}
			got, gotErr := ReceiveNoBody(request, semantics)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("ReceiveNoBody() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr == nil {
				if validationErr := got.Validate(); validationErr != nil {
					t.Fatalf("ReceiveNoBody() result validation error = %v, want nil", validationErr)
				}
				return
			}
			if got != (Received[core.HTTPNoBody]{}) {
				t.Fatalf("ReceiveNoBody() rejected result = %+v, want zero value", got)
			}
		})
	}
}

func TestReceiveNoBodyRejectsNilRequest(t *testing.T) {
	t.Parallel()

	semantics := core.HTTPRouteSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySafe}
	got, gotErr := ReceiveNoBody(nil, semantics)
	if !errors.Is(gotErr, core.ErrExchangeRequest) || !errors.Is(gotErr, core.ErrFoundationContract) {
		t.Fatalf("ReceiveNoBody(nil) error = %v, want request and foundation contract identities", gotErr)
	}
	if got != (Received[core.HTTPNoBody]{}) {
		t.Fatalf("ReceiveNoBody(nil) result = %+v, want zero value", got)
	}
}

func noBodyWithBody(request *http.Request) {
	request.Body = http.NoBody
	request.ContentLength = 1
}

func noBodyWithUnknownLength(request *http.Request) {
	request.Body = http.NoBody
	request.ContentLength = -1
}

func noBodyWithContentType(request *http.Request) {
	request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
}

func noBodyWithContentEncoding(request *http.Request) {
	request.Header.Set(core.HTTPHeaderContentEncoding, "gzip")
}

func noBodyWithTransferEncoding(request *http.Request) {
	request.TransferEncoding = []string{"chunked"}
}

func noBodyWithIdempotencyKey(request *http.Request) {
	request.Header.Set(core.HTTPHeaderIdempotencyKey, "idempotency-key")
}

func noBodyWithMismatchedMethod(request *http.Request) {
	request.Method = http.MethodHead
}

func noBodyWithCancelledContext(request *http.Request) {
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	*request = *request.WithContext(ctx)
}
