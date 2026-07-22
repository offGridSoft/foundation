//go:build !windows

package shutdown

import (
	"os"
	"syscall"
	"testing"
)

func TestOperatingSystemSignalSetsAreExactOnUnix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		want []os.Signal
		set  SignalSet
	}{
		{set: SignalSetInteractive, want: []os.Signal{os.Interrupt}},
		{set: SignalSetStandard, want: []os.Signal{os.Interrupt, syscall.SIGTERM}},
		{set: SignalSetTerminalLifecycle, want: []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}},
	}
	for _, testCase := range cases {
		got := operatingSystemSignals(testCase.set)
		if len(got) != len(testCase.want) {
			t.Fatalf("operatingSystemSignals(%s) length = %d, want %d", testCase.set, len(got), len(testCase.want))
		}
		for index := range got {
			if got[index] != testCase.want[index] {
				t.Fatalf("operatingSystemSignals(%s)[%d] = %v, want %v", testCase.set, index, got[index], testCase.want[index])
			}
		}
	}
	if got := operatingSystemSignals(SignalSetUnknown); got != nil {
		t.Fatalf("operatingSystemSignals(unknown) = %v, want nil", got)
	}
	classifications := []struct {
		signal os.Signal
		kind   SignalKind
	}{
		{signal: os.Interrupt, kind: SignalKindInterrupt},
		{signal: syscall.SIGTERM, kind: SignalKindTerminate},
		{signal: syscall.SIGHUP, kind: SignalKindHangup},
		{signal: syscall.SIGUSR1, kind: SignalKindUnknown},
		{signal: nil, kind: SignalKindUnknown},
	}
	for _, testCase := range classifications {
		if got := classifyOperatingSystemSignal(testCase.signal); got != testCase.kind {
			t.Fatalf("classifyOperatingSystemSignal(%v) = %s, want %s", testCase.signal, got, testCase.kind)
		}
	}
}
