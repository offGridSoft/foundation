package release

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestReadReleasePlanStrictIngressHostileTable(t *testing.T) {
	t.Parallel()
	canonical, err := validReleasePlan(t).Canonical(nil)
	if err != nil {
		t.Fatalf("ReleasePlan.Canonical() error = %v", err)
	}
	cases := []struct {
		name string
		body []byte
		ok   bool
	}{
		{name: "canonical", body: canonical, ok: true},
		{name: "empty", body: nil},
		{name: "oversized", body: bytes.Repeat([]byte{' '}, int(ReleasePlanMaximumBytes)+1)},
		{name: "null", body: []byte("null")},
		{name: "array", body: []byte("[]")},
		{name: "truncated", body: canonical[:len(canonical)-1]},
		{name: "trailing object", body: append(append([]byte(nil), canonical...), []byte("{}")...)},
		{name: "invalid utf8", body: []byte{'{', 0xff, '}'}},
		{name: "unknown field", body: append(append([]byte(nil), canonical[:len(canonical)-1]...), []byte(`,"unknown":true}`)...)},
		{name: "duplicate schema", body: append([]byte(`{"schema":"offgrid-release-plan-v2026",`), canonical[1:]...)},
		{name: "wrong schema", body: bytes.Replace(canonical, []byte(core.SchemaReleasePlan.String()), []byte(core.SchemaReleaseManifest.String()), 1)},
		{name: "case-folded duplicate", body: append(append([]byte(nil), canonical[:len(canonical)-1]...), []byte(`,"Schema":"offgrid-release-plan-v2026"}`)...)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan, err := ReadReleasePlan(bytes.NewReader(tc.body))
			if tc.ok {
				if err != nil {
					t.Fatalf("ReadReleasePlan() error = %v", err)
				}
				if err := plan.Validate(); err != nil {
					t.Fatalf("ReadReleasePlan().Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ReadReleasePlan() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestReleasePlanRejectsContractValidFieldsPastOwnedByteMaximum(t *testing.T) {
	t.Parallel()

	plan := maximumWidthReleasePlan(t)
	if err := validateReleasePlanStructure(plan); err != nil {
		t.Fatalf("wide ReleasePlan structure error = %v", err)
	}
	if err := plan.Validate(); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("ReleasePlan.Validate(oversized canonical form) error = %v, want %v", err, core.ErrReleaseContract)
	}
	if _, err := plan.Canonical(nil); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("ReleasePlan.Canonical(oversized canonical form) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

func maximumWidthReleasePlan(t *testing.T) ReleasePlan {
	t.Helper()
	plan := validReleasePlan(t)
	wide := strings.Repeat("😀", ToolVersionMaxRunes)
	plan.Toolchain = BuildToolchain{GoVersion: mustToolVersion(t, wide), GarbleVersion: mustToolVersion(t, wide)}
	plan.VulnDB.DBVersion = mustToolVersion(t, wide)
	plan.SeedRef = mustGarbleSeedRef(t, strings.Repeat("😀", GarbleSeedRefMaxRunes))
	plan.Evidence = ReleaseGateEvidence{
		FastGateRef:      mustEvidenceRef(t, strings.Repeat("😀", EvidenceRefMaxRunes)),
		FinalCertificate: validFinalCertificateEvidence(t),
	}
	plan.Tools = make([]ToolProvenance, 0, MaxToolProvenanceItems)
	for index := range MaxToolProvenanceItems {
		suffix := fmt.Sprintf("%03d", index)
		plan.Tools = append(plan.Tools, ToolProvenance{
			Module:  mustToolModule(t, strings.Repeat("😀", ToolModuleMaxRunes-len(suffix))+suffix),
			Version: mustToolVersion(t, wide),
			GoSum:   mustGoSumHash(t, GoSumHashPrefix+strings.Repeat("😀", GoSumHashMaxRunes-len(GoSumHashPrefix))),
		})
	}
	plan.ToolCount = uint32(len(plan.Tools))
	plan.Spec.Commands = make([]ReleaseCommand, 0, ReleaseCommandMaximum)
	for index := range ReleaseCommandMaximum {
		suffix := fmt.Sprintf("%03d", index)
		name := strings.Repeat("😀", core.FileNameTokenMaxRunes-len(suffix)) + suffix
		pathPrefix := BuildCommandDirPrefix
		path := pathPrefix + strings.Repeat("😀", BuildImportPathMaxRunes-len(pathPrefix)-len(suffix)) + suffix
		plan.Spec.Commands = append(plan.Spec.Commands, mustReleaseCommand(t, name, path))
	}
	plan.Spec.CommandCount = uint32(len(plan.Spec.Commands))
	return plan
}

func TestReadReleasePlanRejectsReaderFailure(t *testing.T) {
	t.Parallel()
	_, err := ReadReleasePlan(failingReleasePlanReader{})
	if !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("ReadReleasePlan() error = %v, want %v", err, core.ErrReleaseContract)
	}
	if _, err := ReadReleasePlan(nil); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("ReadReleasePlan(nil) error = %v, want %v", err, core.ErrReleaseContract)
	}
}

type failingReleasePlanReader struct{}

func (failingReleasePlanReader) Read([]byte) (int, error) {
	return 0, core.ErrFoundationContract
}
