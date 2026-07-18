package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWritePrivateMaterialHostileFilesystemBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("creates owner-only file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "secret")
		if err := writePrivateMaterial(path, "private"); err != nil {
			t.Fatalf("writePrivateMaterial() error = %v, want nil", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat() error = %v", err)
		}
		if got := info.Mode().Perm(); got != privateMaterialFileMode {
			t.Fatalf("mode = %04o, want %04o", got, privateMaterialFileMode)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile() error = %v", err)
		}
		if got, want := string(data), "private"; got != want {
			t.Fatalf("content = %q, want %q", got, want)
		}
		if got, want := len(data), len("private"); got != want {
			t.Fatalf("content bytes = %d, want exact canonical bytes %d", got, want)
		}
	})

	t.Run("refuses existing permissive file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "existing")
		if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
			t.Fatalf("fixture write error = %v", err)
		}
		if err := writePrivateMaterial(path, "replace"); !errors.Is(err, fs.ErrExist) || !errors.Is(err, errPrivateMaterialFile) {
			t.Fatalf("writePrivateMaterial(existing) error = %v, want errors.Is(..., fs.ErrExist) and errors.Is(..., errPrivateMaterialFile)", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile() error = %v", err)
		}
		if got, want := string(data), "keep"; got != want {
			t.Fatalf("existing content = %q, want %q", got, want)
		}
	})

	t.Run("refuses symlink output", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		link := filepath.Join(dir, "link")
		if err := os.WriteFile(target, []byte("keep"), privateMaterialFileMode); err != nil {
			t.Fatalf("target fixture write error = %v", err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatalf("os.Symlink() error = %v", err)
		}
		if err := writePrivateMaterial(link, "replace"); !errors.Is(err, fs.ErrExist) || !errors.Is(err, errPrivateMaterialFile) {
			t.Fatalf("writePrivateMaterial(symlink) error = %v, want errors.Is(..., fs.ErrExist) and errors.Is(..., errPrivateMaterialFile)", err)
		}
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("os.ReadFile(target) error = %v", err)
		}
		if got, want := string(data), "keep"; got != want {
			t.Fatalf("symlink target content = %q, want %q", got, want)
		}
	})
}
