package release

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestUpdateExecutionBudgetsBindSharedClientContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{name: "release api", got: ReleaseAPIHTTPBudget, want: 15 * time.Second},
		{name: "artifact download", got: UpdateDownloadHTTPBudget, want: 2 * time.Minute},
		{name: "candidate self test", got: UpdateSelfTestBudget, want: 15 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("budget = %s, want %s", tc.got, tc.want)
			}
		})
	}
	if UpdateSelfTestOutputMaximumBytes <= 0 || UpdateSelfTestOutputMaximumBytes > UpdateTransportMaximumBytes {
		t.Fatalf("UpdateSelfTestOutputMaximumBytes = %d, want within (0, %d]", UpdateSelfTestOutputMaximumBytes, UpdateTransportMaximumBytes)
	}
}

func TestUpdateCheckVerificationHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate func(*testing.T, *updateTestChain)
		name   string
		accept bool
	}{
		{name: "exact available publication", accept: true},
		{name: "response signed by foreign server", mutate: func(t *testing.T, chain *updateTestChain) {
			chain.response.Authority = signDeployBody(t, chain.deploy.foreignSigner, chain.response.Authority.Body)
		}},
		{name: "trusted replay for another request", mutate: func(t *testing.T, chain *updateTestChain) {
			body := chain.response.Authority.Body
			body.RequestID = otherUpdateRequestID(t)
			chain.response.Authority = signDeployBody(t, chain.deploy.serverSigner, body)
		}},
		{name: "foreign release authority", mutate: func(t *testing.T, chain *updateTestChain) {
			body := chain.response.Authority.Body
			body.Publication.Manifest = signDeployBody(t, chain.deploy.foreignSigner, body.Publication.Manifest.Body)
			chain.response.Authority = signDeployBody(t, chain.deploy.serverSigner, body)
		}},
		{name: "foreign receipt authority", mutate: func(t *testing.T, chain *updateTestChain) {
			body := chain.response.Authority.Body
			body.Publication.Receipt = signDeployBody(t, chain.deploy.foreignSigner, body.Publication.Receipt.Body)
			chain.response.Authority = signDeployBody(t, chain.deploy.serverSigner, body)
		}},
		{name: "trusted product substitution", mutate: func(_ *testing.T, chain *updateTestChain) {
			body := chain.response.Authority.Body
			body.Product = core.ProductBug
			chain.response.Authority.Body = body
		}},
		{name: "available without publication", mutate: func(_ *testing.T, chain *updateTestChain) {
			body := chain.response.Authority.Body
			body.Publication = nil
			chain.response.Authority.Body = body
		}},
		{name: "publication equals installed release", mutate: func(t *testing.T, chain *updateTestChain) {
			chain.request.InstalledReleaseID = chain.deploy.finalizeResponse.Manifest.Body.ReleaseID
			chain.request.InstalledCommit = chain.deploy.finalizeResponse.Manifest.Body.Commit
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validUpdateTestChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			err := chain.response.Verify(chain.request, chain.deploy.releaseSigner.keyring, chain.deploy.serverSigner.keyring)
			if tc.accept {
				if err != nil {
					t.Fatalf("UpdateCheckResponse.Verify() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("UpdateCheckResponse.Verify() error = %v, want contract identity", err)
			}
		})
	}
}

func TestUpdateEnumsRejectHostileTokensWithoutMutationTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func([]byte) error
		name string
	}{
		{name: "decision", run: func(data []byte) error {
			value := UpdateDecisionCurrent
			err := json.Unmarshal(data, &value)
			if err != nil && value != UpdateDecisionCurrent {
				t.Fatalf("decision mutated")
			}
			return err
		}},
		{name: "self test check", run: func(data []byte) error {
			value := SelfTestCheckProduct
			err := json.Unmarshal(data, &value)
			if err != nil && value != SelfTestCheckProduct {
				t.Fatalf("check mutated")
			}
			return err
		}},
		{name: "self test status", run: func(data []byte) error {
			value := SelfTestStatusPassed
			err := json.Unmarshal(data, &value)
			if err != nil && value != SelfTestStatusPassed {
				t.Fatalf("status mutated")
			}
			return err
		}},
		{name: "phase", run: func(data []byte) error {
			value := UpdatePhaseDownload
			err := json.Unmarshal(data, &value)
			if err != nil && value != UpdatePhaseDownload {
				t.Fatalf("phase mutated")
			}
			return err
		}},
		{name: "failure", run: func(data []byte) error {
			value := UpdateFailureNetwork
			err := json.Unmarshal(data, &value)
			if err != nil && value != UpdateFailureNetwork {
				t.Fatalf("failure mutated")
			}
			return err
		}},
		{name: "rollback", run: func(data []byte) error {
			value := RollbackOutcomeNotRequired
			err := json.Unmarshal(data, &value)
			if err != nil && value != RollbackOutcomeNotRequired {
				t.Fatalf("rollback mutated")
			}
			return err
		}},
		{name: "disposition", run: func(data []byte) error {
			value := DiagnosticDispositionRecorded
			err := json.Unmarshal(data, &value)
			if err != nil && value != DiagnosticDispositionRecorded {
				t.Fatalf("disposition mutated")
			}
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, data := range [][]byte{[]byte(`""`), []byte(`"unknown"`), []byte(`null`), []byte(`1`), []byte(`"available\n"`)} {
				if err := tc.run(data); !errors.Is(err, core.ErrReleaseContract) {
					t.Fatalf("UnmarshalJSON(%s) error = %v, want release contract", data, err)
				}
			}
		})
	}
}

func TestUpdateEndpointsProductBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		build        func(string) (UpdateEndpoints, error)
		buildDefault func() (UpdateEndpoints, error)
		name         string
		checkPath    string
		diagPath     string
		product      core.Product
	}{
		{name: core.ProductTokenBug, product: core.ProductBug, checkPath: OffgridBugUpdateCheckPath, diagPath: OffgridBugUpdateDiagnosticPath, build: BugUpdateEndpointsForBaseURL, buildDefault: BugUpdateEndpoints},
		{name: core.ProductTokenWitness, product: core.ProductWitness, checkPath: OffgridWitnessUpdateCheckPath, diagPath: OffgridWitnessUpdateDiagnosticPath, build: WitnessUpdateEndpointsForBaseURL, buildDefault: WitnessUpdateEndpoints},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpoints, err := tc.build(core.OffgridAPIBaseURL)
			if err != nil {
				t.Fatal(err)
			}
			if endpoints.Product != tc.product {
				t.Fatalf("product = %v, want %v", endpoints.Product, tc.product)
			}
			if got, want := endpoints.Check.String(), core.OffgridAPIBaseURL+tc.checkPath; got != want {
				t.Fatalf("check = %q, want %q", got, want)
			}
			if got, want := endpoints.Diagnostic.String(), core.OffgridAPIBaseURL+tc.diagPath; got != want {
				t.Fatalf("diagnostic = %q, want %q", got, want)
			}
			defaults, err := tc.buildDefault()
			if err != nil {
				t.Fatal(err)
			}
			if defaults != endpoints {
				t.Fatalf("default endpoints = %+v, want %+v", defaults, endpoints)
			}
		})
	}
	for _, baseURL := range []string{"", "http://api.offgridsoftware.ca", core.OffgridAPIBaseURL + "?x=1", "https://user@api.offgridsoftware.ca"} {
		if _, err := BugUpdateEndpointsForBaseURL(baseURL); !errors.Is(err, core.ErrReleaseContract) {
			t.Fatalf("BugUpdateEndpointsForBaseURL(%q) error = %v, want release contract", baseURL, err)
		}
	}
}

func TestSelfTestResultHostileTable(t *testing.T) {
	t.Parallel()
	valid := validPassedSelfTest(t)
	for _, tc := range []struct {
		mutate func(*testing.T, *SelfTestResult)
		name   string
		accept bool
	}{
		{name: "exact complete result", accept: true},
		{name: "missing first check", mutate: func(_ *testing.T, result *SelfTestResult) { result.Checks = result.Checks[1:]; result.CheckCount-- }},
		{name: "duplicate check", mutate: func(_ *testing.T, result *SelfTestResult) { result.Checks[1] = result.Checks[0] }},
		{name: "passed with failure", mutate: func(_ *testing.T, result *SelfTestResult) {
			result.Failure = &SelfTestFailure{Check: SelfTestCheckServerAuthority, Failure: UpdateFailureIntegrity}
		}},
		{name: "failed without failure", mutate: func(_ *testing.T, result *SelfTestResult) { result.Status = SelfTestStatusFailed }},
		{name: "non-release platform", mutate: func(_ *testing.T, result *SelfTestResult) { result.Platform = core.PlatformDarwinAMD64 }},
		{name: "missing authority", mutate: func(_ *testing.T, result *SelfTestResult) { result.ReleaseKeyID = core.SigningKeyID{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := valid
			result.Checks = slices.Clone(valid.Checks)
			if tc.mutate != nil {
				tc.mutate(t, &result)
			}
			err := result.Validate()
			if tc.accept && err != nil {
				t.Fatalf("SelfTestResult.Validate() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("SelfTestResult.Validate() error = %v, want release contract", err)
			}
		})
	}
}

func TestUpdateDiagnosticHostileAndAliasTable(t *testing.T) {
	t.Parallel()
	input := validDiagnosticInput(t)
	diagnostic, err := BuildUpdateDiagnostic(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Target.SHA256 = mustSHA256(t, "f")
	input.SelfTest.Checks[0] = SelfTestCheckServerAuthority
	if err := diagnostic.Validate(); err != nil {
		t.Fatalf("BuildUpdateDiagnostic retained caller alias: %v", err)
	}

	for _, tc := range []struct {
		mutate func(*testing.T, *UpdateDiagnostic)
		name   string
		accept bool
	}{
		{name: "exact diagnostic", accept: true},
		{name: "identity digest drift", mutate: func(_ *testing.T, value *UpdateDiagnostic) { value.Failure = UpdateFailureNetwork }},
		{name: "forged diagnostic id", mutate: func(t *testing.T, value *UpdateDiagnostic) { value.DiagnosticID = mustDiagnosticID(t, "f") }},
		{name: "self test omitted", mutate: func(_ *testing.T, value *UpdateDiagnostic) { value.SelfTest = nil }},
		{name: "self test mutation without diagnostic identity rehash", mutate: func(t *testing.T, value *UpdateDiagnostic) {
			passed := validPassedSelfTest(t)
			value.SelfTest = &passed
		}},
		{name: "cross-product self test", mutate: func(_ *testing.T, value *UpdateDiagnostic) { value.SelfTest.Product = core.ProductBug }},
		{name: "target omitted after publication", mutate: func(_ *testing.T, value *UpdateDiagnostic) { value.Target = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := diagnostic
			value.Target = cloneUpdateTarget(diagnostic.Target)
			value.SelfTest = cloneSelfTestResult(diagnostic.SelfTest)
			if tc.mutate != nil {
				tc.mutate(t, &value)
			}
			err := value.Validate()
			if tc.accept && err != nil {
				t.Fatalf("UpdateDiagnostic.Validate() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UpdateDiagnostic.Validate() error = %v, want release contract", err)
			}
		})
	}

	data, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	withRawLog := append(data[:len(data)-1], []byte(`,"raw_stderr":"secret"}`)...)
	if _, err := core.DecodeStrictJSON[UpdateDiagnostic](withRawLog); !errors.Is(err, core.ErrJSONContract) {
		t.Fatalf("DecodeStrictJSON(raw log) error = %v, want JSON contract", err)
	}
}

func TestUpdateDiagnosticSelfTestOutcomeMatrix(t *testing.T) {
	t.Parallel()
	failed := validFailedSelfTest(t)
	passed := validPassedSelfTest(t)

	for _, tc := range []struct {
		selfTest *SelfTestResult
		name     string
		failure  UpdateFailure
		accept   bool
	}{
		{name: "failed result matches outer integrity failure", selfTest: &failed, failure: UpdateFailureIntegrity, accept: true},
		{name: "failed result cannot disagree with outer failure", selfTest: &failed, failure: UpdateFailureExecution},
		{name: "passed result exposes expected identity mismatch", selfTest: &passed, failure: UpdateFailureIntegrity, accept: true},
		{name: "passed result exposes outer contract mismatch", selfTest: &passed, failure: UpdateFailureContract, accept: true},
		{name: "passed result cannot claim execution failure", selfTest: &passed, failure: UpdateFailureExecution},
		{name: "missing result after candidate crash", failure: UpdateFailureExecution, accept: true},
		{name: "missing result after invalid output", failure: UpdateFailureContract, accept: true},
		{name: "missing result after permission failure", failure: UpdateFailureFilesystem, accept: true},
		{name: "missing result after cancellation", failure: UpdateFailureCancelled, accept: true},
		{name: "missing result after timeout", failure: UpdateFailureTimeout, accept: true},
		{name: "missing result cannot claim observed integrity drift", failure: UpdateFailureIntegrity},
		{name: "missing result cannot claim network failure", failure: UpdateFailureNetwork},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := validDiagnosticInput(t)
			input.Failure = tc.failure
			input.SelfTest = cloneSelfTestResult(tc.selfTest)
			_, err := BuildUpdateDiagnostic(input)
			if tc.accept && err != nil {
				t.Fatalf("BuildUpdateDiagnostic() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("BuildUpdateDiagnostic() error = %v, want release contract", err)
			}
		})
	}
}

func TestUpdateClientBoundarySmokeTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate      func(*testing.T, *updateTestChain)
		name        string
		contentType string
		statusCode  int
		accept      bool
	}{
		{name: "signed response", statusCode: http.StatusOK, contentType: core.HTTPContentTypeJSON, accept: true},
		{name: "foreign response signer", statusCode: http.StatusOK, contentType: core.HTTPContentTypeJSON, mutate: func(t *testing.T, chain *updateTestChain) {
			chain.response.Authority = signDeployBody(t, chain.deploy.foreignSigner, chain.response.Authority.Body)
		}},
		{name: "wrong content type", statusCode: http.StatusOK, contentType: "text/plain"},
		{name: "server refusal", statusCode: http.StatusServiceUnavailable, contentType: core.HTTPContentTypeJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validUpdateTestChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != OffgridWitnessUpdateCheckPath {
					t.Errorf("request = %s %s", request.Method, request.URL.Path)
				}
				w.Header().Set(core.HTTPHeaderContentType, tc.contentType)
				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					writeUpdateEnvelope(t, w, chain.response)
					return
				}
				writeUpdateFailureEnvelope(t, w)
			}))
			defer server.Close()
			endpoints, err := WitnessUpdateEndpointsForBaseURL(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := UpdateClient{HTTP: server.Client(), Endpoints: endpoints, ReleaseKeys: chain.deploy.releaseSigner.keyring, ServerKeys: chain.deploy.serverSigner.keyring}
			_, err = client.Check(context.Background(), chain.request)
			if tc.accept && err != nil {
				t.Fatalf("UpdateClient.Check() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UpdateClient.Check() error = %v, want release contract", err)
			}
		})
	}
}

func TestUpdateDiagnosticClientReceiptBoundaryTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mutate      func(*testing.T, *UpdateDiagnosticReceiptBody)
		name        string
		disposition DiagnosticDisposition
		accept      bool
	}{
		{name: "recorded", disposition: DiagnosticDispositionRecorded, accept: true},
		{name: "idempotent duplicate", disposition: DiagnosticDispositionDuplicate, accept: true},
		{name: "receipt for another diagnostic", disposition: DiagnosticDispositionRecorded, mutate: func(t *testing.T, body *UpdateDiagnosticReceiptBody) { body.DiagnosticID = mustDiagnosticID(t, "f") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validUpdateTestChain(t)
			diagnostic, err := BuildUpdateDiagnostic(validDiagnosticInput(t))
			if err != nil {
				t.Fatal(err)
			}
			body := UpdateDiagnosticReceiptBody{
				Schema: core.SchemaReleaseUpdateDiagnosticReceipt, DiagnosticID: diagnostic.DiagnosticID,
				Disposition: tc.disposition, RecordedAt: core.UnixNanoTimeFromInt64(1_800_000_000_000_000_002),
			}
			if tc.mutate != nil {
				tc.mutate(t, &body)
			}
			receipt := UpdateDiagnosticReceipt{Authority: signDeployBody(t, chain.deploy.serverSigner, body)}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != OffgridWitnessUpdateDiagnosticPath {
					t.Errorf("request = %s %s", request.Method, request.URL.Path)
				}
				w.Header().Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
				data, marshalErr := json.Marshal(core.APIEnvelope[UpdateDiagnosticReceipt]{Data: &receipt, RequestID: core.NewAPIRequestID("update-diagnostic")})
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				_, _ = w.Write(data)
			}))
			defer server.Close()
			endpoints, err := WitnessUpdateEndpointsForBaseURL(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := UpdateClient{HTTP: server.Client(), Endpoints: endpoints, ReleaseKeys: chain.deploy.releaseSigner.keyring, ServerKeys: chain.deploy.serverSigner.keyring}
			_, err = client.Report(context.Background(), diagnostic)
			if tc.accept && err != nil {
				t.Fatalf("UpdateClient.Report() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("UpdateClient.Report() error = %v, want release contract", err)
			}
		})
	}
}

func validUpdateTestChain(t *testing.T) updateTestChain {
	t.Helper()
	deploy := validDeployTransportChain(t)
	installedCommit := mustOtherCommit(t)
	installedReleaseID, err := BuildReleaseID(core.ProductWitness, mustVersion(t), installedCommit)
	if err != nil {
		t.Fatal(err)
	}
	requestID := validUpdateRequestID(t)
	request, err := BuildUpdateCheckRequest(UpdateCheckInput{
		RequestID: requestID, Product: core.ProductWitness, InstalledVersion: mustVersion(t), InstalledReleaseID: installedReleaseID,
		InstalledCommit: installedCommit, InstalledSHA256: mustSHA256(t, "c"), Platform: core.PlatformLinuxAMD64,
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := BuildUpdateCheckResponseBody(request, UpdateDecisionAvailable, &deploy.finalizeResponse)
	if err != nil {
		t.Fatal(err)
	}
	return updateTestChain{deploy: deploy, request: request, response: UpdateCheckResponse{Authority: signDeployBody(t, deploy.serverSigner, body)}}
}

type updateTestChain struct {
	request  UpdateCheckRequest
	response UpdateCheckResponse
	deploy   deployTransportChain
}

func validUpdateRequestID(t *testing.T) UpdateRequestID {
	t.Helper()
	var entropy [core.RandomIdentityEntropyBytes]byte
	for index := range entropy {
		entropy[index] = byte(index + 1)
	}
	id, err := NewUpdateRequestID(entropy)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func otherUpdateRequestID(t *testing.T) UpdateRequestID {
	t.Helper()
	var entropy [core.RandomIdentityEntropyBytes]byte
	for index := range entropy {
		entropy[index] = byte(index + 2)
	}
	id, err := NewUpdateRequestID(entropy)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func validPassedSelfTest(t *testing.T) SelfTestResult {
	t.Helper()
	chain := validDeployTransportChain(t)
	checks := make([]SelfTestCheck, 0, SelfTestCheckCount)
	for check := SelfTestCheckReleaseStamp; check <= SelfTestCheckServerAuthority; check++ {
		checks = append(checks, check)
	}
	result, err := BuildSelfTestResult(SelfTestInput{
		Product: core.ProductWitness, Version: mustVersion(t), Commit: mustCommit(t), Platform: core.PlatformLinuxAMD64,
		BinarySHA256: mustSHA256(t, "b"), ReleaseKeyID: chain.releaseSigner.keyID, ServerKeyID: chain.serverSigner.keyID,
		Status: SelfTestStatusPassed, Checks: checks,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func validFailedSelfTest(t *testing.T) SelfTestResult {
	t.Helper()
	passed := validPassedSelfTest(t)
	failure := SelfTestFailure{Check: SelfTestCheckBinaryDigest, Failure: UpdateFailureIntegrity}
	result, err := BuildSelfTestResult(SelfTestInput{
		Product: passed.Product, Version: passed.Version, Commit: passed.Commit, Platform: passed.Platform,
		BinarySHA256: passed.BinarySHA256, ReleaseKeyID: passed.ReleaseKeyID, ServerKeyID: passed.ServerKeyID,
		Status: SelfTestStatusFailed, Checks: passed.Checks[:int(SelfTestCheckBinaryDigest)], Failure: &failure,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func validDiagnosticInput(t *testing.T) UpdateDiagnosticInput {
	t.Helper()
	failed := validFailedSelfTest(t)
	target := UpdateTargetIdentity{Version: mustVersion(t), ReleaseID: mustReleaseID(t), Commit: mustCommit(t), SHA256: mustSHA256(t, "b")}
	installedCommit := mustOtherCommit(t)
	installedReleaseID, err := BuildReleaseID(core.ProductWitness, mustVersion(t), installedCommit)
	if err != nil {
		t.Fatal(err)
	}
	return UpdateDiagnosticInput{
		Product: core.ProductWitness, InstalledVersion: mustVersion(t), InstalledReleaseID: installedReleaseID,
		InstalledCommit: installedCommit,
		InstalledSHA256: mustSHA256(t, "c"), Platform: core.PlatformLinuxAMD64, Target: &target,
		Phase: UpdatePhaseCandidateSelfTest, Failure: UpdateFailureIntegrity, SelfTest: &failed,
		Rollback: RollbackOutcomeNotRequired, OccurredAt: core.NewUnixNanoTime(time.Unix(1_800_000_000, 1)),
	}
}

func mustDiagnosticID(t *testing.T, digit string) UpdateDiagnosticID {
	t.Helper()
	id, err := ParseUpdateDiagnosticID(strings.Repeat(digit, sha256.Size*2))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func validUpdateDiagnosticReceiptBody(t *testing.T) UpdateDiagnosticReceiptBody {
	t.Helper()
	diagnostic, err := BuildUpdateDiagnostic(validDiagnosticInput(t))
	if err != nil {
		t.Fatal(err)
	}
	body := UpdateDiagnosticReceiptBody{
		Schema: core.SchemaReleaseUpdateDiagnosticReceipt, DiagnosticID: diagnostic.DiagnosticID,
		Disposition: DiagnosticDispositionRecorded, RecordedAt: core.UnixNanoTimeFromInt64(1_800_000_000_000_000_002),
	}
	if err := body.Validate(); err != nil {
		t.Fatal(err)
	}
	return body
}

func writeUpdateEnvelope(t *testing.T, w http.ResponseWriter, response UpdateCheckResponse) {
	t.Helper()
	data, err := json.Marshal(core.APIEnvelope[UpdateCheckResponse]{Data: &response, RequestID: core.NewAPIRequestID("update-check")})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write(data)
}

func writeUpdateFailureEnvelope(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	body := core.APIErrorBody{Code: core.APICodeServiceUnavailable, Message: "temporarily unavailable"}
	data, err := json.Marshal(core.APIEnvelope[UpdateCheckResponse]{Error: &body, RequestID: core.NewAPIRequestID("update-check")})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write(data)
}
