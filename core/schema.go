package core

import (
	"encoding/json"
	"fmt"
)

const (
	ErrFmtSchemaID = "core.SchemaID: %w"
)

type SchemaID uint16

const (
	SchemaUnknown SchemaID = iota
	SchemaBugUsage
	SchemaWitnessUsage
	SchemaBugCheckIn
	SchemaBugSeatLease
	SchemaBugWriterAttestation
	SchemaBugWriterCertificate
	SchemaWitnessCheckIn
	SchemaWitnessSubscription
	SchemaCustodySessionOpenRequest
	SchemaCustodySessionOpenResponse
	SchemaCustodyFinalizeRequest
	SchemaCustodyReceipt
	SchemaReleaseManifest
	SchemaReleaseUploadReceipt
	SchemaReleaseDownloadIndex
	SchemaReleasePlan
	SchemaReleaseRootLayout
	SchemaReleaseCommandRun
)

const (
	SchemaTokenBugUsage                   = ProductTokenBug + "-usage-" + ContractVersionToken
	SchemaTokenWitnessUsage               = ProductTokenWitness + "-usage-" + ContractVersionToken
	SchemaTokenBugCheckIn                 = ProductTokenBug + "-license-check-in-" + ContractVersionToken
	SchemaTokenBugSeatLease               = ProductTokenBug + "-license-lease-" + ContractVersionToken
	SchemaTokenBugWriterAttestation       = ProductTokenBug + "-writer-attestation-" + ContractVersionToken
	SchemaTokenBugWriterCertificate       = ProductTokenBug + "-writer-certificate-" + ContractVersionToken
	SchemaTokenWitnessCheckIn             = ProductTokenWitness + "-subscription-check-in-" + ContractVersionToken
	SchemaTokenWitnessSubscription        = ProductTokenWitness + "-subscription-lease-" + ContractVersionToken
	SchemaTokenCustodySessionOpenRequest  = ProductTokenWitness + "-custody-session-open-" + ContractVersionToken
	SchemaTokenCustodySessionOpenResponse = ProductTokenWitness + "-custody-session-targets-" + ContractVersionToken
	SchemaTokenCustodyFinalizeRequest     = ProductTokenWitness + "-custody-finalize-" + ContractVersionToken
	SchemaTokenCustodyReceipt             = ProductTokenWitness + "-custody-receipt-" + ContractVersionToken
	SchemaTokenReleaseManifest            = "offgrid-release-manifest-" + ContractVersionToken
	SchemaTokenReleaseUploadReceipt       = "offgrid-release-upload-receipt-" + ContractVersionToken
	SchemaTokenReleaseDownloadIndex       = "offgrid-release-download-index-" + ContractVersionToken
	SchemaTokenReleasePlan                = "offgrid-release-plan-" + ContractVersionToken
	SchemaTokenReleaseRootLayout          = "offgrid-release-root-layout-" + ContractVersionToken
	SchemaTokenReleaseCommandRun          = "offgrid-release-command-run-" + ContractVersionToken
)

func schemaNames() [SchemaReleaseCommandRun + 1]string {
	return [...]string{
		SchemaBugUsage:                   SchemaTokenBugUsage,
		SchemaWitnessUsage:               SchemaTokenWitnessUsage,
		SchemaBugCheckIn:                 SchemaTokenBugCheckIn,
		SchemaBugSeatLease:               SchemaTokenBugSeatLease,
		SchemaBugWriterAttestation:       SchemaTokenBugWriterAttestation,
		SchemaBugWriterCertificate:       SchemaTokenBugWriterCertificate,
		SchemaWitnessCheckIn:             SchemaTokenWitnessCheckIn,
		SchemaWitnessSubscription:        SchemaTokenWitnessSubscription,
		SchemaCustodySessionOpenRequest:  SchemaTokenCustodySessionOpenRequest,
		SchemaCustodySessionOpenResponse: SchemaTokenCustodySessionOpenResponse,
		SchemaCustodyFinalizeRequest:     SchemaTokenCustodyFinalizeRequest,
		SchemaCustodyReceipt:             SchemaTokenCustodyReceipt,
		SchemaReleaseManifest:            SchemaTokenReleaseManifest,
		SchemaReleaseUploadReceipt:       SchemaTokenReleaseUploadReceipt,
		SchemaReleaseDownloadIndex:       SchemaTokenReleaseDownloadIndex,
		SchemaReleasePlan:                SchemaTokenReleasePlan,
		SchemaReleaseRootLayout:          SchemaTokenReleaseRootLayout,
		SchemaReleaseCommandRun:          SchemaTokenReleaseCommandRun,
	}
}

func ParseSchemaID(value string) (SchemaID, error) {
	for schema := SchemaBugUsage; int(schema) < len(schemaNames()); schema++ {
		if schemaNames()[schema] == value {
			return schema, nil
		}
	}
	return SchemaUnknown, fmt.Errorf(ErrFmtSchemaID, ErrFoundationContract)
}

func (id SchemaID) String() string {
	if id.IsValid() {
		return schemaNames()[id]
	}
	return ""
}

func (id SchemaID) IsValid() bool {
	return id > SchemaUnknown && int(id) < len(schemaNames()) && schemaNames()[id] != ""
}

func (id SchemaID) Validate() error {
	if !id.IsValid() {
		return fmt.Errorf(ErrFmtSchemaID, ErrFoundationContract)
	}
	return nil
}

func (id SchemaID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.String())
}

func (id *SchemaID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSchemaID, ErrFoundationContract)
	}
	parsed, err := ParseSchemaID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
