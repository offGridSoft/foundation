package peachfuzz

import (
	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/workloadidentity"
)

const (
	OffgridRunStatsPath       = "/" + core.APIVersionToken + "/peachfuzz/stats"
	GoogleCloudServiceAccount = "peachfuzz@" + core.OffgridGoogleCloudProjectID + workloadidentity.GoogleServiceAccountEmailSuffix
	QueryProject              = "project"
)

func OffgridRunStatsEndpoint() (core.APIEndpoint, error) {
	return core.APIEndpointForBaseURL(core.OffgridAPIBaseURL, OffgridRunStatsPath)
}
