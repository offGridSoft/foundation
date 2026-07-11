package custody

import (
	"encoding/json"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

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
	signed := mustSignedReceipt(f)
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
	})
}
