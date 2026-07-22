package durability

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestDurabilityEnumsExhaustiveCompilerDomainTable(t *testing.T) {
	t.Parallel()

	installCases := []struct {
		name  string
		value InstallMode
		valid bool
	}{
		{value: InstallUnknown, name: InstallNameUnknown},
		{value: InstallReplace, name: InstallNameReplace, valid: true},
		{value: InstallCreate, name: InstallNameCreate, valid: true},
		{value: InstallCreate + 1, name: InstallNameUnknown},
		{value: InstallMode(255), name: InstallNameUnknown},
	}
	for _, tc := range installCases {
		if tc.value.String() != tc.name || tc.value.IsValid() != tc.valid || (tc.value.Validate() == nil) != tc.valid {
			t.Fatalf("InstallMode(%d) = string:%q valid:%t err:%v, want %q/%t/errorNil=%t", tc.value, tc.value.String(), tc.value.IsValid(), tc.value.Validate(), tc.name, tc.valid, tc.valid)
		}
	}

	activationCases := []struct {
		name  string
		value ActivationState
		valid bool
	}{
		{value: ActivationUnknown, name: ActivationNameUnknown},
		{value: ActivationNotActivated, name: ActivationNameNotActivated, valid: true},
		{value: ActivationDirectorySyncRequired, name: ActivationNameDirectorySyncRequired, valid: true},
		{value: ActivationDurable, name: ActivationNameDurable, valid: true},
		{value: ActivationDurable + 1, name: ActivationNameUnknown},
		{value: ActivationState(255), name: ActivationNameUnknown},
	}
	for _, tc := range activationCases {
		if tc.value.String() != tc.name || tc.value.IsValid() != tc.valid || (tc.value.Validate() == nil) != tc.valid {
			t.Fatalf("ActivationState(%d) = string:%q valid:%t err:%v, want %q/%t/errorNil=%t", tc.value, tc.value.String(), tc.value.IsValid(), tc.value.Validate(), tc.name, tc.valid, tc.valid)
		}
	}

	temporaryCases := []struct {
		name  string
		value TemporaryState
		valid bool
	}{
		{value: TemporaryUnknown, name: TemporaryNameUnknown},
		{value: TemporaryRetained, name: TemporaryNameRetained, valid: true},
		{value: TemporaryRemovalSyncRequired, name: TemporaryNameRemovalSyncRequired, valid: true},
		{value: TemporaryRemoved, name: TemporaryNameRemoved, valid: true},
		{value: TemporaryRemoved + 1, name: TemporaryNameUnknown},
		{value: TemporaryState(255), name: TemporaryNameUnknown},
	}
	for _, tc := range temporaryCases {
		if tc.value.String() != tc.name || tc.value.IsValid() != tc.valid || (tc.value.Validate() == nil) != tc.valid {
			t.Fatalf("TemporaryState(%d) = string:%q valid:%t err:%v, want %q/%t/errorNil=%t", tc.value, tc.value.String(), tc.value.IsValid(), tc.value.Validate(), tc.name, tc.valid, tc.valid)
		}
	}
}

func TestInstallModeHostileJSONBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     []byte
		want    InstallMode
		wantErr bool
	}{
		{name: "p01_replace", raw: []byte(`"replace"`), want: InstallReplace},
		{name: "p02_create", raw: []byte(`"create"`), want: InstallCreate},
		{name: "n01_empty_bytes", raw: []byte{}, wantErr: true},
		{name: "n02_empty_token", raw: []byte(`""`), wantErr: true},
		{name: "n03_unknown", raw: []byte(`"unknown"`), wantErr: true},
		{name: "n04_case", raw: []byte(`"Replace"`), wantErr: true},
		{name: "n05_number", raw: []byte(`1`), wantErr: true},
		{name: "n06_bool", raw: []byte(`true`), wantErr: true},
		{name: "n07_null", raw: []byte(`null`), wantErr: true},
		{name: "n08_array", raw: []byte(`[]`), wantErr: true},
		{name: "n09_object", raw: []byte(`{}`), wantErr: true},
		{name: "n10_trailing", raw: []byte(`"replace" false`), wantErr: true},
		{name: "b01_truncated", raw: []byte(`"replace`), wantErr: true},
		{name: "b02_whitespace_valid", raw: []byte(" \n\t\"create\"\r"), want: InstallCreate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := InstallCreate
			err := got.UnmarshalJSON(tc.raw)
			if tc.wantErr && (!errors.Is(err, core.ErrDurabilityContract) || got != InstallCreate) {
				t.Fatalf("json.Unmarshal(%q) = value:%d error:%v, want unchanged create and ErrDurabilityContract", tc.raw, got, err)
			}
			if !tc.wantErr && (err != nil || got != tc.want) {
				t.Fatalf("json.Unmarshal(%q) = (%d,%v), want (%d,nil)", tc.raw, got, err, tc.want)
			}
		})
	}
	for _, value := range []InstallMode{InstallReplace, InstallCreate} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json.Marshal(%d) error = %v", value, err)
		}
		parsed, err := ParseInstallMode(value.String())
		if err != nil || parsed != value || string(raw) != `"`+value.String()+`"` {
			t.Fatalf("InstallMode(%d) round trip = raw:%q parsed:%d error:%v", value, raw, parsed, err)
		}
	}
	if _, err := json.Marshal(InstallUnknown); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("json.Marshal(InstallUnknown) error = %v, want ErrDurabilityContract", err)
	}
	var nilReceiver *InstallMode
	if err := nilReceiver.UnmarshalJSON([]byte(`"replace"`)); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("nil InstallMode.UnmarshalJSON() error = %v, want ErrDurabilityContract", err)
	}
}

func TestActivationAndTemporaryStateHostileJSONTables(t *testing.T) {
	t.Parallel()

	activationValid := []ActivationState{ActivationNotActivated, ActivationDirectorySyncRequired, ActivationDurable}
	for _, value := range activationValid {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json.Marshal(ActivationState(%d)) error = %v", value, err)
		}
		got := ActivationNotActivated
		if err := json.Unmarshal(raw, &got); err != nil || got != value {
			t.Fatalf("ActivationState(%d) round trip = (%d,%v)", value, got, err)
		}
	}
	temporaryValid := []TemporaryState{TemporaryRetained, TemporaryRemovalSyncRequired, TemporaryRemoved}
	for _, value := range temporaryValid {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("json.Marshal(TemporaryState(%d)) error = %v", value, err)
		}
		got := TemporaryRetained
		if err := json.Unmarshal(raw, &got); err != nil || got != value {
			t.Fatalf("TemporaryState(%d) round trip = (%d,%v)", value, got, err)
		}
	}
	invalid := [][]byte{nil, {}, []byte(`"unknown"`), []byte(`"DURABLE"`), []byte(`0`), []byte(`null`), []byte(`[]`), []byte(`{}`), []byte(`"durable"{}`), []byte(`"removed`)}
	for index, raw := range invalid {
		activation := ActivationDurable
		if err := activation.UnmarshalJSON(raw); !errors.Is(err, core.ErrDurabilityContract) || activation != ActivationDurable {
			t.Fatalf("activation invalid case %d = (%d,%v), want unchanged durable and ErrDurabilityContract", index, activation, err)
		}
		temporary := TemporaryRemoved
		if err := temporary.UnmarshalJSON(raw); !errors.Is(err, core.ErrDurabilityContract) || temporary != TemporaryRemoved {
			t.Fatalf("temporary invalid case %d = (%d,%v), want unchanged removed and ErrDurabilityContract", index, temporary, err)
		}
	}
	if _, err := json.Marshal(ActivationUnknown); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("json.Marshal(ActivationUnknown) error = %v, want ErrDurabilityContract", err)
	}
	if _, err := json.Marshal(TemporaryUnknown); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("json.Marshal(TemporaryUnknown) error = %v, want ErrDurabilityContract", err)
	}
}
