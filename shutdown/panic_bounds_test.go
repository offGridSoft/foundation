package shutdown

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestPanicDiagnosticHostileUnicodeStaysValidAndBounded(t *testing.T) {
	t.Parallel()

	astralCapacity := (core.ShutdownPanicDiagnosticMaxRunes - 2) / 10
	cases := []struct {
		name               string
		text               string
		wantPrefixOfSource bool
	}{
		{name: "astral runes at source ceiling expand ten to one and stay bounded", text: strings.Repeat("\U0001F600", core.ShutdownPanicSourceMaxRunes), wantPrefixOfSource: true},
		{name: "three byte runes at source ceiling expand six to one and fit", text: strings.Repeat("￿", core.ShutdownPanicSourceMaxRunes), wantPrefixOfSource: true},
		{name: "nul control bytes at source ceiling expand four to one and fit", text: strings.Repeat("\x00", core.ShutdownPanicSourceMaxRunes), wantPrefixOfSource: true},
		{name: "quote backslash storm doubles and fits", text: strings.Repeat("\"\\", core.ShutdownPanicSourceMaxRunes), wantPrefixOfSource: true},
		{name: "invalid utf8 bytes are replaced then bounded", text: strings.Repeat("\xff", 700)},
		{name: "empty text stays valid", text: "", wantPrefixOfSource: true},
		{name: "plain ascii diagnostic survives verbatim", text: "disk handle wedged", wantPrefixOfSource: true},
		{name: "astral rune count exactly at quoted capacity keeps every rune", text: strings.Repeat("\U0001F600", astralCapacity), wantPrefixOfSource: true},
		{name: "astral rune count one above quoted capacity trims to fit", text: strings.Repeat("\U0001F600", astralCapacity+1), wantPrefixOfSource: true},
		{name: "oversized astral payload far above every ceiling trims to fit", text: strings.Repeat("\U0001F600", core.ShutdownPanicSourceMaxRunes*4), wantPrefixOfSource: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := newStepPanicError(newPanicValue("string", tc.text))
			force := newForcePanicError(newPanicValue("string", tc.text))
			if stepErr, forceErr := step.Validate(), force.Validate(); stepErr != nil || forceErr != nil {
				t.Fatalf("panic value validate = step %v force %v (diagnostic %q), want nil for authentic capture", stepErr, forceErr, step.Value.Diagnostic)
			}
			diagnostic := step.Value.Diagnostic.String()
			if runes := utf8.RuneCountInString(diagnostic); runes > core.ShutdownPanicDiagnosticMaxRunes {
				t.Fatalf("diagnostic runes = %d, want at most %d", runes, core.ShutdownPanicDiagnosticMaxRunes)
			}
			decoded, err := strconv.Unquote(diagnostic)
			if err != nil {
				t.Fatalf("Unquote(diagnostic) error = %v, want round-trippable ASCII quoting", err)
			}
			if runes := utf8.RuneCountInString(decoded); runes > core.ShutdownPanicSourceMaxRunes {
				t.Fatalf("decoded source runes = %d, want at most %d", runes, core.ShutdownPanicSourceMaxRunes)
			}
			if !utf8.ValidString(decoded) {
				t.Fatalf("decoded diagnostic = %q, want valid UTF-8", decoded)
			}
			if tc.wantPrefixOfSource && !strings.HasPrefix(tc.text, decoded) {
				t.Fatalf("decoded diagnostic = %q, want truncation-only prefix of source %q", decoded, tc.text)
			}
		})
	}
}

func TestPlanStepPanicWithAstralRunePayloadClassifiesPanicked(t *testing.T) {
	t.Parallel()

	id, err := ParseStepID("emoji-panic")
	if err != nil {
		t.Fatalf("ParseStepID() error = %v, want nil", err)
	}
	payload := strings.Repeat("\U0001F600", 600)
	plan := mustTestPlan(t, testPlanPolicy(time.Second, time.Second), Step{ID: id, Run: func(context.Context) error { panic(payload) }})
	report, runErr := plan.Run(t.Context())
	if !errors.Is(runErr, core.ErrShutdownStepPanic) {
		t.Fatalf("Plan.Run(astral panic) error = %v, want %v", runErr, core.ErrShutdownStepPanic)
	}
	if len(report.Results) != 1 || report.Results[0].Outcome != StepOutcomePanicked {
		t.Fatalf("Plan.Run(astral panic) report = %+v, want single result classified %d (panicked)", report, StepOutcomePanicked)
	}
	var panicErr StepPanicError
	if !errors.As(report.Results[0].Err, &panicErr) || panicErr.Validate() != nil {
		t.Fatalf("Plan.Run(astral panic) result error = %v validate=%v, want authentic valid typed panic", report.Results[0].Err, panicErr.Validate())
	}
	if !strings.HasPrefix(panicErr.Value.Diagnostic.String(), `"\U0001f600`) {
		t.Fatalf("panic diagnostic = %q, want quoted astral payload prefix preserved", panicErr.Value.Diagnostic)
	}
}
