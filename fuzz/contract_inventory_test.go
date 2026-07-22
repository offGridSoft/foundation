package fuzz

import foundationcore "github.com/offGridSoft/foundation/v2026/core"

var (
	_ foundationcore.Validatable = FuzzCorpusEntryName{}
	_ foundationcore.Validatable = FuzzCorpusSelection{}
	_ foundationcore.Validatable = FuzzCorpusDirectorySelection{}
)
