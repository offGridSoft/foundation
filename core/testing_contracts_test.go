package core

import (
	"errors"
	"testing"
)

func TestTestSerialReasonHostileTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		reason  TestSerialReason
		want    string
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
		})
	}
}

func TestTestSerialFunctionContract(t *testing.T) {
	t.Parallel()

	if GoTestSerialPackageName == "" {
		t.Fatal("GoTestSerialPackageName is empty")
	}
	if GoTestSerialFunction == "" {
		t.Fatal("GoTestSerialFunction is empty")
	}
	if GoWitnessWaiverDirective == "" {
		t.Fatal("GoWitnessWaiverDirective is empty")
	}
	if GoLegacySerialDirective == "" {
		t.Fatal("GoLegacySerialDirective is empty")
	}
	if GoTestParallelDefaultRule == "" {
		t.Fatal("GoTestParallelDefaultRule is empty")
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
		if reason.Valid() {
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
