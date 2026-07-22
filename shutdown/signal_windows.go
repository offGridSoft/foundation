//go:build windows

package shutdown

import "os"

func operatingSystemSignals(SignalSet) []os.Signal {
	return []os.Signal{os.Interrupt}
}

func classifyOperatingSystemSignal(observed os.Signal) SignalKind {
	if observed == os.Interrupt {
		return SignalKindInterrupt
	}
	return SignalKindUnknown
}
