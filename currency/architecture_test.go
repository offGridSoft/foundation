package currency

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

	got := collectCurrencyPublicAPI(t)
	want := []string{
		"const:CodeAUD",
		"const:CodeBHD",
		"const:CodeCAD",
		"const:CodeCHF",
		"const:CodeCLF",
		"const:CodeEUR",
		"const:CodeGBP",
		"const:CodeHKD",
		"const:CodeJPY",
		"const:CodeNZD",
		"const:CodeSGD",
		"const:CodeUSD",
		"const:CodeUnknown",
		"const:DisplayUnitAutomatic",
		"const:DisplayUnitBillions",
		"const:DisplayUnitHundreds",
		"const:DisplayUnitMajor",
		"const:DisplayUnitMillions",
		"const:DisplayUnitMinor",
		"const:DisplayUnitThousands",
		"const:DisplayUnitUnknown",
		"const:OrderEqual",
		"const:OrderGreater",
		"const:OrderLess",
		"const:OrderUnknown",
		"field:FirestoreAmount.Currency:string",
		"field:FirestoreAmount.MinorUnits:int64",
		"field:HumanizePolicy.FractionDigits:uint8",
		"field:HumanizePolicy.Unit:DisplayUnit",
		"field:PostgreSQLAmount.Currency:string",
		"field:PostgreSQLAmount.MinorUnits:int64",
		"func:FromFirestore@public.go",
		"func:FromPostgreSQL@public.go",
		"func:New@public.go",
		"func:Parse@public.go",
		"func:ParseCode@public.go",
		"func:ParseDisplayUnit@public.go",
		"method:Amount.Add",
		"method:Amount.Code",
		"method:Amount.Compare",
		"method:Amount.Decimal",
		"method:Amount.Firestore",
		"method:Amount.Humanize",
		"method:Amount.IsNegative",
		"method:Amount.IsPositive",
		"method:Amount.IsZero",
		"method:Amount.MarshalJSON",
		"method:Amount.MinorUnits",
		"method:Amount.Multiply",
		"method:Amount.PostgreSQL",
		"method:Amount.Subtract",
		"method:Amount.UnmarshalJSON",
		"method:Amount.Validate",
		"method:Code.FractionDigits",
		"method:Code.IsValid",
		"method:Code.MarshalJSON",
		"method:Code.String",
		"method:Code.UnmarshalJSON",
		"method:Code.Validate",
		"method:DisplayUnit.IsValid",
		"method:DisplayUnit.MarshalJSON",
		"method:DisplayUnit.String",
		"method:DisplayUnit.UnmarshalJSON",
		"method:DisplayUnit.Validate",
		"method:FirestoreAmount.Validate",
		"method:HumanizePolicy.Validate",
		"method:Humanized.Code",
		"method:Humanized.Number",
		"method:Humanized.Unit",
		"method:Humanized.Validate",
		"method:Order.IsValid",
		"method:Order.MarshalJSON",
		"method:Order.String",
		"method:Order.UnmarshalJSON",
		"method:Order.Validate",
		"method:PostgreSQLAmount.Validate",
		"type:Amount",
		"type:Code",
		"type:DisplayUnit",
		"type:FirestoreAmount",
		"type:HumanizePolicy",
		"type:Humanized",
		"type:Order",
		"type:PostgreSQLAmount",
	}
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("public API drifted\n got: %s\nwant: %s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestProductionImportsRemainCoreAndStandardLibraryOnlyRatchet(t *testing.T) {
	t.Parallel()

	got := collectCurrencyProductionImports(t)
	want := []string{
		"encoding/json",
		"errors",
		"fmt",
		"github.com/offGridSoft/foundation/v2026/core",
		"math",
		"strconv",
		"strings",
	}
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("production imports drifted\n got: %s\nwant: %s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

type structRole uint8

const (
	structRoleUnknown structRole = iota
	structRoleClosedValue
	structRoleOpenRequest
	structRoleExternalProjection
	structRoleWireProjection
)

func TestDataFlowStructInventoryRatchet(t *testing.T) {
	t.Parallel()

	got := collectCurrencyStructs(t)
	want := []structInventoryEntry{
		{name: "Amount", role: structRoleClosedValue},
		{name: "FirestoreAmount", role: structRoleExternalProjection},
		{name: "HumanizePolicy", role: structRoleOpenRequest},
		{name: "Humanized", role: structRoleClosedValue},
		{name: "PostgreSQLAmount", role: structRoleExternalProjection},
		{name: "amountJSON", role: structRoleWireProjection},
		{name: "canonicalMinorUnits", role: structRoleWireProjection},
	}
	if len(got) != len(want) {
		t.Fatalf("production struct count = %d, want %d; got %v", len(got), len(want), got)
	}
	for index, entry := range want {
		if entry.role == structRoleUnknown || got[index] != entry.name {
			t.Fatalf("production struct inventory row %d = (%q,%d), want (%q,nonzero role)", index, got[index], entry.role, entry.name)
		}
	}
}

func TestRetiredCoreMoneySurfaceCannotReenterRatchet(t *testing.T) {
	t.Parallel()

	retired := []string{"ErrFmtMoneyPennies", "MoneyPennies", "NewMoneyPennies"}
	if got := declaredCoreIdentifiers(t, retired); len(got) != 0 {
		t.Fatalf("retired core money declarations re-entered production: %s", strings.Join(got, ", "))
	}
}

type structInventoryEntry struct {
	name string
	role structRole
}

func collectCurrencyPublicAPI(t *testing.T) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	files := parseCurrencyProductionFiles(t, fileSet)
	var api []string
	for fileName, file := range files {
		for _, declaration := range file.Decls {
			api = append(api, currencyExportedDeclarations(t, fileSet, fileName, declaration)...)
		}
	}
	sort.Strings(api)
	return api
}

func currencyExportedDeclarations(t *testing.T, fileSet *token.FileSet, fileName string, declaration ast.Decl) []string {
	t.Helper()

	switch value := declaration.(type) {
	case *ast.FuncDecl:
		if !value.Name.IsExported() {
			return nil
		}
		if value.Recv == nil {
			return []string{"func:" + value.Name.Name + "@" + fileName}
		}
		receiver := receiverName(value.Recv.List[0].Type)
		if !ast.IsExported(receiver) {
			return nil
		}
		return []string{"method:" + receiver + "." + value.Name.Name}
	case *ast.GenDecl:
		return currencyExportedGeneral(t, fileSet, value)
	default:
		return nil
	}
}

func currencyExportedGeneral(t *testing.T, fileSet *token.FileSet, declaration *ast.GenDecl) []string {
	t.Helper()

	var api []string
	for _, specification := range declaration.Specs {
		switch value := specification.(type) {
		case *ast.TypeSpec:
			if !value.Name.IsExported() {
				continue
			}
			api = append(api, "type:"+value.Name.Name)
			if structure, ok := value.Type.(*ast.StructType); ok {
				api = append(api, exportedStructFields(t, fileSet, value.Name.Name, structure)...)
			}
		case *ast.ValueSpec:
			for _, name := range value.Names {
				if name.IsExported() {
					api = append(api, "const:"+name.Name)
				}
			}
		}
	}
	return api
}

func exportedStructFields(t *testing.T, fileSet *token.FileSet, typeName string, structure *ast.StructType) []string {
	t.Helper()

	var fields []string
	for _, field := range structure.Fields.List {
		typeNameText := formatExpression(t, fileSet, field.Type)
		for _, name := range field.Names {
			if name.IsExported() {
				fields = append(fields, "field:"+typeName+"."+name.Name+":"+typeNameText)
			}
		}
	}
	return fields
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
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}

func collectCurrencyProductionImports(t *testing.T) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	files := parseCurrencyProductionFiles(t, fileSet)
	var imports []string
	for _, file := range files {
		for _, specification := range file.Imports {
			path, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(imports, path) {
				imports = append(imports, path)
			}
		}
	}
	sort.Strings(imports)
	return imports
}

func collectCurrencyStructs(t *testing.T) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	files := parseCurrencyProductionFiles(t, fileSet)
	var structures []string
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range general.Specs {
				named, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := named.Type.(*ast.StructType); ok {
					structures = append(structures, named.Name.Name)
				}
			}
		}
	}
	sort.Strings(structures)
	return structures
}

func parseCurrencyProductionFiles(t *testing.T, fileSet *token.FileSet) map[string]*ast.File {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string]*ast.File)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, entry.Name(), nil, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		files[entry.Name()] = file
	}
	return files
}

func declaredCoreIdentifiers(t *testing.T, retired []string) []string {
	t.Helper()

	fileSet := token.NewFileSet()
	entries, err := os.ReadDir("../core")
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, filepath.Join("../core", entry.Name()), nil, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && slices.Contains(retired, identifier.Name) {
				found = append(found, identifier.Name)
			}
			return true
		})
	}
	sort.Strings(found)
	return found
}
