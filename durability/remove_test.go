package durability

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestTreeRemovalRequestContainmentBoundaryTable(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root := core.AbsoluteDirectoryPath(rootText)
	tests := []struct {
		wantErr error
		request TreeRemovalRequest
		name    string
	}{
		{name: "direct child", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "child"))}},
		{name: "deep child", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "a", "b", "c"))}},
		{name: "unicode child", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "界"))}},
		{name: "dot-prefixed child", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, ".stage"))}},
		{name: "prefix sibling inside root", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "root-copy"))}},
		{name: "zero root", request: TreeRemovalRequest{Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "zero target", request: TreeRemovalRequest{Root: root}, wantErr: core.ErrDurabilityContract},
		{name: "relative root", request: TreeRemovalRequest{Root: "relative", Target: core.AbsoluteDirectoryPath(filepath.Join(rootText, "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "relative target", request: TreeRemovalRequest{Root: root, Target: "relative"}, wantErr: core.ErrDurabilityContract},
		{name: "unclean target", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(rootText + string(filepath.Separator) + "a" + string(filepath.Separator) + ".." + string(filepath.Separator) + "b")}, wantErr: core.ErrDurabilityContract},
		{name: "root itself", request: TreeRemovalRequest{Root: root, Target: root}, wantErr: core.ErrDurabilityContract},
		{name: "parent of root", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Dir(rootText))}, wantErr: core.ErrDurabilityContract},
		{name: "prefix sibling outside root", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(rootText + "-copy")}, wantErr: core.ErrDurabilityContract},
		{name: "distant outside root", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(filepath.Join(string(filepath.Separator), "outside", "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "nul target", request: TreeRemovalRequest{Root: root, Target: core.AbsoluteDirectoryPath(rootText + string(filepath.Separator) + "\x00")}, wantErr: core.ErrDurabilityContract},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotErr := testCase.request.Validate()
			if !errors.Is(gotErr, testCase.wantErr) {
				t.Fatalf("TreeRemovalRequest.Validate() error = %v, want %v", gotErr, testCase.wantErr)
			}
		})
	}
}

func TestFileRemovalRequestContainmentBoundaryTable(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root := core.AbsoluteDirectoryPath(rootText)
	tests := []struct {
		wantErr error
		request FileRemovalRequest
		name    string
	}{
		{name: "direct child", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(rootText, "child"))}},
		{name: "deep child", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(rootText, "a", "b", "child"))}},
		{name: "unicode child", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(rootText, "界"))}},
		{name: "dot-prefixed child", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(rootText, ".stage"))}},
		{name: "zero root", request: FileRemovalRequest{Target: core.AbsoluteFilePath(filepath.Join(rootText, "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "zero target", request: FileRemovalRequest{Root: root}, wantErr: core.ErrDurabilityContract},
		{name: "relative root", request: FileRemovalRequest{Root: "relative", Target: core.AbsoluteFilePath(filepath.Join(rootText, "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "relative target", request: FileRemovalRequest{Root: root, Target: "relative"}, wantErr: core.ErrDurabilityContract},
		{name: "unclean target", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(rootText + string(filepath.Separator) + "a" + string(filepath.Separator) + ".." + string(filepath.Separator) + "child")}, wantErr: core.ErrDurabilityContract},
		{name: "root itself", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(rootText)}, wantErr: core.ErrDurabilityContract},
		{name: "parent of root", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Dir(rootText))}, wantErr: core.ErrDurabilityContract},
		{name: "prefix sibling outside root", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(rootText + "-copy")}, wantErr: core.ErrDurabilityContract},
		{name: "distant outside root", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(string(filepath.Separator), "outside", "child"))}, wantErr: core.ErrDurabilityContract},
		{name: "nul target", request: FileRemovalRequest{Root: root, Target: core.AbsoluteFilePath(rootText + string(filepath.Separator) + "\x00")}, wantErr: core.ErrDurabilityContract},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			gotErr := testCase.request.Validate()
			if !errors.Is(gotErr, testCase.wantErr) {
				t.Fatalf("FileRemovalRequest.Validate() error = %v, want %v", gotErr, testCase.wantErr)
			}
		})
	}
}

func TestRemoveTreeRealFilesystemHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup func(*testing.T, string) (core.AbsoluteDirectoryPath, context.Context)
		name  string
	}{
		{
			name: "missing target is an exact no-op",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				return core.AbsoluteDirectoryPath(filepath.Join(root, "missing")), t.Context()
			},
		},
		{
			name: "empty directory is removed",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				target := filepath.Join(root, "empty")
				mustCreateRemovalDirectory(t, target)
				return core.AbsoluteDirectoryPath(target), t.Context()
			},
		},
		{
			name: "nested tree is removed bottom-up",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				target := filepath.Join(root, "nested")
				mustCreateRemovalDirectory(t, filepath.Join(target, "a", "b", "c"))
				mustWriteRemovalFile(t, filepath.Join(target, "a", "b", "c", "data"))
				return core.AbsoluteDirectoryPath(target), t.Context()
			},
		},
		{
			name: "directory wider than one standard-library batch is removed",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				target := filepath.Join(root, "wide")
				mustCreateRemovalDirectory(t, target)
				for index := range 1025 {
					mustWriteRemovalFile(t, filepath.Join(target, strconv.Itoa(index)))
				}
				return core.AbsoluteDirectoryPath(target), t.Context()
			},
		},
		{
			name: "deep bounded path is removed",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				target := filepath.Join(root, "deep")
				deepest := target
				for index := range 64 {
					deepest = filepath.Join(deepest, strconv.Itoa(index))
				}
				mustCreateRemovalDirectory(t, deepest)
				mustWriteRemovalFile(t, filepath.Join(deepest, "data"))
				return core.AbsoluteDirectoryPath(target), t.Context()
			},
		},
		{
			name: "cancelled context preserves target",
			setup: func(t *testing.T, root string) (core.AbsoluteDirectoryPath, context.Context) {
				target := filepath.Join(root, "cancelled")
				mustCreateRemovalDirectory(t, target)
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return core.AbsoluteDirectoryPath(target), ctx
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			sibling := filepath.Join(root, "sibling")
			mustWriteRemovalFile(t, sibling)
			target, ctx := testCase.setup(t, root)
			gotErr := RemoveTree(ctx, TreeRemovalRequest{Root: core.AbsoluteDirectoryPath(root), Target: target})
			cancelled := errors.Is(context.Cause(ctx), context.Canceled)
			if cancelled {
				if !errors.Is(gotErr, context.Canceled) {
					t.Fatalf("RemoveTree(cancelled) error = %v, want context.Canceled", gotErr)
				}
				if _, err := os.Lstat(target.String()); err != nil {
					t.Fatalf("cancelled target Lstat() error = %v, want target preserved", err)
				}
			} else {
				if gotErr != nil {
					t.Fatalf("RemoveTree() error = %v, want nil", gotErr)
				}
				if _, err := os.Lstat(target.String()); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("removed target Lstat() error = %v, want fs.ErrNotExist", err)
				}
			}
			if data, err := os.ReadFile(sibling); err != nil || string(data) != "evidence" {
				t.Fatalf("sibling after RemoveTree() = (%q,%v), want preserved evidence", data, err)
			}
		})
	}
}

func TestRemoveTreeRejectsRootMutationAndNonDirectoryRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr   error
		setupRoot func(*testing.T) core.AbsoluteDirectoryPath
		name      string
	}{
		{
			name: "missing root",
			setupRoot: func(t *testing.T) core.AbsoluteDirectoryPath {
				return core.AbsoluteDirectoryPath(filepath.Join(t.TempDir(), "missing"))
			},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "regular file root",
			setupRoot: func(t *testing.T) core.AbsoluteDirectoryPath {
				root := filepath.Join(t.TempDir(), "file")
				mustWriteRemovalFile(t, root)
				return core.AbsoluteDirectoryPath(root)
			},
			wantErr: core.ErrDurabilityContract,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := testCase.setupRoot(t)
			target := core.AbsoluteDirectoryPath(filepath.Join(root.String(), "child"))
			gotErr := RemoveTree(t.Context(), TreeRemovalRequest{Root: root, Target: target})
			if !errors.Is(gotErr, testCase.wantErr) {
				t.Fatalf("RemoveTree() error = %v, want %v", gotErr, testCase.wantErr)
			}
		})
	}
}

func mustCreateRemovalDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustWriteRemovalFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("evidence"), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestRemoveTreeTerminalSymlinkDoesNotRemoveTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "evidence")
	mustWriteRemovalFile(t, outsideFile)
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}
	gotErr := RemoveTree(t.Context(), TreeRemovalRequest{
		Root: core.AbsoluteDirectoryPath(root), Target: core.AbsoluteDirectoryPath(link),
	})
	if !errors.Is(gotErr, core.ErrDurabilityContract) {
		t.Fatalf("RemoveTree(symlink) error = %v, want ErrDurabilityContract", gotErr)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("rejected symlink Lstat() error = %v, want link preserved", err)
	}
	if data, err := os.ReadFile(outsideFile); err != nil || string(data) != "evidence" {
		t.Fatalf("outside target after symlink removal = (%q,%v), want preserved evidence", data, err)
	}
}

func TestRemoveTreeIntermediateSymlinkCannotEscapeRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	outsideChild := filepath.Join(outside, "child")
	mustWriteRemovalFile(t, outsideChild)
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}
	gotErr := RemoveTree(t.Context(), TreeRemovalRequest{
		Root: core.AbsoluteDirectoryPath(root), Target: core.AbsoluteDirectoryPath(filepath.Join(link, "child")),
	})
	if !errors.Is(gotErr, core.ErrDurabilityContract) {
		t.Fatalf("RemoveTree(intermediate symlink escape) error = %v, want ErrDurabilityContract", gotErr)
	}
	if data, err := os.ReadFile(outsideChild); err != nil || string(data) != "evidence" {
		t.Fatalf("outside child after rejected removal = (%q,%v), want preserved evidence", data, err)
	}
}

func TestRemoveFileRealFilesystemHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup func(*testing.T, string) core.AbsoluteFilePath
		name  string
	}{
		{
			name: "missing file is an exact no-op",
			setup: func(_ *testing.T, root string) core.AbsoluteFilePath {
				return core.AbsoluteFilePath(filepath.Join(root, "missing"))
			},
		},
		{
			name: "empty regular file is removed",
			setup: func(t *testing.T, root string) core.AbsoluteFilePath {
				target := filepath.Join(root, "empty")
				if err := os.WriteFile(target, nil, 0o600); err != nil {
					t.Fatalf("WriteFile(empty) error = %v", err)
				}
				return core.AbsoluteFilePath(target)
			},
		},
		{
			name: "nonempty regular file is removed",
			setup: func(t *testing.T, root string) core.AbsoluteFilePath {
				target := filepath.Join(root, "evidence")
				mustWriteRemovalFile(t, target)
				return core.AbsoluteFilePath(target)
			},
		},
		{
			name: "terminal symlink is unlinked without following target",
			setup: func(t *testing.T, root string) core.AbsoluteFilePath {
				outside := t.TempDir()
				mustWriteRemovalFile(t, filepath.Join(outside, "evidence"))
				target := filepath.Join(root, "link")
				if err := os.Symlink(outside, target); err != nil {
					t.Skipf("Symlink() error = %v", err)
				}
				return core.AbsoluteFilePath(target)
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			sibling := filepath.Join(root, "sibling")
			mustWriteRemovalFile(t, sibling)
			target := testCase.setup(t, root)
			gotErr := RemoveFile(t.Context(), FileRemovalRequest{
				Root: core.AbsoluteDirectoryPath(root), Target: target,
			})
			if gotErr != nil {
				t.Fatalf("RemoveFile() error = %v, want nil", gotErr)
			}
			if _, err := os.Lstat(target.String()); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("removed file Lstat() error = %v, want fs.ErrNotExist", err)
			}
			if data, err := os.ReadFile(sibling); err != nil || string(data) != "evidence" {
				t.Fatalf("sibling after RemoveFile() = (%q,%v), want preserved evidence", data, err)
			}
		})
	}
}

func TestRemoveFileRejectsDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "directory")
	mustCreateRemovalDirectory(t, target)
	gotErr := RemoveFile(t.Context(), FileRemovalRequest{
		Root: core.AbsoluteDirectoryPath(root), Target: core.AbsoluteFilePath(target),
	})
	if !errors.Is(gotErr, core.ErrDurabilityContract) {
		t.Fatalf("RemoveFile(directory) error = %v, want ErrDurabilityContract", gotErr)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("directory after rejected RemoveFile() = (%v,%v), want preserved directory", info, err)
	}
}
