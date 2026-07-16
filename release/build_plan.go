package release

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	GarbleSeedBytes                         = 8
	GarbleSeedMaxInputBytes                 = 4096
	GarbleSeedRandom                        = "random"
	GarbleSeedFlagPrefix                    = "-seed="
	GarbleArgLiterals                       = "-literals"
	GarbleArgTiny                           = "-tiny"
	GoArgBuild                              = "build"
	GoArgTrimPath                           = "-trimpath"
	GoArgBuildModeExecutable                = "-buildmode=exe"
	GoArgBuildVCS                           = "-buildvcs=true"
	GoBuildOutputFlag                       = "-o"
	GoBuildTagsPrefix                       = "-tags="
	GoBuildLDFlagsPrefix                    = "-ldflags="
	LDFlagStripSymbols                      = "-s"
	LDFlagStripDebug                        = "-w"
	LDFlagClearBuildID                      = "-buildid="
	LDFlagSetVariable                       = "-X"
	BuildCommandDirPrefix                   = "./cmd/"
	WindowsExecutableSuffix                 = ".exe"
	DistDirName                             = "dist"
	BuildImportPathMaxRunes                 = 256
	BuildTagMaxRunes                        = 64
	LinkerSymbolMaxRunes                    = 256
	DefaultReleaseBuildPlatformCount        = 4
	ReleaseBuildTagMaximum           uint32 = 64
	ReleaseCommandMaximum            uint32 = 64
)

func defaultReleaseBuildPlatforms() [DefaultReleaseBuildPlatformCount]core.Platform {
	return [...]core.Platform{
		core.PlatformDarwinARM64,
		core.PlatformLinuxAMD64,
		core.PlatformLinuxARM64,
		core.PlatformWindowsAMD64,
	}
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
	if err != nil || len(raw) != GarbleSeedBytes {
		return GarbleSeed{}, fmt.Errorf(ErrFmtGarbleSeed, core.ErrReleaseContract)
	}
	return GarbleSeed{value: base64.StdEncoding.EncodeToString(raw)}, nil
}

func (s GarbleSeed) String() string {
	if s.IsRandom() {
		return GarbleSeedRandom
	}
	parsed, err := ParseGarbleSeed(s.value)
	if err != nil {
		return s.value
	}
	return parsed.value
}

func (s GarbleSeed) IsRandom() bool {
	trimmed := strings.TrimSpace(s.value)
	return trimmed == "" || trimmed == GarbleSeedRandom
}

func (s GarbleSeed) Validate() error {
	_, err := ParseGarbleSeed(s.value)
	return err
}

func (s GarbleSeed) MarshalJSON() ([]byte, error) {
	parsed, err := ParseGarbleSeed(s.value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(parsed.value)
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
	if !strings.HasPrefix(value, BuildCommandDirPrefix) {
		return BuildImportPath{}, fmt.Errorf(ErrFmtBuildCommand, core.ErrReleaseContract)
	}
	if err := validateBuildImportPathSegments(value); err != nil {
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

func (p BuildImportPath) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.value)
}

func (p *BuildImportPath) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBuildCommand, core.ErrReleaseContract)
	}
	parsed, err := ParseBuildImportPath(value)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type BuildOutputPath struct {
	value string
}

func ParseBuildOutputPath(value string) (BuildOutputPath, error) {
	if err := validateLocalOutputPath(value); err != nil {
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

func (p BuildOutputPath) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.value)
}

func (p *BuildOutputPath) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBuildOutput, core.ErrReleaseContract)
	}
	parsed, err := ParseBuildOutputPath(value)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type BuildTag struct {
	value string
}

func ParseBuildTag(value string) (BuildTag, error) {
	if err := validateBuildTagToken(value); err != nil {
		return BuildTag{}, fmt.Errorf(ErrFmtBuildTag, core.ErrReleaseContract)
	}
	return BuildTag{value: value}, nil
}

func validateBuildTagToken(value string) error {
	if err := core.ValidateFileNameToken(value, BuildTagMaxRunes); err != nil {
		return err
	}
	if strings.Contains(value, ",") || strings.ContainsAny(value, " \t\n\r") {
		return core.ErrReleaseContract
	}
	return nil
}

func (t BuildTag) String() string {
	return t.value
}

func (t BuildTag) Validate() error {
	_, err := ParseBuildTag(t.value)
	return err
}

func (t BuildTag) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t.value)
}

func (t *BuildTag) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBuildTag, core.ErrReleaseContract)
	}
	parsed, err := ParseBuildTag(value)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
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
	if strings.Contains(value, "=") {
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

func (s LinkerSymbol) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.value)
}

func (s *LinkerSymbol) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtLinkerSymbol, core.ErrReleaseContract)
	}
	parsed, err := ParseLinkerSymbol(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
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

func (s BuildCommitStamp) MarshalJSON() ([]byte, error) {
	if s.IsZero() {
		return []byte(core.JSONLiteralNull), nil
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldSymbol, s.Symbol)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, s.Commit)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
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

func (p ReleaseBuildPolicy) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldCommitStamp, p.CommitStamp)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTags, p.Tags)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTagCount, p.TagCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldBuildVCS, p.BuildVCS)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldClearBuildID, p.ClearBuildID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldStrip, p.Strip)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateBuildTags(p ReleaseBuildPolicy) error {
	if err := (core.CollectionCardinality{
		Length:          len(p.Tags),
		DeclaredCount:   p.TagCount,
		Maximum:         ReleaseBuildTagMaximum,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtBuildPolicy, core.ErrReleaseContract)
	}
	for index, tag := range p.Tags {
		if err := tag.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtBuildPolicy, err)
		}
		if slices.Contains(p.Tags[:index], tag) {
			return fmt.Errorf(ErrFmtBuildPolicy, core.ErrReleaseContract)
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

func (c ReleaseCommand) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldName, c.Name)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldImportPath, c.ImportPath)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
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

func (r ReleaseGarbleBuildRequest) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldCommand, r.Command)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSeed, r.Seed)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldOutput, r.Output)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPolicy, r.Policy)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatform, r.Platform)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
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
		GoArgBuildModeExecutable,
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

func (s ProductReleaseSpec) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldProduct, s.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, s.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommands, s.Commands)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommandCount, s.CommandCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatforms, s.Platforms)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPlatformCount, s.PlatformCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldPolicy, s.Policy)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (s ProductReleaseSpec) GarbleBuildRequests(seed GarbleSeed, layout ReleaseRootLayout) ([]ReleaseGarbleBuildRequest, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if err := seed.Validate(); err != nil || seed.IsRandom() {
		return nil, fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	if err := layout.Validate(); err != nil {
		return nil, wrapReleaseContract(ErrFmtReleaseSpec, err)
	}
	if !specMatchesLayout(s, layout) {
		return nil, fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	return buildRequestsForSpec(s, seed, layout)
}

func buildRequestsForSpec(s ProductReleaseSpec, seed GarbleSeed, layout ReleaseRootLayout) ([]ReleaseGarbleBuildRequest, error) {
	requests := make([]ReleaseGarbleBuildRequest, 0, len(s.Commands)*len(s.Platforms))
	for _, platform := range s.Platforms {
		for _, command := range s.Commands {
			request, err := buildRequestForCommand(buildRequestInput{
				Spec:     s,
				Seed:     seed,
				Layout:   layout,
				Platform: platform,
				Command:  command,
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
	Command  ReleaseCommand
	Seed     GarbleSeed
	Layout   ReleaseRootLayout
	Spec     ProductReleaseSpec
	Platform core.Platform
}

func buildRequestForCommand(input buildRequestInput) (ReleaseGarbleBuildRequest, error) {
	binary, err := input.Command.BinaryName(input.Platform)
	if err != nil {
		return ReleaseGarbleBuildRequest{}, err
	}
	output, err := releaseRootPath(input.Layout.Platforms.String(), input.Platform.String(), binary.String())
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

func specMatchesLayout(s ProductReleaseSpec, layout ReleaseRootLayout) bool {
	return s.Product == layout.Product && s.Version.String() == layout.Version.String()
}

func validateSpecCommands(s ProductReleaseSpec) error {
	if err := (core.CollectionCardinality{
		Length:          len(s.Commands),
		DeclaredCount:   s.CommandCount,
		Minimum:         1,
		Maximum:         ReleaseCommandMaximum,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	for index, command := range s.Commands {
		if err := command.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtReleaseSpec, err)
		}
		for _, prior := range s.Commands[:index] {
			if prior.Name == command.Name || prior.ImportPath == command.ImportPath {
				return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
			}
		}
	}
	return nil
}

func validateSpecPlatforms(s ProductReleaseSpec) error {
	if err := (core.CollectionCardinality{
		Length:          len(s.Platforms),
		DeclaredCount:   s.PlatformCount,
		Minimum:         1,
		Maximum:         core.PlatformMaximumDefault,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
	}
	for index, platform := range s.Platforms {
		if err := validateReleasePlatform(platform, ErrFmtReleaseSpec); err != nil {
			return err
		}
		if slices.Contains(s.Platforms[:index], platform) {
			return fmt.Errorf(ErrFmtReleaseSpec, core.ErrReleaseContract)
		}
	}
	return nil
}

func BuildPlatforms() []core.Platform {
	defaults := defaultReleaseBuildPlatforms()
	platforms := make([]core.Platform, len(defaults))
	copy(platforms, defaults[:])
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

func validateBuildToken(value string, maxRunes int) error {
	if err := core.ValidateOpaqueToken(value, maxRunes); err != nil {
		return err
	}
	if strings.ContainsAny(value, " \\") {
		return core.ErrFoundationContract
	}
	return nil
}

func validateLocalOutputPath(value string) error {
	if err := core.ValidateOpaqueToken(value, core.PathTokenMaxRunes); err != nil {
		return err
	}
	return validatePathSegments(value)
}

func validateBuildImportPathSegments(value string) error {
	relative, ok := strings.CutPrefix(value, BuildCommandDirPrefix)
	if !ok || relative == "" {
		return core.ErrFoundationContract
	}
	for segment := range strings.SplitSeq(relative, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return core.ErrFoundationContract
		}
	}
	return nil
}

func validatePathSegments(value string) error {
	clean := strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(clean, "/") || hasWindowsVolumePrefix(clean) {
		return core.ErrFoundationContract
	}
	for segment := range strings.SplitSeq(clean, "/") {
		if segment == "" {
			continue
		}
		if segment == "." || segment == ".." {
			return core.ErrFoundationContract
		}
	}
	return nil
}

func hasWindowsVolumePrefix(value string) bool {
	return len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && value[2] == '/'
}
