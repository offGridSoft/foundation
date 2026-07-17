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
	SchemaBugCheckInResponse
	SchemaBugSeatLease
	SchemaBugWriterAttestation
	SchemaBugWriterCertificate
	SchemaBugWriterRevocation
	SchemaWitnessCheckIn
	SchemaWitnessCheckInResponse
	SchemaWitnessSubscription
	SchemaCustodySessionOpenRequest
	SchemaCustodySessionOpenResponse
	SchemaCustodyFinalizeRequest
	SchemaCustodyReceipt
	SchemaReleaseManifest
	SchemaReleaseUploadReceipt
	SchemaReleaseDownloadIndex
	SchemaReleaseDeployPlan
	SchemaReleasePlan
	SchemaReleaseRootLayout
	SchemaReleaseCommandRun
	SchemaReleaseSeedGrant
	SchemaReleaseDeployPrepareRequest
	SchemaReleaseDeployPrepareResponse
	SchemaReleaseDeployFinalizeRequest
	SchemaReleaseDeployFinalizeResponse
	SchemaReleaseUpdateCheckRequest
	SchemaReleaseUpdateCheckResponse
	SchemaReleaseSelfTestResult
	SchemaReleaseUpdateDiagnostic
	SchemaReleaseUpdateDiagnosticReceipt
	SchemaReleaseGateReport
)

const (
	SchemaTokenBugUsage                       = ProductTokenBug + "-usage-" + ContractVersionToken
	SchemaTokenWitnessUsage                   = ProductTokenWitness + "-usage-" + ContractVersionToken
	SchemaTokenBugCheckIn                     = ProductTokenBug + "-license-check-in-" + ContractVersionToken
	SchemaTokenBugCheckInResponse             = ProductTokenBug + "-license-check-in-response-" + ContractVersionToken
	SchemaTokenBugSeatLease                   = ProductTokenBug + "-license-lease-" + ContractVersionToken
	SchemaTokenBugWriterAttestation           = ProductTokenBug + "-writer-attestation-" + ContractVersionToken
	SchemaTokenBugWriterCertificate           = ProductTokenBug + "-writer-certificate-" + ContractVersionToken
	SchemaTokenBugWriterRevocation            = ProductTokenBug + "-writer-revocation-" + ContractVersionToken
	SchemaTokenWitnessCheckIn                 = ProductTokenWitness + "-subscription-check-in-" + ContractVersionToken
	SchemaTokenWitnessCheckInResponse         = ProductTokenWitness + "-subscription-check-in-response-" + ContractVersionToken
	SchemaTokenWitnessSubscription            = ProductTokenWitness + "-subscription-lease-" + ContractVersionToken
	SchemaTokenCustodySessionOpenRequest      = ProductTokenWitness + "-custody-session-open-" + ContractVersionToken
	SchemaTokenCustodySessionOpenResponse     = ProductTokenWitness + "-custody-session-targets-" + ContractVersionToken
	SchemaTokenCustodyFinalizeRequest         = ProductTokenWitness + "-custody-finalize-" + ContractVersionToken
	SchemaTokenCustodyReceipt                 = ProductTokenWitness + "-custody-receipt-" + ContractVersionToken
	SchemaTokenReleaseManifest                = "offgrid-release-manifest-" + ContractVersionToken
	SchemaTokenReleaseUploadReceipt           = "offgrid-release-upload-receipt-" + ContractVersionToken
	SchemaTokenReleaseDownloadIndex           = "offgrid-release-download-index-" + ContractVersionToken
	SchemaTokenReleaseDeployPlan              = "offgrid-release-deploy-plan-" + ContractVersionToken
	SchemaTokenReleasePlan                    = "offgrid-release-plan-" + ContractVersionToken
	SchemaTokenReleaseRootLayout              = "offgrid-release-root-layout-" + ContractVersionToken
	SchemaTokenReleaseCommandRun              = "offgrid-release-command-run-" + ContractVersionToken
	SchemaTokenReleaseSeedGrant               = "offgrid-release-seed-grant-" + ContractVersionToken
	SchemaTokenReleaseDeployPrepareRequest    = "offgrid-release-deploy-prepare-request-" + ContractVersionToken
	SchemaTokenReleaseDeployPrepareResponse   = "offgrid-release-deploy-prepare-response-" + ContractVersionToken
	SchemaTokenReleaseDeployFinalizeRequest   = "offgrid-release-deploy-finalize-request-" + ContractVersionToken
	SchemaTokenReleaseDeployFinalizeResponse  = "offgrid-release-deploy-finalize-response-" + ContractVersionToken
	SchemaTokenReleaseUpdateCheckRequest      = "offgrid-release-update-check-request-" + ContractVersionToken
	SchemaTokenReleaseUpdateCheckResponse     = "offgrid-release-update-check-response-" + ContractVersionToken
	SchemaTokenReleaseSelfTestResult          = "offgrid-release-self-test-result-" + ContractVersionToken
	SchemaTokenReleaseUpdateDiagnostic        = "offgrid-release-update-diagnostic-" + ContractVersionToken
	SchemaTokenReleaseUpdateDiagnosticReceipt = "offgrid-release-update-diagnostic-receipt-" + ContractVersionToken
	SchemaTokenReleaseGateReport              = "offgrid-release-gate-report-" + ContractVersionToken
)

func schemaNames() [SchemaReleaseGateReport + 1]string {
	return [...]string{
		SchemaBugUsage:                       SchemaTokenBugUsage,
		SchemaWitnessUsage:                   SchemaTokenWitnessUsage,
		SchemaBugCheckIn:                     SchemaTokenBugCheckIn,
		SchemaBugCheckInResponse:             SchemaTokenBugCheckInResponse,
		SchemaBugSeatLease:                   SchemaTokenBugSeatLease,
		SchemaBugWriterAttestation:           SchemaTokenBugWriterAttestation,
		SchemaBugWriterCertificate:           SchemaTokenBugWriterCertificate,
		SchemaBugWriterRevocation:            SchemaTokenBugWriterRevocation,
		SchemaWitnessCheckIn:                 SchemaTokenWitnessCheckIn,
		SchemaWitnessCheckInResponse:         SchemaTokenWitnessCheckInResponse,
		SchemaWitnessSubscription:            SchemaTokenWitnessSubscription,
		SchemaCustodySessionOpenRequest:      SchemaTokenCustodySessionOpenRequest,
		SchemaCustodySessionOpenResponse:     SchemaTokenCustodySessionOpenResponse,
		SchemaCustodyFinalizeRequest:         SchemaTokenCustodyFinalizeRequest,
		SchemaCustodyReceipt:                 SchemaTokenCustodyReceipt,
		SchemaReleaseManifest:                SchemaTokenReleaseManifest,
		SchemaReleaseUploadReceipt:           SchemaTokenReleaseUploadReceipt,
		SchemaReleaseDownloadIndex:           SchemaTokenReleaseDownloadIndex,
		SchemaReleaseDeployPlan:              SchemaTokenReleaseDeployPlan,
		SchemaReleasePlan:                    SchemaTokenReleasePlan,
		SchemaReleaseRootLayout:              SchemaTokenReleaseRootLayout,
		SchemaReleaseCommandRun:              SchemaTokenReleaseCommandRun,
		SchemaReleaseSeedGrant:               SchemaTokenReleaseSeedGrant,
		SchemaReleaseDeployPrepareRequest:    SchemaTokenReleaseDeployPrepareRequest,
		SchemaReleaseDeployPrepareResponse:   SchemaTokenReleaseDeployPrepareResponse,
		SchemaReleaseDeployFinalizeRequest:   SchemaTokenReleaseDeployFinalizeRequest,
		SchemaReleaseDeployFinalizeResponse:  SchemaTokenReleaseDeployFinalizeResponse,
		SchemaReleaseUpdateCheckRequest:      SchemaTokenReleaseUpdateCheckRequest,
		SchemaReleaseUpdateCheckResponse:     SchemaTokenReleaseUpdateCheckResponse,
		SchemaReleaseSelfTestResult:          SchemaTokenReleaseSelfTestResult,
		SchemaReleaseUpdateDiagnostic:        SchemaTokenReleaseUpdateDiagnostic,
		SchemaReleaseUpdateDiagnosticReceipt: SchemaTokenReleaseUpdateDiagnosticReceipt,
		SchemaReleaseGateReport:              SchemaTokenReleaseGateReport,
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
