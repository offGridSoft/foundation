package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSignedBodyCanonicalProjectionCoversEveryTaggedField(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body core.CanonicalBody
		name string
	}{
		{name: "manifest", body: validManifest(t)},
		{name: "upload receipt", body: validUploadReceipt(t)},
		{name: "download index", body: validDownloadIndex(t)},
		{name: "deploy plan", body: validDeployPlan(t)},
		{name: "release plan", body: validReleasePlan(t)},
		{name: "release root", body: validWitnessReleaseRootLayout(t)},
		{name: "command run", body: validCommandRun(t)},
		{name: "seed grant", body: validSeedGrant(t)},
		{name: "update check response", body: validUpdateTestChain(t).response.Authority.Body},
		{name: "update diagnostic receipt", body: validUpdateDiagnosticReceiptBody(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requireCanonicalJSONFieldCoverage(t, tc.body)
		})
	}
}

func TestUpdateTransportJSONProjectionCoversEveryTaggedField(t *testing.T) {
	t.Parallel()
	chain := validUpdateTestChain(t)
	requested := chain.deploy.finalizeResponse.Manifest.Body.ReleaseID
	requestWithTarget := chain.request
	requestWithTarget.RequestedReleaseID = &requested
	diagnostic, err := BuildUpdateDiagnostic(validDiagnosticInput(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value any
		name  string
	}{
		{name: "check request", value: requestWithTarget},
		{name: "self test result", value: validFailedSelfTest(t)},
		{name: "update diagnostic", value: diagnostic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			want := taggedJSONFieldNames(reflect.TypeOf(tc.value))
			got, err := topLevelJSONFieldNames(encoded)
			if err != nil {
				t.Fatalf("JSON field scan error = %v", err)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("JSON fields = %v, want %v", got, want)
			}
		})
	}
}

func TestDeployTransportJSONProjectionCoversEveryTaggedField(t *testing.T) {
	t.Parallel()
	chain := validDeployTransportChain(t)
	for _, tc := range []struct {
		value any
		name  string
	}{
		{name: "prepare request", value: chain.prepareRequest},
		{name: "prepare response", value: chain.prepareResponse},
		{name: "finalize request", value: chain.finalizeRequest},
		{name: "finalize response", value: chain.finalizeResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			want := taggedJSONFieldNames(reflect.TypeOf(tc.value))
			got, err := topLevelJSONFieldNames(encoded)
			if err != nil {
				t.Fatalf("JSON field scan error = %v", err)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("JSON fields = %v, want %v", got, want)
			}
		})
	}
}

func requireCanonicalJSONFieldCoverage(t *testing.T, body core.CanonicalBody) {
	t.Helper()
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatalf("Canonical() error = %v", err)
	}
	want := taggedJSONFieldNames(reflect.TypeOf(body))
	got, err := topLevelJSONFieldNames(canonical)
	if err != nil {
		t.Fatalf("canonical field scan error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("canonical fields = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("canonical fields = %v, want %v", got, want)
		}
	}
}

func taggedJSONFieldNames(valueType reflect.Type) []string {
	fields := make([]string, 0, valueType.NumField())
	for field := range valueType.Fields() {
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	sort.Strings(fields)
	return fields
}

func topLevelJSONFieldNames(data []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return nil, fmt.Errorf("open object: %w", err)
	}
	fields := make([]string, 0)
	for decoder.More() {
		token, tokenErr := decoder.Token()
		if tokenErr != nil {
			return nil, tokenErr
		}
		field, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("field token contract")
		}
		fields = append(fields, field)
		if err := skipJSONValue(decoder); err != nil {
			return nil, err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON contract")
	}
	sort.Strings(fields)
	return fields, nil
}

func skipJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, nested := token.(json.Delim)
	if !nested {
		return nil
	}
	for decoder.More() {
		if delim == '{' {
			if _, err := decoder.Token(); err != nil {
				return err
			}
		}
		if err := skipJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
