package testserial

import (
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSerialOGSAcceptsEveryCompilerOwnedReason(t *testing.T) {
	t.Parallel()

	reasons := []core.TestSerialReason{
		core.TestSerialReasonProcessEnvironment,
		core.TestSerialReasonProcessWorkingDirectory,
		core.TestSerialReasonProcessSignal,
		core.TestSerialReasonProcessOutput,
		core.TestSerialReasonProcessLogger,
		core.TestSerialReasonRuntimeAllocation,
		core.TestSerialReasonSharedCompiledAssets,
		core.TestSerialReasonSharedCompiledPages,
		core.TestSerialReasonSharedRegistry,
		core.TestSerialReasonExternalService,
		core.TestSerialReasonOrderedState,
		core.TestSerialReasonPackageSeam,
	}
	for _, reason := range reasons {
		if err := reason.Validate(); err != nil {
			t.Fatalf("TestSerialReason(%d).Validate() error = %v", reason, err)
		}
		Serial(t, reason)
	}
}
