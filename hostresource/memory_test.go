package hostresource

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestMemoryAssessmentHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		wantError error
		name      string
		limit     MemoryLimit
		snapshot  MemorySnapshot
		wantState MemoryPressureState
	}{
		{name: "p01_zero_usage_healthy", snapshot: MemorySnapshot{}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 90}, wantState: MemoryPressureHealthy},
		{name: "p02_one_below", snapshot: MemorySnapshot{ManagedBytes: 89}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 90}, wantState: MemoryPressureHealthy},
		{name: "p03_equal_reached", snapshot: MemorySnapshot{ManagedBytes: 90}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 90}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "p04_one_above", snapshot: MemorySnapshot{ManagedBytes: 91}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 90}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "p05_one_percent", snapshot: MemorySnapshot{ManagedBytes: 1}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 1}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "p06_hundred_percent_below", snapshot: MemorySnapshot{ManagedBytes: 99}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 100}, wantState: MemoryPressureHealthy},
		{name: "p07_hundred_percent_equal", snapshot: MemorySnapshot{ManagedBytes: 100}, limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 100}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "p08_non_divisible_one_below_ceiling", snapshot: MemorySnapshot{ManagedBytes: 2}, limit: MemoryLimit{LimitBytes: 3, TriggerPercent: 90}, wantState: MemoryPressureHealthy},
		{name: "p09_non_divisible_equal_ceiling", snapshot: MemorySnapshot{ManagedBytes: 3}, limit: MemoryLimit{LimitBytes: 3, TriggerPercent: 90}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "p10_max_signed_no_overflow", snapshot: MemorySnapshot{ManagedBytes: math.MaxInt64 - 1}, limit: MemoryLimit{LimitBytes: math.MaxInt64, TriggerPercent: 100}, wantState: MemoryPressureHealthy},
		{name: "p11_max_signed_equal", snapshot: MemorySnapshot{ManagedBytes: math.MaxInt64}, limit: MemoryLimit{LimitBytes: math.MaxInt64, TriggerPercent: 100}, wantState: MemoryPressureReached, wantError: core.ErrMemoryLimitReached},
		{name: "n01_zero_limit", limit: MemoryLimit{TriggerPercent: 90}, wantError: core.ErrHostResourceContract},
		{name: "n02_zero_percent", limit: MemoryLimit{LimitBytes: 100}, wantError: core.ErrHostResourceContract},
		{name: "n03_percent_101", limit: MemoryLimit{LimitBytes: 100, TriggerPercent: 101}, wantError: core.ErrHostResourceContract},
		{name: "n04_limit_above_signed", limit: MemoryLimit{LimitBytes: uint64(math.MaxInt64) + 1, TriggerPercent: 90}, wantError: core.ErrHostResourceContract},
		{name: "n05_snapshot_above_signed", snapshot: MemorySnapshot{ManagedBytes: uint64(math.MaxInt64) + 1}, limit: MemoryLimit{LimitBytes: math.MaxInt64, TriggerPercent: 90}, wantError: core.ErrHostResourceContract},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewMemoryAssessment(tc.snapshot, tc.limit)
			if tc.wantState == MemoryPressureUnknown {
				if !errors.Is(err, tc.wantError) {
					t.Fatalf("NewMemoryAssessment() error = %v, want errors.Is %v", err, tc.wantError)
				}
				return
			}
			if err != nil || got.State != tc.wantState || got.Validate() != nil {
				t.Fatalf("NewMemoryAssessment() = (%+v,%v), want state %d and valid", got, err, tc.wantState)
			}
			checkErr := CheckMemoryLimit(got)
			if tc.wantError == nil && checkErr != nil {
				t.Fatalf("CheckMemoryLimit() error = %v, want nil", checkErr)
			}
			if tc.wantError != nil && !errors.Is(checkErr, tc.wantError) {
				t.Fatalf("CheckMemoryLimit() error = %v, want %v", checkErr, tc.wantError)
			}
		})
	}
}

func TestMemoryEnumsSnapshotsAndImpossibleStates(t *testing.T) {
	t.Parallel()

	for _, state := range []MemoryPressureState{MemoryPressureUnknown, 3, 127, 128, math.MaxUint8} {
		if !errors.Is(state.Validate(), core.ErrHostResourceContract) {
			t.Fatalf("MemoryPressureState(%d).Validate() error = %v, want ErrHostResourceContract", state, state.Validate())
		}
	}
	for _, kind := range []RuntimeOOMKind{RuntimeOOMUnknown, 4, 127, 128, math.MaxUint8} {
		if !errors.Is(kind.Validate(), core.ErrHostResourceContract) {
			t.Fatalf("RuntimeOOMKind(%d).Validate() error = %v, want ErrHostResourceContract", kind, kind.Validate())
		}
	}
	impossible := MemoryAssessment{
		Snapshot: MemorySnapshot{ManagedBytes: 90},
		Limit:    MemoryLimit{LimitBytes: 100, TriggerPercent: 90},
		State:    MemoryPressureHealthy,
	}
	if !errors.Is(impossible.Validate(), core.ErrHostResourceContract) {
		t.Fatalf("impossible MemoryAssessment.Validate() error = %v, want ErrHostResourceContract", impossible.Validate())
	}
	snapshot, err := ReadMemorySnapshot()
	if err != nil || snapshot.Validate() != nil {
		t.Fatalf("ReadMemorySnapshot() = (%+v,%v), want valid host observation", snapshot, err)
	}
}

func TestRuntimeOOMDetectionHostileEvidenceTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data []byte
		want RuntimeOOMKind
	}{
		{name: "p01_allocator_exact", data: []byte(core.GoRuntimeOOMAllocatorBanner), want: RuntimeOOMGoAllocator},
		{name: "p02_gc_exact", data: []byte(core.GoRuntimeOOMGCBanner), want: RuntimeOOMGoGC},
		{name: "p03_allocator_after_noise", data: []byte("noise\n" + core.GoRuntimeOOMAllocatorBanner), want: RuntimeOOMGoAllocator},
		{name: "p04_gc_after_noise", data: []byte("noise\n" + core.GoRuntimeOOMGCBanner), want: RuntimeOOMGoGC},
		{name: "p05_allocator_before_stack", data: []byte(core.GoRuntimeOOMAllocatorBanner + "\ngoroutine 1"), want: RuntimeOOMGoAllocator},
		{name: "p06_gc_before_stack", data: []byte(core.GoRuntimeOOMGCBanner + "\ngoroutine 1"), want: RuntimeOOMGoGC},
		{name: "p07_binary_prefix", data: append([]byte{0, 0xff}, []byte(core.GoRuntimeOOMAllocatorBanner)...), want: RuntimeOOMGoAllocator},
		{name: "p08_binary_suffix", data: append([]byte(core.GoRuntimeOOMGCBanner), 0, 0xff), want: RuntimeOOMGoGC},
		{name: "p09_both_prefers_specific_allocator", data: []byte(core.GoRuntimeOOMGCBanner + "\n" + core.GoRuntimeOOMAllocatorBanner), want: RuntimeOOMGoAllocator},
		{name: "p10_repeated_allocator", data: []byte(strings.Repeat(core.GoRuntimeOOMAllocatorBanner, 2)), want: RuntimeOOMGoAllocator},
		{name: "n01_empty", want: RuntimeOOMNone},
		{name: "n02_nil", data: nil, want: RuntimeOOMNone},
		{name: "n03_case_changed", data: []byte("Fatal error: runtime: out of memory"), want: RuntimeOOMNone},
		{name: "n04_allocator_missing_first_byte", data: []byte(core.GoRuntimeOOMAllocatorBanner[1:]), want: RuntimeOOMNone},
		{name: "n05_allocator_missing_last_byte", data: []byte(core.GoRuntimeOOMAllocatorBanner[:len(core.GoRuntimeOOMAllocatorBanner)-1]), want: RuntimeOOMNone},
		{name: "n06_gc_missing_first_byte", data: []byte(core.GoRuntimeOOMGCBanner[1:]), want: RuntimeOOMNone},
		{name: "n07_gc_missing_last_byte", data: []byte(core.GoRuntimeOOMGCBanner[:len(core.GoRuntimeOOMGCBanner)-1]), want: RuntimeOOMNone},
		{name: "n08_spaces_changed", data: []byte("fatal error: runtime:  out of memory"), want: RuntimeOOMNone},
		{name: "n09_words_only", data: []byte("runtime out of memory"), want: RuntimeOOMNone},
		{name: "n10_json_claim", data: []byte(`{"error":"oom"}`), want: RuntimeOOMNone},
		{name: "b01_banner_at_end", data: []byte("prefix" + core.GoRuntimeOOMGCBanner), want: RuntimeOOMGoGC},
		{name: "b02_banner_at_start", data: []byte(core.GoRuntimeOOMAllocatorBanner + "suffix"), want: RuntimeOOMGoAllocator},
		{name: "b03_nul_inside_breaks", data: []byte("fatal error:\x00 out of memory"), want: RuntimeOOMNone},
		{name: "b04_newline_inside_breaks", data: []byte("fatal error:\n out of memory"), want: RuntimeOOMNone},
		{name: "b05_similar_heap_exhausted", data: []byte("fatal error: runtime: heap exhausted"), want: RuntimeOOMNone},
		{name: "b06_plain_oom_killed", data: []byte("OOMKilled"), want: RuntimeOOMNone},
		{name: "b07_linux_kernel_oom", data: []byte("Out of memory: Killed process"), want: RuntimeOOMNone},
		{name: "b08_go_panic_memory", data: []byte("panic: out of memory"), want: RuntimeOOMNone},
		{name: "b09_allocator_with_crlf", data: []byte(core.GoRuntimeOOMAllocatorBanner + "\r\n"), want: RuntimeOOMGoAllocator},
		{name: "b10_gc_embedded_no_boundaries_required", data: []byte("x" + core.GoRuntimeOOMGCBanner + "y"), want: RuntimeOOMGoGC},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DetectRuntimeOOM(tc.data)
			if got.Kind != tc.want || got.Validate() != nil {
				t.Fatalf("DetectRuntimeOOM(%q) = %+v, want kind %d and valid evidence", tc.data, got, tc.want)
			}
		})
	}
}
