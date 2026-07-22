package shutdown

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type SignalKind uint8

const (
	SignalKindUnknown SignalKind = iota
	SignalKindInterrupt
	SignalKindTerminate
	SignalKindHangup
	signalKindLimit
)

const (
	SignalKindNameUnknown   = "unknown"
	SignalKindNameInterrupt = "interrupt"
	SignalKindNameTerminate = "terminate"
	SignalKindNameHangup    = "hangup"
	ErrFmtSignalKind        = "shutdown.SignalKind: %w"
)

func signalKindNames() [signalKindLimit]string {
	return [...]string{SignalKindInterrupt: SignalKindNameInterrupt, SignalKindTerminate: SignalKindNameTerminate, SignalKindHangup: SignalKindNameHangup}
}

func (k SignalKind) IsValid() bool {
	return k > SignalKindUnknown && k < signalKindLimit && signalKindNames()[k] != ""
}
func (k SignalKind) Validate() error {
	if !k.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (k SignalKind) String() string {
	if !k.IsValid() {
		return SignalKindNameUnknown
	}
	return signalKindNames()[k]
}
func ParseSignalKind(token string) (SignalKind, error) {
	for value := SignalKindInterrupt; value < signalKindLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return SignalKindUnknown, fmt.Errorf(ErrFmtSignalKind, core.ErrShutdownContract)
}
func (k SignalKind) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(k.String(), k.Validate(), ErrFmtSignalKind)
}
func (k *SignalKind) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, k, ParseSignalKind, ErrFmtSignalKind)
}

type SignalSet uint8

const (
	SignalSetUnknown SignalSet = iota
	SignalSetInteractive
	SignalSetStandard
	SignalSetTerminalLifecycle
	signalSetLimit
)

const (
	SignalSetNameUnknown           = "unknown"
	SignalSetNameInteractive       = "interactive"
	SignalSetNameStandard          = "standard"
	SignalSetNameTerminalLifecycle = "terminal-lifecycle"
	ErrFmtSignalSet                = "shutdown.SignalSet: %w"
)

func signalSetNames() [signalSetLimit]string {
	return [...]string{SignalSetInteractive: SignalSetNameInteractive, SignalSetStandard: SignalSetNameStandard, SignalSetTerminalLifecycle: SignalSetNameTerminalLifecycle}
}
func (s SignalSet) IsValid() bool {
	return s > SignalSetUnknown && s < signalSetLimit && signalSetNames()[s] != ""
}
func (s SignalSet) Validate() error {
	if !s.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (s SignalSet) String() string {
	if !s.IsValid() {
		return SignalSetNameUnknown
	}
	return signalSetNames()[s]
}
func ParseSignalSet(token string) (SignalSet, error) {
	for value := SignalSetInteractive; value < signalSetLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return SignalSetUnknown, fmt.Errorf(ErrFmtSignalSet, core.ErrShutdownContract)
}
func (s SignalSet) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(s.String(), s.Validate(), ErrFmtSignalSet)
}
func (s *SignalSet) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, s, ParseSignalSet, ErrFmtSignalSet)
}

type SecondSignalAction uint8

const (
	SecondSignalUnknown SecondSignalAction = iota
	SecondSignalOperatingSystemDefault
	SecondSignalForce
	secondSignalLimit
)

const (
	SecondSignalNameUnknown                = "unknown"
	SecondSignalNameOperatingSystemDefault = "operating-system-default"
	SecondSignalNameForce                  = "force"
	ErrFmtSecondSignalAction               = "shutdown.SecondSignalAction: %w"
)

func secondSignalNames() [secondSignalLimit]string {
	return [...]string{SecondSignalOperatingSystemDefault: SecondSignalNameOperatingSystemDefault, SecondSignalForce: SecondSignalNameForce}
}
func (a SecondSignalAction) IsValid() bool {
	return a > SecondSignalUnknown && a < secondSignalLimit && secondSignalNames()[a] != ""
}
func (a SecondSignalAction) Validate() error {
	if !a.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (a SecondSignalAction) String() string {
	if !a.IsValid() {
		return SecondSignalNameUnknown
	}
	return secondSignalNames()[a]
}
func ParseSecondSignalAction(token string) (SecondSignalAction, error) {
	for value := SecondSignalOperatingSystemDefault; value < secondSignalLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return SecondSignalUnknown, fmt.Errorf(ErrFmtSecondSignalAction, core.ErrShutdownContract)
}
func (a SecondSignalAction) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(a.String(), a.Validate(), ErrFmtSecondSignalAction)
}
func (a *SecondSignalAction) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, a, ParseSecondSignalAction, ErrFmtSecondSignalAction)
}

type GraceExpiryAction uint8

const (
	GraceExpiryUnknown GraceExpiryAction = iota
	GraceExpiryDisabled
	GraceExpiryForce
	graceExpiryLimit
)

const (
	GraceExpiryNameUnknown  = "unknown"
	GraceExpiryNameDisabled = "disabled"
	GraceExpiryNameForce    = "force"
	ErrFmtGraceExpiryAction = "shutdown.GraceExpiryAction: %w"
)

func graceExpiryNames() [graceExpiryLimit]string {
	return [...]string{GraceExpiryDisabled: GraceExpiryNameDisabled, GraceExpiryForce: GraceExpiryNameForce}
}
func (a GraceExpiryAction) IsValid() bool {
	return a > GraceExpiryUnknown && a < graceExpiryLimit && graceExpiryNames()[a] != ""
}
func (a GraceExpiryAction) Validate() error {
	if !a.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (a GraceExpiryAction) String() string {
	if !a.IsValid() {
		return GraceExpiryNameUnknown
	}
	return graceExpiryNames()[a]
}
func ParseGraceExpiryAction(token string) (GraceExpiryAction, error) {
	for value := GraceExpiryDisabled; value < graceExpiryLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return GraceExpiryUnknown, fmt.Errorf(ErrFmtGraceExpiryAction, core.ErrShutdownContract)
}
func (a GraceExpiryAction) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(a.String(), a.Validate(), ErrFmtGraceExpiryAction)
}
func (a *GraceExpiryAction) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, a, ParseGraceExpiryAction, ErrFmtGraceExpiryAction)
}

type ForceReason uint8

const (
	ForceReasonUnknown ForceReason = iota
	ForceReasonSecondSignal
	ForceReasonGraceExpired
	forceReasonLimit
)

const (
	ForceReasonNameUnknown      = "unknown"
	ForceReasonNameSecondSignal = "second-signal"
	ForceReasonNameGraceExpired = "grace-expired"
	ErrFmtForceReason           = "shutdown.ForceReason: %w"
)

func forceReasonNames() [forceReasonLimit]string {
	return [...]string{ForceReasonSecondSignal: ForceReasonNameSecondSignal, ForceReasonGraceExpired: ForceReasonNameGraceExpired}
}
func (r ForceReason) IsValid() bool {
	return r > ForceReasonUnknown && r < forceReasonLimit && forceReasonNames()[r] != ""
}
func (r ForceReason) Validate() error {
	if !r.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (r ForceReason) String() string {
	if !r.IsValid() {
		return ForceReasonNameUnknown
	}
	return forceReasonNames()[r]
}
func ParseForceReason(token string) (ForceReason, error) {
	for value := ForceReasonSecondSignal; value < forceReasonLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return ForceReasonUnknown, fmt.Errorf(ErrFmtForceReason, core.ErrShutdownContract)
}
func (r ForceReason) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(r.String(), r.Validate(), ErrFmtForceReason)
}
func (r *ForceReason) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, r, ParseForceReason, ErrFmtForceReason)
}

type ForceOutcome uint8

const (
	ForceOutcomeUnknown ForceOutcome = iota
	ForceOutcomeCompleted
	ForceOutcomeFailed
	ForceOutcomeTimedOut
	ForceOutcomePanicked
	forceOutcomeLimit
)

const (
	ForceOutcomeNameUnknown   = "unknown"
	ForceOutcomeNameCompleted = "completed"
	ForceOutcomeNameFailed    = "failed"
	ForceOutcomeNameTimedOut  = "timed-out"
	ForceOutcomeNamePanicked  = "panicked"
	ErrFmtForceOutcome        = "shutdown.ForceOutcome: %w"
)

func forceOutcomeNames() [forceOutcomeLimit]string {
	return [...]string{ForceOutcomeCompleted: ForceOutcomeNameCompleted, ForceOutcomeFailed: ForceOutcomeNameFailed, ForceOutcomeTimedOut: ForceOutcomeNameTimedOut, ForceOutcomePanicked: ForceOutcomeNamePanicked}
}
func (o ForceOutcome) IsValid() bool {
	return o > ForceOutcomeUnknown && o < forceOutcomeLimit && forceOutcomeNames()[o] != ""
}
func (o ForceOutcome) Validate() error {
	if !o.IsValid() {
		return core.ErrShutdownContract
	}
	return nil
}
func (o ForceOutcome) String() string {
	if !o.IsValid() {
		return ForceOutcomeNameUnknown
	}
	return forceOutcomeNames()[o]
}
func ParseForceOutcome(token string) (ForceOutcome, error) {
	for value := ForceOutcomeCompleted; value < forceOutcomeLimit; value++ {
		if value.String() == token {
			return value, nil
		}
	}
	return ForceOutcomeUnknown, fmt.Errorf(ErrFmtForceOutcome, core.ErrShutdownContract)
}
func (o ForceOutcome) MarshalJSON() ([]byte, error) {
	return marshalShutdownEnum(o.String(), o.Validate(), ErrFmtForceOutcome)
}
func (o *ForceOutcome) UnmarshalJSON(data []byte) error {
	return unmarshalShutdownEnum(data, o, ParseForceOutcome, ErrFmtForceOutcome)
}

func marshalShutdownEnum(token string, validationErr error, format string) ([]byte, error) {
	if validationErr != nil {
		return nil, fmt.Errorf(format, validationErr)
	}
	return json.Marshal(token)
}

func unmarshalShutdownEnum[T any](data []byte, target *T, parse func(string) (T, error), format string) error {
	if target == nil {
		return fmt.Errorf(format, core.ErrShutdownContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(format, errors.Join(core.ErrShutdownContract, err))
	}
	parsed, err := parse(token)
	if err != nil {
		return err
	}
	*target = parsed
	return nil
}
