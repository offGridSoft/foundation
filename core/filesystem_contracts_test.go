package core

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAbsoluteFilesystemPathsHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	maximum := "/" + strings.Repeat("a", FilesystemPathMaxRunes-1)
	overMaximum := maximum + "a"
	invalidUTF8 := "/" + string([]byte{0xff})
	cases := []struct {
		name          string
		value         string
		wantFile      bool
		wantDirectory bool
	}{
		{name: "p01_file_at_root", value: "/a", wantFile: true, wantDirectory: true},
		{name: "p02_nested_file", value: "/var/lib/tool/state.json", wantFile: true, wantDirectory: true},
		{name: "p03_unicode", value: "/tmp/日本語/évidence", wantFile: true, wantDirectory: true},
		{name: "p04_dotfile", value: "/tmp/.state", wantFile: true, wantDirectory: true},
		{name: "p05_spaces_are_explicit", value: "/tmp/evidence file", wantFile: true, wantDirectory: true},
		{name: "p06_shell_metacharacters_are_opaque", value: "/tmp/$literal;name", wantFile: true, wantDirectory: true},
		{name: "p07_single_quote_is_opaque", value: "/tmp/client's-file", wantFile: true, wantDirectory: true},
		{name: "p08_maximum_runes", value: maximum, wantFile: true, wantDirectory: true},
		{name: "p09_root_directory", value: "/", wantDirectory: true},
		{name: "p10_dot_prefixed_segment", value: "/tmp/.cache/item", wantFile: true, wantDirectory: true},
		{name: "n01_empty"},
		{name: "n02_relative", value: "tmp/file"},
		{name: "n03_dot_relative", value: "./file"},
		{name: "n04_parent_relative", value: "../file"},
		{name: "n05_unclean_dot", value: "/tmp/./file"},
		{name: "n06_unclean_parent", value: "/tmp/../file"},
		{name: "n07_repeated_separator", value: "/tmp//file"},
		{name: "n08_trailing_separator", value: "/tmp/file/"},
		{name: "n09_nul", value: "/tmp/a\x00b"},
		{name: "n10_invalid_utf8", value: invalidUTF8},
		{name: "n11_over_maximum", value: overMaximum},
		{name: "n12_root_is_not_file", value: "/", wantDirectory: true},
		{name: "b01_one_rune_file", value: "/x", wantFile: true, wantDirectory: true},
		{name: "b02_two_dots_inside_name", value: "/tmp/a..b", wantFile: true, wantDirectory: true},
		{name: "b03_dot_segment_prefix", value: "/tmp/.a", wantFile: true, wantDirectory: true},
		{name: "b04_parent_token_inside_name", value: "/tmp/..a", wantFile: true, wantDirectory: true},
		{name: "b05_newline_is_opaque_path_data", value: "/tmp/a\nb", wantFile: true, wantDirectory: true},
		{name: "b06_tab_is_opaque_path_data", value: "/tmp/a\tb", wantFile: true, wantDirectory: true},
		{name: "b07_backslash_is_filename_on_unix", value: "/tmp/a\\b", wantFile: true, wantDirectory: true},
		{name: "b08_leading_space_segment", value: "/tmp/ file", wantFile: true, wantDirectory: true},
		{name: "b09_trailing_space_segment", value: "/tmp/file ", wantFile: true, wantDirectory: true},
		{name: "b10_multibyte_counts_as_runes", value: "/" + strings.Repeat("界", FilesystemPathMaxRunes-1), wantFile: true, wantDirectory: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			file, fileErr := ParseAbsoluteFilePath(tc.value)
			if tc.wantFile && fileErr != nil {
				t.Fatalf("ParseAbsoluteFilePath(%q) error = %v, want nil", tc.value, fileErr)
			}
			if !tc.wantFile && (!errors.Is(fileErr, ErrFilesystemContract) || !errors.Is(fileErr, ErrFoundationContract)) {
				t.Fatalf("ParseAbsoluteFilePath(%q) error = %v, want filesystem and foundation identities", tc.value, fileErr)
			}
			if tc.wantFile && (file.String() != tc.value || file.Validate() != nil) {
				t.Fatalf("file path round trip = (%q, %v), want (%q, nil)", file.String(), file.Validate(), tc.value)
			}

			directory, directoryErr := ParseAbsoluteDirectoryPath(tc.value)
			if tc.wantDirectory && directoryErr != nil {
				t.Fatalf("ParseAbsoluteDirectoryPath(%q) error = %v, want nil", tc.value, directoryErr)
			}
			if !tc.wantDirectory && (!errors.Is(directoryErr, ErrFilesystemContract) || !errors.Is(directoryErr, ErrFoundationContract)) {
				t.Fatalf("ParseAbsoluteDirectoryPath(%q) error = %v, want filesystem and foundation identities", tc.value, directoryErr)
			}
			if tc.wantDirectory && (directory.String() != tc.value || directory.Validate() != nil) {
				t.Fatalf("directory path round trip = (%q, %v), want (%q, nil)", directory.String(), directory.Validate(), tc.value)
			}
		})
	}

	if utf8.RuneCountInString(maximum) != FilesystemPathMaxRunes {
		t.Fatalf("maximum fixture runes = %d, want %d", utf8.RuneCountInString(maximum), FilesystemPathMaxRunes)
	}
}
