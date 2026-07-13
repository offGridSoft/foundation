package core

import (
	"errors"
	"strings"
	"testing"
)

func TestSchemaIDsUseSingleCalendarGenerationAuthority(t *testing.T) {
	t.Parallel()

	names := schemaNames()
	for schema := SchemaUnknown + 1; int(schema) < len(names); schema++ {
		name := names[schema]
		if !strings.HasSuffix(name, "-"+ContractVersionToken) {
			t.Fatalf("SchemaID(%d).String() = %q, want suffix %q", schema, name, "-"+ContractVersionToken)
		}
		parsed, err := ParseSchemaID(name)
		if err != nil {
			t.Fatalf("ParseSchemaID(%q) error = %v, want nil", name, err)
		}
		if parsed != schema {
			t.Fatalf("ParseSchemaID(%q) = %v, want %v", name, parsed, schema)
		}
	}
}

func TestParseSchemaIDRejectsHostileAndRetiredGenerations(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"bug-usage-v1",
		"bug-usage-v2",
		"bug-usage-v2025",
		"bug-usage-v2027",
		"bug-usage-2026",
		"BUG-usage-v2026",
		"bug_usage_v2026",
		"bug-usage-v02026",
		"bug-usage-v2026 ",
		" bug-usage-v2026",
		"bug-usage-v2026\n",
		"bug-usage-v2026\x00",
		"bug-usage-v２０２６",
		"witness-usage-v1",
		"witness-usage-v2",
		"offgrid-release-manifest-v1",
		"offgrid-release-manifest-v2",
		"offgrid-release-manifest-v2025",
		"offgrid-release-manifest-v2026-extra",
	} {
		if _, err := ParseSchemaID(value); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("ParseSchemaID(%q) error = %v, want %v", value, err, ErrFoundationContract)
		}
	}
}
