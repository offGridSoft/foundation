package release

import (
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzDecodeDeployPrepareHTTPResponse(f *testing.F) {
	f.Add(core.HTTPStatusOK, core.HTTPContentTypeJSON, []byte(`{}`))
	f.Add(400, core.HTTPContentTypeJSON, []byte(`{"data":null,"error":{"code":"invalid_input","message":"refused"},"request_id":"fuzz"}`))
	f.Add(core.HTTPStatusOK, "text/plain", []byte("not json"))

	f.Fuzz(func(t *testing.T, status int, contentType string, body []byte) {
		response, err := decodeDeployHTTPResponse[DeployPrepareResponse](status, contentType, body)
		if err == nil {
			if status != core.HTTPStatusOK {
				t.Fatalf("accepted non-success status %d", status)
			}
			if err := response.Validate(); err != nil {
				t.Fatalf("accepted response failed Validate(): %v", err)
			}
		}
	})
}

func FuzzDecodeDeployFinalizeHTTPResponse(f *testing.F) {
	f.Add(core.HTTPStatusOK, core.HTTPContentTypeJSON, []byte(`{}`))
	f.Add(409, core.HTTPContentTypeJSON, []byte(`{"data":null,"error":{"code":"conflict","message":"refused"},"request_id":"fuzz"}`))
	f.Add(core.HTTPStatusOK, "application/octet-stream", []byte{0, 1, 2, 3})

	f.Fuzz(func(t *testing.T, status int, contentType string, body []byte) {
		response, err := decodeDeployHTTPResponse[DeployFinalizeResponse](status, contentType, body)
		if err == nil {
			if status != core.HTTPStatusOK {
				t.Fatalf("accepted non-success status %d", status)
			}
			if err := response.Validate(); err != nil {
				t.Fatalf("accepted response failed Validate(): %v", err)
			}
		}
	})
}

func FuzzParseDeployPrepareRequestMITMBoundary(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"schema":"release.deploy.prepare.request.v1","request_id":"00"}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		request, err := ParseDeployPrepareRequest(body)
		if err == nil {
			if err := request.Validate(); err != nil {
				t.Fatalf("accepted request failed Validate(): %v", err)
			}
		}
	})
}

func FuzzParseDeployFinalizeRequestMITMBoundary(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"schema":"release.deploy.finalize.request.v1","objects":[]}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		request, err := ParseDeployFinalizeRequest(body)
		if err == nil {
			if err := request.Validate(); err != nil {
				t.Fatalf("accepted request failed Validate(): %v", err)
			}
		}
	})
}
