package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	WitnessBronzeMachineLimit MachineLimit = 5
	WitnessSilverMachineLimit MachineLimit = 10
	WitnessGoldMachineLimit   MachineLimit = 10

	WitnessInitialRetentionDuration = 90 * 24 * time.Hour
	WitnessPaymentExtensionDuration = 30 * 24 * time.Hour
	WitnessBronzeRetentionCap       = 90 * 24 * time.Hour
	WitnessSilverRetentionCap       = 3 * 365 * 24 * time.Hour
	WitnessGoldRetentionCap         = 10 * 365 * 24 * time.Hour
	WitnessFirstWarningMissedCount  = 1
	WitnessExportWarningMissedCount = 2

	RetentionActionTokenRetain         = "retain"
	RetentionActionTokenPaymentWarning = "payment_warning"
	RetentionActionTokenExportWarning  = "export_warning"
	RetentionActionTokenDeleteEligible = "delete_eligible"
	RetentionActionTokenLegalHold      = "legal_hold"

	ErrFmtMachineLimit             = "core.MachineLimit: %w"
	ErrFmtWitnessRetentionPolicy   = "core.WitnessRetentionPolicy: %w"
	ErrFmtWitnessRetentionDecision = "core.WitnessRetentionDecision: %w"
)

type MachineLimit uint16

func NewMachineLimit(value uint16) (MachineLimit, error) {
	limit := MachineLimit(value)
	if err := limit.Validate(); err != nil {
		return 0, err
	}
	return limit, nil
}

func (l MachineLimit) Uint16() uint16 {
	return uint16(l)
}

func (l MachineLimit) String() string {
	if l.IsValid() {
		return strconv.FormatUint(uint64(l), 10)
	}
	return ""
}

func (l MachineLimit) IsValid() bool {
	return l > 0
}

func (l MachineLimit) Validate() error {
	if !l.IsValid() {
		return fmt.Errorf(ErrFmtMachineLimit, ErrWitnessPolicyContract)
	}
	return nil
}

func (l MachineLimit) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(uint16(l))
}

func (l *MachineLimit) UnmarshalJSON(data []byte) error {
	var value uint16
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtMachineLimit, ErrWitnessPolicyContract)
	}
	parsed, err := NewMachineLimit(value)
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

type WitnessRetentionPolicy struct {
	InitialRetention        NanosecondsDuration `json:"initial_retention_ns"`
	PaymentExtension        NanosecondsDuration `json:"payment_extension_ns"`
	RetentionCap            NanosecondsDuration `json:"retention_cap_ns"`
	FirstWarningAt          uint8               `json:"first_warning_missed_payments"`
	ExportWarningAt         uint8               `json:"export_warning_missed_payments"`
	DeletionEventRequired   bool                `json:"deletion_event_required"`
	ExportCureRequired      bool                `json:"export_cure_required"`
	LegalHoldBlocksDeletion bool                `json:"legal_hold_blocks_deletion"`
	RetentionExpiryRequired bool                `json:"retention_expiry_required"`
}

func NewWitnessRetentionPolicy(retentionCap time.Duration) WitnessRetentionPolicy {
	return WitnessRetentionPolicy{
		InitialRetention:        NewNanosecondsDuration(WitnessInitialRetentionDuration),
		PaymentExtension:        NewNanosecondsDuration(WitnessPaymentExtensionDuration),
		RetentionCap:            NewNanosecondsDuration(retentionCap),
		FirstWarningAt:          WitnessFirstWarningMissedCount,
		ExportWarningAt:         WitnessExportWarningMissedCount,
		DeletionEventRequired:   true,
		ExportCureRequired:      true,
		LegalHoldBlocksDeletion: true,
		RetentionExpiryRequired: true,
	}
}

func (p WitnessRetentionPolicy) Validate() error {
	if err := validateWitnessRetentionDuration(p.InitialRetention); err != nil {
		return err
	}
	if err := validateWitnessRetentionDuration(p.PaymentExtension); err != nil {
		return err
	}
	if err := validateWitnessRetentionDuration(p.RetentionCap); err != nil {
		return err
	}
	return validateWitnessRetentionRules(p)
}

func validateWitnessRetentionRules(p WitnessRetentionPolicy) error {
	if p.InitialRetention.Duration() > p.RetentionCap.Duration() {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	if p.FirstWarningAt != WitnessFirstWarningMissedCount || p.ExportWarningAt != WitnessExportWarningMissedCount {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	if !p.DeletionEventRequired || !p.ExportCureRequired || !p.LegalHoldBlocksDeletion || !p.RetentionExpiryRequired {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return nil
}

func validateWitnessRetentionDuration(duration NanosecondsDuration) error {
	if err := duration.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, errors.Join(ErrWitnessPolicyContract, err))
	}
	if duration.IsZero() {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return nil
}

type WitnessRetentionWindow struct {
	AcceptedAt         UnixNanoTime `json:"accepted_at_unix_ns"`
	RetainUntil        UnixNanoTime `json:"retain_until_unix_ns"`
	MaximumRetainUntil UnixNanoTime `json:"maximum_retain_until_unix_ns"`
}

func NewWitnessRetentionWindow(acceptedAt UnixNanoTime, policy WitnessRetentionPolicy) (WitnessRetentionWindow, error) {
	if err := ValidateRequiredUnixNanoTime(acceptedAt); err != nil {
		return WitnessRetentionWindow{}, fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	if err := policy.Validate(); err != nil {
		return WitnessRetentionWindow{}, err
	}
	maximum, err := addWitnessRetentionDuration(acceptedAt, policy.RetentionCap)
	if err != nil {
		return WitnessRetentionWindow{}, err
	}
	retainUntil, err := addWitnessRetentionDuration(acceptedAt, policy.InitialRetention)
	if err != nil {
		return WitnessRetentionWindow{}, err
	}
	return WitnessRetentionWindow{AcceptedAt: acceptedAt, RetainUntil: retainUntil, MaximumRetainUntil: maximum}, nil
}

func (w WitnessRetentionWindow) Validate() error {
	if err := validateWitnessRetentionTime(w.AcceptedAt); err != nil {
		return err
	}
	if err := validateWitnessRetentionTime(w.RetainUntil); err != nil {
		return err
	}
	if err := validateWitnessRetentionTime(w.MaximumRetainUntil); err != nil {
		return err
	}
	if w.RetainUntil.Before(w.AcceptedAt) || w.MaximumRetainUntil.Before(w.RetainUntil) {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return nil
}

func validateWitnessRetentionTime(value UnixNanoTime) error {
	if err := value.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, errors.Join(ErrWitnessPolicyContract, err))
	}
	return nil
}

func (w WitnessRetentionWindow) ValidatePolicy(policy WitnessRetentionPolicy) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	wantMaximum, err := addWitnessRetentionDuration(w.AcceptedAt, policy.RetentionCap)
	if err != nil || !w.MaximumRetainUntil.Equal(wantMaximum) {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return nil
}

func ExtendWitnessRetention(w WitnessRetentionWindow, policy WitnessRetentionPolicy) (WitnessRetentionWindow, error) {
	if err := w.ValidatePolicy(policy); err != nil {
		return WitnessRetentionWindow{}, err
	}
	candidate, err := addWitnessRetentionDuration(w.RetainUntil, policy.PaymentExtension)
	if err != nil || candidate.After(w.MaximumRetainUntil) {
		w.RetainUntil = w.MaximumRetainUntil
		return w, nil
	}
	w.RetainUntil = candidate
	return w, nil
}

func addWitnessRetentionDuration(base UnixNanoTime, duration NanosecondsDuration) (UnixNanoTime, error) {
	if base.UnixNano() > math.MaxInt64-duration.Nanoseconds() {
		return UnixNanoTime{}, fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return base.Add(duration.Duration()), nil
}

type RetentionAction uint8

const (
	retentionActionInvalid RetentionAction = iota
	RetentionActionRetain
	RetentionActionPaymentWarning
	RetentionActionExportWarning
	RetentionActionDeleteEligible
	RetentionActionLegalHold
)

func retentionActionNames() [RetentionActionLegalHold + 1]string {
	return [...]string{
		RetentionActionRetain:         RetentionActionTokenRetain,
		RetentionActionPaymentWarning: RetentionActionTokenPaymentWarning,
		RetentionActionExportWarning:  RetentionActionTokenExportWarning,
		RetentionActionDeleteEligible: RetentionActionTokenDeleteEligible,
		RetentionActionLegalHold:      RetentionActionTokenLegalHold,
	}
}

func (a RetentionAction) String() string {
	if a.IsValid() {
		return retentionActionNames()[a]
	}
	return ""
}

func (a RetentionAction) IsValid() bool {
	return a > retentionActionInvalid && int(a) < len(retentionActionNames()) && retentionActionNames()[a] != ""
}

func (a RetentionAction) Validate() error {
	if !a.IsValid() {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func ParseRetentionAction(token string) (RetentionAction, error) {
	for action := RetentionActionRetain; int(action) < len(retentionActionNames()); action++ {
		if retentionActionNames()[action] == token {
			return action, nil
		}
	}
	return retentionActionInvalid, fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
}

func (a RetentionAction) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(a.String())
}

func (a *RetentionAction) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	parsed, err := ParseRetentionAction(token)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

type WitnessRetentionDecisionInput struct {
	Now         UnixNanoTime
	RetainUntil UnixNanoTime
	//validate:ignore reason="every uint8 missed-payment count is a valid decision input"
	MissedPayments uint8
	//validate:ignore reason="both export-cure states are valid decision inputs"
	ExportCureComplete bool
	//validate:ignore reason="both legal-hold states are valid decision inputs"
	LegalHold bool
}

func (in WitnessRetentionDecisionInput) Validate() error {
	if err := in.Now.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	if err := in.RetainUntil.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	return nil
}

func DecideWitnessRetention(in WitnessRetentionDecisionInput) (RetentionAction, error) {
	if err := in.Validate(); err != nil {
		return retentionActionInvalid, err
	}
	if in.LegalHold {
		return RetentionActionLegalHold, nil
	}
	if witnessDeletionEligible(in) {
		return RetentionActionDeleteEligible, nil
	}
	if in.MissedPayments >= WitnessExportWarningMissedCount {
		return RetentionActionExportWarning, nil
	}
	if in.MissedPayments >= WitnessFirstWarningMissedCount {
		return RetentionActionPaymentWarning, nil
	}
	return RetentionActionRetain, nil
}

func witnessDeletionEligible(in WitnessRetentionDecisionInput) bool {
	return in.MissedPayments >= WitnessExportWarningMissedCount &&
		in.Now.After(in.RetainUntil) && in.ExportCureComplete
}
