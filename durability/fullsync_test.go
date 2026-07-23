package durability

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFullSyncRealFileKindsAndInvalidHandles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cases := []struct {
		open      func() (*os.File, error)
		name      string
		wantError bool
	}{
		{name: "p01_empty_regular", open: func() (*os.File, error) { return os.Create(filepath.Join(dir, "empty")) }},
		{name: "p02_written_regular", open: func() (*os.File, error) {
			file, err := os.Create(filepath.Join(dir, "written"))
			if err == nil {
				_, err = file.Write([]byte("evidence"))
			}
			return file, err
		}},
		{name: "p03_append_regular", open: func() (*os.File, error) {
			return os.OpenFile(filepath.Join(dir, "append"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		}},
		{name: "p04_read_only_regular", open: func() (*os.File, error) {
			path := filepath.Join(dir, "read-only")
			if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
				return nil, err
			}
			return os.Open(path)
		}},
		{name: "p05_directory", open: func() (*os.File, error) { return os.Open(dir) }},
		{name: "n01_closed_regular", open: func() (*os.File, error) {
			file, err := os.Create(filepath.Join(dir, "closed"))
			if err == nil {
				err = file.Close()
			}
			return file, err
		}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			file, err := tc.open()
			if err != nil {
				t.Fatalf("open fixture error = %v", err)
			}
			if !tc.wantError {
				t.Cleanup(func() { _ = file.Close() })
			}
			err = CommitFile(file)
			if tc.wantError && err == nil {
				t.Fatalf("CommitFile() error = nil, want typed handle failure")
			}
			if !tc.wantError && err != nil {
				var pathErr *os.PathError
				if tc.name != "p05_directory" || !errors.As(err, &pathErr) {
					t.Fatalf("CommitFile() error = %v, want nil (or typed directory limitation)", err)
				}
			}
		})
	}
	if err := CommitFile(nil); !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("CommitFile(nil) error = %v, want os.ErrInvalid", err)
	}
}
