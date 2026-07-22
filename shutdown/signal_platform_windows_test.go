//go:build windows

package shutdown

import (
	"os"
	"testing"
)

func TestOperatingSystemSignalSetsAreExactOnWindows(t *testing.T) {
	t.Parallel()

	sets := []SignalSet{SignalSetInteractive, SignalSetStandard, SignalSetTerminalLifecycle, SignalSetUnknown}
	for _, set := range sets {
		got := operatingSystemSignals(set)
		if len(got) != 1 || got[0] != os.Interrupt {
			t.Fatalf("operatingSystemSignals(%s) = %v, want [os.Interrupt]", set, got)
		}
	}
	if got := classifyOperatingSystemSignal(os.Interrupt); got != SignalKindInterrupt {
		t.Fatalf("classifyOperatingSystemSignal(os.Interrupt) = %s, want interrupt", got)
	}
	if got := classifyOperatingSystemSignal(unknownOperatingSystemSignal{}); got != SignalKindUnknown {
		t.Fatalf("classifyOperatingSystemSignal(unknown) = %s, want unknown", got)
	}
	if got := classifyOperatingSystemSignal(nil); got != SignalKindUnknown {
		t.Fatalf("classifyOperatingSystemSignal(nil) = %s, want unknown", got)
	}
}
