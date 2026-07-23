package hostresource

import (
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestDiskAssessmentHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantError error
		name      string
		capacity  DiskCapacity
		policy    DiskPressurePolicy
		wantState DiskPressureState
	}{
		{name: "p01_disabled_zero_floor", capacity: DiskCapacity{FreeBytes: 50, TotalBytes: 100}, wantState: DiskPressureDisabled},
		{name: "p02_healthy_one_above", capacity: DiskCapacity{FreeBytes: 51, TotalBytes: 100}, policy: DiskPressurePolicy{FloorBytes: 50}, wantState: DiskPressureHealthy},
		{name: "p03_reached_equal", capacity: DiskCapacity{FreeBytes: 50, TotalBytes: 100}, policy: DiskPressurePolicy{FloorBytes: 50}, wantState: DiskPressureReached, wantError: core.ErrDiskFloorReached},
		{name: "p04_reached_one_below", capacity: DiskCapacity{FreeBytes: 49, TotalBytes: 100}, policy: DiskPressurePolicy{FloorBytes: 50}, wantState: DiskPressureReached, wantError: core.ErrDiskFloorReached},
		{name: "p05_zero_free", capacity: DiskCapacity{TotalBytes: 100}, policy: DiskPressurePolicy{FloorBytes: 1}, wantState: DiskPressureReached, wantError: core.ErrDiskFloorReached},
		{name: "p06_all_free", capacity: DiskCapacity{FreeBytes: 100, TotalBytes: 100}, policy: DiskPressurePolicy{FloorBytes: 99}, wantState: DiskPressureHealthy},
		{name: "p07_one_byte_volume_disabled", capacity: DiskCapacity{FreeBytes: 1, TotalBytes: 1}, wantState: DiskPressureDisabled},
		{name: "p08_max_signed_volume", capacity: DiskCapacity{FreeBytes: math.MaxInt64, TotalBytes: math.MaxInt64}, policy: DiskPressurePolicy{FloorBytes: math.MaxInt64 - 1}, wantState: DiskPressureHealthy},
		{name: "p09_max_floor_equal", capacity: DiskCapacity{FreeBytes: math.MaxInt64, TotalBytes: math.MaxInt64}, policy: DiskPressurePolicy{FloorBytes: math.MaxInt64}, wantState: DiskPressureReached, wantError: core.ErrDiskFloorReached},
		{name: "p10_floor_above_total_reaches", capacity: DiskCapacity{FreeBytes: 10, TotalBytes: 10}, policy: DiskPressurePolicy{FloorBytes: 11}, wantState: DiskPressureReached, wantError: core.ErrDiskFloorReached},
		{name: "n01_zero_total", capacity: DiskCapacity{}, wantError: core.ErrHostResourceContract},
		{name: "n02_free_above_total", capacity: DiskCapacity{FreeBytes: 2, TotalBytes: 1}, wantError: core.ErrHostResourceContract},
		{name: "n03_total_above_signed", capacity: DiskCapacity{TotalBytes: uint64(math.MaxInt64) + 1}, wantError: core.ErrHostResourceContract},
		{name: "n04_free_max_uint", capacity: DiskCapacity{FreeBytes: math.MaxUint64, TotalBytes: math.MaxUint64}, wantError: core.ErrHostResourceContract},
		{name: "n05_floor_above_signed", capacity: DiskCapacity{TotalBytes: 1}, policy: DiskPressurePolicy{FloorBytes: uint64(math.MaxInt64) + 1}, wantError: core.ErrHostResourceContract},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newDiskAssessment(tc.capacity, tc.policy)
			if tc.wantState == DiskPressureUnknown {
				if !errors.Is(err, tc.wantError) {
					t.Fatalf("newDiskAssessment() error = %v, want errors.Is %v", err, tc.wantError)
				}
				return
			}
			if err != nil || got.State != tc.wantState {
				t.Fatalf("newDiskAssessment() = (%+v, %v), want state %d and nil", got, err, tc.wantState)
			}
			checkErr := diskAssessmentError(got)
			if tc.wantError == nil && checkErr != nil {
				t.Fatalf("diskAssessmentError() error = %v, want nil", checkErr)
			}
			if tc.wantError != nil && !errors.Is(checkErr, tc.wantError) {
				t.Fatalf("diskAssessmentError() error = %v, want errors.Is %v", checkErr, tc.wantError)
			}
		})
	}
}

func TestDiskEnumsAndImpossibleAssessmentsAreRejected(t *testing.T) {
	t.Parallel()

	states := []DiskPressureState{DiskPressureUnknown, 4, 127, 128, math.MaxUint8}
	for _, state := range states {
		if !errors.Is(state.Validate(), core.ErrHostResourceContract) {
			t.Fatalf("DiskPressureState(%d).Validate() error = %v, want ErrHostResourceContract", state, state.Validate())
		}
	}
	policies := []MissingPathPolicy{MissingPathUnknown, 3, 127, 128, math.MaxUint8}
	for _, policy := range policies {
		if !errors.Is(policy.Validate(), core.ErrHostResourceContract) {
			t.Fatalf("MissingPathPolicy(%d).Validate() error = %v, want ErrHostResourceContract", policy, policy.Validate())
		}
	}
	impossible := []DiskAssessment{
		{},
		{Capacity: DiskCapacity{FreeBytes: 60, TotalBytes: 100}, Policy: DiskPressurePolicy{FloorBytes: 50}, State: DiskPressureReached},
		{Capacity: DiskCapacity{FreeBytes: 50, TotalBytes: 100}, Policy: DiskPressurePolicy{FloorBytes: 50}, State: DiskPressureHealthy},
		{Capacity: DiskCapacity{FreeBytes: 50, TotalBytes: 100}, State: DiskPressureHealthy},
	}
	for _, assessment := range impossible {
		if !errors.Is(assessment.Validate(), core.ErrHostResourceContract) {
			t.Fatalf("DiskAssessment(%+v).Validate() error = %v, want ErrHostResourceContract", assessment, assessment.Validate())
		}
	}
}

func TestAssessDiskRealVolumeAndIngressFailures(t *testing.T) {
	t.Parallel()

	dir, err := core.ParseAbsoluteDirectoryPath(t.TempDir())
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(temp) error = %v, want nil", err)
	}
	assessment, err := AssessDisk(t.Context(), DiskAssessmentRequest{Path: dir})
	if err != nil || assessment.Capacity.FreeBytes == 0 || assessment.Capacity.TotalBytes < assessment.Capacity.FreeBytes || assessment.Validate() != nil {
		t.Fatalf("AssessDisk(real temp volume) = (%+v, %v), want valid disabled assessment", assessment, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := AssessDisk(cancelled, DiskAssessmentRequest{Path: dir}); !errors.Is(err, context.Canceled) {
		t.Fatalf("AssessDisk(cancelled) error = %v, want context.Canceled", err)
	}
	var nilContext context.Context
	if _, err := AssessDisk(nilContext, DiskAssessmentRequest{Path: dir}); !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("AssessDisk(nil) error = %v, want ErrNilContext", err)
	}
	if _, err := AssessDisk(t.Context(), DiskAssessmentRequest{Path: "relative"}); !errors.Is(err, core.ErrFilesystemContract) {
		t.Fatalf("AssessDisk(relative) error = %v, want ErrFilesystemContract", err)
	}
	missing, err := core.ParseAbsoluteDirectoryPath(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(missing) error = %v, want nil", err)
	}
	if _, err := AssessDisk(t.Context(), DiskAssessmentRequest{Path: missing}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("AssessDisk(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestMeasureTreeHostileFilesystemMatrix(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root, err := core.ParseAbsoluteDirectoryPath(rootText)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(root) error = %v, want nil", err)
	}
	paths := []struct {
		path string
		data []byte
	}{
		{path: "empty", data: []byte{}},
		{path: "one", data: []byte{1}},
		{path: filepath.Join("nested", "ten"), data: make([]byte, 10)},
		{path: filepath.Join("nested", "deep", "thirty-one"), data: make([]byte, 31)},
	}
	for _, item := range paths {
		path := filepath.Join(rootText, item.path)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", path, err)
		}
		if err := os.WriteFile(path, item.data, 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
	if err := os.Symlink(filepath.Join(rootText, "one"), filepath.Join(rootText, "symlink")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	usage, err := MeasureTree(t.Context(), TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject})
	if err != nil || usage.RegularFileBytes != 42 || usage.RegularFileCount != 4 {
		t.Fatalf("MeasureTree() = (%+v, %v), want 42 bytes across 4 regular files and symlink excluded", usage, err)
	}

	missing, err := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "absent"))
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(absent) error = %v", err)
	}
	empty, err := MeasureTree(t.Context(), TreeUsageRequest{Root: missing, MissingPolicy: MissingPathIsEmpty})
	if err != nil || empty != (TreeUsage{}) {
		t.Fatalf("MeasureTree(missing-is-empty) = (%+v, %v), want zero/nil", empty, err)
	}
	if _, err := MeasureTree(t.Context(), TreeUsageRequest{Root: missing, MissingPolicy: MissingPathReject}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("MeasureTree(missing-reject) error = %v, want fs.ErrNotExist", err)
	}
}

func TestMeasureTreeCancellationAndWalkFailures(t *testing.T) {
	t.Parallel()

	root, err := core.ParseAbsoluteDirectoryPath(t.TempDir())
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(root) error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	walk := func(path string, visit fs.WalkDirFunc) error {
		cancel()
		return visit(path, syntheticDirEntry{name: "late", mode: 0}, nil)
	}
	if _, err := measureTree(cancelled, TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject}, walk); !errors.Is(err, context.Canceled) {
		t.Fatalf("measureTree(cancelled during walk) error = %v, want context.Canceled", err)
	}

	sentinel := errors.New("hostile walk failure")
	walkFailure := func(string, fs.WalkDirFunc) error { return sentinel }
	if _, err := measureTree(t.Context(), TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject}, walkFailure); !errors.Is(err, sentinel) {
		t.Fatalf("measureTree(walk failure) error = %v, want sentinel", err)
	}
	callbackFailure := func(path string, visit fs.WalkDirFunc) error {
		return visit(path, syntheticDirEntry{name: "bad"}, sentinel)
	}
	if _, err := measureTree(t.Context(), TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject}, callbackFailure); !errors.Is(err, sentinel) {
		t.Fatalf("measureTree(callback failure) error = %v, want sentinel", err)
	}
}

type syntheticDirEntry struct {
	name string
	mode fs.FileMode
}

func (e syntheticDirEntry) Name() string               { return e.name }
func (e syntheticDirEntry) IsDir() bool                { return e.mode.IsDir() }
func (e syntheticDirEntry) Type() fs.FileMode          { return e.mode }
func (e syntheticDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func TestDiskArithmeticRejectsOverflowAndImpossibleSizes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantError error
		name      string
		blocks    uint64
		blockSize uint64
		want      uint64
	}{
		{name: "p01_zero_blocks", blocks: 0, blockSize: 4096, want: 0},
		{name: "p02_one_byte", blocks: 1, blockSize: 1, want: 1},
		{name: "p03_common_4k", blocks: 10, blockSize: 4096, want: 40960},
		{name: "p04_exact_max_signed", blocks: math.MaxInt64, blockSize: 1, want: math.MaxInt64},
		{name: "n01_one_over_signed", blocks: uint64(math.MaxInt64) + 1, blockSize: 1, wantError: core.ErrNumericOverflow},
		{name: "n02_uint_multiply_overflow", blocks: math.MaxUint64, blockSize: 2, wantError: core.ErrNumericOverflow},
		{name: "n03_max_times_max", blocks: math.MaxUint64, blockSize: math.MaxUint64, wantError: core.ErrNumericOverflow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := blocksToBytes(tc.blocks, tc.blockSize)
			if tc.wantError != nil && !errors.Is(err, tc.wantError) {
				t.Fatalf("blocksToBytes(%d,%d) error = %v, want %v", tc.blocks, tc.blockSize, err, tc.wantError)
			}
			if tc.wantError == nil && (err != nil || got != tc.want) {
				t.Fatalf("blocksToBytes(%d,%d) = (%d,%v), want (%d,nil)", tc.blocks, tc.blockSize, got, err, tc.want)
			}
		})
	}

	usage := TreeUsage{RegularFileBytes: math.MaxInt64, RegularFileCount: 1}
	if err := addRegularFile(&usage, 1); !errors.Is(err, core.ErrNumericOverflow) {
		t.Fatalf("addRegularFile(byte overflow) error = %v, want ErrNumericOverflow", err)
	}
	usage = TreeUsage{RegularFileCount: math.MaxUint64}
	if err := addRegularFile(&usage, 0); !errors.Is(err, core.ErrNumericOverflow) {
		t.Fatalf("addRegularFile(count overflow) error = %v, want ErrNumericOverflow", err)
	}
	if err := addRegularFile(&TreeUsage{}, -1); !errors.Is(err, core.ErrNumericOverflow) {
		t.Fatalf("addRegularFile(negative) error = %v, want ErrNumericOverflow", err)
	}
}
