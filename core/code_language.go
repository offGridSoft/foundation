package core

import (
	"encoding/json"
	"fmt"
)

// CodeLanguage is the shared identity of one source-language family supported
// by OffGrid tooling. Product packages own their language-specific behavior;
// core owns the stable identity, order, token, and display name.
type CodeLanguage uint8

const (
	CodeLanguageUnknown CodeLanguage = iota
	CodeLanguageGo
	CodeLanguageJavaScript
	CodeLanguageTypeScript
	CodeLanguagePython
	CodeLanguageC
	CodeLanguageCPP
	CodeLanguageSwift
	CodeLanguageCSharp
	CodeLanguageJava
	CodeLanguagePerl
	CodeLanguagePHP
	CodeLanguageSQL
	CodeLanguageRust
	CodeLanguageZig
	CodeLanguageRuby
	CodeLanguageKotlin
	CodeLanguageDart
	CodeLanguageScala
	CodeLanguageR
	CodeLanguageShell
	CodeLanguageLua
	CodeLanguageObjectiveC
	CodeLanguageLisp
	CodeLanguageHaskell
	CodeLanguageAssembly
	CodeLanguageVisualBasic
	CodeLanguagePascal
	CodeLanguageFortran
	CodeLanguageCOBOL
	CodeLanguageAda
	CodeLanguageJulia
	CodeLanguageGroovy
	CodeLanguageClojure
	CodeLanguageElixir
	CodeLanguageErlang
	CodeLanguageFSharp
	CodeLanguagePowerShell
	CodeLanguageRacket
	CodeLanguageOCaml
	CodeLanguageNim
	CodeLanguageCrystal
	CodeLanguageV
	CodeLanguageSolidity
	CodeLanguageTerraform
	CodeLanguageMake
	CodeLanguageDockerfile
	CodeLanguageYAML
	CodeLanguageVue
	CodeLanguageSvelte
	CodeLanguageDelphi
)

const CodeLanguageCount = int(CodeLanguageDelphi)

func codeLanguageTokens() [CodeLanguageDelphi + 1]string {
	return [...]string{
		CodeLanguageGo:          "go",
		CodeLanguageJavaScript:  "javascript",
		CodeLanguageTypeScript:  "typescript",
		CodeLanguagePython:      "python",
		CodeLanguageC:           "c",
		CodeLanguageCPP:         "cpp",
		CodeLanguageSwift:       "swift",
		CodeLanguageCSharp:      "csharp",
		CodeLanguageJava:        "java",
		CodeLanguagePerl:        "perl",
		CodeLanguagePHP:         "php",
		CodeLanguageSQL:         "sql",
		CodeLanguageRust:        "rust",
		CodeLanguageZig:         "zig",
		CodeLanguageRuby:        "ruby",
		CodeLanguageKotlin:      "kotlin",
		CodeLanguageDart:        "dart",
		CodeLanguageScala:       "scala",
		CodeLanguageR:           "r",
		CodeLanguageShell:       "shell",
		CodeLanguageLua:         "lua",
		CodeLanguageObjectiveC:  "objective-c",
		CodeLanguageLisp:        "lisp",
		CodeLanguageHaskell:     "haskell",
		CodeLanguageAssembly:    "assembly",
		CodeLanguageVisualBasic: "visual-basic",
		CodeLanguagePascal:      "pascal",
		CodeLanguageFortran:     "fortran",
		CodeLanguageCOBOL:       "cobol",
		CodeLanguageAda:         "ada",
		CodeLanguageJulia:       "julia",
		CodeLanguageGroovy:      "groovy",
		CodeLanguageClojure:     "clojure",
		CodeLanguageElixir:      "elixir",
		CodeLanguageErlang:      "erlang",
		CodeLanguageFSharp:      "fsharp",
		CodeLanguagePowerShell:  "powershell",
		CodeLanguageRacket:      "racket",
		CodeLanguageOCaml:       "ocaml",
		CodeLanguageNim:         "nim",
		CodeLanguageCrystal:     "crystal",
		CodeLanguageV:           "v",
		CodeLanguageSolidity:    "solidity",
		CodeLanguageTerraform:   "terraform",
		CodeLanguageMake:        "make",
		CodeLanguageDockerfile:  "dockerfile",
		CodeLanguageYAML:        "yaml",
		CodeLanguageVue:         "vue",
		CodeLanguageSvelte:      "svelte",
		CodeLanguageDelphi:      "delphi",
	}
}

func codeLanguageDisplayNames() [CodeLanguageDelphi + 1]string {
	return [...]string{
		CodeLanguageGo:          "Go",
		CodeLanguageJavaScript:  "JavaScript",
		CodeLanguageTypeScript:  "TypeScript",
		CodeLanguagePython:      "Python",
		CodeLanguageC:           "C",
		CodeLanguageCPP:         "C++",
		CodeLanguageSwift:       "Swift",
		CodeLanguageCSharp:      "C#",
		CodeLanguageJava:        "Java",
		CodeLanguagePerl:        "Perl",
		CodeLanguagePHP:         "PHP",
		CodeLanguageSQL:         "SQL",
		CodeLanguageRust:        "Rust",
		CodeLanguageZig:         "Zig",
		CodeLanguageRuby:        "Ruby",
		CodeLanguageKotlin:      "Kotlin",
		CodeLanguageDart:        "Dart",
		CodeLanguageScala:       "Scala",
		CodeLanguageR:           "R",
		CodeLanguageShell:       "Shell",
		CodeLanguageLua:         "Lua",
		CodeLanguageObjectiveC:  "Objective-C",
		CodeLanguageLisp:        "Lisp",
		CodeLanguageHaskell:     "Haskell",
		CodeLanguageAssembly:    "Assembly",
		CodeLanguageVisualBasic: "Visual Basic",
		CodeLanguagePascal:      "Pascal",
		CodeLanguageFortran:     "Fortran",
		CodeLanguageCOBOL:       "COBOL",
		CodeLanguageAda:         "Ada",
		CodeLanguageJulia:       "Julia",
		CodeLanguageGroovy:      "Groovy",
		CodeLanguageClojure:     "Clojure",
		CodeLanguageElixir:      "Elixir",
		CodeLanguageErlang:      "Erlang",
		CodeLanguageFSharp:      "F#",
		CodeLanguagePowerShell:  "PowerShell",
		CodeLanguageRacket:      "Racket",
		CodeLanguageOCaml:       "OCaml",
		CodeLanguageNim:         "Nim",
		CodeLanguageCrystal:     "Crystal",
		CodeLanguageV:           "V",
		CodeLanguageSolidity:    "Solidity",
		CodeLanguageTerraform:   "Terraform",
		CodeLanguageMake:        "Make",
		CodeLanguageDockerfile:  "Dockerfile",
		CodeLanguageYAML:        "YAML",
		CodeLanguageVue:         "Vue",
		CodeLanguageSvelte:      "Svelte",
		CodeLanguageDelphi:      "Delphi",
	}
}

func AllCodeLanguages() [CodeLanguageCount]CodeLanguage {
	var languages [CodeLanguageCount]CodeLanguage
	for index := range languages {
		languages[index] = CodeLanguage(index + 1)
	}
	return languages
}

func (l CodeLanguage) String() string {
	if !l.IsValid() {
		return ""
	}
	return codeLanguageTokens()[l]
}

func (l CodeLanguage) DisplayName() string {
	if !l.IsValid() {
		return ""
	}
	return codeLanguageDisplayNames()[l]
}

func (l CodeLanguage) IsValid() bool {
	return l > CodeLanguageUnknown && int(l) < len(codeLanguageTokens()) && codeLanguageTokens()[l] != "" && codeLanguageDisplayNames()[l] != ""
}

func (l CodeLanguage) Validate() error {
	if !l.IsValid() {
		return fmt.Errorf(ErrFmtCodeLanguage, ErrFoundationContract)
	}
	return nil
}

func ParseCodeLanguage(token string) (CodeLanguage, error) {
	for _, language := range AllCodeLanguages() {
		if language.String() == token {
			return language, nil
		}
	}
	return CodeLanguageUnknown, fmt.Errorf(ErrFmtCodeLanguage, ErrFoundationContract)
}

func (l CodeLanguage) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(l.String())
}

func (l *CodeLanguage) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCodeLanguage, ErrFoundationContract)
	}
	parsed, err := ParseCodeLanguage(token)
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}
