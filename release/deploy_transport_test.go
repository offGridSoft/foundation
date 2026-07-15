package release

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type deployTestSigner struct {
	keyID   core.SigningKeyID
	private ed25519.PrivateKey
	keyring core.SigningKeyring
}

type deployTransportChain struct {
	prepareResponse  DeployPrepareResponse
	finalizeResponse DeployFinalizeResponse
	prepareRequest   DeployPrepareRequest
	releaseSigner    deployTestSigner
	serverSigner     deployTestSigner
	foreignSigner    deployTestSigner
	finalizeRequest  DeployFinalizeRequest
}

func TestDeployPrepareVerificationRejectsHostileChainTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *deployTransportChain)
		name   string
		accept bool
	}{
		{name: "exact signed chain accepted", accept: true},
		{name: "foreign manifest signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.prepareRequest.Manifest = signDeployBody(t, chain.foreignSigner, chain.prepareRequest.Manifest.Body)
		}},
		{name: "foreign plan signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.prepareResponse.Plan = signDeployBody(t, chain.foreignSigner, chain.prepareResponse.Plan.Body)
		}},
		{name: "request id replay", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.prepareRequest.RequestID = otherDeployRequestID(t)
		}},
		{name: "plan changed after signing", mutate: func(_ *testing.T, chain *deployTransportChain) {
			chain.prepareResponse.Plan.Body.Targets[0].ExpiresAt = core.UnixNanoTimeFromInt64(1782302400000000001)
		}},
		{name: "trusted plan embeds different manifest", mutate: replacePlanWithTrustedManifestSwap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validDeployTransportChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			err := chain.prepareResponse.Verify(
				chain.prepareRequest, chain.releaseSigner.keyring, chain.serverSigner.keyring,
			)
			if tc.accept {
				if err != nil {
					t.Fatalf("DeployPrepareResponse.Verify() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployPrepareResponse.Verify() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestDeployFinalizeVerificationRejectsHostileChainTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *deployTransportChain)
		name   string
		accept bool
	}{
		{name: "exact finalized chain accepted", accept: true},
		{name: "foreign request manifest signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeRequest.Manifest = signDeployBody(t, chain.foreignSigner, chain.finalizeRequest.Manifest.Body)
		}},
		{name: "foreign plan signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeRequest.Plan = signDeployBody(t, chain.foreignSigner, chain.finalizeRequest.Plan.Body)
		}},
		{name: "finalize request id replay", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeRequest.RequestID = otherDeployRequestID(t)
		}},
		{name: "missing completion", mutate: func(_ *testing.T, chain *deployTransportChain) {
			chain.finalizeRequest.Objects = nil
			chain.finalizeRequest.ObjectCount = 0
		}},
		{name: "completion object swap", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeRequest.Objects[0].Object = mustOtherObjectKey(t)
		}},
		{name: "foreign receipt signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Receipt = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Receipt.Body)
		}},
		{name: "foreign response manifest signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Manifest = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Manifest.Body)
		}},
		{name: "trusted response manifest substitution", mutate: replaceResponseWithTrustedManifestSwap},
		{name: "foreign index signer", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Index = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Index.Body)
		}},
		{name: "finalize response id replay", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.RequestID = otherDeployRequestID(t)
		}},
		{name: "trusted receipt object substitution", mutate: replaceReceiptWithTrustedObjectSwap},
		{name: "trusted receipt attempt substitution", mutate: replaceReceiptWithTrustedAttemptSwap},
		{name: "trusted index manifest substitution", mutate: replaceIndexWithTrustedManifestSwap},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validDeployTransportChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			err := chain.finalizeResponse.Verify(chain.finalizeRequest, chain.releaseSigner.keyring, chain.serverSigner.keyring)
			if tc.accept {
				if err != nil {
					t.Fatalf("DeployFinalizeResponse.Verify() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployFinalizeResponse.Verify() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestDeployPublicationVerificationRejectsHostileChainTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mutate func(*testing.T, *deployTransportChain)
		name   string
		accept bool
	}{
		{name: "exact persisted publication accepted", accept: true},
		{name: "foreign manifest signature", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Manifest = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Manifest.Body)
		}},
		{name: "foreign receipt signature", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Receipt = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Receipt.Body)
		}},
		{name: "foreign index signature", mutate: func(t *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Index = signDeployBody(t, chain.foreignSigner, chain.finalizeResponse.Index.Body)
		}},
		{name: "manifest changed after release signature", mutate: func(_ *testing.T, chain *deployTransportChain) {
			chain.finalizeResponse.Manifest.Body.CreatedAt = core.UnixNanoTimeFromInt64(chain.finalizeResponse.Manifest.Body.CreatedAt.UnixNano() + 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			chain := validDeployTransportChain(t)
			if tc.mutate != nil {
				tc.mutate(t, &chain)
			}
			err := chain.finalizeResponse.VerifyPublication(chain.releaseSigner.keyring, chain.serverSigner.keyring)
			if tc.accept {
				if err != nil {
					t.Fatalf("DeployFinalizeResponse.VerifyPublication() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) && !errors.Is(err, core.ErrFoundationContract) {
				t.Fatalf("DeployFinalizeResponse.VerifyPublication() error = %v, want release/foundation contract", err)
			}
		})
	}
}

func TestDeployTransportStrictJSONIngressHostileTable(t *testing.T) {
	t.Parallel()

	chain := validDeployTransportChain(t)
	for _, endpoint := range []struct {
		parse func([]byte) error
		value any
		name  string
	}{
		{name: "prepare request", value: chain.prepareRequest, parse: func(data []byte) error {
			_, err := ParseDeployPrepareRequest(data)
			return err
		}},
		{name: "prepare response", value: chain.prepareResponse, parse: func(data []byte) error {
			_, err := ParseDeployPrepareResponse(data)
			return err
		}},
		{name: "finalize request", value: chain.finalizeRequest, parse: func(data []byte) error {
			_, err := ParseDeployFinalizeRequest(data)
			return err
		}},
		{name: "finalize response", value: chain.finalizeResponse, parse: func(data []byte) error {
			_, err := ParseDeployFinalizeResponse(data)
			return err
		}},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			t.Parallel()
			valid, err := json.Marshal(endpoint.value)
			if err != nil {
				t.Fatal(err)
			}
			for _, hostile := range []struct {
				data   func([]byte) []byte
				name   string
				accept bool
			}{
				{name: "valid accepted", data: bytes.Clone, accept: true},
				{name: "unknown field", data: appendUnknownDeployField},
				{name: "duplicate schema", data: appendDuplicateDeploySchema},
				{name: "trailing value", data: func(data []byte) []byte { return append(bytes.Clone(data), '0') }},
				{name: "oversized", data: func([]byte) []byte { return bytes.Repeat([]byte{' '}, core.StrictJSONMaxBytes+1) }},
			} {
				t.Run(hostile.name, func(t *testing.T) {
					t.Parallel()
					err := endpoint.parse(hostile.data(valid))
					if hostile.accept {
						if err != nil {
							t.Fatalf("parse valid error = %v", err)
						}
						return
					}
					if !errors.Is(err, core.ErrJSONContract) {
						t.Fatalf("parse hostile error = %v, want %v", err, core.ErrJSONContract)
					}
				})
			}
		})
	}
}

func TestDeployTransportBuildersDoNotRetainCallerSlices(t *testing.T) {
	t.Parallel()

	chain := validDeployTransportChain(t)
	manifest := chain.prepareRequest.Manifest
	request, err := BuildDeployPrepareRequest(chain.prepareRequest.RequestID, manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Body.Artifacts[0] = Artifact{}
	if err := request.Validate(); err != nil {
		t.Fatalf("BuildDeployPrepareRequest() retained manifest alias: %v", err)
	}

	plan := chain.finalizeRequest.Plan
	objects := append([]UploadedArtifact(nil), chain.finalizeRequest.Objects...)
	finalize, err := BuildDeployFinalizeRequest(chain.finalizeRequest.RequestID, plan, chain.finalizeRequest.Manifest, objects)
	if err != nil {
		t.Fatal(err)
	}
	plan.Body.Targets[0].Headers[0] = core.UploadHeader{}
	objects[0] = UploadedArtifact{}
	if err := finalize.Validate(); err != nil {
		t.Fatalf("BuildDeployFinalizeRequest() retained caller alias: %v", err)
	}

	responseManifest := chain.finalizeRequest.Manifest
	response, err := BuildDeployFinalizeResponse(
		chain.finalizeResponse.RequestID,
		responseManifest,
		chain.finalizeResponse.Receipt,
		chain.finalizeResponse.Index,
	)
	if err != nil {
		t.Fatal(err)
	}
	responseManifest.Body.Artifacts[0] = Artifact{}
	if err := response.Validate(); err != nil {
		t.Fatalf("BuildDeployFinalizeResponse() retained manifest alias: %v", err)
	}
}

func TestDeployTransportRejectsValidatedNestedBodyAboveIngressCap(t *testing.T) {
	t.Parallel()

	chain := validDeployTransportChain(t)
	manifest := validManifest(t)
	manifest.Artifacts = []Artifact{
		validArtifactWithSize(t, "alpha.tar.gz", 12),
		validArtifactWithSize(t, "bravo.tar.gz", 12),
		validArtifactWithSize(t, "charlie.tar.gz", 12),
	}
	manifest.ArtifactCount = uint32(len(manifest.Artifacts))
	manifest.TotalBytes = core.NewByteCount(36)
	attemptID := validUploadAttemptID(t)
	targets := make([]UploadTarget, 0, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		targets = append(targets, maximumHeaderUploadTarget(t, manifest, artifact, attemptID))
	}
	plan, err := BuildDeployPlan(DeployPlanInput{
		RequestID: chain.prepareRequest.RequestID,
		AttemptID: attemptID,
		Layout:    validWitnessReleaseRootLayout(t),
		Targets:   targets,
		Manifest:  manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("large nested DeployPlan.Validate() error = %v", err)
	}
	structuralPlan := chain.prepareResponse.Plan
	structuralPlan.Body = plan
	response := DeployPrepareResponse{
		Schema: core.SchemaReleaseDeployPrepareResponse, RequestID: plan.RequestID, Plan: structuralPlan,
	}
	if err := response.Validate(); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("oversized DeployPrepareResponse.Validate() error = %v, want %v", err, core.ErrReleaseContract)
	}
	if _, err := json.Marshal(response); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("oversized DeployPrepareResponse.MarshalJSON() error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func FuzzDeployRequestIDJSONBoundary(f *testing.F) {
	for _, seed := range []string{
		bytesToHexDigit('a'), bytesToHexDigit('0'), "", "not-hex", bytesToHexDigit('A'),
	} {
		f.Add(seed)
	}
	known, err := ParseDeployRequestID(bytesToHexDigit('b'))
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, value string) {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		decoded := known
		err = decoded.UnmarshalJSON(data)
		parsed, parseErr := ParseDeployRequestID(value)
		if parseErr != nil {
			if !errors.Is(parseErr, core.ErrReleaseContract) || !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("invalid value errors = (%v, %v), want %v", parseErr, err, core.ErrReleaseContract)
			}
			if decoded != known {
				t.Fatalf("rejected value mutated receiver: got %q, want %q", decoded, known)
			}
			return
		}
		if err != nil || decoded != parsed {
			t.Fatalf("valid value round trip = (%q, %v), want (%q, nil)", decoded, err, parsed)
		}
	})
}

func validDeployTransportChain(t *testing.T) deployTransportChain {
	t.Helper()
	releaseSigner := newDeployTestSigner(t, 1)
	serverSigner := newDeployTestSigner(t, 2)
	foreignSigner := newDeployTestSigner(t, 3)
	manifest := validManifest(t)
	requestID := validDeployRequestID(t)
	prepareRequest, err := BuildDeployPrepareRequest(requestID, signDeployBody(t, releaseSigner, manifest))
	if err != nil {
		t.Fatal(err)
	}
	plan := validDeployPlan(t)
	plan.RequestID = requestID
	prepareResponse, err := BuildDeployPrepareResponse(requestID, signDeployBody(t, serverSigner, plan))
	if err != nil {
		t.Fatal(err)
	}
	receipt := validUploadReceipt(t)
	finalizeRequest, err := BuildDeployFinalizeRequest(requestID, prepareResponse.Plan, prepareRequest.Manifest, receipt.Objects)
	if err != nil {
		t.Fatal(err)
	}
	finalizeResponse, err := BuildDeployFinalizeResponse(
		requestID, prepareRequest.Manifest, signDeployBody(t, serverSigner, receipt), signDeployBody(t, serverSigner, validDownloadIndex(t)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return deployTransportChain{
		prepareRequest: prepareRequest, prepareResponse: prepareResponse,
		finalizeRequest: finalizeRequest, finalizeResponse: finalizeResponse,
		releaseSigner: releaseSigner, serverSigner: serverSigner, foreignSigner: foreignSigner,
	}
}

func newDeployTestSigner(t *testing.T, seedByte byte) deployTestSigner {
	t.Helper()
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seedByte}, ed25519.SeedSize))
	publicHex, err := core.NewEd25519PublicKeyHex(private.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := core.ParseSigningKeyID(publicHex.String())
	if err != nil {
		t.Fatal(err)
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicHex}}}
	if err := keyring.Validate(); err != nil {
		t.Fatal(err)
	}
	return deployTestSigner{private: private, keyring: keyring, keyID: keyID}
}

func signDeployBody[B core.CanonicalBody](t *testing.T, signer deployTestSigner, body B) core.Signed[B] {
	t.Helper()
	message, err := core.AppendSignedMessage(nil, signer.keyID, body)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(signer.private, message))
	if err != nil {
		t.Fatal(err)
	}
	return core.Signed[B]{Body: body, KeyID: signer.keyID, Signature: signature}
}

func replacePlanWithTrustedManifestSwap(t *testing.T, chain *deployTransportChain) {
	t.Helper()
	manifest := chain.prepareRequest.Manifest.Body
	manifest.CreatedAt = core.UnixNanoTimeFromInt64(manifest.CreatedAt.UnixNano() + 1)
	target := validUploadTargetFor(t, manifest, core.StorageProviderGCS, validUploadAttemptID(t))
	plan, err := BuildDeployPlan(DeployPlanInput{
		RequestID: chain.prepareRequest.RequestID, AttemptID: validUploadAttemptID(t),
		Layout: validWitnessReleaseRootLayout(t), Targets: []UploadTarget{target}, Manifest: manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	chain.prepareResponse.Plan = signDeployBody(t, chain.serverSigner, plan)
}

func replaceReceiptWithTrustedObjectSwap(t *testing.T, chain *deployTransportChain) {
	t.Helper()
	receipt := chain.finalizeResponse.Receipt.Body
	object := receipt.Objects[0]
	object.Object = mustOtherObjectKey(t)
	binding, err := DeriveUploadBinding(UploadBindingInput{
		Product: receipt.Product, ReleaseID: receipt.ReleaseID, ManifestSHA256: receipt.ManifestSHA256,
		Artifact: object.Artifact, ArtifactSHA256: object.SHA256, ArtifactSize: object.Size,
		Provider: object.Provider, Bucket: object.Bucket, Object: object.Object, AttemptID: object.AttemptID,
	})
	if err != nil {
		t.Fatal(err)
	}
	object.Binding = binding
	receipt.Objects[0] = object
	chain.finalizeResponse.Receipt = signDeployBody(t, chain.serverSigner, receipt)
}

func replaceResponseWithTrustedManifestSwap(t *testing.T, chain *deployTransportChain) {
	t.Helper()
	manifest := chain.finalizeResponse.Manifest.Body
	manifest.CreatedAt = core.UnixNanoTimeFromInt64(manifest.CreatedAt.UnixNano() + 1)
	chain.finalizeResponse.Manifest = signDeployBody(t, chain.releaseSigner, manifest)
}

func replaceReceiptWithTrustedAttemptSwap(t *testing.T, chain *deployTransportChain) {
	t.Helper()
	receipt := chain.finalizeResponse.Receipt.Body
	receipt.AttemptID = mustOtherUploadAttemptID(t)
	object := receipt.Objects[0]
	object.AttemptID = receipt.AttemptID
	binding, err := DeriveUploadBinding(UploadBindingInput{
		Product: receipt.Product, ReleaseID: receipt.ReleaseID, ManifestSHA256: receipt.ManifestSHA256,
		Artifact: object.Artifact, ArtifactSHA256: object.SHA256, ArtifactSize: object.Size,
		Provider: object.Provider, Bucket: object.Bucket, Object: object.Object, AttemptID: object.AttemptID,
	})
	if err != nil {
		t.Fatal(err)
	}
	object.Binding = binding
	receipt.Objects[0] = object
	chain.finalizeResponse.Receipt = signDeployBody(t, chain.serverSigner, receipt)
}

func replaceIndexWithTrustedManifestSwap(t *testing.T, chain *deployTransportChain) {
	t.Helper()
	index := chain.finalizeResponse.Index.Body
	index.Commit = mustOtherCommit(t)
	index.ReleaseID = mustOtherReleaseID(t)
	chain.finalizeResponse.Index = signDeployBody(t, chain.serverSigner, index)
}

func appendUnknownDeployField(data []byte) []byte {
	return appendDeployTopLevelField(data, []byte(`"unknown_contract_field":true`))
}

func appendDuplicateDeploySchema(data []byte) []byte {
	return appendDeployTopLevelField(data, []byte(`"`+core.JSONFieldSchema+`":"`+core.SchemaTokenReleaseDeployPrepareRequest+`"`))
}

func appendDeployTopLevelField(data, field []byte) []byte {
	result := make([]byte, 0, len(data)+len(field)+1)
	result = append(result, data[:len(data)-1]...)
	result = append(result, ',')
	result = append(result, field...)
	return append(result, '}')
}

func otherDeployRequestID(t *testing.T) DeployRequestID {
	t.Helper()
	id, err := ParseDeployRequestID(bytesToHexDigit('e'))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func bytesToHexDigit(digit byte) string {
	return string(bytes.Repeat([]byte{digit}, core.RandomIdentityEntropyBytes*2))
}

func maximumHeaderUploadTarget(
	t *testing.T,
	manifest Manifest,
	artifact Artifact,
	attemptID UploadAttemptID,
) UploadTarget {
	t.Helper()
	object, err := BuildObjectKey(ObjectKeyInput{
		Product: manifest.Product, Date: manifest.Date, ReleaseID: manifest.ReleaseID,
		Visibility: VisibilityPublic, Artifact: artifact.Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestCanonical, err := manifest.Canonical(nil)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := DeriveUploadBinding(UploadBindingInput{
		Product: manifest.Product, ReleaseID: manifest.ReleaseID,
		ManifestSHA256: core.NewSHA256Hex(sha256.Sum256(manifestCanonical)),
		Artifact:       artifact.Name, ArtifactSHA256: artifact.SHA256, ArtifactSize: artifact.Size,
		Provider: core.StorageProviderGCS, Bucket: mustBucket(t), Object: object, AttemptID: attemptID,
	})
	if err != nil {
		t.Fatal(err)
	}
	headers := requiredUploadHeaders(t, core.StorageProviderGCS, attemptID, binding)
	for index := len(headers); index < int(core.HTTPHeaderMaximumDefault); index++ {
		headers = append(headers, core.UploadHeader{
			Name: fmt.Sprintf("X-Deploy-Boundary-%02d", index), Value: strings.Repeat("x", core.HTTPHeaderValueMaxRunes),
		})
	}
	uploadURL, err := core.ParseSignedUploadURL("https://storage.googleapis.com/offgrid-release/artifact?signature=abc")
	if err != nil {
		t.Fatal(err)
	}
	return UploadTarget{
		Artifact: artifact.Name, Object: object, Bucket: mustBucket(t), URL: uploadURL,
		Headers: headers, AttemptID: attemptID, Binding: binding,
		ExpiresAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		Provider:  core.StorageProviderGCS, Method: core.UploadMethodSignedPUT,
	}
}

func requiredUploadHeaders(
	t *testing.T,
	provider core.StorageProvider,
	attemptID UploadAttemptID,
	binding UploadBinding,
) []core.UploadHeader {
	t.Helper()
	attemptHeader, err := UploadAttemptHeader(provider, attemptID)
	if err != nil {
		t.Fatal(err)
	}
	bindingHeader, err := UploadBindingHeader(provider, binding)
	if err != nil {
		t.Fatal(err)
	}
	createOnlyHeader, err := UploadCreateOnlyHeader(provider)
	if err != nil {
		t.Fatal(err)
	}
	return []core.UploadHeader{
		{Name: core.HTTPHeaderContentType, Value: "application/octet-stream"},
		attemptHeader,
		bindingHeader,
		createOnlyHeader,
	}
}
