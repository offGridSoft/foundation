package custody

import (
	"fmt"
	"math"
	"strings"

	"github.com/offGridSoft/foundation/core"
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

type SessionOpenRequest struct {
	Schema        string               `json:"schema"`
	Customer      CustomerID           `json:"customer_id"`
	Lease         OpenLeaseRef         `json:"lease"`
	Release       ReleaseIdentity      `json:"release"`
	Artifacts     []ArtifactDescriptor `json:"artifacts"`
	TotalBytes    core.ByteCount       `json:"total_bytes"`
	ArtifactCount uint32               `json:"artifact_count"`
}

func (r SessionOpenRequest) Validate() error {
	if r.Schema != SchemaSessionOpenRequest {
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
	return validateArtifactSet(r.Artifacts, r.ArtifactCount, r.TotalBytes, ErrFmtOpenRequest)
}

func validateArtifactSet(
	artifacts []ArtifactDescriptor,
	count uint32,
	total core.ByteCount,
	errFmt string,
) error {
	if count == 0 || int(count) != len(artifacts) {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	var sum uint64
	names := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if err := artifact.Validate(); err != nil {
			return fmt.Errorf(errFmt, err)
		}
		name := artifact.Name.String()
		if _, exists := names[name]; exists {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
		names[name] = struct{}{}
		size := artifact.Size.Uint64()
		if size > math.MaxUint64-sum {
			return fmt.Errorf(errFmt, core.ErrCustodyContract)
		}
		sum += size
	}
	if err := total.Validate(); err != nil {
		return fmt.Errorf(errFmt, err)
	}
	if sum != total.Uint64() {
		return fmt.Errorf(errFmt, core.ErrCustodyContract)
	}
	return nil
}

type SessionOpenResponse struct {
	Retention RetentionPolicy   `json:"retention"`
	ExpiresAt core.UnixNanoTime `json:"expires_at"`
	Schema    string            `json:"schema"`
	Session   SessionID         `json:"session_id"`
	Targets   []UploadTarget    `json:"targets"`
}

func (r SessionOpenResponse) Validate() error {
	if r.Schema != SchemaSessionOpenResponse {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	if err := r.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if len(r.Targets) == 0 {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	for _, target := range r.Targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf(ErrFmtOpenResponse, err)
		}
	}
	if err := r.Retention.Validate(); err != nil {
		return fmt.Errorf(ErrFmtOpenResponse, err)
	}
	if r.ExpiresAt.IsZero() {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	return nil
}

type UploadTarget struct {
	Artifact ArtifactName    `json:"artifact"`
	Object   ObjectPath      `json:"object"`
	URL      SignedUploadURL `json:"url"`
	Headers  []UploadHeader  `json:"headers"`
	Provider StorageProvider `json:"provider"`
	Method   UploadMethod    `json:"method"`
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
	if strings.TrimSpace(h.Name) == "" {
		return fmt.Errorf(ErrFmtUploadHeader, core.ErrCustodyContract)
	}
	if strings.ContainsAny(h.Name, "\r\n") || strings.ContainsAny(h.Value, "\r\n") {
		return fmt.Errorf(ErrFmtUploadHeader, core.ErrCustodyContract)
	}
	return nil
}

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

type FinalizeRequest struct {
	Schema  string           `json:"schema"`
	Session SessionID        `json:"session_id"`
	Objects []UploadedObject `json:"objects"`
}

func (r FinalizeRequest) Validate() error {
	if r.Schema != SchemaFinalizeRequest {
		return fmt.Errorf(ErrFmtFinalize, core.ErrCustodyContract)
	}
	if err := r.Session.Validate(); err != nil {
		return fmt.Errorf(ErrFmtFinalize, err)
	}
	if len(r.Objects) == 0 {
		return fmt.Errorf(ErrFmtFinalize, core.ErrCustodyContract)
	}
	for _, object := range r.Objects {
		if err := object.Validate(); err != nil {
			return fmt.Errorf(ErrFmtFinalize, err)
		}
	}
	return nil
}

type ReceiptBody struct {
	Retention RetentionPolicy   `json:"retention"`
	IssuedAt  core.UnixNanoTime `json:"issued_at"`
	Release   ReleaseIdentity   `json:"release"`
	Schema    string            `json:"schema"`
	Customer  CustomerID        `json:"customer_id"`
	Session   SessionID         `json:"session_id"`
	ChainHash core.SHA256Hex    `json:"chain_hash"`
	Objects   []UploadedObject  `json:"objects"`
	LedgerSeq int64             `json:"ledger_seq"`
	Provider  StorageProvider   `json:"provider"`
}

func (b ReceiptBody) Validate() error {
	if b.Schema != SchemaReceipt {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	return validateReceiptFields(b)
}

func validateReceiptFields(b ReceiptBody) error {
	if err := validateReceiptIdentity(b); err != nil {
		return err
	}
	if err := validateReceiptObjects(b.Objects); err != nil {
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

func validateReceiptObjects(objects []UploadedObject) error {
	if len(objects) == 0 {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	for _, object := range objects {
		if err := object.Validate(); err != nil {
			return fmt.Errorf(ErrFmtReceipt, err)
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
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return core.AppendCanonicalJSON(dst, b)
}
