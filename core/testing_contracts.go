package core

import (
	"encoding/json"
	"fmt"
)

// TestSerialReason is the compiler-owned reason a Go test cannot safely call
// testing.T.Parallel. It replaces informal serial comments with a closed
// contract that Witness-lint can recognize and downstream test helpers can
// validate.
type TestSerialReason uint8

const (
	TestSerialReasonInvalid TestSerialReason = iota
	TestSerialReasonProcessEnvironment
	TestSerialReasonProcessWorkingDirectory
	TestSerialReasonProcessSignal
	TestSerialReasonProcessOutput
	TestSerialReasonProcessLogger
	TestSerialReasonRuntimeAllocation
	TestSerialReasonSharedCompiledAssets
	TestSerialReasonSharedCompiledPages
	TestSerialReasonSharedRegistry
	TestSerialReasonExternalService
	TestSerialReasonOrderedState
	TestSerialReasonPackageSeam
	testSerialReasonLimit
)

const (
	GoTestSerialPackageName   = "testserial"
	GoTestSerialFunction      = "Serial"
	GoWitnessWaiverDirective  = "witness:waiver"
	GoLegacySerialDirective   = "serial:"
	GoTestParallelDefaultRule = "test/parallel/default"
)

func testSerialReasonNames() [testSerialReasonLimit]string {
	return [...]string{
		TestSerialReasonProcessEnvironment:      "process_environment",
		TestSerialReasonProcessWorkingDirectory: "process_working_directory",
		TestSerialReasonProcessSignal:           "process_signal",
		TestSerialReasonProcessOutput:           "process_output",
		TestSerialReasonProcessLogger:           "process_logger",
		TestSerialReasonRuntimeAllocation:       "runtime_allocation",
		TestSerialReasonSharedCompiledAssets:    "shared_compiled_assets",
		TestSerialReasonSharedCompiledPages:     "shared_compiled_pages",
		TestSerialReasonSharedRegistry:          "shared_registry",
		TestSerialReasonExternalService:         "external_service",
		TestSerialReasonOrderedState:            "ordered_state",
		TestSerialReasonPackageSeam:             "package_seam",
	}
}

func (r TestSerialReason) String() string {
	if !r.IsValid() {
		return ""
	}
	return testSerialReasonNames()[r]
}

func (r TestSerialReason) IsValid() bool {
	return r > TestSerialReasonInvalid && r < testSerialReasonLimit && testSerialReasonNames()[r] != ""
}

// Validate rejects the zero value and values outside the closed reason set.
func (r TestSerialReason) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtTestSerialReason, ErrTestSerialContract)
	}
	return nil
}

func (r TestSerialReason) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func ParseTestSerialReason(token string) (TestSerialReason, error) {
	for reason := TestSerialReasonProcessEnvironment; reason < testSerialReasonLimit; reason++ {
		if reason.String() == token {
			return reason, nil
		}
	}
	return TestSerialReasonInvalid, fmt.Errorf(ErrFmtTestSerialReason, ErrTestSerialContract)
}

func (r *TestSerialReason) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtTestSerialReason, ErrTestSerialContract)
	}
	parsed, err := ParseTestSerialReason(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
