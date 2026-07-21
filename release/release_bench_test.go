package release

import (
	"fmt"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func BenchmarkReleasePlanValidate(b *testing.B) {
	plan := benchmarkReleasePlan(b, 2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := plan.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReleasePlanGarbleBuildRequests(b *testing.B) {
	plan := benchmarkReleasePlan(b, 2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		requests, err := plan.GarbleBuildRequests()
		if err != nil {
			b.Fatal(err)
		}
		if len(requests) != len(BuildPlatforms())*2 {
			b.Fatalf("request count = %d", len(requests))
		}
	}
}

func BenchmarkToolProvenanceSetValidate64(b *testing.B) {
	tools := benchmarkTools(b, int(MaxToolProvenanceItems))
	set := ToolProvenanceSet{Tools: tools, ToolCount: uint32(len(tools))}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := set.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkReleasePlan(b *testing.B, toolCount int) ReleasePlan {
	b.Helper()
	tools := benchmarkTools(b, toolCount)
	return ReleasePlan{
		Schema:    core.SchemaReleasePlan,
		Product:   core.ProductWitness,
		Version:   benchmarkVersion(b),
		ReleaseID: benchmarkReleaseID(b),
		Date:      benchmarkDate(b),
		Commit:    benchmarkCommit(b, "a"),
		Seed:      benchmarkSeed(b),
		SeedRef:   benchmarkSeedRef(b),
		Layout:    benchmarkLayout(b),
		Spec:      benchmarkSpec(b),
		Toolchain: BuildToolchain{
			GoVersion:     benchmarkToolVersion(b, "go1.25.0"),
			GarbleVersion: benchmarkToolVersion(b, "v0.15.0"),
		},
		Evidence: ReleaseGateEvidence{
			FastGateRef:      benchmarkEvidenceRef(b, "witness://release/fast/green"),
			FinalCertificate: benchmarkFinalCertificateEvidence(b),
		},
		VulnDB: VulnDBSnapshot{
			DBVersion:  benchmarkToolVersion(b, "2026-07-08T00:00:00Z"),
			SnapshotAt: core.UnixNanoTimeFromInt64(1782302400000000000),
		},
		Tools:     tools,
		ToolCount: uint32(len(tools)),
	}
}

func benchmarkFinalCertificateEvidence(b *testing.B) FinalCertificateEvidence {
	b.Helper()
	evidence, err := BuildCertifiedFinalCertificateEvidence(benchmarkSHA256(b, "f"), benchmarkCommit(b, "a"))
	if err != nil {
		b.Fatal(err)
	}
	return evidence
}

func benchmarkSpec(b *testing.B) ProductReleaseSpec {
	b.Helper()
	commands := []ReleaseCommand{
		benchmarkCommand(b, core.ProductTokenWitness, "./cmd/witness"),
		benchmarkCommand(b, "witness-sign", "./cmd/witness-sign"),
	}
	tags := []BuildTag{
		benchmarkBuildTag(b, "netgo"),
		benchmarkBuildTag(b, "osusergo"),
	}
	return ProductReleaseSpec{
		Product:       core.ProductWitness,
		Version:       benchmarkVersion(b),
		Commands:      commands,
		CommandCount:  uint32(len(commands)),
		Platforms:     BuildPlatforms(),
		PlatformCount: uint32(len(BuildPlatforms())),
		Policy: ReleaseBuildPolicy{
			BuildVCS:     true,
			ClearBuildID: true,
			Strip:        true,
			Tags:         tags,
			TagCount:     uint32(len(tags)),
			CommitStamp: BuildCommitStamp{
				Symbol: benchmarkLinkerSymbol(b, "github.com/offGridSoft/witness/internal/release.BuildCommit"),
				Commit: benchmarkCommit(b, "a"),
			},
		},
	}
}

func benchmarkTools(b *testing.B, count int) []ToolProvenance {
	b.Helper()
	tools := make([]ToolProvenance, count)
	for idx := range tools {
		module := fmt.Sprintf("github.com/offGridSoft/tool%02d", idx)
		tools[idx] = ToolProvenance{
			Module:  benchmarkToolModule(b, module),
			Version: benchmarkToolVersion(b, "v2026.0.0"),
			GoSum:   benchmarkGoSumHash(b, fmt.Sprintf("h1:%043d=", idx)),
		}
	}
	return tools
}

func benchmarkLayout(b *testing.B) ReleaseRootLayout {
	b.Helper()
	commit := benchmarkCommit(b, "a")
	id, err := BuildReleaseID(core.ProductWitness, benchmarkVersion(b), commit)
	if err != nil {
		b.Fatal(err)
	}
	layout, err := BuildReleaseRootLayout(ReleaseRootInput{
		Product:   core.ProductWitness,
		Version:   benchmarkVersion(b),
		Date:      benchmarkDate(b),
		ReleaseID: id,
		Commit:    commit,
	})
	if err != nil {
		b.Fatal(err)
	}
	return layout
}

func benchmarkCommand(b *testing.B, name, path string) ReleaseCommand {
	b.Helper()
	command, err := NewReleaseCommand(name, path)
	if err != nil {
		b.Fatal(err)
	}
	return command
}

func benchmarkVersion(b *testing.B) core.ProductVersion {
	b.Helper()
	version, err := core.ParseProductVersion(core.FoundationVersion2026)
	if err != nil {
		b.Fatal(err)
	}
	return version
}

func benchmarkReleaseID(b *testing.B) ReleaseID {
	b.Helper()
	id, err := BuildReleaseID(core.ProductWitness, benchmarkVersion(b), benchmarkCommit(b, "a"))
	if err != nil {
		b.Fatal(err)
	}
	return id
}

func benchmarkDate(b *testing.B) ReleaseDate {
	b.Helper()
	date, err := ParseReleaseDate("2026-07-08")
	if err != nil {
		b.Fatal(err)
	}
	return date
}

func benchmarkCommit(b *testing.B, digit string) core.BuildCommit {
	b.Helper()
	commit, err := core.ParseBuildCommit(strings.Repeat(digit, 40))
	if err != nil {
		b.Fatal(err)
	}
	return commit
}

func benchmarkSHA256(b *testing.B, digit string) core.SHA256Hex {
	b.Helper()
	digest, err := core.ParseSHA256Hex(strings.Repeat(digit, 64))
	if err != nil {
		b.Fatal(err)
	}
	return digest
}

func benchmarkSeed(b *testing.B) GarbleSeed {
	b.Helper()
	seed, err := ParseRequiredGarbleSeed("AQIDBAUGBwg")
	if err != nil {
		b.Fatal(err)
	}
	return seed
}

func benchmarkSeedRef(b *testing.B) GarbleSeedRef {
	b.Helper()
	ref, err := ParseGarbleSeedRef("release/2026/07/08/seed")
	if err != nil {
		b.Fatal(err)
	}
	return ref
}

func benchmarkEvidenceRef(b *testing.B, value string) EvidenceRef {
	b.Helper()
	ref, err := ParseEvidenceRef(value)
	if err != nil {
		b.Fatal(err)
	}
	return ref
}

func benchmarkToolVersion(b *testing.B, value string) ToolVersion {
	b.Helper()
	version, err := ParseToolVersion(value)
	if err != nil {
		b.Fatal(err)
	}
	return version
}

func benchmarkToolModule(b *testing.B, value string) ToolModule {
	b.Helper()
	module, err := ParseToolModule(value)
	if err != nil {
		b.Fatal(err)
	}
	return module
}

func benchmarkGoSumHash(b *testing.B, value string) GoSumHash {
	b.Helper()
	hash, err := ParseGoSumHash(value)
	if err != nil {
		b.Fatal(err)
	}
	return hash
}

func benchmarkBuildTag(b *testing.B, value string) BuildTag {
	b.Helper()
	tag, err := ParseBuildTag(value)
	if err != nil {
		b.Fatal(err)
	}
	return tag
}

func benchmarkLinkerSymbol(b *testing.B, value string) LinkerSymbol {
	b.Helper()
	symbol, err := ParseLinkerSymbol(value)
	if err != nil {
		b.Fatal(err)
	}
	return symbol
}
