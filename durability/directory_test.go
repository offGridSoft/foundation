package durability

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestEnsureDirectoryRealFilesystemHostileTable(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root, err := core.ParseAbsoluteDirectoryPath(rootText)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(root) error = %v", err)
	}
	targetText := filepath.Join(rootText, "a", "b", "c")
	target, err := core.ParseAbsoluteDirectoryPath(targetText)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(target) error = %v", err)
	}
	request := DirectoryRequest{Root: root, Target: target, Mode: 0o700}
	if err := EnsureDirectory(t.Context(), request); err != nil {
		t.Fatalf("EnsureDirectory(nested) error = %v, want nil", err)
	}
	if err := EnsureDirectory(t.Context(), request); err != nil {
		t.Fatalf("EnsureDirectory(existing idempotent) error = %v, want nil", err)
	}
	info, err := os.Stat(targetText)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("target stat = (%v,%v), want directory mode 0700", info, err)
	}
	if err := CommitDirectory(target); err != nil {
		t.Fatalf("CommitDirectory(real directory) error = %v, want nil", err)
	}

	fileCollision := filepath.Join(rootText, "file")
	if err := os.WriteFile(fileCollision, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(collision) error = %v", err)
	}
	collision, _ := core.ParseAbsoluteDirectoryPath(fileCollision)
	if err := EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: collision, Mode: 0o700}); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("EnsureDirectory(file collision) error = %v, want ErrDurabilityContract", err)
	}
	if err := CommitDirectory(collision); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("CommitDirectory(file) error = %v, want ErrDurabilityContract", err)
	}
}

func TestDirectoryRequestRejectsEveryInvalidBoundary(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root, _ := core.ParseAbsoluteDirectoryPath(rootText)
	inside, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "inside"))
	outside, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(filepath.Dir(rootText), "outside"))
	cases := []struct {
		name    string
		request DirectoryRequest
	}{
		{name: "n01_zero"},
		{name: "n02_relative_root", request: DirectoryRequest{Root: "relative", Target: inside, Mode: 0o700}},
		{name: "n03_relative_target", request: DirectoryRequest{Root: root, Target: "relative", Mode: 0o700}},
		{name: "n04_target_escape", request: DirectoryRequest{Root: root, Target: outside, Mode: 0o700}},
		{name: "n05_zero_mode", request: DirectoryRequest{Root: root, Target: inside}},
		{name: "n06_type_bit", request: DirectoryRequest{Root: root, Target: inside, Mode: fs.ModeDir | 0o700}},
		{name: "n07_setuid", request: DirectoryRequest{Root: root, Target: inside, Mode: fs.ModeSetuid | 0o700}},
		{name: "n08_unclean_root", request: DirectoryRequest{Root: core.AbsoluteDirectoryPath(rootText + "/."), Target: inside, Mode: 0o700}},
		{name: "n09_unclean_target", request: DirectoryRequest{Root: root, Target: core.AbsoluteDirectoryPath(rootText + "/a/../inside"), Mode: 0o700}},
		{name: "n10_nul_target", request: DirectoryRequest{Root: root, Target: core.AbsoluteDirectoryPath(rootText + "/a\x00b"), Mode: 0o700}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.request.Validate(); !errors.Is(err, core.ErrDurabilityContract) {
				t.Fatalf("DirectoryRequest.Validate() error = %v, want ErrDurabilityContract", err)
			}
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := EnsureDirectory(cancelled, DirectoryRequest{Root: root, Target: inside, Mode: 0o700}); !errors.Is(err, context.Canceled) {
		t.Fatalf("EnsureDirectory(cancelled) error = %v, want context.Canceled", err)
	}
	missingRoot, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "missing"))
	if err := EnsureDirectory(t.Context(), DirectoryRequest{Root: missingRoot, Target: missingRoot, Mode: 0o700}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("EnsureDirectory(missing root) error = %v, want fs.ErrNotExist", err)
	}
}
