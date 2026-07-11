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
	SchemaTokenBugUsage                   = ProductTokenBug + "-usage-v1"
	SchemaTokenWitnessUsage               = ProductTokenWitness + "-usage-v1"
	SchemaTokenBugCheckIn                 = ProductTokenBug + "-license-check-in-v1"
	SchemaTokenBugSeatLease               = ProductTokenBug + "-license-lease-v1"
	SchemaTokenWitnessCheckIn             = ProductTokenWitness + "-subscription-check-in-v1"
	SchemaTokenWitnessSubscription        = ProductTokenWitness + "-subscription-lease-v1"
	SchemaTokenCustodySessionOpenRequest  = ProductTokenWitness + "-custody-session-open-v2"
	SchemaTokenCustodySessionOpenResponse = ProductTokenWitness + "-custody-session-targets-v2"
	SchemaTokenCustodyFinalizeRequest     = ProductTokenWitness + "-custody-finalize-v2"
	SchemaTokenCustodyReceipt             = ProductTokenWitness + "-custody-receipt-v2"
	SchemaTokenReleaseManifest            = "offgrid-release-manifest-v1"
	SchemaTokenReleaseUploadReceipt       = "offgrid-release-upload-receipt-v1"
	SchemaTokenReleaseDownloadIndex       = "offgrid-release-download-index-v1"
	SchemaTokenReleasePlan                = "offgrid-release-plan-v1"
	SchemaTokenReleaseRootLayout          = "offgrid-release-root-layout-v1"
	SchemaTokenReleaseCommandRun          = "offgrid-release-command-run-v1"
)

func schemaNames() [SchemaReleaseCommandRun + 1]string {
	return [...]string{
		SchemaBugUsage:                   SchemaTokenBugUsage,
		SchemaWitnessUsage:               SchemaTokenWitnessUsage,
		SchemaBugCheckIn:                 SchemaTokenBugCheckIn,
		SchemaBugSeatLease:               SchemaTokenBugSeatLease,
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
