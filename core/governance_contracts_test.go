package core

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGovernanceDocumentContractHostileTable(t *testing.T) {
	t.Parallel()

	root := governanceModuleRoot(t)
	for _, document := range GovernanceDocuments() {
		t.Run(document.Path(), func(t *testing.T) {
			t.Parallel()
			requireGovernanceDocument(t, root, document)
		})
	}
}

func TestGovernanceDocumentRejectsInvalidOrdinals(t *testing.T) {
	t.Parallel()
	for _, document := range []GovernanceDocument{GovernanceDocumentUnknown, GovernanceDocument(3), GovernanceDocument(255)} {
		if err := document.Validate(); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("GovernanceDocument(%d).Validate() error = %v, want ErrFoundationContract", document, err)
		}
		if document.Path() != "" {
			t.Fatalf("GovernanceDocument(%d).Path() = %q, want empty", document, document.Path())
		}
		if _, err := document.ExpectedSHA256(); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("GovernanceDocument(%d).ExpectedSHA256() error = %v, want ErrFoundationContract", document, err)
		}
	}
}

func requireGovernanceDocument(t *testing.T, root string, document GovernanceDocument) {
	t.Helper()
	if err := document.Validate(); err != nil {
		t.Fatalf("GovernanceDocument.Validate() error = %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(document.Path()))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%s) error = %v", document.Path(), err)
	}
	if info.Size() <= 0 || info.Size() > GovernanceDocumentDefaultMaxBytes {
		t.Fatalf("%s bytes = %d, want 1..%d", document.Path(), info.Size(), GovernanceDocumentDefaultMaxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", document.Path(), err)
	}
	want, err := document.ExpectedSHA256()
	if err != nil {
		t.Fatalf("GovernanceDocument.ExpectedSHA256() error = %v", err)
	}
	if got := NewSHA256Hex(sha256.Sum256(data)); got != want {
		t.Fatalf("%s sha256 = %s, want %s", document.Path(), got, want)
	}
}

func governanceModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, GoModuleFileName)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("module root not found from %s", dir)
		}
		dir = parent
	}
}
