package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const FilesystemPathMaxRunes = 4096

type AbsoluteFilePath string

type AbsoluteDirectoryPath string

func ParseAbsoluteFilePath(value string) (AbsoluteFilePath, error) {
	if err := validateAbsoluteFilesystemPath(value); err != nil || filepath.Base(value) == string(filepath.Separator) {
		return "", fmt.Errorf(ErrFmtAbsoluteFilePath, ErrFilesystemContract)
	}
	return AbsoluteFilePath(value), nil
}

func (p AbsoluteFilePath) Validate() error {
	_, err := ParseAbsoluteFilePath(string(p))
	return err
}

func (p AbsoluteFilePath) String() string {
	return string(p)
}

func ParseAbsoluteDirectoryPath(value string) (AbsoluteDirectoryPath, error) {
	if err := validateAbsoluteFilesystemPath(value); err != nil {
		return "", fmt.Errorf(ErrFmtAbsoluteDirectoryPath, ErrFilesystemContract)
	}
	return AbsoluteDirectoryPath(value), nil
}

func (p AbsoluteDirectoryPath) Validate() error {
	_, err := ParseAbsoluteDirectoryPath(string(p))
	return err
}

func (p AbsoluteDirectoryPath) String() string {
	return string(p)
}

func validateAbsoluteFilesystemPath(value string) error {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > FilesystemPathMaxRunes {
		return ErrFilesystemContract
	}
	if strings.IndexByte(value, 0) >= 0 || !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return ErrFilesystemContract
	}
	return nil
}
