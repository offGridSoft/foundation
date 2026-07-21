package peachfuzz

import (
	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/workloadidentity"
)

const (
	OffgridRunStatsPath               = "/" + core.APIVersionToken + "/peachfuzz/stats"
	OffgridRunEvidenceUploadPath      = "/" + core.APIVersionToken + "/peachfuzz/evidence/uploads"
	OffgridRunEvidenceFinalizePath    = "/" + core.APIVersionToken + "/peachfuzz/evidence/finalize"
	OffgridRunEvidenceMaterializePath = "/" + core.APIVersionToken + "/peachfuzz/evidence/materialize"
	GoogleCloudServiceAccount         = "peachfuzz@" + core.OffgridGoogleCloudProjectID + workloadidentity.GoogleServiceAccountEmailSuffix
	QueryProject                      = "project"
)

func OffgridRunStatsEndpoint() (core.APIEndpoint, error) {
	return core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, OffgridRunStatsPath)
}

func OffgridRunEvidenceUploadEndpoint() (core.APIEndpoint, error) {
	return core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, OffgridRunEvidenceUploadPath)
}

func OffgridRunEvidenceFinalizeEndpoint() (core.APIEndpoint, error) {
	return core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, OffgridRunEvidenceFinalizePath)
}

func OffgridRunEvidenceMaterializeEndpoint() (core.APIEndpoint, error) {
	return core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, OffgridRunEvidenceMaterializePath)
}
