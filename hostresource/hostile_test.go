package hostresource

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestMeasureTreeRejectsNonDirectoryRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "regular")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	fileRoot, err := core.ParseAbsoluteDirectoryPath(filePath)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(file) error = %v", err)
	}
	for _, policy := range []MissingPathPolicy{MissingPathReject, MissingPathIsEmpty} {
		usage, err := MeasureTree(t.Context(), TreeUsageRequest{Root: fileRoot, MissingPolicy: policy})
		if !errors.Is(err, core.ErrHostResourceContract) || usage != (TreeUsage{}) {
			t.Fatalf("MeasureTree(file root, policy %s) = (%+v,%v), want zero usage and ErrHostResourceContract", policy, usage, err)
		}
	}
}

func TestMemoryLimitTriggerBytesExhaustiveCeilingParity(t *testing.T) {
	t.Parallel()

	for limit := uint64(1); limit <= 256; limit++ {
		previous := uint64(0)
		for percent := uint8(1); percent <= 100; percent++ {
			memoryLimit := MemoryLimit{LimitBytes: limit, TriggerPercent: percent}
			got, err := memoryLimit.TriggerBytes()
			want := (limit*uint64(percent) + 99) / 100
			if err != nil || got != want {
				t.Fatalf("TriggerBytes(limit=%d,percent=%d) = (%d,%v), want (%d,nil) exact ceiling", limit, percent, got, err, want)
			}
			if got < previous {
				t.Fatalf("TriggerBytes(limit=%d,percent=%d) = %d below previous %d, want monotonic in percent", limit, percent, got, previous)
			}
			if got == 0 || got > limit {
				t.Fatalf("TriggerBytes(limit=%d,percent=%d) = %d, want within (0,%d]", limit, percent, got, limit)
			}
			previous = got
			assessment, err := newMemoryAssessment(MemorySnapshot{ManagedBytes: got}, memoryLimit)
			if err != nil || assessment.State != MemoryPressureReached {
				t.Fatalf("newMemoryAssessment(at trigger %d) = (%+v,%v), want reached", got, assessment, err)
			}
			below, err := newMemoryAssessment(MemorySnapshot{ManagedBytes: got - 1}, memoryLimit)
			if err != nil || below.State != MemoryPressureHealthy {
				t.Fatalf("newMemoryAssessment(one below trigger %d) = (%+v,%v), want healthy", got, below, err)
			}
		}
	}
}
