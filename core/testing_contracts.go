package core

import "fmt"

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
	GoTestSerialPackageName = "testlock"
	GoTestSerialFunction    = "Serial"
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
	if !r.Valid() {
		return ""
	}
	return testSerialReasonNames()[r]
}

func (r TestSerialReason) Valid() bool {
	return r > TestSerialReasonInvalid && r < testSerialReasonLimit && testSerialReasonNames()[r] != ""
}

// Validate rejects the zero value and values outside the closed reason set.
func (r TestSerialReason) Validate() error {
	if !r.Valid() {
		return fmt.Errorf(ErrFmtTestSerialReason, ErrTestSerialContract)
	}
	return nil
}
