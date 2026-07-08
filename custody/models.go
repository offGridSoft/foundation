package custody

import (
	"errors"
	"fmt"

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
	Session   SessionID         `json:"session_id"`
	Targets   []UploadTarget    `json:"targets"`
	Retention RetentionPolicy   `json:"retention"`
	ExpiresAt core.UnixNanoTime `json:"expires_at"`
	Schema    core.SchemaID     `json:"schema"`
}

func (r SessionOpenResponse) Validate() error {
	if r.Schema != core.SchemaCustodySessionOpenResponse {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	if err := r.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if err := validateUploadTargetSet(r.Targets, ErrFmtOpenResponse); err != nil {
		return err
	}
	if err := r.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if r.ExpiresAt.IsZero() {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	return nil
}

func validateUploadTargetSet(targets []UploadTarget, errFmt string) error {
	if len(targets) == 0 {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	artifacts := core.NewUniqueStringSet(len(targets))
	objects := core.NewUniqueStringSet(len(targets))
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		if err := artifacts.Add(target.Artifact.String()); err != nil {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
		if err := objects.Add(target.Object.String()); err != nil {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
	}
	return nil
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
	for _, header := range t.Headers {
		if err := header.Validate(); err != nil {
			return fmt.Errorf(ErrFmtStorage, err)
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
	RetainUntil core.UnixNanoTime `json:"retain_until"`
	Class       RetentionClass    `json:"class"`
}

func (p RetentionPolicy) Validate() error {
	if err := p.Class.Validate(); err != nil {
		return err
	}
	if p.RetainUntil.IsZero() {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	return nil
}

// Field order is signature-load-bearing when nested inside ReceiptBody.
type UploadedObject struct {
	Artifact   ArtifactName   `json:"artifact"`
	Object     ObjectPath     `json:"object"`
	Generation Generation     `json:"generation"`
	SHA256     core.SHA256Hex `json:"sha256"`
	BLAKE3     core.BLAKE3Hex `json:"blake3"`
	Size       core.ByteCount `json:"size_bytes"`
}

func (o UploadedObject) Validate() error {
	if err := o.Artifact.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadedObject, err)
	}
	if err := o.Object.Validate(); err != nil {
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
	Session SessionID        `json:"session_id"`
	Objects []UploadedObject `json:"objects"`
	Schema  core.SchemaID    `json:"schema"`
}

func (r FinalizeRequest) Validate() error {
	if r.Schema != core.SchemaCustodyFinalizeRequest {
		return fmt.Errorf(ErrFmtFinalize, core.ErrCustodyContract)
	}
	if err := r.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFinalize, err)
	}
	return validateUploadedObjectSet(r.Objects, ErrFmtFinalize)
}

// Field order is storage-only; MarshalJSON owns the signature-load-bearing order.
type ReceiptBody struct {
	Release   ReleaseIdentity      `json:"release"`
	Customer  CustomerID           `json:"customer_id"`
	Session   SessionID            `json:"session_id"`
	ChainHash core.SHA256Hex       `json:"chain_hash"`
	Objects   []UploadedObject     `json:"objects"`
	Retention RetentionPolicy      `json:"retention"`
	IssuedAt  core.UnixNanoTime    `json:"issued_at"`
	LedgerSeq int64                `json:"ledger_seq"`
	Schema    core.SchemaID        `json:"schema"`
	Provider  core.StorageProvider `json:"provider"`
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
	if err := validateUploadedObjectSet(b.Objects, ErrFmtReceipt); err != nil {
		return err
	}
	if err := validateReceiptStorage(b); err != nil {
		return err
	}
	return validateReceiptLedger(b)
}

func validateReceiptIdentity(b ReceiptBody) error {
	if err := b.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Release.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func validateUploadedObjectSet(objects []UploadedObject, errFmt string) error {
	if len(objects) == 0 {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	artifacts := core.NewUniqueStringSet(len(objects))
	paths := core.NewUniqueStringSet(len(objects))
	for _, object := range objects {
		if err := object.Validate(); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		if err := artifacts.Add(object.Artifact.String()); err != nil {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
		if err := paths.Add(object.Object.String()); err != nil {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
	}
	return nil
}

func validateReceiptStorage(b ReceiptBody) error {
	if err := b.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	if err := b.Provider.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func validateReceiptLedger(b ReceiptBody) error {
	if b.LedgerSeq < 1 || b.IssuedAt.IsZero() {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	if err := b.ChainHash.Validate(); err != nil {
		return fmt.Errorf(ErrFmtReceipt, err)
	}
	return nil
}

func (b ReceiptBody) Canonical(dst []byte) ([]byte, error) {
	return core.AppendCanonicalJSON(dst, b)
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
	dst, err = core.AppendJSONField(dst, "release", b.Release)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "customer_id", b.Customer)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "session_id", b.Session)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "chain_hash", b.ChainHash)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "objects", b.Objects)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "retention", b.Retention)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "issued_at", b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "ledger_seq", b.LedgerSeq)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, "provider", b.Provider)
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
