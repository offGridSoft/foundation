package license

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestBugWriterRevocationHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	valid := testBugWriterRevocation(t)
	validJSON, err := json.Marshal(valid)
	if err != nil {
		t.Fatalf("json.Marshal(valid) error = %v", err)
	}
	cases := []struct {
		attack func() error
		name   string
	}{
		{name: "missing schema", attack: func() error { value := valid; value.Schema = core.SchemaUnknown; return value.Validate() }},
		{name: "attestation schema substitution", attack: func() error { value := valid; value.Schema = core.SchemaBugWriterAttestation; return value.Validate() }},
		{name: "certificate schema substitution", attack: func() error { value := valid; value.Schema = core.SchemaBugWriterCertificate; return value.Validate() }},
		{name: "missing writer key", attack: func() error { value := valid; value.WriterKeyID = core.SigningKeyID{}; return value.Validate() }},
		{name: "missing revocation time", attack: func() error { value := valid; value.RevokedAt = core.UnixNanoTime{}; return value.Validate() }},
		{name: "negative revocation time", attack: func() error {
			value := valid
			value.RevokedAt = core.UnixNanoTimeFromInt64(-1)
			return value.Validate()
		}},
		{name: "duplicate schema field", attack: func() error {
			_, err := core.DecodeStrictJSON[BugWriterRevocationBody]([]byte(strings.Replace(string(validJSON), `"schema":`, `"schema":"`+core.SchemaTokenBugWriterRevocation+`","schema":`, 1)))
			return err
		}},
		{name: "unknown field", attack: func() error {
			_, err := core.DecodeStrictJSON[BugWriterRevocationBody]([]byte(strings.TrimSuffix(string(validJSON), "}") + `,"admin":true}`))
			return err
		}},
		{name: "trailing document", attack: func() error {
			data := append([]byte{}, validJSON...)
			data = append(data, validJSON...)
			_, err := core.DecodeStrictJSON[BugWriterRevocationBody](data)
			return err
		}},
		{name: "numeric schema", attack: func() error {
			_, err := core.DecodeStrictJSON[BugWriterRevocationBody]([]byte(strings.Replace(string(validJSON), `"`+core.SchemaTokenBugWriterRevocation+`"`, `1`, 1)))
			return err
		}},
		{name: "string revocation time", attack: func() error {
			_, err := core.DecodeStrictJSON[BugWriterRevocationBody]([]byte(strings.Replace(string(validJSON), strconv.FormatInt(valid.RevokedAt.UnixNano(), 10), `"invalid"`, 1)))
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.attack(); !errors.Is(err, core.ErrFoundationContract) && !errors.Is(err, core.ErrJSONContract) {
				t.Fatalf("attack error = %v, want typed Foundation or JSON contract", err)
			}
		})
	}
}

func TestBugWriterRevocationCanonicalLayerTriad(t *testing.T) {
	t.Parallel()

	body := testBugWriterRevocation(t)
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatalf("Canonical() error = %v", err)
	}
	marshaled, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(canonical) != string(marshaled) {
		t.Fatalf("canonical bytes = %q, MarshalJSON bytes = %q", canonical, marshaled)
	}
	if body.SigningSchema() != core.SchemaBugWriterRevocation {
		t.Fatalf("SigningSchema() = %v, want %v", body.SigningSchema(), core.SchemaBugWriterRevocation)
	}
}

func TestBugWriterRevocationCutoffTable(t *testing.T) {
	t.Parallel()

	revocation := testBugWriterRevocation(t)
	base := testWriterAttestationBody(t)
	base.WriterKeyID = revocation.WriterKeyID
	cases := []struct {
		name    string
		key     core.SigningKeyID
		at      core.UnixNanoTime
		wantErr bool
	}{
		{name: "before cutoff remains historical evidence", at: revocation.RevokedAt.Add(-time.Nanosecond), key: revocation.WriterKeyID},
		{name: "exact cutoff refused", at: revocation.RevokedAt, key: revocation.WriterKeyID, wantErr: true},
		{name: "after cutoff refused", at: revocation.RevokedAt.Add(time.Nanosecond), key: revocation.WriterKeyID, wantErr: true},
		{name: "other writer unaffected", at: revocation.RevokedAt.Add(time.Hour), key: base.WriterKeyID},
	}
	otherKey, err := core.ParseSigningKeyID(BugWriterKeyIDPrefix + strings.Repeat("b", 64))
	if err != nil {
		t.Fatalf("ParseSigningKeyID(other) error = %v", err)
	}
	cases[3].key = otherKey
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			attestation := base
			attestation.WriterKeyID = tc.key
			attestation.OccurredAt = tc.at
			err := revocation.VerifyAttestationAllowed(attestation)
			if tc.wantErr && !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("VerifyAttestationAllowed() error = %v, want %v", err, core.ErrLicenseContract)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("VerifyAttestationAllowed() error = %v, want nil", err)
			}
		})
	}
}

func testBugWriterRevocation(t testing.TB) BugWriterRevocationBody {
	t.Helper()
	key, err := core.ParseSigningKeyID(BugWriterKeyIDPrefix + strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("ParseSigningKeyID() error = %v", err)
	}
	return BugWriterRevocationBody{
		Schema: core.SchemaBugWriterRevocation, WriterKeyID: key,
		RevokedAt: core.NewUnixNanoTime(time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)),
	}
}
