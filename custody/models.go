package custody

import (
	"errors"
	"fmt"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

type ArtifactDescriptor struct {
	Name   ArtifactName   `json:"name"`
	SHA256 core.SHA256Hex `json:"sha256"`
	BLAKE3 core.BLAKE3Hex `json:"blake3"`
	Size   core.ByteCount `json:"size_bytes"`
}

func (a ArtifactDescriptor) Validate() error {
	if err := a.Name.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifact, err)
	}
	if err := a.Size.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifact, err)
	}
	if err := a.SHA256.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifact, err)
	}
	if err := a.BLAKE3.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifact, err)
	}
	return nil
}

func (a ArtifactDescriptor) ArtifactSetName() string {
	return a.Name.String()
}

func (a ArtifactDescriptor) ArtifactSetSize() core.ByteCount {
	return a.Size
}

type SessionOpenRequest struct {
	Release       ReleaseIdentity      `json:"release"`
	Lease         OpenLeaseRef         `json:"lease"`
	Customer      CustomerID           `json:"customer_id"`
	BundleRoot    core.BLAKE3Hex       `json:"bundle_root"`
	Artifacts     []ArtifactDescriptor `json:"artifacts"`
	TotalBytes    core.ByteCount       `json:"total_bytes"`
	ArtifactCount uint32               `json:"artifact_count"`
	Schema        core.SchemaID        `json:"schema"`
}

func (r SessionOpenRequest) Validate() error {
	if r.Schema != core.SchemaCustodySessionOpenRequest {
		return fmt.Errorf(ErrFmtOpenRequest, core.ErrCustodyContract)
	}
	if err := r.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenRequest, err)
	}
	if err := r.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenRequest, err)
	}
	if err := r.Lease.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenRequest, err)
	}
	if err := r.Release.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenRequest, err)
	}
	return validateCoreArtifactSet(core.ArtifactSet[ArtifactDescriptor]{
		Items:      r.Artifacts,
		Count:      r.ArtifactCount,
		TotalBytes: r.TotalBytes,
	}, ErrFmtOpenRequest)
}

type SessionOpenResponse struct {
	ExistingReceipt *core.Signed[ReceiptBody] `json:"existing_receipt,omitempty"`
	Upload          *SessionUploadGrant       `json:"upload,omitempty"`
	Customer        CustomerID                `json:"customer_id"`
	BundleRoot      core.BLAKE3Hex            `json:"bundle_root"`
	Schema          core.SchemaID             `json:"schema"`
	Disposition     SessionOpenDisposition    `json:"disposition"`
}

type SessionUploadGrant struct {
	Session   SessionID         `json:"session_id"`
	Targets   []UploadTarget    `json:"targets"`
	Retention RetentionPolicy   `json:"retention"`
	ExpiresAt core.UnixNanoTime `json:"expires_at"`
}

func (g SessionUploadGrant) Validate(customer CustomerID, bundleRoot core.BLAKE3Hex) error {
	if err := g.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if err := validateUploadTargetSet(g.Targets, customer, bundleRoot, ErrFmtOpenResponse); err != nil {
		return err
	}
	if err := g.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if err := core.ValidateRequiredUnixNanoTime(g.ExpiresAt); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	return nil
}

func (r SessionOpenResponse) Validate() error {
	if r.Schema != core.SchemaCustodySessionOpenResponse {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	if err := r.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if err := r.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if err := r.Disposition.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	switch r.Disposition {
	case SessionOpenDispositionUploadRequired:
		return r.validateUploadRequired()
	case SessionOpenDispositionReceiptReused:
		return r.validateReceiptReused()
	default:
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
}

func (r SessionOpenResponse) validateUploadRequired() error {
	if r.ExistingReceipt != nil || r.Upload == nil {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	return r.Upload.Validate(r.Customer, r.BundleRoot)
}

func (r SessionOpenResponse) validateReceiptReused() error {
	if r.ExistingReceipt == nil || r.Upload != nil {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	if err := r.ExistingReceipt.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if r.ExistingReceipt.Body.Customer != r.Customer || r.ExistingReceipt.Body.BundleRoot != r.BundleRoot {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	return nil
}

func validateUploadTargetSet(targets []UploadTarget, customer CustomerID, bundleRoot core.BLAKE3Hex, errFmt string) error {
	if err := (core.CollectionCardinality{
		Length:  len(targets),
		Minimum: 1,
		Maximum: core.CollectionMaximumDefault,
	}).Validate(); err != nil {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	for index, target := range targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		if err := target.Object.ValidateWitnessIdentity(customer, bundleRoot, target.Artifact); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		for _, prior := range targets[:index] {
			if prior.Provider == target.Provider && (prior.Artifact == target.Artifact || prior.Object == target.Object) {
				return fmt.Errorf(errFmt, core.ErrCustodyContract)
			}
		}
		if !hasMatchingUploadTarget(targets, target) {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
	}
	return nil
}

func hasMatchingUploadTarget(targets []UploadTarget, target UploadTarget) bool {
	for _, candidate := range targets {
		if candidate.Provider != target.Provider && candidate.Artifact == target.Artifact && candidate.Object == target.Object {
			return true
		}
	}
	return false
}

type UploadTarget struct {
	Artifact ArtifactName         `json:"artifact"`
	Object   ObjectPath           `json:"object"`
	URL      SignedUploadURL      `json:"url"`
	Headers  []UploadHeader       `json:"headers"`
	Provider core.StorageProvider `json:"provider"`
	Method   core.UploadMethod    `json:"method"`
}

func (t UploadTarget) Validate() error {
	if err := t.Artifact.Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, err)
	}
	if err := t.Provider.Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, err)
	}
	if err := t.Object.Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, err)
	}
	if err := t.URL.Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, err)
	}
	if err := t.Method.Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, err)
	}
	return validateUploadHeaders(t.Headers)
}

func validateUploadHeaders(headers []UploadHeader) error {
	if err := (core.CollectionCardinality{
		Length:  len(headers),
		Maximum: core.HTTPHeaderMaximumDefault,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
	}
	for index, header := range headers {
		if err := header.Validate(); err != nil {
			return fmt.Errorf(ErrFmtStorage, err)
		}
		for _, prior := range headers[:index] {
			if strings.EqualFold(prior.Name, header.Name) {
				return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
			}
		}
	}
	return nil
}

type UploadHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h UploadHeader) Validate() error {
	if err := core.ValidateHTTPHeaderName(h.Name); err != nil {
		return fmt.Errorf(ErrFmtUploadHeader, core.ErrCustodyContract)
	}
	if err := core.ValidateHTTPHeaderValue(h.Value); err != nil {
		return fmt.Errorf(ErrFmtUploadHeader, core.ErrCustodyContract)
	}
	return nil
}

// Field order is signature-load-bearing when nested inside ReceiptBody.
type RetentionPolicy struct {
	RetainUntil        core.UnixNanoTime `json:"retain_until"`
	MaximumRetainUntil core.UnixNanoTime `json:"maximum_retain_until"`
	Class              RetentionClass    `json:"class"`
}

func (p RetentionPolicy) Validate() error {
	if err := p.Class.Validate(); err != nil {
		return err
	}
	if err := core.ValidateRequiredUnixNanoTime(p.RetainUntil); err != nil {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(p.MaximumRetainUntil); err != nil || p.MaximumRetainUntil.Before(p.RetainUntil) {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	return nil
}

// Field order is signature-load-bearing when nested inside ReceiptBody.
type UploadedObject struct {
	Artifact   ArtifactName         `json:"artifact"`
	Object     ObjectPath           `json:"object"`
	Generation Generation           `json:"generation"`
	SHA256     core.SHA256Hex       `json:"sha256"`
	BLAKE3     core.BLAKE3Hex       `json:"blake3"`
	Size       core.ByteCount       `json:"size_bytes"`
	Provider   core.StorageProvider `json:"provider"`
}

func (o UploadedObject) Validate() error {
	if err := o.Artifact.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.Object.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.Provider.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.Generation.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.Size.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.SHA256.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.BLAKE3.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	return nil
}

func (o UploadedObject) ArtifactSetName() string {
	return o.Artifact.String()
}

func (o UploadedObject) ArtifactSetSize() core.ByteCount {
	return o.Size
}

type FinalizeRequest struct {
	Customer   CustomerID       `json:"customer_id"`
	BundleRoot core.BLAKE3Hex   `json:"bundle_root"`
	Session    SessionID        `json:"session_id"`
	Objects    []UploadedObject `json:"objects"`
	Schema     core.SchemaID    `json:"schema"`
}

func (r FinalizeRequest) Validate() error {
	if r.Schema != core.SchemaCustodyFinalizeRequest {
		return fmt.Errorf(ErrFmtFinalize, core.ErrCustodyContract)
	}
	if err := r.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFinalize, err)
	}
	if err := r.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFinalize, err)
	}
	if err := r.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFinalize, err)
	}
	return validateUploadedObjectSet(r.Objects, r.Customer, r.BundleRoot, ErrFmtFinalize)
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type ReceiptBody struct {
	ReceiptID  ReceiptID         `json:"receipt_id"`
	Release    ReleaseIdentity   `json:"release"`
	Customer   CustomerID        `json:"customer_id"`
	BundleRoot core.BLAKE3Hex    `json:"bundle_root"`
	Session    SessionID         `json:"session_id"`
	ChainHash  core.SHA256Hex    `json:"chain_hash"`
	Objects    []UploadedObject  `json:"objects"`
	Retention  RetentionPolicy   `json:"retention"`
	IssuedAt   core.UnixNanoTime `json:"issued_at"`
	AcceptedAt core.UnixNanoTime `json:"accepted_at"`
	LedgerSeq  LedgerSeq         `json:"ledger_seq"`
	Schema     core.SchemaID     `json:"schema"`
}

func (b ReceiptBody) Validate() error {
	if b.Schema != core.SchemaCustodyReceipt {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	return validateReceiptFields(b)
}

func validateReceiptFields(b ReceiptBody) error {
	if err := validateReceiptIdentity(b); err != nil {
		return err
	}
	if err := validateUploadedObjectSet(b.Objects, b.Customer, b.BundleRoot, ErrFmtReceipt); err != nil {
		return err
	}
	if err := validateReceiptStorage(b); err != nil {
		return err
	}
	return validateReceiptLedger(b)
}

func validateReceiptIdentity(b ReceiptBody) error {
	if err := b.ReceiptID.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Release.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func validateUploadedObjectSet(objects []UploadedObject, customer CustomerID, bundleRoot core.BLAKE3Hex, errFmt string) error {
	if err := (core.CollectionCardinality{
		Length:  len(objects),
		Minimum: 1,
		Maximum: core.CollectionMaximumDefault,
	}).Validate(); err != nil {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	for index, object := range objects {
		if err := object.Validate(); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		if err := object.Object.ValidateWitnessIdentity(customer, bundleRoot, object.Artifact); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		for _, prior := range objects[:index] {
			if prior.Provider == object.Provider && (prior.Artifact == object.Artifact || prior.Object == object.Object) {
				return fmt.Errorf(errFmt, core.ErrCustodyContract)
			}
		}
		if !hasMatchingUploadedObject(objects, object) {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
	}
	return nil
}

func hasMatchingUploadedObject(objects []UploadedObject, object UploadedObject) bool {
	for _, candidate := range objects {
		if candidate.Provider != object.Provider && candidate.Artifact == object.Artifact && candidate.Object == object.Object && candidate.SHA256 == object.SHA256 && candidate.BLAKE3 == object.BLAKE3 && candidate.Size == object.Size {
			return true
		}
	}
	return false
}

func validateReceiptStorage(b ReceiptBody) error {
	if err := b.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func validateReceiptLedger(b ReceiptBody) error {
	if err := core.ValidateRequiredUnixNanoTime(b.IssuedAt); err != nil {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.AcceptedAt); err != nil || b.IssuedAt.Before(b.AcceptedAt) {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	if err := b.LedgerSeq.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.ChainHash.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func (b ReceiptBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendReceiptBodyJSON(dst, b)
}

func (b ReceiptBody) SigningSchema() core.SchemaID {
	return b.Schema
}

func (b ReceiptBody) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return appendReceiptBodyJSON(nil, b)
}

func appendReceiptBodyJSON(dst []byte, b ReceiptBody) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldRelease, b.Release)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldReceiptID, b.ReceiptID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldCustomerID, b.Customer)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldBundleRoot, b.BundleRoot)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldSessionID, b.Session)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldChainHash, b.ChainHash)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldObjects, b.Objects)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldRetention, b.Retention)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldAcceptedAt, b.AcceptedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldLedgerSeq, b.LedgerSeq)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateCoreArtifactSet[T core.ArtifactSetItem](set core.ArtifactSet[T], errFmt string) error {
	if err := core.ValidateArtifactSet(set); err != nil {
		return fmt.Errorf(errFmt, errors.Join(core.ErrCustodyContract, err))
	}
	return nil
}
