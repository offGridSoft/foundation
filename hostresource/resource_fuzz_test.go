package hostresource

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzRuntimeOOMEvidenceClassification(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte(core.GoRuntimeOOMAllocatorBanner))
	f.Add([]byte(core.GoRuntimeOOMGCBanner))
	f.Add([]byte("fatal error: runtime: heap exhausted"))
	f.Fuzz(func(t *testing.T, stderr []byte) {
		evidence := DetectRuntimeOOM(stderr)
		if err := evidence.Validate(); err != nil {
			t.Fatalf("DetectRuntimeOOM(%q).Validate() error = %v, want closed valid evidence", stderr, err)
		}
		if evidence.Kind == RuntimeOOMGoAllocator && !bytes.Contains(stderr, []byte(core.GoRuntimeOOMAllocatorBanner)) {
			t.Fatalf("DetectRuntimeOOM(%q) = allocator without compiler-owned allocator banner", stderr)
		}
		if evidence.Kind == RuntimeOOMGoGC && !bytes.Contains(stderr, []byte(core.GoRuntimeOOMGCBanner)) {
			t.Fatalf("DetectRuntimeOOM(%q) = GC without compiler-owned GC banner", stderr)
		}
	})
}

func FuzzHostResourceEnumJSONNeverMutatesOnRejection(f *testing.F) {
	for _, seed := range [][]byte{[]byte(`"healthy"`), []byte(`"reached"`), []byte(`null`), []byte{}, []byte(`{}`)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		disk := DiskPressureHealthy
		diskErr := disk.UnmarshalJSON(data)
		if diskErr != nil && (disk != DiskPressureHealthy || !errors.Is(diskErr, core.ErrHostResourceContract)) {
			t.Fatalf("DiskPressureState.UnmarshalJSON(%q) = (%d,%v), want unchanged on typed rejection", data, disk, diskErr)
		}
		if diskErr == nil {
			raw, err := json.Marshal(disk)
			if err != nil || !json.Valid(raw) || disk.Validate() != nil {
				t.Fatalf("accepted DiskPressureState = %d raw:%q errors:%v/%v", disk, raw, err, disk.Validate())
			}
		}

		oom := RuntimeOOMNone
		oomErr := oom.UnmarshalJSON(data)
		if oomErr != nil && (oom != RuntimeOOMNone || !errors.Is(oomErr, core.ErrHostResourceContract)) {
			t.Fatalf("RuntimeOOMKind.UnmarshalJSON(%q) = (%d,%v), want unchanged on typed rejection", data, oom, oomErr)
		}
		if oomErr == nil && oom.Validate() != nil {
			t.Fatalf("accepted RuntimeOOMKind(%d).Validate() error = %v", oom, oom.Validate())
		}
	})
}
