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

	WitnessInitialRetentionDuration = 180 * 24 * time.Hour
	WitnessPaymentExtensionDuration = 30 * 24 * time.Hour
	WitnessBronzeRetentionCap       = 180 * 24 * time.Hour
	WitnessSilverRetentionCap       = 3 * 365 * 24 * time.Hour
	WitnessGoldRetentionCap         = 10 * 365 * 24 * time.Hour
	WitnessDeletionRiskNoticeAfter  = 60 * 24 * time.Hour
	WitnessRetrievalOnlyNoticeAfter = 90 * 24 * time.Hour
	WitnessDeletionEligibleAfter    = 180 * 24 * time.Hour

	RetentionActionTokenRetain              = "retain"
	RetentionActionTokenPaymentWarning      = "payment_warning"
	RetentionActionTokenDeletionRiskWarning = "deletion_notice"
	RetentionActionTokenRetrievalOnly       = "retrieval_only"
	RetentionActionTokenDeleteEligible      = "delete_eligible"
	RetentionActionTokenLegalHold           = "legal_hold"

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
	value, err := parseStrictUint64JSON(data)
	if err != nil || value > math.MaxUint16 {
		return fmt.Errorf(ErrFmtMachineLimit, ErrWitnessPolicyContract)
	}
	parsed, err := NewMachineLimit(uint16(value))
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
	DeletionRiskNoticeAfter NanosecondsDuration `json:"deletion_risk_notice_after_ns"`
	RetrievalOnlyAfter      NanosecondsDuration `json:"retrieval_only_after_ns"`
	DeletionEligibleAfter   NanosecondsDuration `json:"deletion_eligible_after_ns"`
	PaymentNoticeImmediate  bool                `json:"payment_notice_immediate"`
	NoticeOutboxRequired    bool                `json:"notice_outbox_required"`
	DeletionEventRequired   bool                `json:"deletion_event_required"`
	LegalHoldBlocksDeletion bool                `json:"legal_hold_blocks_deletion"`
	RetentionExpiryRequired bool                `json:"retention_expiry_required"`
}

func NewWitnessRetentionPolicy(retentionCap time.Duration) WitnessRetentionPolicy {
	return WitnessRetentionPolicy{
		InitialRetention:        NewNanosecondsDuration(WitnessInitialRetentionDuration),
		PaymentExtension:        NewNanosecondsDuration(WitnessPaymentExtensionDuration),
		RetentionCap:            NewNanosecondsDuration(retentionCap),
		DeletionRiskNoticeAfter: NewNanosecondsDuration(WitnessDeletionRiskNoticeAfter),
		RetrievalOnlyAfter:      NewNanosecondsDuration(WitnessRetrievalOnlyNoticeAfter),
		DeletionEligibleAfter:   NewNanosecondsDuration(WitnessDeletionEligibleAfter),
		PaymentNoticeImmediate:  true,
		NoticeOutboxRequired:    true,
		DeletionEventRequired:   true,
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
	if err := validateWitnessRetentionDuration(p.DeletionRiskNoticeAfter); err != nil {
		return err
	}
	if err := validateWitnessRetentionDuration(p.RetrievalOnlyAfter); err != nil {
		return err
	}
	if err := validateWitnessRetentionDuration(p.DeletionEligibleAfter); err != nil {
		return err
	}
	return validateWitnessRetentionRules(p)
}

func validateWitnessRetentionRules(p WitnessRetentionPolicy) error {
	if !validWitnessRetentionRange(p) {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	if !validWitnessNoticeSchedule(p) {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	if !validWitnessRetentionRequirements(p) {
		return fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return nil
}

func validWitnessRetentionRange(policy WitnessRetentionPolicy) bool {
	return policy.InitialRetention.Duration() <= policy.RetentionCap.Duration() &&
		policy.InitialRetention.Duration() >= WitnessDeletionEligibleAfter
}

func validWitnessNoticeSchedule(policy WitnessRetentionPolicy) bool {
	return policy.DeletionRiskNoticeAfter.Duration() == WitnessDeletionRiskNoticeAfter &&
		policy.RetrievalOnlyAfter.Duration() == WitnessRetrievalOnlyNoticeAfter &&
		policy.DeletionEligibleAfter.Duration() == WitnessDeletionEligibleAfter
}

func validWitnessRetentionRequirements(policy WitnessRetentionPolicy) bool {
	return policy.PaymentNoticeImmediate && policy.NoticeOutboxRequired && policy.DeletionEventRequired &&
		policy.LegalHoldBlocksDeletion && policy.RetentionExpiryRequired
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
	result, err := AddUnixNanoDuration(base, duration.Duration())
	if err != nil {
		return UnixNanoTime{}, fmt.Errorf(ErrFmtWitnessRetentionPolicy, ErrWitnessPolicyContract)
	}
	return result, nil
}

type RetentionAction uint8

const (
	retentionActionInvalid RetentionAction = iota
	RetentionActionRetain
	RetentionActionPaymentWarning
	RetentionActionDeletionRiskWarning
	RetentionActionRetrievalOnly
	RetentionActionDeleteEligible
	RetentionActionLegalHold
)

func retentionActionNames() [RetentionActionLegalHold + 1]string {
	return [...]string{
		RetentionActionRetain:              RetentionActionTokenRetain,
		RetentionActionPaymentWarning:      RetentionActionTokenPaymentWarning,
		RetentionActionDeletionRiskWarning: RetentionActionTokenDeletionRiskWarning,
		RetentionActionRetrievalOnly:       RetentionActionTokenRetrievalOnly,
		RetentionActionDeleteEligible:      RetentionActionTokenDeleteEligible,
		RetentionActionLegalHold:           RetentionActionTokenLegalHold,
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
	Now             UnixNanoTime
	MissedPaymentAt UnixNanoTime
	NoticeHistory   WitnessCustodyNoticeHistory
	MissedPayment   bool
	LegalHold       bool
}

func (in WitnessRetentionDecisionInput) Validate() error {
	if err := in.Now.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	if err := validateWitnessMissedPaymentAt(in.MissedPaymentAt, in.Now, in.MissedPayment); err != nil {
		return err
	}
	if err := validateWitnessNoticeHistory(in.NoticeHistory, in.MissedPaymentAt, in.Now, in.MissedPayment); err != nil {
		return err
	}
	if !in.MissedPayment && in.LegalHold {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func validateWitnessMissedPaymentAt(value, now UnixNanoTime, required bool) error {
	if value.IsZero() {
		if required {
			return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
		}
		return nil
	}
	if err := value.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	if !required || value.After(now) {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func validateWitnessNoticeHistory(history WitnessCustodyNoticeHistory, missedPaymentAt, now UnixNanoTime, required bool) error {
	if err := history.Validate(); err != nil {
		return err
	}
	if !required {
		if !history.IsZero() {
			return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
		}
		return nil
	}
	return history.ValidateWindow(missedPaymentAt, now)
}

type WitnessCustodyNoticeHistory struct {
	PaymentWarningAt      UnixNanoTime `json:"payment_warning_at"`
	DeletionRiskWarningAt UnixNanoTime `json:"deletion_risk_warning_at"`
	RetrievalOnlyAt       UnixNanoTime `json:"retrieval_only_at"`
}

func (h WitnessCustodyNoticeHistory) IsZero() bool {
	return h.PaymentWarningAt.IsZero() && h.DeletionRiskWarningAt.IsZero() && h.RetrievalOnlyAt.IsZero()
}

func (h WitnessCustodyNoticeHistory) Validate() error {
	if h.PaymentWarningAt.IsZero() {
		return validateWitnessMissingPaymentNotice(h)
	}
	if err := h.PaymentWarningAt.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	if err := validateOptionalWitnessNotice(h.DeletionRiskWarningAt, h.PaymentWarningAt); err != nil {
		return err
	}
	return validateOptionalWitnessNotice(h.RetrievalOnlyAt, h.DeletionRiskWarningAt)
}

func validateOptionalWitnessNotice(value, prior UnixNanoTime) error {
	if value.IsZero() {
		return nil
	}
	if err := value.Validate(); err != nil {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, errors.Join(ErrWitnessPolicyContract, err))
	}
	if prior.IsZero() || value.Before(prior) {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func (h WitnessCustodyNoticeHistory) ValidateWindow(missedPaymentAt, now UnixNanoTime) error {
	if err := h.Validate(); err != nil {
		return err
	}
	if h.PaymentWarningAt.IsZero() {
		return nil
	}
	if !witnessNoticeTimeValid(h.PaymentWarningAt, missedPaymentAt, now) {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	if !validWitnessDeletionRiskNotice(h, missedPaymentAt, now) {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	if !validWitnessRetrievalOnlyNotice(h, missedPaymentAt, now) {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func validateWitnessMissingPaymentNotice(history WitnessCustodyNoticeHistory) error {
	if !history.DeletionRiskWarningAt.IsZero() || !history.RetrievalOnlyAt.IsZero() {
		return fmt.Errorf(ErrFmtWitnessRetentionDecision, ErrWitnessPolicyContract)
	}
	return nil
}

func validWitnessDeletionRiskNotice(history WitnessCustodyNoticeHistory, missedPaymentAt, now UnixNanoTime) bool {
	return history.DeletionRiskWarningAt.IsZero() ||
		(witnessNoticeTimeValid(history.DeletionRiskWarningAt, history.PaymentWarningAt, now) &&
			witnessNoticeThresholdValid(history.DeletionRiskWarningAt, missedPaymentAt, WitnessDeletionRiskNoticeAfter))
}

func validWitnessRetrievalOnlyNotice(history WitnessCustodyNoticeHistory, missedPaymentAt, now UnixNanoTime) bool {
	return history.RetrievalOnlyAt.IsZero() || (!history.DeletionRiskWarningAt.IsZero() &&
		witnessNoticeTimeValid(history.RetrievalOnlyAt, history.DeletionRiskWarningAt, now) &&
		witnessNoticeThresholdValid(history.RetrievalOnlyAt, missedPaymentAt, WitnessRetrievalOnlyNoticeAfter))
}

func witnessNoticeThresholdValid(value, missedPaymentAt UnixNanoTime, delay time.Duration) bool {
	threshold, err := AddUnixNanoDuration(missedPaymentAt, delay)
	return err == nil && !value.Before(threshold)
}

func witnessNoticeTimeValid(value, earliest, now UnixNanoTime) bool {
	return value.Validate() == nil && !value.Before(earliest) && !value.After(now)
}

func DecideWitnessRetention(in WitnessRetentionDecisionInput, policy WitnessRetentionPolicy) (RetentionAction, error) {
	if err := in.Validate(); err != nil {
		return retentionActionInvalid, err
	}
	if err := policy.Validate(); err != nil {
		return retentionActionInvalid, err
	}
	if in.LegalHold {
		return RetentionActionLegalHold, nil
	}
	if !in.MissedPayment {
		return RetentionActionRetain, nil
	}
	return decideMissedPaymentRetention(in, policy), nil
}

func decideMissedPaymentRetention(in WitnessRetentionDecisionInput, policy WitnessRetentionPolicy) RetentionAction {
	if in.NoticeHistory.PaymentWarningAt.IsZero() {
		return RetentionActionPaymentWarning
	}
	elapsed := in.Now.Sub(in.MissedPaymentAt)
	if elapsed >= policy.DeletionRiskNoticeAfter.Duration() && in.NoticeHistory.DeletionRiskWarningAt.IsZero() {
		return RetentionActionDeletionRiskWarning
	}
	if elapsed >= policy.DeletionEligibleAfter.Duration() && !in.NoticeHistory.RetrievalOnlyAt.IsZero() {
		return RetentionActionDeleteEligible
	}
	if elapsed >= policy.RetrievalOnlyAfter.Duration() && in.NoticeHistory.RetrievalOnlyAt.IsZero() {
		return RetentionActionRetrievalOnly
	}
	return RetentionActionRetain
}
