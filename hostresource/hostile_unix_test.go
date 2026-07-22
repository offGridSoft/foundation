//go:build !windows

package hostresource

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestMeasureTreeUnreadableSubdirectoryFailsLoudly(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permission enforcement")
	}
	rootText := t.TempDir()
	root, err := core.ParseAbsoluteDirectoryPath(rootText)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath(root) error = %v", err)
	}
	locked := filepath.Join(rootText, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatalf("Mkdir(locked) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(locked, "hidden"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("WriteFile(hidden) error = %v", err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("Chmod(locked, 0) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	usage, err := MeasureTree(t.Context(), TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject})
	if !errors.Is(err, fs.ErrPermission) || usage != (TreeUsage{}) {
		t.Fatalf("MeasureTree(unreadable subtree) = (%+v,%v), want zero usage and fs.ErrPermission instead of silent undercount", usage, err)
	}
}

func TestMeasureTreeRejectsSymlinkRootWithoutTraversingTarget(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "outside"), []byte("payload"), 0o600); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	link := filepath.Join(parent, "tree-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink(tree root) error = %v", err)
	}
	root, _ := core.ParseAbsoluteDirectoryPath(link)
	usage, err := MeasureTree(t.Context(), TreeUsageRequest{Root: root, MissingPolicy: MissingPathReject})
	if !errors.Is(err, core.ErrHostResourceContract) || usage != (TreeUsage{}) {
		t.Fatalf("MeasureTree(symlink root) = (%+v,%v), want zero usage and ErrHostResourceContract", usage, err)
	}
}
