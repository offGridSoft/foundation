package core

import "testing"

func TestFoundationModulePathContractsTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "module generation", got: FoundationModulePath, want: "github.com/offGridSoft/foundation/v" + ContractYear},
		{name: "core package", got: FoundationCorePackagePath, want: FoundationModulePath + "/core"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("contract = %q, want %q", tc.got, tc.want)
			}
		})
	}
}
