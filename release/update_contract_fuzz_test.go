package release

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzUpdateCheckRequestStrictBoundary(f *testing.F) {
	seed, _, err := updateFuzzSeeds()
	if err != nil {
		f.Fatal(err)
	}
	data, err := json.Marshal(seed)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Fuzz(func(t *testing.T, candidate []byte) {
		decoded, err := core.DecodeStrictJSON[UpdateCheckRequest](candidate)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("decoded request invalid: %v", err)
		}
		if _, err := json.Marshal(decoded); err != nil {
			t.Fatalf("decoded request marshal: %v", err)
		}
	})
}

func FuzzUpdateDiagnosticStrictBoundary(f *testing.F) {
	_, seed, err := updateFuzzSeeds()
	if err != nil {
		f.Fatal(err)
	}
	data, err := json.Marshal(seed)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	f.Fuzz(func(t *testing.T, candidate []byte) {
		decoded, err := core.DecodeStrictJSON[UpdateDiagnostic](candidate)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("decoded diagnostic invalid: %v", err)
		}
		if _, err := json.Marshal(decoded); err != nil {
			t.Fatalf("decoded diagnostic marshal: %v", err)
		}
	})
}

func FuzzUpdateCheckResponseMITMBoundary(f *testing.F) {
	request, _, err := updateFuzzSeeds()
	if err != nil {
		f.Fatal(err)
	}
	signer, err := updateFuzzSigner()
	if err != nil {
		f.Fatal(err)
	}
	body, err := BuildUpdateCheckResponseBody(request, UpdateDecisionCurrent, nil)
	if err != nil {
		f.Fatal(err)
	}
	signed, err := signUpdateFuzzBody(signer, body)
	if err != nil {
		f.Fatal(err)
	}
	seed, err := json.Marshal(UpdateCheckResponse{Authority: signed})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, candidate []byte) {
		decoded, err := core.DecodeStrictJSON[UpdateCheckResponse](candidate)
		if err != nil {
			return
		}
		if err := decoded.Verify(request, signer.keyring, signer.keyring); err != nil {
			return
		}
		if _, err := json.Marshal(decoded); err != nil {
			t.Fatalf("verified response marshal: %v", err)
		}
	})
}

func FuzzUpdateDiagnosticReceiptMITMBoundary(f *testing.F) {
	_, diagnostic, err := updateFuzzSeeds()
	if err != nil {
		f.Fatal(err)
	}
	signer, err := updateFuzzSigner()
	if err != nil {
		f.Fatal(err)
	}
	body := UpdateDiagnosticReceiptBody{Schema: core.SchemaReleaseUpdateDiagnosticReceipt, DiagnosticID: diagnostic.DiagnosticID, Disposition: DiagnosticDispositionRecorded, RecordedAt: core.UnixNanoTimeFromInt64(1_800_000_000_000_000_002)}
	signed, err := signUpdateFuzzBody(signer, body)
	if err != nil {
		f.Fatal(err)
	}
	seed, err := json.Marshal(UpdateDiagnosticReceipt{Authority: signed})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, candidate []byte) {
		decoded, err := core.DecodeStrictJSON[UpdateDiagnosticReceipt](candidate)
		if err != nil {
			return
		}
		if err := decoded.Verify(diagnostic, signer.keyring); err != nil {
			return
		}
		if _, err := json.Marshal(decoded); err != nil {
			t.Fatalf("verified receipt marshal: %v", err)
		}
	})
}

type updateFuzzSigningAuthority struct {
	keyID   core.SigningKeyID
	private ed25519.PrivateKey
	keyring core.SigningKeyring
}

func updateFuzzSigner() (updateFuzzSigningAuthority, error) {
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, ed25519.SeedSize))
	publicKey, err := core.NewEd25519PublicKeyHex(private.Public().(ed25519.PublicKey))
	if err != nil {
		return updateFuzzSigningAuthority{}, err
	}
	keyID, err := core.ParseSigningKeyID(publicKey.String())
	if err != nil {
		return updateFuzzSigningAuthority{}, err
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := keyring.Validate(); err != nil {
		return updateFuzzSigningAuthority{}, err
	}
	return updateFuzzSigningAuthority{keyID: keyID, private: private, keyring: keyring}, nil
}

func signUpdateFuzzBody[B core.CanonicalBody](authority updateFuzzSigningAuthority, body B) (core.Signed[B], error) {
	message, err := core.AppendSignedMessage(nil, authority.keyID, body)
	if err != nil {
		return core.Signed[B]{}, err
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(authority.private, message))
	if err != nil {
		return core.Signed[B]{}, err
	}
	return core.Signed[B]{Body: body, KeyID: authority.keyID, Signature: signature}, nil
}

func updateFuzzSeeds() (UpdateCheckRequest, UpdateDiagnostic, error) {
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	installedCommit, err := core.ParseBuildCommit(strings.Repeat("b", 40))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	targetCommit, err := core.ParseBuildCommit(strings.Repeat("a", 40))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	installedRelease, err := BuildReleaseID(core.ProductWitness, version, installedCommit)
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	targetRelease, err := BuildReleaseID(core.ProductWitness, version, targetCommit)
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	installedSHA, err := core.ParseSHA256Hex(strings.Repeat("c", 64))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	targetSHA, err := core.ParseSHA256Hex(strings.Repeat("d", 64))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	requestID, err := ParseUpdateRequestID(strings.Repeat("1", core.RandomIdentityEntropyBytes*2))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	request, err := BuildUpdateCheckRequest(UpdateCheckInput{
		RequestID: requestID, Product: core.ProductWitness, InstalledVersion: version, InstalledReleaseID: installedRelease,
		InstalledCommit: installedCommit, InstalledSHA256: installedSHA, Platform: core.PlatformLinuxAMD64,
	})
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	keyID, err := core.ParseSigningKeyID(strings.Repeat("e", 64))
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	failure := SelfTestFailure{Check: SelfTestCheckBinaryDigest, Failure: UpdateFailureIntegrity}
	selfTest, err := BuildSelfTestResult(SelfTestInput{
		Product: core.ProductWitness, Version: version, Commit: targetCommit, Platform: core.PlatformLinuxAMD64,
		BinarySHA256: targetSHA, ReleaseKeyID: keyID, ServerKeyID: keyID, Status: SelfTestStatusFailed,
		Checks: []SelfTestCheck{SelfTestCheckReleaseStamp, SelfTestCheckProduct, SelfTestCheckVersion, SelfTestCheckCommit, SelfTestCheckPlatform, SelfTestCheckBinaryDigest}, Failure: &failure,
	})
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, fmt.Errorf("self test seed: %w", err)
	}
	target := UpdateTargetIdentity{Version: version, ReleaseID: targetRelease, Commit: targetCommit, SHA256: targetSHA}
	diagnostic, err := BuildUpdateDiagnostic(UpdateDiagnosticInput{
		Product: core.ProductWitness, InstalledVersion: version, InstalledReleaseID: installedRelease,
		InstalledCommit: installedCommit,
		InstalledSHA256: installedSHA, Platform: core.PlatformLinuxAMD64, Target: &target,
		Phase: UpdatePhaseCandidateSelfTest, Failure: UpdateFailureIntegrity, SelfTest: &selfTest,
		Rollback: RollbackOutcomeNotRequired, OccurredAt: core.UnixNanoTimeFromInt64(1_800_000_000_000_000_001),
	})
	if err != nil {
		return UpdateCheckRequest{}, UpdateDiagnostic{}, err
	}
	return request, diagnostic, nil
}
