package core

import (
	"errors"
	"testing"
)

func FuzzAbsoluteFilesystemPathContracts(f *testing.F) {
	seeds := []string{
		"",
		"/",
		"/tmp/evidence",
		"relative",
		"/tmp/../escape",
		"/tmp/a\x00b",
		"/日本語/évidence",
		string([]byte{'/', 0xff}),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		file, fileErr := ParseAbsoluteFilePath(value)
		if fileErr == nil {
			if file.String() != value || file.Validate() != nil {
				t.Fatalf("ParseAbsoluteFilePath(%q) success produced (%q,%v), want exact valid round trip", value, file.String(), file.Validate())
			}
		} else if !errors.Is(fileErr, ErrFilesystemContract) || !errors.Is(fileErr, ErrFoundationContract) {
			t.Fatalf("ParseAbsoluteFilePath(%q) error = %v, want filesystem and foundation identities", value, fileErr)
		}

		directory, directoryErr := ParseAbsoluteDirectoryPath(value)
		if directoryErr == nil {
			if directory.String() != value || directory.Validate() != nil {
				t.Fatalf("ParseAbsoluteDirectoryPath(%q) success produced (%q,%v), want exact valid round trip", value, directory.String(), directory.Validate())
			}
		} else if !errors.Is(directoryErr, ErrFilesystemContract) || !errors.Is(directoryErr, ErrFoundationContract) {
			t.Fatalf("ParseAbsoluteDirectoryPath(%q) error = %v, want filesystem and foundation identities", value, directoryErr)
		}
	})
}
