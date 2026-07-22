package release

import "testing"

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
