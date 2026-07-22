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
		{name: "custody package", got: FoundationCustodyPackagePath, want: FoundationModulePath + "/custody"},
		{name: "durability package", got: FoundationDurabilityPackagePath, want: FoundationModulePath + "/durability"},
		{name: "durability test package", got: FoundationDurabilityTestPackagePath, want: FoundationModulePath + "/durabilitytest"},
		{name: "host resource package", got: FoundationHostResourcePackagePath, want: FoundationModulePath + "/hostresource"},
		{name: "license package", got: FoundationLicensePackagePath, want: FoundationModulePath + "/license"},
		{name: "release package", got: FoundationReleasePackagePath, want: FoundationModulePath + "/release"},
		{name: "shutdown package", got: FoundationShutdownPackagePath, want: FoundationModulePath + "/shutdown"},
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
