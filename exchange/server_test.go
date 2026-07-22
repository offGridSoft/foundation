package exchange

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	receiveFixtureNameMaxRunes = 32
	receiveFixtureCountMaximum = 100
	receiveFixtureBodyLimit    = 256
)

type receiveFixture struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (f receiveFixture) Validate() error {
	if !utf8.ValidString(f.Name) || f.Name == "" || strings.TrimSpace(f.Name) != f.Name || utf8.RuneCountInString(f.Name) > receiveFixtureNameMaxRunes {
		return core.ErrFoundationContract
	}
	if f.Count < 0 || f.Count > receiveFixtureCountMaximum {
		return core.ErrFoundationContract
	}
	return nil
}

func TestReceiveJSONHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	semantics := core.HTTPRouteSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt}
	policy := ServerPolicy{RequestBodyLimit: core.NewByteCount(receiveFixtureBodyLimit)}
	cases := []struct {
		setup   func(*testing.T) *http.Request
		wantErr error
		name    string
		want    receiveFixture
	}{
		{name: "one rune and zero count accept exact semantic floors", setup: receiveRequest(`{"name":"a","count":0}`), want: receiveFixture{Name: "a", Count: 0}},
		{name: "thirty two runes and count one hundred accept exact ceilings", setup: receiveRequest(`{"name":"` + strings.Repeat("n", receiveFixtureNameMaxRunes) + `","count":100}`), want: receiveFixture{Name: strings.Repeat("n", receiveFixtureNameMaxRunes), Count: 100}},
		{name: "middle values accept canonical object", setup: receiveRequest(`{"name":"kernel","count":50}`), want: receiveFixture{Name: "kernel", Count: 50}},
		{name: "reordered fields preserve typed meaning", setup: receiveRequest(`{"count":1,"name":"witness"}`), want: receiveFixture{Name: "witness", Count: 1}},
		{name: "json whitespace remains wire neutral", setup: receiveRequest(" \n\t{\"name\":\"bug\",\"count\":99}\r\n"), want: receiveFixture{Name: "bug", Count: 99}},
		{name: "escaped quote decodes before validation", setup: receiveRequest(`{"name":"peach\"fuzz","count":2}`), want: receiveFixture{Name: `peach"fuzz`, Count: 2}},
		{name: "multibyte runes count as runes not bytes", setup: receiveRequest(`{"name":"é界","count":3}`), want: receiveFixture{Name: "é界", Count: 3}},
		{name: "one count accepts one above floor", setup: receiveRequest(`{"name":"a","count":1}`), want: receiveFixture{Name: "a", Count: 1}},
		{name: "ninety nine count accepts one below ceiling", setup: receiveRequest(`{"name":"z","count":99}`), want: receiveFixture{Name: "z", Count: 99}},
		{name: "body exactly at byte limit accepts trailing whitespace", setup: receiveRequestAtExactLimit, want: receiveFixture{Name: "limit", Count: 4}},
		{name: "empty body rejects missing object", setup: receiveRequest(""), wantErr: core.ErrExchangeRequest},
		{name: "whitespace body rejects missing object", setup: receiveRequest(" \n\t"), wantErr: core.ErrExchangeRequest},
		{name: "truncated object rejects partial transport", setup: receiveRequest(`{"name":"a"`), wantErr: core.ErrExchangeRequest},
		{name: "truncated string rejects partial token", setup: receiveRequest(`{"name":"a,"count":1}`), wantErr: core.ErrExchangeRequest},
		{name: "unknown field rejects schema drift", setup: receiveRequest(`{"name":"a","count":1,"future":true}`), wantErr: core.ErrExchangeRequest},
		{name: "duplicate name rejects ambiguous identity", setup: receiveRequest(`{"name":"a","name":"b","count":1}`), wantErr: core.ErrExchangeRequest},
		{name: "duplicate count rejects ambiguous quantity", setup: receiveRequest(`{"name":"a","count":1,"count":2}`), wantErr: core.ErrExchangeRequest},
		{name: "string count rejects type confusion", setup: receiveRequest(`{"name":"a","count":"1"}`), wantErr: core.ErrExchangeRequest},
		{name: "numeric name rejects type confusion", setup: receiveRequest(`{"name":1,"count":1}`), wantErr: core.ErrExchangeRequest},
		{name: "null rejects required object", setup: receiveRequest(`null`), wantErr: core.ErrExchangeRequest},
		{name: "array rejects object substitution", setup: receiveRequest(`[{"name":"a","count":1}]`), wantErr: core.ErrExchangeRequest},
		{name: "scalar rejects object substitution", setup: receiveRequest(`1`), wantErr: core.ErrExchangeRequest},
		{name: "boolean rejects object substitution", setup: receiveRequest(`true`), wantErr: core.ErrExchangeRequest},
		{name: "invalid utf8 rejects hostile wire bytes", setup: receiveByteRequest([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', ',', '"', 'c', 'o', 'u', 'n', 't', '"', ':', '1', '}'}), wantErr: core.ErrExchangeRequest},
		{name: "empty name rejects semantic zero", setup: receiveRequest(`{"name":"","count":1}`), wantErr: core.ErrExchangeRequest},
		{name: "thirty three runes reject one above semantic ceiling", setup: receiveRequest(`{"name":"` + strings.Repeat("n", receiveFixtureNameMaxRunes+1) + `","count":1}`), wantErr: core.ErrExchangeRequest},
		{name: "negative count rejects one below semantic floor", setup: receiveRequest(`{"name":"a","count":-1}`), wantErr: core.ErrExchangeRequest},
		{name: "count one hundred one rejects one above semantic ceiling", setup: receiveRequest(`{"name":"a","count":101}`), wantErr: core.ErrExchangeRequest},
		{name: "maximum integer rejects far above semantic ceiling", setup: receiveRequest(`{"name":"a","count":` + intWire(math.MaxInt) + `}`), wantErr: core.ErrExchangeRequest},
		{name: "trailing object rejects request smuggling", setup: receiveRequest(`{"name":"a","count":1}{"name":"b","count":2}`), wantErr: core.ErrExchangeRequest},
		{name: "body one byte above byte limit rejects streaming overflow", setup: receiveRequest(strings.Repeat(" ", receiveFixtureBodyLimit+1)), wantErr: core.ErrExchangeBodyLimit},
		{name: "declared length one above limit rejects before read", setup: requestWithDeclaredLength(receiveFixtureBodyLimit + 1), wantErr: core.ErrExchangeBodyLimit},
		{name: "declared maximum length rejects far above limit", setup: requestWithDeclaredLength(math.MaxInt64), wantErr: core.ErrExchangeBodyLimit},
		{name: "nil body rejects missing stream", setup: requestWithNilBody, wantErr: core.ErrExchangeRequest},
		{name: "text content type rejects parser confusion", setup: requestWithContentType("text/plain"), wantErr: core.ErrExchangeContentType},
		{name: "missing content type rejects implicit protocol", setup: requestWithContentType(""), wantErr: core.ErrExchangeContentType},
		{name: "get method rejects route contract mismatch", setup: requestWithMethod(http.MethodGet), wantErr: core.ErrExchangeRequest},
		{name: "connect method rejects unsupported future method", setup: requestWithMethod(http.MethodConnect), wantErr: core.ErrExchangeRequest},
		{name: "cancelled request rejects before consuming body", setup: cancelledReceiveRequest, wantErr: core.ErrExchangeCancelled},
		{name: "gzip content encoding rejects undeclared transform", setup: requestWithContentEncoding("gzip"), wantErr: core.ErrExchangeContentType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := tc.setup(t)
			got, gotErr := ReceiveJSON[receiveFixture](request, semantics, policy)
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("ReceiveJSON() error = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr == nil && got.Body != tc.want {
				t.Fatalf("ReceiveJSON() body = %+v, want %+v", got.Body, tc.want)
			}
			if tc.wantErr != nil && got != (Received[receiveFixture]{}) {
				t.Fatalf("ReceiveJSON() rejected value = %+v, want zero value", got)
			}
		})
	}
}

func receiveRequest(body string) func(*testing.T) *http.Request {
	return receiveByteRequest([]byte(body))
}

func receiveByteRequest(body []byte) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "https://api.offgridsoftware.ca/v1/exchange", bytes.NewReader(body))
		request.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
		return request
	}
}

func receiveRequestAtExactLimit(t *testing.T) *http.Request {
	t.Helper()
	prefix := `{"name":"limit","count":4}`
	body := prefix + strings.Repeat(" ", receiveFixtureBodyLimit-len(prefix))
	return receiveRequest(body)(t)
}

func requestWithDeclaredLength(length int64) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := receiveRequest(`{"name":"unread","count":1}`)(t)
		request.ContentLength = length
		request.Body = io.NopCloser(bodyReadMustNotRun{})
		return request
	}
}

type bodyReadMustNotRun struct{}

func (bodyReadMustNotRun) Read([]byte) (int, error) {
	return 0, core.ErrFoundationContract
}

func requestWithNilBody(t *testing.T) *http.Request {
	t.Helper()
	request := receiveRequest(`{"name":"a","count":1}`)(t)
	request.Body = nil
	return request
}

func requestWithContentType(contentType string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := receiveRequest(`{"name":"a","count":1}`)(t)
		request.Header.Set(core.HTTPHeaderContentType, contentType)
		return request
	}
}

func requestWithMethod(method string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := receiveRequest(`{"name":"a","count":1}`)(t)
		request.Method = method
		return request
	}
}

func cancelledReceiveRequest(t *testing.T) *http.Request {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return receiveRequest(`{"name":"a","count":1}`)(t).WithContext(ctx)
}

func requestWithContentEncoding(encoding string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		t.Helper()
		request := receiveRequest(`{"name":"a","count":1}`)(t)
		request.Header.Set("Content-Encoding", encoding)
		return request
	}
}

func intWire(value int) string {
	return strconv.Itoa(value)
}
