package hostresource

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestHostResourceEnumsExhaustiveDomainAndJSONTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value   any
		marshal func() ([]byte, error)
		name    string
		token   string
		valid   bool
	}{
		{name: "disk_unknown", value: DiskPressureUnknown, token: DiskPressureNameUnknown, marshal: func() ([]byte, error) { return json.Marshal(DiskPressureUnknown) }},
		{name: "disk_disabled", value: DiskPressureDisabled, token: DiskPressureNameDisabled, valid: true, marshal: func() ([]byte, error) { return json.Marshal(DiskPressureDisabled) }},
		{name: "disk_healthy", value: DiskPressureHealthy, token: DiskPressureNameHealthy, valid: true, marshal: func() ([]byte, error) { return json.Marshal(DiskPressureHealthy) }},
		{name: "disk_reached", value: DiskPressureReached, token: DiskPressureNameReached, valid: true, marshal: func() ([]byte, error) { return json.Marshal(DiskPressureReached) }},
		{name: "missing_unknown", value: MissingPathUnknown, token: MissingPathNameUnknown, marshal: func() ([]byte, error) { return json.Marshal(MissingPathUnknown) }},
		{name: "missing_reject", value: MissingPathReject, token: MissingPathNameReject, valid: true, marshal: func() ([]byte, error) { return json.Marshal(MissingPathReject) }},
		{name: "missing_empty", value: MissingPathIsEmpty, token: MissingPathNameIsEmpty, valid: true, marshal: func() ([]byte, error) { return json.Marshal(MissingPathIsEmpty) }},
		{name: "memory_unknown", value: MemoryPressureUnknown, token: MemoryPressureNameUnknown, marshal: func() ([]byte, error) { return json.Marshal(MemoryPressureUnknown) }},
		{name: "memory_healthy", value: MemoryPressureHealthy, token: MemoryPressureNameHealthy, valid: true, marshal: func() ([]byte, error) { return json.Marshal(MemoryPressureHealthy) }},
		{name: "memory_reached", value: MemoryPressureReached, token: MemoryPressureNameReached, valid: true, marshal: func() ([]byte, error) { return json.Marshal(MemoryPressureReached) }},
		{name: "oom_unknown", value: RuntimeOOMUnknown, token: RuntimeOOMNameUnknown, marshal: func() ([]byte, error) { return json.Marshal(RuntimeOOMUnknown) }},
		{name: "oom_none", value: RuntimeOOMNone, token: RuntimeOOMNameNone, valid: true, marshal: func() ([]byte, error) { return json.Marshal(RuntimeOOMNone) }},
		{name: "oom_allocator", value: RuntimeOOMGoAllocator, token: RuntimeOOMNameGoAllocator, valid: true, marshal: func() ([]byte, error) { return json.Marshal(RuntimeOOMGoAllocator) }},
		{name: "oom_gc", value: RuntimeOOMGoGC, token: RuntimeOOMNameGoGC, valid: true, marshal: func() ([]byte, error) { return json.Marshal(RuntimeOOMGoGC) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, err := tc.marshal()
			if tc.valid && (err != nil || string(raw) != `"`+tc.token+`"`) {
				t.Fatalf("marshal(%v) = (%q,%v), want token %q", tc.value, raw, err, tc.token)
			}
			if !tc.valid && !errors.Is(err, core.ErrHostResourceContract) {
				t.Fatalf("marshal(%v) error = %v, want ErrHostResourceContract", tc.value, err)
			}
		})
	}
}

func TestHostResourceEnumsRejectMalformedJSONWithoutMutation(t *testing.T) {
	t.Parallel()

	invalid := [][]byte{
		nil,
		{},
		[]byte(`""`),
		[]byte(`"unknown"`),
		[]byte(`"Healthy"`),
		[]byte(`0`),
		[]byte(`true`),
		[]byte(`null`),
		[]byte(`[]`),
		[]byte(`{}`),
		[]byte(`"healthy" false`),
		[]byte(`"healthy`),
	}
	for index, raw := range invalid {
		disk := DiskPressureHealthy
		if err := disk.UnmarshalJSON(raw); !errors.Is(err, core.ErrHostResourceContract) || disk != DiskPressureHealthy {
			t.Fatalf("disk invalid case %d = (%d,%v), want unchanged healthy and ErrHostResourceContract", index, disk, err)
		}
		missing := MissingPathReject
		if err := missing.UnmarshalJSON(raw); !errors.Is(err, core.ErrHostResourceContract) || missing != MissingPathReject {
			t.Fatalf("missing invalid case %d = (%d,%v), want unchanged reject and ErrHostResourceContract", index, missing, err)
		}
		memory := MemoryPressureHealthy
		if err := memory.UnmarshalJSON(raw); !errors.Is(err, core.ErrHostResourceContract) || memory != MemoryPressureHealthy {
			t.Fatalf("memory invalid case %d = (%d,%v), want unchanged healthy and ErrHostResourceContract", index, memory, err)
		}
		oom := RuntimeOOMNone
		if err := oom.UnmarshalJSON(raw); !errors.Is(err, core.ErrHostResourceContract) || oom != RuntimeOOMNone {
			t.Fatalf("oom invalid case %d = (%d,%v), want unchanged none and ErrHostResourceContract", index, oom, err)
		}
	}
}

func TestHostResourceEnumValidJSONRoundTrips(t *testing.T) {
	t.Parallel()

	diskValues := []DiskPressureState{DiskPressureDisabled, DiskPressureHealthy, DiskPressureReached}
	for _, value := range diskValues {
		raw, err := json.Marshal(value)
		got := DiskPressureDisabled
		unmarshalErr := json.Unmarshal(raw, &got)
		if err != nil || unmarshalErr != nil || got != value || !got.IsValid() {
			t.Fatalf("DiskPressureState(%d) round trip = raw:%q got:%d errors:%v/%v", value, raw, got, err, unmarshalErr)
		}
	}
	missingValues := []MissingPathPolicy{MissingPathReject, MissingPathIsEmpty}
	for _, value := range missingValues {
		raw, err := json.Marshal(value)
		got := MissingPathReject
		unmarshalErr := json.Unmarshal(raw, &got)
		if err != nil || unmarshalErr != nil || got != value || !got.IsValid() {
			t.Fatalf("MissingPathPolicy(%d) round trip = raw:%q got:%d errors:%v/%v", value, raw, got, err, unmarshalErr)
		}
	}
	memoryValues := []MemoryPressureState{MemoryPressureHealthy, MemoryPressureReached}
	for _, value := range memoryValues {
		raw, err := json.Marshal(value)
		got := MemoryPressureHealthy
		unmarshalErr := json.Unmarshal(raw, &got)
		if err != nil || unmarshalErr != nil || got != value || !got.IsValid() {
			t.Fatalf("MemoryPressureState(%d) round trip = raw:%q got:%d errors:%v/%v", value, raw, got, err, unmarshalErr)
		}
	}
	oomValues := []RuntimeOOMKind{RuntimeOOMNone, RuntimeOOMGoAllocator, RuntimeOOMGoGC}
	for _, value := range oomValues {
		raw, err := json.Marshal(value)
		got := RuntimeOOMNone
		unmarshalErr := json.Unmarshal(raw, &got)
		if err != nil || unmarshalErr != nil || got != value || !got.IsValid() {
			t.Fatalf("RuntimeOOMKind(%d) round trip = raw:%q got:%d errors:%v/%v", value, raw, got, err, unmarshalErr)
		}
	}
}
