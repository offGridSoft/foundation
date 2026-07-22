package shutdown

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestStepIDHostileScalarBoundaryTable(t *testing.T) {
	t.Parallel()

	maximum := strings.Repeat("a", core.ShutdownStepIDMaxRunes)
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "p01_one_rune", value: "x"},
		{name: "p02_words", value: "persistence flush"},
		{name: "p03_hyphen", value: "close-scheduler"},
		{name: "p04_underscore", value: "close_scheduler"},
		{name: "p05_dot", value: "storage.close"},
		{name: "p06_slash_is_opaque", value: "storage/close"},
		{name: "p07_unicode", value: "fermer-évidence"},
		{name: "p08_maximum", value: maximum},
		{name: "p09_shell_metacharacters_are_opaque", value: "$literal;step"},
		{name: "p10_parent_token_is_opaque", value: "../not-a-path"},
		{name: "n01_empty", wantErr: true},
		{name: "n02_space_only", value: " ", wantErr: true},
		{name: "n03_tab_only", value: "\t", wantErr: true},
		{name: "n04_newline_only", value: "\n", wantErr: true},
		{name: "n05_leading_space", value: " close", wantErr: true},
		{name: "n06_trailing_space", value: "close ", wantErr: true},
		{name: "n07_nul", value: "close\x00store", wantErr: true},
		{name: "n08_control", value: "close\x1fstore", wantErr: true},
		{name: "n09_invalid_utf8", value: string([]byte{0xff}), wantErr: true},
		{name: "n10_over_maximum", value: maximum + "a", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseStepID(tc.value)
			if tc.wantErr && (!errors.Is(err, core.ErrShutdownContract) || !errors.Is(err, core.ErrFoundationContract)) {
				t.Fatalf("ParseStepID(%q) error = %v, want shutdown and foundation identities", tc.value, err)
			}
			if !tc.wantErr && (err != nil || got.String() != tc.value || got.Validate() != nil) {
				t.Fatalf("ParseStepID(%q) = (%q,%v), validate=%v", tc.value, got, err, got.Validate())
			}
		})
	}
	if utf8.RuneCountInString(maximum) != core.ShutdownStepIDMaxRunes {
		t.Fatalf("maximum fixture runes = %d, want %d", utf8.RuneCountInString(maximum), core.ShutdownStepIDMaxRunes)
	}
}

func TestStepIDHostileJSONDoesNotMutateOnRejection(t *testing.T) {
	t.Parallel()

	valid, err := ParseStepID("storage.close")
	if err != nil {
		t.Fatalf("ParseStepID(valid) error = %v", err)
	}
	raw, err := json.Marshal(valid)
	if err != nil || string(raw) != `"storage.close"` {
		t.Fatalf("json.Marshal(valid) = (%q,%v)", raw, err)
	}
	invalid := [][]byte{nil, {}, []byte(`""`), []byte(`" close"`), []byte(`null`), []byte(`1`), []byte(`true`), []byte(`[]`), []byte(`{}`), []byte(`"close" false`), []byte(`"close`)}
	for index, data := range invalid {
		got := valid
		if err := got.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || got != valid {
			t.Fatalf("invalid case %d UnmarshalJSON(%q) = (%q,%v), want unchanged and ErrShutdownContract", index, data, got, err)
		}
	}
	var nilID *StepID
	if err := nilID.UnmarshalJSON(raw); !errors.Is(err, core.ErrShutdownContract) {
		t.Fatalf("nil StepID.UnmarshalJSON() error = %v, want ErrShutdownContract", err)
	}
}
