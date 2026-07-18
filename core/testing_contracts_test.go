package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestWitnessLintWaiverFlagsAreExact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "show", got: GoWitnessLintShowWaivedFlag, want: "--show-waived"},
		{name: "hide", got: GoWitnessLintHideWaivedFlag, want: "--hide-waived"},
		{name: "fail", got: GoWitnessLintFailWaivedFlag, want: "--fail-waived"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Fatalf("flag = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestTestSerialReasonHostileTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		want    string
		name    string
		reason  TestSerialReason
		wantErr bool
	}{
		{name: "reject zero", reason: TestSerialReasonInvalid, wantErr: true},
		{name: "reject limit", reason: testSerialReasonLimit, wantErr: true},
		{name: "reject maximum", reason: TestSerialReason(^uint8(0)), wantErr: true},
		{name: "environment", reason: TestSerialReasonProcessEnvironment, want: "process_environment"},
		{name: "working directory", reason: TestSerialReasonProcessWorkingDirectory, want: "process_working_directory"},
		{name: "signal", reason: TestSerialReasonProcessSignal, want: "process_signal"},
		{name: "output", reason: TestSerialReasonProcessOutput, want: "process_output"},
		{name: "logger", reason: TestSerialReasonProcessLogger, want: "process_logger"},
		{name: "allocation", reason: TestSerialReasonRuntimeAllocation, want: "runtime_allocation"},
		{name: "compiled assets", reason: TestSerialReasonSharedCompiledAssets, want: "shared_compiled_assets"},
		{name: "compiled pages", reason: TestSerialReasonSharedCompiledPages, want: "shared_compiled_pages"},
		{name: "registry", reason: TestSerialReasonSharedRegistry, want: "shared_registry"},
		{name: "external service", reason: TestSerialReasonExternalService, want: "external_service"},
		{name: "ordered state", reason: TestSerialReasonOrderedState, want: "ordered_state"},
		{name: "package seam", reason: TestSerialReasonPackageSeam, want: "package_seam"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.reason.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrTestSerialContract) || !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("Validate() error = %v, want TestSerial and Foundation identities", err)
				}
				if tc.reason.String() != "" {
					t.Fatalf("String() = %q, want empty", tc.reason.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if got := tc.reason.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
			identifier := tc.reason.GoIdentifier()
			parsed, parseErr := ParseTestSerialReasonGoIdentifier(identifier)
			if parseErr != nil {
				t.Fatalf("ParseTestSerialReasonGoIdentifier(%q) error = %v", identifier, parseErr)
			}
			if parsed != tc.reason {
				t.Fatalf("ParseTestSerialReasonGoIdentifier(%q) = %d, want %d", identifier, parsed, tc.reason)
			}
		})
	}
}

func TestParseTestSerialReasonGoIdentifierOGSRejectsLookalikes(t *testing.T) {
	t.Parallel()

	for _, identifier := range []string{
		"",
		"TestSerialReasonInvalid",
		"TestSerialReasonProcessEnvironmentExtra",
		"LocalTestSerialReasonProcessEnvironment",
	} {
		if _, err := ParseTestSerialReasonGoIdentifier(identifier); !errors.Is(err, ErrTestSerialContract) {
			t.Fatalf("ParseTestSerialReasonGoIdentifier(%q) error = %v, want ErrTestSerialContract", identifier, err)
		}
	}
}

func TestTestSerialFunctionContract(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		GoTestSerialPackageName,
		GoTestSerialFunction,
		GoWitnessWaiverDirective,
		GoLegacySerialDirective,
		GoTestParallelDefaultRule,
	} {
		if value == "" {
			t.Fatalf("compiler-owned test protocol value = %q, want non-empty", value)
		}
	}
}

func TestTestSerialReasonJSONHostileTable(t *testing.T) {
	t.Parallel()

	for reason := TestSerialReasonProcessEnvironment; reason < testSerialReasonLimit; reason++ {
		encoded, err := json.Marshal(reason)
		if err != nil {
			t.Fatalf("json.Marshal(%d) error = %v", reason, err)
		}
		var decoded TestSerialReason
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(%d) error = %v", reason, err)
		}
		if decoded != reason {
			t.Fatalf("JSON round trip = %d, want %d", decoded, reason)
		}
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`""`), []byte(`"unknown"`), []byte(`1`)} {
		var decoded TestSerialReason
		if err := json.Unmarshal(raw, &decoded); !errors.Is(err, ErrTestSerialContract) {
			t.Fatalf("json.Unmarshal(%q) error = %v, want ErrTestSerialContract", raw, err)
		}
	}
	var decoded TestSerialReason
	if err := json.Unmarshal(nil, &decoded); err == nil {
		t.Fatalf("json.Unmarshal(nil) error = %v, want malformed JSON refusal", err)
	}
}

func FuzzTestSerialReasonValidateNeverPanics(f *testing.F) {
	for reason := TestSerialReasonInvalid; reason <= testSerialReasonLimit; reason++ {
		f.Add(uint8(reason))
	}
	f.Add(^uint8(0))

	f.Fuzz(func(t *testing.T, raw uint8) {
		reason := TestSerialReason(raw)
		err := reason.Validate()
		if reason.IsValid() {
			if err != nil {
				t.Fatalf("valid reason %d Validate() error = %v", raw, err)
			}
			if reason.String() == "" {
				t.Fatalf("valid reason %d String() is empty", raw)
			}
			return
		}
		if !errors.Is(err, ErrTestSerialContract) {
			t.Fatalf("invalid reason %d error = %v, want ErrTestSerialContract", raw, err)
		}
		if reason.String() != "" {
			t.Fatalf("invalid reason %d String() = %q, want empty", raw, reason.String())
		}
	})
}
