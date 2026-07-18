package custody

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzSignedReceiptDecodeVerifyBoundary(f *testing.F) {
	keyID, err := core.ParseSigningKeyID("custody-receipt-key-2026")
	if err != nil {
		f.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	publicKey, err := core.NewEd25519PublicKeyHex(privateKey.Public().(ed25519.PublicKey))
	if err != nil {
		f.Fatal(err)
	}
	body := validReceipt(f)
	message, err := core.AppendSignedMessage(nil, keyID, body)
	if err != nil {
		f.Fatal(err)
	}
	signature, err := core.NewEd25519SignatureHex(ed25519.Sign(privateKey, message))
	if err != nil {
		f.Fatal(err)
	}
	signed := core.Signed[ReceiptBody]{Body: body, KeyID: keyID, Signature: signature}
	seed, err := json.Marshal(signed)
	if err != nil {
		f.Fatal(err)
	}
	keyring := core.SigningKeyring{Keys: []core.SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, decodeErr := core.DecodeStrictJSON[core.Signed[ReceiptBody]](data)
		if decodeErr != nil || decoded.Verify(keyring) != nil {
			return
		}
		canonical, canonicalErr := decoded.Body.Canonical(nil)
		if canonicalErr != nil {
			t.Fatalf("verified receipt canonicalization error = %v", canonicalErr)
		}
		roundTrip, roundTripErr := core.DecodeStrictJSON[ReceiptBody](canonical)
		if roundTripErr != nil {
			t.Fatalf("canonical receipt decode error = %v", roundTripErr)
		}
		again, againErr := roundTrip.Canonical(nil)
		if againErr != nil || !bytes.Equal(canonical, again) {
			t.Fatalf("canonical receipt instability: error = %v", againErr)
		}
	})
}

func FuzzSessionOpenRequestBoundary(f *testing.F) {
	seed, err := json.Marshal(validOpenRequest(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[SessionOpenRequest](data)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("accepted SessionOpenRequest validation = %v, want nil", err)
		}
	})
}

func FuzzSessionOpenResponseBoundary(f *testing.F) {
	verifiedReceipt, keyring := verifiedSignedReceipt(f)
	uploadSeed, err := json.Marshal(SessionOpenResponse{
		Schema:     core.SchemaCustodySessionOpenResponse,
		Customer:   mustCustomerID(f),
		BundleRoot: mustBundleRoot(f),
		Upload: &SessionUploadGrant{
			Session:   mustSessionID(f),
			Targets:   validUploadTargets(f),
			Retention: mustRetention(),
			ExpiresAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		},
		Disposition: SessionOpenDispositionUploadRequired,
	})
	if err != nil {
		f.Fatal(err)
	}
	signed := verifiedReceipt
	reuseSeed, err := json.Marshal(SessionOpenResponse{
		Schema:          core.SchemaCustodySessionOpenResponse,
		Customer:        signed.Body.Customer,
		BundleRoot:      signed.Body.BundleRoot,
		ExistingReceipt: &signed,
		Disposition:     SessionOpenDispositionReceiptReused,
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(uploadSeed)
	f.Add(reuseSeed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[SessionOpenResponse](data)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("accepted SessionOpenResponse validation = %v, want nil", err)
		}
		if decoded.Disposition == SessionOpenDispositionReceiptReused {
			if err := decoded.Verify(keyring); err != nil {
				return
			}
		}
	})
}

func FuzzFinalizeRequestBoundary(f *testing.F) {
	seed, err := json.Marshal(FinalizeRequest{
		Schema:     core.SchemaCustodyFinalizeRequest,
		Customer:   mustCustomerID(f),
		BundleRoot: mustBundleRoot(f),
		Session:    mustSessionID(f),
		Objects:    mustUploadedObjects(f),
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[FinalizeRequest](data)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("accepted FinalizeRequest validation = %v, want nil", err)
		}
	})
}

func FuzzReceiptBoundary(f *testing.F) {
	seed, err := json.Marshal(validReceipt(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[ReceiptBody](data)
		if err != nil {
			return
		}
		canonical, err := decoded.Canonical(nil)
		if err != nil {
			t.Fatalf("accepted ReceiptBody canonicalization = %v, want nil", err)
		}
		if len(canonical) > core.StrictJSONMaxBytes {
			t.Fatalf("ReceiptBody canonical bytes = %d, want <= %d", len(canonical), core.StrictJSONMaxBytes)
		}
	})
}

func FuzzDownloadGrantBoundary(f *testing.F) {
	seed, err := json.Marshal(validDownloadGrantBody(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[DownloadGrantBody](data)
		if err != nil {
			return
		}
		canonical, err := decoded.Canonical(nil)
		if err != nil {
			t.Fatalf("accepted DownloadGrantBody canonicalization = %v, want nil", err)
		}
		roundTrip, err := core.DecodeStrictJSON[DownloadGrantBody](canonical)
		if err != nil {
			t.Fatalf("canonical DownloadGrantBody decode = %v, want nil", err)
		}
		again, err := roundTrip.Canonical(nil)
		if err != nil || !bytes.Equal(canonical, again) {
			t.Fatalf("canonical DownloadGrantBody instability: error = %v", err)
		}
	})
}

func FuzzRFC3161TimestampProofBoundary(f *testing.F) {
	seed, err := json.Marshal(mustTimestampProof(f))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[TimestampProof](data)
		if err != nil {
			return
		}
		canonical, err := decoded.Canonical(nil)
		if err != nil {
			t.Fatalf("accepted TimestampProof canonicalization = %v, want nil", err)
		}
		roundTrip, err := core.DecodeStrictJSON[TimestampProof](canonical)
		if err != nil {
			t.Fatalf("canonical TimestampProof decode = %v, want nil", err)
		}
		again, err := roundTrip.Canonical(nil)
		if err != nil || !bytes.Equal(canonical, again) {
			t.Fatalf("canonical TimestampProof instability: error = %v", err)
		}
	})
}

func FuzzCustodyScalarAndEnumReceivers(f *testing.F) {
	pathSeed, err := json.Marshal(witnessObjectPath(f))
	if err != nil {
		f.Fatal(err)
	}
	dispositionSeed, err := json.Marshal(SessionOpenDispositionReceiptReused)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(pathSeed)
	f.Add(dispositionSeed)
	f.Fuzz(func(t *testing.T, data []byte) {
		path := mustObjectPath(t)
		priorPath := path
		if err := path.UnmarshalJSON(data); err != nil && path != priorPath {
			t.Fatalf("ObjectPath.UnmarshalJSON() mutated receiver after rejection")
		}

		disposition := SessionOpenDispositionUploadRequired
		priorDisposition := disposition
		if err := disposition.UnmarshalJSON(data); err != nil && disposition != priorDisposition {
			t.Fatalf("SessionOpenDisposition.UnmarshalJSON() mutated receiver after rejection")
		}

		authority := TimestampAuthorityFreeTSA
		priorAuthority := authority
		if err := authority.UnmarshalJSON(data); err != nil && authority != priorAuthority {
			t.Fatalf("TimestampAuthority.UnmarshalJSON() mutated receiver after rejection")
		}

		token := mustTimestampProof(t).Token
		priorToken := token.String()
		if err := token.UnmarshalJSON(data); err != nil && token.String() != priorToken {
			t.Fatalf("RFC3161Token.UnmarshalJSON() mutated receiver after rejection")
		}

		response := mustTimestampProof(t).Response
		priorResponse := response.String()
		if err := response.UnmarshalJSON(data); err != nil && response.String() != priorResponse {
			t.Fatalf("RFC3161Response.UnmarshalJSON() mutated receiver after rejection")
		}
	})
}
