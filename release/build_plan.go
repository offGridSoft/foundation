package release

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	GarbleSeedBytes         = 8
	GarbleSeedMaxInputBytes = 4096
	GarbleSeedRandom        = "random"
	GarbleSeedFlagPrefix    = "-seed="
	GarbleArgLiterals       = "-literals"
	GarbleArgTiny           = "-tiny"
	GoArgBuild              = "build"
	GoArgTrimPath           = "-trimpath"
	GoArgBuildVCS           = "-buildvcs=true"
	GoBuildOutputFlag       = "-o"
	GoBuildTagsPrefix       = "-tags="
	GoBuildLDFlagsPrefix    = "-ldflags="
	LDFlagStripSymbols      = "-s"
	LDFlagStripDebug        = "-w"
	LDFlagClearBuildID      = "-buildid="
	LDFlagSetVariable       = "-X"
	WindowsExecutableSuffix = ".exe"
	DistDirName             = "dist"
	BuildImportPathMaxRunes = 256
	BuildTagMaxRunes        = 64
	LinkerSymbolMaxRunes    = 256
)

var defaultReleaseBuildPlatforms = [...]core.Platform{
	core.PlatformDarwinARM64,
	core.PlatformLinuxAMD64,
	core.PlatformLinuxARM64,
	core.PlatformWindowsAMD64,
}

type GarbleSeed struct {
	value string
}

func ParseGarbleSeed(value string) (GarbleSeed, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == GarbleSeedRandom {
		return GarbleSeed{value: GarbleSeedRandom}, nil
	}
	return parseConcreteGarbleSeed(trimmed)
}

func ParseRequiredGarbleSeed(value string) (GarbleSeed, error) {
	seed, err := ParseGarbleSeed(value)
	if err != nil {
		return GarbleSeed{}, err
	}
	if seed.IsRandom() {
		return GarbleSeed{}, fmt.Errorf(ErrFmtGarbleSeed, core.ErrReleaseContract)
	}
	return seed, nil
}

func parseConcreteGarbleSeed(value string) (GarbleSeed, error) {
	if len(value) > GarbleSeedMaxInputBytes {
		return GarbleSeed{}, fmt.Errorf(ErrFmtGarbleSeed, core.ErrReleaseContract)
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) < GarbleSeedBytes {
		return GarbleSeed{}, fmt.Errorf(ErrFmtGarbleSeed, core.ErrReleaseContract)
	}
	return GarbleSeed{value: base64.StdEncoding.EncodeToString(raw[:GarbleSeedBytes])}, nil
}

func (s GarbleSeed) String() string {
	return s.value
}

func (s GarbleSeed) IsRandom() bool {
	return s.value == GarbleSeedRandom
}

func (s GarbleSeed) Validate() error {
	_, err := ParseGarbleSeed(s.value)
	return err
}

func (s GarbleSeed) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.value)
}

func (s *GarbleSeed) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGarbleSeed, core.ErrReleaseContract)
	}
	parsed, err := ParseGarbleSeed(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type BuildImportPath struct {
	value string
}

func ParseBuildImportPath(value string) (BuildImportPath, error) {
	if err := validateBuildToken(value, BuildImportPathMaxRunes); err != nil {
		return BuildImportPath{}, fmt.Errorf(ErrFmtBuildCommand, core.ErrReleaseContract)
	}
	if !strings.HasPrefix(value, "./cmd/") {
		return BuildImportPath{}, fmt.Errorf(ErrFmtBuildCommand, core.ErrReleaseContract)
	}
	return BuildImportPath{value: value}, nil
}

func (p BuildImportPath) String() string {
	return p.value
}

func (p BuildImportPath) Validate() error {
	_, err := ParseBuildImportPath(p.value)
	return err
}

type BuildOutputPath struct {
	value string
}

func ParseBuildOutputPath(value string) (BuildOutputPath, error) {
	if err := core.ValidatePathToken(value, core.PathTokenMaxRunes); err != nil {
		return BuildOutputPath{}, fmt.Errorf(ErrFmtBuildOutput, core.ErrReleaseContract)
	}
	return BuildOutputPath{value: value}, nil
}

func (p BuildOutputPath) String() string {
	return p.value
}

func (p BuildOutputPath) Validate() error {
	_, err := ParseBuildOutputPath(p.value)
	return err
}

type BuildTag struct {
	value string
}

func ParseBuildTag(value string) (BuildTag, error) {
	if err := core.ValidateFileNameToken(value, BuildTagMaxRunes); err != nil {
		return BuildTag{}, fmt.Errorf(ErrFmtBuildTag, core.ErrReleaseContract)
	}
	return BuildTag{value: value}, nil
}

func (t BuildTag) String() string {
	return t.value
}

func (t BuildTag) Validate() error {
	_, err := ParseBuildTag(t.value)
	return err
}

type LinkerSymbol struct {
	value string
}

func ParseLinkerSymbol(value string) (LinkerSymbol, error) {
	if err := validateBuildToken(value, LinkerSymbolMaxRunes); err != nil {
		return LinkerSymbol{}, fmt.Errorf(ErrFmtLinkerSymbol, core.ErrReleaseContract)
	}
	if !strings.Contains(value, ".") {
		return LinkerSymbol{}, fmt.Errorf(ErrFmtLinkerSymbol, core.ErrReleaseContract)
	}
	return LinkerSymbol{value: value}, nil
}

func (s LinkerSymbol) String() string {
	return s.value
}

func (s LinkerSymbol) Validate() error {
	_, err := ParseLinkerSymbol(s.value)
	return err
}

type BuildCommitStamp struct {
	Symbol LinkerSymbol     `json:"symbol"`
	Commit core.BuildCommit `json:"commit"`
}

func (s BuildCommitStamp) IsZero() bool {
	return s.Symbol.String() == "" && s.Commit.String() == ""
}

func (s BuildCommitStamp) Validate() error {
	if s.IsZero() {
		return nil
	}
	if err := s.Symbol.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildPolicy, err)
	}
	if err := s.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildPolicy, err)
	}
	return nil
}

type ReleaseBuildPolicy struct {
	CommitStamp  BuildCommitStamp `json:"commit_stamp"`
	Tags         []BuildTag       `json:"tags"`
	TagCount     uint32           `json:"tag_count"`
	BuildVCS     bool             `json:"build_vcs"`
	ClearBuildID bool             `json:"clear_build_id"`
	Strip        bool             `json:"strip"`
}

func (p ReleaseBuildPolicy) Validate() error {
	if !p.Strip {
		return fmt.Errorf(ErrFmtBuildPolicy, core.ErrReleaseContract)
	}
	if err := p.CommitStamp.Validate(); err != nil {
		return err
	}
	return validateBuildTags(p)
}

func validateBuildTags(p ReleaseBuildPolicy) error {
	if int(p.TagCount) != len(p.Tags) {
		return fmt.Errorf(ErrFmtBuildPolicy, core.ErrReleaseContract)
	}
	seen := core.NewUniqueStringSet(len(p.Tags))
	for _, tag := range p.Tags {
		if err := tag.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtBuildPolicy, err)
		}
		if err := seen.Add(tag.String()); err != nil {
			return wrapReleaseContract(ErrFmtBuildPolicy, err)
		}
	}
	return nil
}

type ReleaseCommand struct {
	Name       ArtifactName    `json:"name"`
	ImportPath BuildImportPath `json:"import_path"`
}

func (c ReleaseCommand) Validate() error {
	if err := c.Name.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildCommand, err)
	}
	if err := c.ImportPath.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildCommand, err)
	}
	return nil
}

func (c ReleaseCommand) BinaryName(platform core.Platform) (ArtifactName, error) {
	if err := c.Validate(); err != nil {
		return ArtifactName{}, err
	}
	if platform == core.PlatformWindowsAMD64 || platform == core.PlatformWindowsARM64 {
		return ParseArtifactName(c.Name.String() + WindowsExecutableSuffix)
	}
	if err := platform.Validate(); err != nil {
		return ArtifactName{}, wrapReleaseContract(ErrFmtBuildCommand, err)
	}
	return c.Name, nil
}

type ReleaseGarbleBuildRequest struct {
	Command  ReleaseCommand     `json:"command"`
	Seed     GarbleSeed         `json:"seed"`
	Output   BuildOutputPath    `json:"output"`
	Policy   ReleaseBuildPolicy `json:"policy"`
	Platform core.Platform      `json:"platform"`
}

func (r ReleaseGarbleBuildRequest) Validate() error {
	if err := r.Seed.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildRequest, err)
	}
	if r.Seed.IsRandom() {
		return fmt.Errorf(ErrFmtBuildRequest, core.ErrReleaseContract)
	}
	if err := validateReleasePlatform(r.Platform, ErrFmtBuildRequest); err != nil {
		return err
	}
	if err := r.Command.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildRequest, err)
	}
	if err := r.Output.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildRequest, err)
	}
	if err := r.Policy.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtBuildRequest, err)
	}
	return nil
}

func GarbleBuildArgs(r ReleaseGarbleBuildRequest) ([]string, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	args := []string{
		GarbleSeedFlagPrefix + r.Seed.String(),
		GarbleArgLiterals,
		GarbleArgTiny,
		GoArgBuild,
		GoArgTrimPath,
	}
	args = appendBuildPolicyArgs(args, r.Policy)
	args = append(args, GoBuildOutputFlag, r.Output.String(), r.Command.ImportPath.String())
	return args, nil
}

func appendBuildPolicyArgs(args []string, policy ReleaseBuildPolicy) []string {
	if policy.BuildVCS {
		args = append(args, GoArgBuildVCS)
	}
	if policy.TagCount > 0 {
		args = append(args, GoBuildTagsPrefix+joinBuildTags(policy.Tags))
	}
	ldflags := buildLDFlags(policy)
	if ldflags != "" {
		args = append(args, GoBuildLDFlagsPrefix+ldflags)
	}
	return args
}

func buildLDFlags(policy ReleaseBuildPolicy) string {
	parts := []string{LDFlagStripSymbols, LDFlagStripDebug}
	if policy.ClearBuildID {
		parts = append(parts, LDFlagClearBuildID)
	}
	if !policy.CommitStamp.IsZero() {
		assignment := policy.CommitStamp.Symbol.String() + "=" + policy.CommitStamp.Commit.String()
		parts = append(parts, LDFlagSetVariable, assignment)
	}
	return strings.Join(parts, " ")
}

func joinBuildTags(tags []BuildTag) string {
	values := make([]string, 0, len(tags))
	for _, tag := range tags {
		values = append(values, tag.String())
	}
	return strings.Join(values, ",")
}

type ProductReleaseSpec struct {
	Version       core.ProductVersion `json:"version"`
	Commands      []ReleaseCommand    `json:"commands"`
	Platforms     []core.Platform     `json:"platforms"`
	Policy        ReleaseBuildPolicy  `json:"policy"`
	CommandCount  uint32              `json:"command_count"`
	PlatformCount uint32              `json:"platform_count"`
	Product       core.Product        `json:"product"`
}

func (s ProductReleaseSpec) Validate() error {
	if err := s.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseSpec, err)
	}
	if err := s.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseSpec, err)
	}
	if err := s.Policy.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleaseSpec, err)
	}
	if err := validateSpecCommands(s); err != nil {
		return err
	}
	return validateSpecPlatforms(s)
}

func (s ProductReleaseSpec) GarbleBuildRequests(seed GarbleSeed, outputRoot ArtifactName) ([]ReleaseGarbleBuildRequest, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if err := seed.Validate(); err != nil || seed.IsRandom() {
		return nil, fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	return buildRequestsForSpec(s, seed, outputRoot)
}

func buildRequestsForSpec(s ProductReleaseSpec, seed GarbleSeed, outputRoot ArtifactName) ([]ReleaseGarbleBuildRequest, error) {
	requests := make([]ReleaseGarbleBuildRequest, 0, len(s.Commands)*len(s.Platforms))
	for _, platform := range s.Platforms {
		for _, command := range s.Commands {
			request, err := buildRequestForCommand(buildRequestInput{
				Spec:       s,
				Seed:       seed,
				OutputRoot: outputRoot,
				Platform:   platform,
				Command:    command,
			})
			if err != nil {
				return nil, err
			}
			requests = append(requests, request)
		}
	}
	return requests, nil
}

type buildRequestInput struct {
	Command    ReleaseCommand
	Seed       GarbleSeed
	OutputRoot ArtifactName
	Spec       ProductReleaseSpec
	Platform   core.Platform
}

func buildRequestForCommand(input buildRequestInput) (ReleaseGarbleBuildRequest, error) {
	binary, err := input.Command.BinaryName(input.Platform)
	if err != nil {
		return ReleaseGarbleBuildRequest{}, err
	}
	output, err := ParseBuildOutputPath(strings.Join([]string{input.OutputRoot.String(), input.Platform.String(), binary.String()}, "/"))
	if err != nil {
		return ReleaseGarbleBuildRequest{}, err
	}
	return ReleaseGarbleBuildRequest{
		Seed:     input.Seed,
		Command:  input.Command,
		Output:   output,
		Platform: input.Platform,
		Policy:   input.Spec.Policy,
	}, nil
}

func validateSpecCommands(s ProductReleaseSpec) error {
	if s.CommandCount == 0 || int(s.CommandCount) != len(s.Commands) {
		return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	names := core.NewUniqueStringSet(len(s.Commands))
	imports := core.NewUniqueStringSet(len(s.Commands))
	for _, command := range s.Commands {
		if err := command.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtReleaseSpec, err)
		}
		if err := names.Add(command.Name.String()); err != nil {
			return wrapReleaseContract(ErrFmtReleaseSpec, err)
		}
		if err := imports.Add(command.ImportPath.String()); err != nil {
			return wrapReleaseContract(ErrFmtReleaseSpec, err)
		}
	}
	return nil
}

func validateSpecPlatforms(s ProductReleaseSpec) error {
	if s.PlatformCount == 0 || int(s.PlatformCount) != len(s.Platforms) {
		return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	seen := core.NewUniqueStringSet(len(s.Platforms))
	for _, platform := range s.Platforms {
		if err := validateReleasePlatform(platform, ErrFmtReleaseSpec); err != nil {
			return err
		}
		if err := seen.Add(platform.String()); err != nil {
			return wrapReleaseContract(ErrFmtReleaseSpec, err)
		}
	}
	return nil
}

func BuildPlatforms() []core.Platform {
	platforms := make([]core.Platform, len(defaultReleaseBuildPlatforms))
	copy(platforms, defaultReleaseBuildPlatforms[:])
	return platforms
}

func NewReleaseCommand(name, importPath string) (ReleaseCommand, error) {
	artifactName, err := ParseArtifactName(name)
	if err != nil {
		return ReleaseCommand{}, err
	}
	parsedPath, err := ParseBuildImportPath(importPath)
	if err != nil {
		return ReleaseCommand{}, err
	}
	return ReleaseCommand{Name: artifactName, ImportPath: parsedPath}, nil
}

func DefaultOutputRoot() (ArtifactName, error) {
	return ParseArtifactName(DistDirName)
}

func validateBuildToken(value string, maxRunes int) error {
	if err := core.ValidateOpaqueToken(value, maxRunes); err != nil {
		return err
	}
	if strings.ContainsAny(value, " \\") {
		return core.ErrFoundationContract
	}
	return nil
}
