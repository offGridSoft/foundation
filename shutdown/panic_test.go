package shutdown

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestPanicTypeAndDiagnosticHostileScalarBoundaries(t *testing.T) {
	t.Parallel()

	typeCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "simple built in type", value: "string"},
		{name: "qualified private type", value: "shutdown.hostilePanic"},
		{name: "pointer type", value: "*shutdown.hostilePanic"},
		{name: "generic type", value: "pkg.Container[int]"},
		{name: "unicode identifier", value: "pkg.Δiagnostic"},
		{name: "exact maximum", value: strings.Repeat("t", core.ShutdownPanicTypeMaxRunes)},
		{name: "empty", wantErr: true},
		{name: "newline control", value: "type\nname", wantErr: true},
		{name: "nul control", value: "type\x00name", wantErr: true},
		{name: "invalid utf8", value: string([]byte{0xff}), wantErr: true},
		{name: "one over maximum", value: strings.Repeat("t", core.ShutdownPanicTypeMaxRunes+1), wantErr: true},
	}
	for _, testCase := range typeCases {
		t.Run("type "+testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePanicTypeName(testCase.value)
			if testCase.wantErr {
				if got != "" || !errors.Is(err, core.ErrShutdownContract) {
					t.Fatalf("ParsePanicTypeName(%q) = (%q,%v), want zero and ErrShutdownContract", testCase.value, got, err)
				}
				return
			}
			if err != nil || got.String() != testCase.value || got.Validate() != nil {
				t.Fatalf("ParsePanicTypeName(%q) = (%q,%v) validate=%v, want exact valid value", testCase.value, got, err, got.Validate())
			}
		})
	}

	diagnosticCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty source is quoted", value: strconv.QuoteToASCII("")},
		{name: "plain string is canonical quoted", value: strconv.QuoteToASCII("disk wedged")},
		{name: "controls are escaped", value: strconv.QuoteToASCII("line\nnext\x00")},
		{name: "unicode is ascii escaped", value: strconv.QuoteToASCII("界")},
		{name: "exact source maximum", value: strconv.QuoteToASCII(strings.Repeat("d", core.ShutdownPanicSourceMaxRunes))},
		{name: "zero representation", wantErr: true},
		{name: "unquoted text", value: "disk wedged", wantErr: true},
		{name: "noncanonical unicode quote", value: `"界"`, wantErr: true},
		{name: "truncated quote", value: `"disk`, wantErr: true},
		{name: "raw control", value: "\"disk\nwedged\"", wantErr: true},
		{name: "one over source maximum", value: strconv.QuoteToASCII(strings.Repeat("d", core.ShutdownPanicSourceMaxRunes+1)), wantErr: true},
		{name: "invalid escape", value: `"\q"`, wantErr: true},
	}
	for _, testCase := range diagnosticCases {
		t.Run("diagnostic "+testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePanicDiagnostic(testCase.value)
			if testCase.wantErr {
				if got != "" || !errors.Is(err, core.ErrShutdownContract) {
					t.Fatalf("ParsePanicDiagnostic(%q) = (%q,%v), want zero and ErrShutdownContract", testCase.value, got, err)
				}
				return
			}
			if err != nil || got.String() != testCase.value || got.Validate() != nil {
				t.Fatalf("ParsePanicDiagnostic(%q) = (%q,%v) validate=%v, want exact canonical value", testCase.value, got, err, got.Validate())
			}
		})
	}
}
