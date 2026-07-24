package temporal

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestPublicAPIMatchesReviewedSpecificationRatchet(t *testing.T) {
	t.Parallel()

	got := collectPublicAPI(t)
	want := []string{
		"const:HumanUnitAutomatic",
		"const:HumanUnitDays",
		"const:HumanUnitHours",
		"const:HumanUnitMicroseconds",
		"const:HumanUnitMilliseconds",
		"const:HumanUnitMinutes",
		"const:HumanUnitNanoseconds",
		"const:HumanUnitSeconds",
		"const:HumanUnitUnknown",
		"const:HumanUnitYears",
		"const:HumanizeStyleCompact",
		"const:HumanizeStyleLong",
		"const:HumanizeStyleUnknown",
		"const:OrderAfter",
		"const:OrderBefore",
		"const:OrderEqual",
		"const:OrderUnknown",
		"field:FirestoreAggregateDuration.Nanoseconds:string",
		"field:FirestoreDuration.Nanoseconds:int64",
		"field:FirestoreInstant.Nanoseconds:int64",
		"field:FirestoreInstant.QueryTimestamp:time.Time",
		"field:HumanizePolicy.FractionDigits:uint8",
		"field:HumanizePolicy.Style:HumanizeStyle",
		"field:HumanizePolicy.Unit:HumanUnit",
		"field:PostgreSQLAggregateDuration.Nanoseconds:string",
		"field:PostgreSQLDuration.Nanoseconds:int64",
		"field:PostgreSQLInstant.Nanoseconds:int64",
		"field:PostgreSQLInstant.QueryTimestamp:time.Time",
		"func:AggregateDurationFromDuration@public.go",
		"func:AggregateDurationFromFirestore@public.go",
		"func:AggregateDurationFromPostgreSQL@public.go",
		"func:DurationFromFirestore@public.go",
		"func:DurationFromNanoseconds@public.go",
		"func:DurationFromPostgreSQL@public.go",
		"func:InstantFromFirestore@public.go",
		"func:InstantFromNanoseconds@public.go",
		"func:InstantFromPostgreSQL@public.go",
		"func:NewDuration@public.go",
		"func:NewInstant@public.go",
		"func:ParseAggregateDuration@public.go",
		"func:ParseHumanUnit@public.go",
		"func:ParseHumanizeStyle@public.go",
		"func:ParseOrder@public.go",
		"method:AggregateDuration.Add",
		"method:AggregateDuration.AddDuration",
		"method:AggregateDuration.Compare",
		"method:AggregateDuration.Decimal",
		"method:AggregateDuration.Firestore",
		"method:AggregateDuration.Humanize",
		"method:AggregateDuration.IsZero",
		"method:AggregateDuration.MarshalJSON",
		"method:AggregateDuration.Multiply",
		"method:AggregateDuration.PostgreSQL",
		"method:AggregateDuration.UnmarshalJSON",
		"method:AggregateDuration.Validate",
		"method:Duration.Add",
		"method:Duration.Aggregate",
		"method:Duration.Compare",
		"method:Duration.Firestore",
		"method:Duration.Humanize",
		"method:Duration.IsZero",
		"method:Duration.MarshalJSON",
		"method:Duration.Multiply",
		"method:Duration.Nanoseconds",
		"method:Duration.PostgreSQL",
		"method:Duration.Stdlib",
		"method:Duration.UnmarshalJSON",
		"method:Duration.Validate",
		"method:FirestoreAggregateDuration.Validate",
		"method:FirestoreDuration.Validate",
		"method:FirestoreInstant.Validate",
		"method:HumanUnit.IsValid",
		"method:HumanUnit.MarshalJSON",
		"method:HumanUnit.String",
		"method:HumanUnit.UnmarshalJSON",
		"method:HumanUnit.Validate",
		"method:HumanizePolicy.Validate",
		"method:HumanizeStyle.IsValid",
		"method:HumanizeStyle.MarshalJSON",
		"method:HumanizeStyle.String",
		"method:HumanizeStyle.UnmarshalJSON",
		"method:HumanizeStyle.Validate",
		"method:Humanized.Number",
		"method:Humanized.Text",
		"method:Humanized.Unit",
		"method:Humanized.Validate",
		"method:Instant.Add",
		"method:Instant.Compare",
		"method:Instant.Firestore",
		"method:Instant.IsSet",
		"method:Instant.MarshalJSON",
		"method:Instant.Nanoseconds",
		"method:Instant.PostgreSQL",
		"method:Instant.Since",
		"method:Instant.Subtract",
		"method:Instant.Time",
		"method:Instant.UnmarshalJSON",
		"method:Instant.Validate",
		"method:Order.IsValid",
		"method:Order.MarshalJSON",
		"method:Order.String",
		"method:Order.UnmarshalJSON",
		"method:Order.Validate",
		"method:PostgreSQLAggregateDuration.Validate",
		"method:PostgreSQLDuration.Validate",
		"method:PostgreSQLInstant.Validate",
		"type:AggregateDuration",
		"type:Duration",
		"type:FirestoreAggregateDuration",
		"type:FirestoreDuration",
		"type:FirestoreInstant",
		"type:HumanUnit",
		"type:HumanizePolicy",
		"type:HumanizeStyle",
		"type:Humanized",
		"type:Instant",
		"type:Order",
		"type:PostgreSQLAggregateDuration",
		"type:PostgreSQLDuration",
		"type:PostgreSQLInstant",
	}
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("public API drifted\n got: %s\nwant: %s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDependencySurfaceIsPrimitiveOnlyRatchet(t *testing.T) {
	t.Parallel()

	got := collectProductionImports(t)
	want := []string{
		"encoding/json",
		"errors",
		"fmt",
		"github.com/offGridSoft/foundation/v2026/core",
		"math",
		"math/bits",
		"strconv",
		"strings",
		"time",
	}
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("production imports drifted\n got: %s\nwant: %s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func collectPublicAPI(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	var api []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, entry.Name(), nil, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		api = append(api, exportedDeclarations(t, fileSet, entry.Name(), file)...)
	}
	sort.Strings(api)
	return api
}

func collectProductionImports(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	var imports []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, entry.Name(), nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		for _, specification := range file.Imports {
			path, unquoteErr := strconv.Unquote(specification.Path.Value)
			if unquoteErr != nil {
				t.Fatal(unquoteErr)
			}
			if !slices.Contains(imports, path) {
				imports = append(imports, path)
			}
		}
	}
	sort.Strings(imports)
	return imports
}

func exportedDeclarations(t *testing.T, fileSet *token.FileSet, fileName string, file *ast.File) []string {
	t.Helper()

	var api []string
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.FuncDecl:
			api = appendExportedFunction(api, fileName, value)
		case *ast.GenDecl:
			api = appendExportedGeneral(t, api, fileSet, value)
		}
	}
	return api
}

func appendExportedFunction(api []string, fileName string, function *ast.FuncDecl) []string {
	if !function.Name.IsExported() {
		return api
	}
	if function.Recv == nil {
		return append(api, "func:"+function.Name.Name+"@"+filepath.Base(fileName))
	}
	return append(api, "method:"+receiverName(function.Recv.List[0].Type)+"."+function.Name.Name)
}

func appendExportedGeneral(t *testing.T, api []string, fileSet *token.FileSet, declaration *ast.GenDecl) []string {
	t.Helper()

	for _, specification := range declaration.Specs {
		switch value := specification.(type) {
		case *ast.TypeSpec:
			if value.Name.IsExported() {
				api = append(api, "type:"+value.Name.Name)
				api = append(api, exportedFields(t, fileSet, value)...)
			}
		case *ast.ValueSpec:
			api = append(api, exportedValues(declaration.Tok, value)...)
		}
	}
	return api
}

func exportedFields(t *testing.T, fileSet *token.FileSet, specification *ast.TypeSpec) []string {
	t.Helper()

	structure, ok := specification.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	var fields []string
	for _, field := range structure.Fields.List {
		fieldType := formatExpression(t, fileSet, field.Type)
		for _, name := range field.Names {
			if name.IsExported() {
				fields = append(fields, "field:"+specification.Name.Name+"."+name.Name+":"+fieldType)
			}
		}
	}
	return fields
}

func exportedValues(kind token.Token, specification *ast.ValueSpec) []string {
	var values []string
	for _, name := range specification.Names {
		if name.IsExported() {
			values = append(values, strings.ToLower(kind.String())+":"+name.Name)
		}
	}
	return values
}

func formatExpression(t *testing.T, fileSet *token.FileSet, expression ast.Expr) string {
	t.Helper()

	var buffer bytes.Buffer
	if err := format.Node(&buffer, fileSet, expression); err != nil {
		t.Fatal(err)
	}
	return buffer.String()
}

func receiverName(expression ast.Expr) string {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}
