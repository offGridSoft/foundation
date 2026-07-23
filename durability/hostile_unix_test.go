//go:build !windows

package durability

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestEnsureDirectorySymlinkComponentsCannotEscapeRoot(t *testing.T) {
	t.Parallel()

	t.Run("intermediate symlink to outside directory is rejected typed", func(t *testing.T) {
		t.Parallel()
		rootText := t.TempDir()
		outsideText := t.TempDir()
		root, err := core.ParseAbsoluteDirectoryPath(rootText)
		if err != nil {
			t.Fatalf("ParseAbsoluteDirectoryPath(root) error = %v", err)
		}
		if err := os.Symlink(outsideText, filepath.Join(rootText, "a")); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		target, err := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "a", "b"))
		if err != nil {
			t.Fatalf("ParseAbsoluteDirectoryPath(target) error = %v", err)
		}
		err = EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: target, Mode: 0o700})
		if !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("EnsureDirectory(symlinked component) error = %v, want ErrDurabilityContract", err)
		}
		if _, statErr := os.Stat(filepath.Join(outsideText, "b")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(outside escape) error = %v, want fs.ErrNotExist proving containment held", statErr)
		}
	})

	t.Run("final component symlink to directory is rejected typed", func(t *testing.T) {
		t.Parallel()
		rootText := t.TempDir()
		outsideText := t.TempDir()
		root, _ := core.ParseAbsoluteDirectoryPath(rootText)
		if err := os.Symlink(outsideText, filepath.Join(rootText, "leaf")); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		target, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "leaf"))
		err := EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: target, Mode: 0o700})
		if !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("EnsureDirectory(symlink leaf) error = %v, want ErrDurabilityContract", err)
		}
	})

	t.Run("dangling symlink component is rejected typed", func(t *testing.T) {
		t.Parallel()
		rootText := t.TempDir()
		root, _ := core.ParseAbsoluteDirectoryPath(rootText)
		if err := os.Symlink(filepath.Join(rootText, "missing-target"), filepath.Join(rootText, "gone")); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		target, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "gone", "child"))
		err := EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: target, Mode: 0o700})
		if !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("EnsureDirectory(dangling symlink component) error = %v, want ErrDurabilityContract", err)
		}
	})

	t.Run("symlink root is rejected before any child creation", func(t *testing.T) {
		t.Parallel()
		parent := t.TempDir()
		outside := t.TempDir()
		rootText := filepath.Join(parent, "root-link")
		if err := os.Symlink(outside, rootText); err != nil {
			t.Fatalf("Symlink(root) error = %v", err)
		}
		root, _ := core.ParseAbsoluteDirectoryPath(rootText)
		target, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "child"))
		err := EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: target, Mode: 0o700})
		if !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("EnsureDirectory(symlink root) error = %v, want ErrDurabilityContract", err)
		}
		if _, statErr := os.Stat(filepath.Join(outside, "child")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Stat(outside child) error = %v, want fs.ErrNotExist", statErr)
		}
	})

	t.Run("mid chain regular file component fails with not-a-directory identity", func(t *testing.T) {
		t.Parallel()
		rootText := t.TempDir()
		root, _ := core.ParseAbsoluteDirectoryPath(rootText)
		if err := os.WriteFile(filepath.Join(rootText, "a"), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		target, _ := core.ParseAbsoluteDirectoryPath(filepath.Join(rootText, "a", "b"))
		err := EnsureDirectory(t.Context(), DirectoryRequest{Root: root, Target: target, Mode: 0o700})
		if !errors.Is(err, syscall.ENOTDIR) && !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("EnsureDirectory(file mid-chain) error = %v, want ENOTDIR or ErrDurabilityContract", err)
		}
	})
}

func TestContentStageRejectsSymlinkedReadWriteBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup                func(*testing.T, string, string) (ContentStageRequest, core.AbsoluteFilePath)
		name                 string
		wantConstructorError bool
	}{
		{name: "staging directory symlink to outside is rejected before file creation", wantConstructorError: true, setup: func(t *testing.T, root, outside string) (ContentStageRequest, core.AbsoluteFilePath) {
			stage := filepath.Join(root, "stage")
			if err := os.Symlink(outside, stage); err != nil {
				t.Fatalf("Symlink(stage) error = %v", err)
			}
			return contentStageRequest(root, stage), core.AbsoluteFilePath(filepath.Join(root, "target"))
		}},
		{name: "target parent symlink to outside is rejected before activation", setup: func(t *testing.T, root, outside string) (ContentStageRequest, core.AbsoluteFilePath) {
			stage := filepath.Join(root, "stage")
			if err := os.Mkdir(stage, 0o700); err != nil {
				t.Fatalf("Mkdir(stage) error = %v", err)
			}
			link := filepath.Join(root, "objects")
			if err := os.Symlink(outside, link); err != nil {
				t.Fatalf("Symlink(target parent) error = %v", err)
			}
			return contentStageRequest(root, stage), core.AbsoluteFilePath(filepath.Join(link, "object"))
		}},
		{name: "intermediate target symlink inside root is rejected as an informal path alias", setup: func(t *testing.T, root, _ string) (ContentStageRequest, core.AbsoluteFilePath) {
			stage := filepath.Join(root, "stage")
			realTarget := filepath.Join(root, "real")
			for _, directory := range []string{stage, realTarget} {
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatalf("Mkdir(%q) error = %v", directory, err)
				}
			}
			link := filepath.Join(root, "alias")
			if err := os.Symlink("real", link); err != nil {
				t.Fatalf("Symlink(in-root alias) error = %v", err)
			}
			return contentStageRequest(root, stage), core.AbsoluteFilePath(filepath.Join(link, "object"))
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			outside := t.TempDir()
			request, target := testCase.setup(t, root, outside)
			stage, err := NewContentStage(t.Context(), request)
			if testCase.wantConstructorError {
				if stage != nil || !errors.Is(err, core.ErrDurabilityContract) {
					t.Fatalf("NewContentStage() = (%v,%v), want nil/ErrDurabilityContract", stage, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewContentStage() error = %v, want nil before target validation", err)
			}
			if _, err := stage.Write([]byte("must stay contained")); err != nil {
				t.Fatalf("Write() error = %v, want nil", err)
			}
			result, commitErr := stage.Commit(t.Context(), target)
			if !errors.Is(commitErr, core.ErrDurabilityContract) ||
				result.Activation != ActivationNotActivated ||
				result.Temporary != TemporaryRetained {
				t.Fatalf("Commit(symlink target) = (%+v,%v), want retained/ErrDurabilityContract", result, commitErr)
			}
			if abortErr := stage.Abort(); abortErr != nil {
				t.Fatalf("Abort() error = %v, want nil", abortErr)
			}
			if _, statErr := os.Stat(filepath.Join(outside, "object")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("Stat(outside object) error = %v, want not-exist proving no escape", statErr)
			}
		})
	}
}

func TestReadBoundedRejectsNonRegularFilesWithoutHanging(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fifoPath := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	fifo, err := core.ParseAbsoluteFilePath(fifoPath)
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath(fifo) error = %v", err)
	}
	if _, err := ReadBounded(t.Context(), ReadRequest{Path: fifo, MaximumBytes: core.NewByteCount(8)}); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("ReadBounded(fifo) error = %v, want ErrDurabilityContract without blocking", err)
	}

	directory, err := core.ParseAbsoluteFilePath(dir)
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath(directory) error = %v", err)
	}
	if _, err := ReadBounded(t.Context(), ReadRequest{Path: directory, MaximumBytes: core.NewByteCount(8)}); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("ReadBounded(directory) error = %v, want ErrDurabilityContract", err)
	}

	realPath := filepath.Join(dir, "real")
	if err := os.WriteFile(realPath, []byte("linked"), 0o600); err != nil {
		t.Fatalf("WriteFile(real) error = %v", err)
	}
	linkPath := filepath.Join(dir, "link")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	link, err := core.ParseAbsoluteFilePath(linkPath)
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath(link) error = %v", err)
	}
	data, err := ReadBounded(t.Context(), ReadRequest{Path: link, MaximumBytes: core.NewByteCount(8)})
	if err != nil || string(data) != "linked" {
		t.Fatalf("ReadBounded(symlink to regular) = (%q,%v), want (linked,nil)", data, err)
	}

	missing, err := core.ParseAbsoluteFilePath(filepath.Join(dir, "absent"))
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath(missing) error = %v", err)
	}
	if _, err := ReadBounded(t.Context(), ReadRequest{Path: missing, MaximumBytes: core.NewByteCount(8)}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadBounded(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestOpenBoundedFileUsesNonblockingDescriptorForFIFO(t *testing.T) {
	t.Parallel()

	fifoPath := filepath.Join(t.TempDir(), "fifo-race-target")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	file, err := openBoundedFile(fifoPath)
	if err != nil {
		t.Fatalf("openBoundedFile(FIFO) error = %v", err)
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		t.Fatalf("FIFO descriptor errors = stat:%v close:%v, want nil/nil", statErr, closeErr)
	}
	if info.Mode().IsRegular() {
		t.Fatalf("FIFO descriptor mode = %v, want non-regular", info.Mode())
	}
}
