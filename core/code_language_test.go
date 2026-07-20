package core

import (
	"errors"
	"testing"
)

func TestCodeLanguageCatalogContract(t *testing.T) {
	t.Parallel()
	languages := AllCodeLanguages()
	if len(languages) != CodeLanguageCount {
		t.Fatalf("len(AllCodeLanguages()) = %d, want %d", len(languages), CodeLanguageCount)
	}
	seenTokens := make([]string, 0, CodeLanguageCount)
	seenNames := make([]string, 0, CodeLanguageCount)
	for index, language := range languages {
		if language != CodeLanguage(index+1) {
			t.Fatalf("AllCodeLanguages()[%d] = %d, want %d", index, language, index+1)
		}
		if err := language.Validate(); err != nil {
			t.Fatalf("CodeLanguage(%d).Validate() error = %v", language, err)
		}
		for _, token := range seenTokens {
			if language.String() == token {
				t.Fatalf("CodeLanguage(%d).String() = duplicate %q", language, token)
			}
		}
		for _, name := range seenNames {
			if language.DisplayName() == name {
				t.Fatalf("CodeLanguage(%d).DisplayName() = duplicate %q", language, name)
			}
		}
		seenTokens = append(seenTokens, language.String())
		seenNames = append(seenNames, language.DisplayName())
		parsed, err := ParseCodeLanguage(language.String())
		if err != nil || parsed != language {
			t.Fatalf("ParseCodeLanguage(%q) = %v, %v; want %v, nil", language.String(), parsed, err, language)
		}
	}
}

func TestCodeLanguageRejectsInvalidBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		token    string
		language CodeLanguage
	}{
		{name: "zero", language: CodeLanguageUnknown},
		{name: "past catalog", language: CodeLanguage(CodeLanguageCount + 1)},
		{name: "blank token", token: ""},
		{name: "unknown token", token: "brainfuck"},
		{name: "case drift", token: "Go"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.token != "" || tc.name == "blank token" {
				_, err := ParseCodeLanguage(tc.token)
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseCodeLanguage(%q) error = %v, want ErrFoundationContract", tc.token, err)
				}
				return
			}
			if err := tc.language.Validate(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("CodeLanguage(%d).Validate() error = %v, want ErrFoundationContract", tc.language, err)
			}
		})
	}
}
