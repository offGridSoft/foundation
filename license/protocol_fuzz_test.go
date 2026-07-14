package license

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzParseCheckInEndpointNeverPanics(f *testing.F) {
	f.Add(mustDefaultCheckInEndpoint(f, core.ProductBug).String())
	f.Add(mustDefaultCheckInEndpoint(f, core.ProductWitness).String())
	f.Fuzz(func(t *testing.T, value string) {
		endpoint, err := ParseCheckInEndpoint(value)
		if err != nil {
			return
		}
		if err := endpoint.Validate(); err != nil {
			t.Fatalf("parsed endpoint validation = %v", err)
		}
		if endpoint.String() != value {
			t.Fatalf("parsed endpoint = %q, want %q", endpoint.String(), value)
		}
	})
}

type checkInTestHelper interface {
	Helper()
	Fatal(args ...any)
}

func mustDefaultCheckInEndpoint(t checkInTestHelper, product core.Product) core.APIEndpoint {
	t.Helper()
	endpoint, err := CheckInEndpointForBaseURL(core.OffgridAPIBaseURL, product)
	if err != nil {
		t.Fatal(err)
	}
	return endpoint
}

func FuzzRefusalAndRemediationReceivers(f *testing.F) {
	refusalSeed, err := json.Marshal(RefusalStorageVerificationFailure)
	if err != nil {
		f.Fatal(err)
	}
	remediationSeed, err := json.Marshal(RemediationRetryUpload)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(refusalSeed)
	f.Add(remediationSeed)
	f.Fuzz(func(t *testing.T, data []byte) {
		refusal := RefusalPaymentFailing
		priorRefusal := refusal
		if err := refusal.UnmarshalJSON(data); err != nil && refusal != priorRefusal {
			t.Fatalf("Refusal.UnmarshalJSON() mutated receiver after rejection")
		}

		remediation := RemediationUpdatePayment
		priorRemediation := remediation
		if err := remediation.UnmarshalJSON(data); err != nil && remediation != priorRemediation {
			t.Fatalf("Remediation.UnmarshalJSON() mutated receiver after rejection")
		}
	})
}

func FuzzWitnessUsageBoundary(f *testing.F) {
	seed, err := json.Marshal(WitnessUsage{
		Schema:      core.SchemaWitnessUsage,
		WindowStart: core.NewUnixNanoTime(time.Unix(1, 0)),
		WindowEnd:   core.NewUnixNanoTime(time.Unix(2, 0)),
		Quiz:        1,
		Test:        2,
		Midterm:     3,
		Final:       4,
		Store:       5,
		Verify:      6,
	})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte(nil))
	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := core.DecodeStrictJSON[WitnessUsage](data)
		if err != nil {
			return
		}
		if err := decoded.Validate(); err != nil {
			t.Fatalf("accepted WitnessUsage validation = %v, want nil", err)
		}
	})
}
