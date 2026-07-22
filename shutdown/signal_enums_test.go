package shutdown

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSignalEnumsExhaustiveClosedStatesAndJSON(t *testing.T) {
	t.Parallel()

	signalKinds := []SignalKind{SignalKindInterrupt, SignalKindTerminate, SignalKindHangup}
	for _, value := range signalKinds {
		raw, err := json.Marshal(value)
		got := SignalKindInterrupt
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == SignalKindNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("SignalKind(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidSignalKinds := []SignalKind{SignalKindUnknown, signalKindLimit, 127, 128, 255}
	for _, value := range invalidSignalKinds {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != SignalKindNameUnknown {
			t.Fatalf("SignalKind(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(SignalKind(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}

	signalSets := []SignalSet{SignalSetInteractive, SignalSetStandard, SignalSetTerminalLifecycle}
	for _, value := range signalSets {
		raw, err := json.Marshal(value)
		got := SignalSetInteractive
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == SignalSetNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("SignalSet(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidSignalSets := []SignalSet{SignalSetUnknown, signalSetLimit, 127, 128, 255}
	for _, value := range invalidSignalSets {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != SignalSetNameUnknown {
			t.Fatalf("SignalSet(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(SignalSet(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}

	secondActions := []SecondSignalAction{SecondSignalOperatingSystemDefault, SecondSignalForce}
	for _, value := range secondActions {
		raw, err := json.Marshal(value)
		got := SecondSignalOperatingSystemDefault
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == SecondSignalNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("SecondSignalAction(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidSecondActions := []SecondSignalAction{SecondSignalUnknown, secondSignalLimit, 127, 128, 255}
	for _, value := range invalidSecondActions {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != SecondSignalNameUnknown {
			t.Fatalf("SecondSignalAction(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(SecondSignalAction(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}

	graceActions := []GraceExpiryAction{GraceExpiryDisabled, GraceExpiryForce}
	for _, value := range graceActions {
		raw, err := json.Marshal(value)
		got := GraceExpiryDisabled
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == GraceExpiryNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("GraceExpiryAction(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidGraceActions := []GraceExpiryAction{GraceExpiryUnknown, graceExpiryLimit, 127, 128, 255}
	for _, value := range invalidGraceActions {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != GraceExpiryNameUnknown {
			t.Fatalf("GraceExpiryAction(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(GraceExpiryAction(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}

	forceReasons := []ForceReason{ForceReasonSecondSignal, ForceReasonGraceExpired}
	for _, value := range forceReasons {
		raw, err := json.Marshal(value)
		got := ForceReasonSecondSignal
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == ForceReasonNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("ForceReason(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidForceReasons := []ForceReason{ForceReasonUnknown, forceReasonLimit, 127, 128, 255}
	for _, value := range invalidForceReasons {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != ForceReasonNameUnknown {
			t.Fatalf("ForceReason(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(ForceReason(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}

	forceOutcomes := []ForceOutcome{ForceOutcomeCompleted, ForceOutcomeFailed, ForceOutcomeTimedOut, ForceOutcomePanicked}
	for _, value := range forceOutcomes {
		raw, err := json.Marshal(value)
		got := ForceOutcomeCompleted
		unmarshalErr := json.Unmarshal(raw, &got)
		if !value.IsValid() || value.Validate() != nil || value.String() == ForceOutcomeNameUnknown || err != nil || unmarshalErr != nil || got != value {
			t.Fatalf("ForceOutcome(%d) state/round-trip = valid:%t string:%q raw:%q got:%d errors:%v/%v", value, value.IsValid(), value.String(), raw, got, err, unmarshalErr)
		}
	}
	invalidForceOutcomes := []ForceOutcome{ForceOutcomeUnknown, forceOutcomeLimit, 127, 128, 255}
	for _, value := range invalidForceOutcomes {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrShutdownContract) || value.String() != ForceOutcomeNameUnknown {
			t.Fatalf("ForceOutcome(%d) = valid:%t string:%q error:%v, want rejected unknown", value, value.IsValid(), value.String(), value.Validate())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrShutdownContract) {
			t.Fatalf("json.Marshal(ForceOutcome(%d)) error = %v, want ErrShutdownContract", value, err)
		}
	}
}

func TestSignalEnumsRejectMalformedJSONWithoutMutation(t *testing.T) {
	t.Parallel()

	invalidJSON := [][]byte{nil, {}, []byte(`""`), []byte(`"unknown"`), []byte(`"Interrupt"`), []byte(`0`), []byte(`true`), []byte(`null`), []byte(`[]`), []byte(`{}`), []byte(`"interrupt" false`), []byte(`"interrupt`)}
	for index, data := range invalidJSON {
		kind := SignalKindInterrupt
		if err := kind.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || kind != SignalKindInterrupt {
			t.Fatalf("SignalKind invalid JSON %d = value:%d error:%v, want unchanged interrupt and ErrShutdownContract", index, kind, err)
		}
		set := SignalSetStandard
		if err := set.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || set != SignalSetStandard {
			t.Fatalf("SignalSet invalid JSON %d = value:%d error:%v, want unchanged standard and ErrShutdownContract", index, set, err)
		}
		second := SecondSignalForce
		if err := second.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || second != SecondSignalForce {
			t.Fatalf("SecondSignalAction invalid JSON %d = value:%d error:%v, want unchanged force and ErrShutdownContract", index, second, err)
		}
		grace := GraceExpiryForce
		if err := grace.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || grace != GraceExpiryForce {
			t.Fatalf("GraceExpiryAction invalid JSON %d = value:%d error:%v, want unchanged force and ErrShutdownContract", index, grace, err)
		}
		reason := ForceReasonGraceExpired
		if err := reason.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || reason != ForceReasonGraceExpired {
			t.Fatalf("ForceReason invalid JSON %d = value:%d error:%v, want unchanged grace-expired and ErrShutdownContract", index, reason, err)
		}
		outcome := ForceOutcomePanicked
		if err := outcome.UnmarshalJSON(data); !errors.Is(err, core.ErrShutdownContract) || outcome != ForceOutcomePanicked {
			t.Fatalf("ForceOutcome invalid JSON %d = value:%d error:%v, want unchanged panicked and ErrShutdownContract", index, outcome, err)
		}
	}

	var nilKind *SignalKind
	var nilSet *SignalSet
	var nilSecond *SecondSignalAction
	var nilGrace *GraceExpiryAction
	var nilReason *ForceReason
	var nilOutcome *ForceOutcome
	errorsByEnum := []error{
		nilKind.UnmarshalJSON([]byte(`"interrupt"`)),
		nilSet.UnmarshalJSON([]byte(`"standard"`)),
		nilSecond.UnmarshalJSON([]byte(`"force"`)),
		nilGrace.UnmarshalJSON([]byte(`"force"`)),
		nilReason.UnmarshalJSON([]byte(`"grace-expired"`)),
		nilOutcome.UnmarshalJSON([]byte(`"panicked"`)),
	}
	for index, gotErr := range errorsByEnum {
		if !errors.Is(gotErr, core.ErrShutdownContract) {
			t.Fatalf("nil signal enum receiver %d error = %v, want %v", index, gotErr, core.ErrShutdownContract)
		}
	}
}

func FuzzSignalEnumJSONNeverMutatesOnRejection(f *testing.F) {
	for _, seed := range [][]byte{nil, []byte(`"interrupt"`), []byte(`"terminal-lifecycle"`), []byte(`"force"`), []byte(`"grace-expired"`), []byte(`"panicked"`), []byte(`null`), []byte{0xff}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		kind := SignalKindInterrupt
		if err := kind.UnmarshalJSON(data); err != nil && kind != SignalKindInterrupt {
			t.Fatalf("SignalKind.UnmarshalJSON(%q) mutated rejected receiver to %d", data, kind)
		}
		set := SignalSetStandard
		if err := set.UnmarshalJSON(data); err != nil && set != SignalSetStandard {
			t.Fatalf("SignalSet.UnmarshalJSON(%q) mutated rejected receiver to %d", data, set)
		}
		second := SecondSignalForce
		if err := second.UnmarshalJSON(data); err != nil && second != SecondSignalForce {
			t.Fatalf("SecondSignalAction.UnmarshalJSON(%q) mutated rejected receiver to %d", data, second)
		}
		grace := GraceExpiryForce
		if err := grace.UnmarshalJSON(data); err != nil && grace != GraceExpiryForce {
			t.Fatalf("GraceExpiryAction.UnmarshalJSON(%q) mutated rejected receiver to %d", data, grace)
		}
		reason := ForceReasonGraceExpired
		if err := reason.UnmarshalJSON(data); err != nil && reason != ForceReasonGraceExpired {
			t.Fatalf("ForceReason.UnmarshalJSON(%q) mutated rejected receiver to %d", data, reason)
		}
		outcome := ForceOutcomePanicked
		if err := outcome.UnmarshalJSON(data); err != nil && outcome != ForceOutcomePanicked {
			t.Fatalf("ForceOutcome.UnmarshalJSON(%q) mutated rejected receiver to %d", data, outcome)
		}
	})
}
