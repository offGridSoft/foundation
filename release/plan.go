package release

import (
	"crypto/sha256"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var _ core.CanonicalBody = ReleasePlan{}

const (
	ReleasePlanValidateAllocationBudget     float64 = 12
	ReleasePlanBuildRequestAllocationBudget float64 = 34
)

type ReleasePreflightInput struct {
	Toolchain       BuildToolchain
	Evidence        ReleaseGateEvidence
	HeadCommit      core.BuildCommit
	RequestedCommit core.BuildCommit
	Seed            GarbleSeed
	VulnDB          VulnDBSnapshot
	TreeState       TreeState
}

func Preflight(input ReleasePreflightInput) error {
	if err := validatePreflightIdentity(input); err != nil {
		return err
	}
	if err := validatePreflightFacts(input); err != nil {
		return err
	}
	return validateRequiredReleaseSeed(input.Seed, ErrFmtPreflight)
}

func validatePreflightIdentity(input ReleasePreflightInput) error {
	if err := input.RequestedCommit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtPreflight, err)
	}
	if err := input.HeadCommit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtPreflight, err)
	}
	if input.TreeState != TreeStateClean {
		return fmt.Errorf(ErrFmtPreflight, core.ErrReleaseContract)
	}
	if input.HeadCommit.String() != input.RequestedCommit.String() {
		return fmt.Errorf(ErrFmtPreflight, core.ErrReleaseContract)
	}
	return nil
}

func validatePreflightFacts(input ReleasePreflightInput) error {
	if err := input.Toolchain.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtPreflight, err)
	}
	if err := input.VulnDB.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtPreflight, err)
	}
	if err := input.Evidence.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtPreflight, err)
	}
	return nil
}

// Field order is storage-only; MarshalJSON owns the release-plan wire order.
type ReleasePlan struct {
	Toolchain BuildToolchain      `json:"toolchain"`
	Evidence  ReleaseGateEvidence `json:"evidence"`
	Commit    core.BuildCommit    `json:"commit"`
	Version   core.ProductVersion `json:"version"`
	Seed      GarbleSeed          `json:"seed"`
	SeedRef   GarbleSeedRef       `json:"seed_ref"`
	Date      ReleaseDate         `json:"date"`
	ReleaseID ReleaseID           `json:"release_id"`
	Layout    ReleaseRootLayout   `json:"layout"`
	VulnDB    VulnDBSnapshot      `json:"vuln_db"`
	Tools     []ToolProvenance    `json:"tools"`
	Spec      ProductReleaseSpec  `json:"spec"`
	ToolCount uint32              `json:"tool_count"`
	Schema    core.SchemaID       `json:"schema"`
	Product   core.Product        `json:"product"`
}

func (p ReleasePlan) Validate() error {
	if err := validateReleasePlanStructure(p); err != nil {
		return err
	}
	if releasePlanRawStringBytes(p) <= ReleasePlanFastRawByteMaximum {
		return nil
	}
	_, err := appendBoundedReleasePlanJSON(make([]byte, 0, ReleasePlanMaximumBytes), p)
	return err
}

func releasePlanRawStringBytes(p ReleasePlan) int64 {
	total := int64(len(p.Version.String()) + len(p.ReleaseID.String()) + len(p.Date.String()) +
		len(p.Commit.String()) + len(p.Seed.value) + len(p.SeedRef.String()))
	total += releaseRootRawStringBytes(p.Layout)
	total += releaseSpecRawStringBytes(p.Spec)
	total += int64(len(p.Toolchain.GoVersion.String()) + len(p.Toolchain.GarbleVersion.String()) +
		len(p.VulnDB.DBVersion.String()) + len(p.Evidence.FastGateRef.String()) + len(p.Evidence.FinalEvidenceRef.String()))
	for _, tool := range p.Tools {
		total += int64(len(tool.Module.String()) + len(tool.Version.String()) + len(tool.GoSum.String()))
	}
	return total
}

func releaseRootRawStringBytes(layout ReleaseRootLayout) int64 {
	return int64(len(layout.Version.String()) + len(layout.Date.String()) + len(layout.ReleaseID.String()) + len(layout.Commit.String()) +
		len(layout.Root.String()) + len(layout.Private.String()) + len(layout.Public.String()) +
		len(layout.Platforms.String()) + len(layout.Receipts.String()) + len(layout.Manifests.String()) +
		len(layout.Dogfood.String()))
}

func releaseSpecRawStringBytes(spec ProductReleaseSpec) int64 {
	total := int64(len(spec.Version.String()) + len(spec.Policy.CommitStamp.Symbol.String()) +
		len(spec.Policy.CommitStamp.Commit.String()))
	for _, command := range spec.Commands {
		total += int64(len(command.Name.String()) + len(command.ImportPath.String()))
	}
	for _, tag := range spec.Policy.Tags {
		total += int64(len(tag.String()))
	}
	return total
}

func validateReleasePlanStructure(p ReleasePlan) error {
	if p.Schema != core.SchemaReleasePlan {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	if err := validateReleasePlanIdentity(p); err != nil {
		return err
	}
	if err := validateReleasePlanFacts(p); err != nil {
		return err
	}
	return validateReleasePlanTools(p)
}

func (p ReleasePlan) GarbleBuildRequests() ([]ReleaseGarbleBuildRequest, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p.Spec.GarbleBuildRequests(p.Seed, p.Layout)
}

func (p ReleasePlan) Canonical(dst []byte) ([]byte, error) {
	if err := validateReleasePlanStructure(p); err != nil {
		return nil, err
	}
	return appendBoundedReleasePlanJSON(dst, p)
}

func (p ReleasePlan) SigningSchema() core.SchemaID {
	return p.Schema
}

func (p ReleasePlan) MarshalJSON() ([]byte, error) {
	return p.Canonical(nil)
}

func appendBoundedReleasePlanJSON(dst []byte, p ReleasePlan) ([]byte, error) {
	start := len(dst)
	encoded, err := appendReleasePlanJSON(dst, p)
	if err != nil {
		return nil, err
	}
	if int64(len(encoded)-start) > ReleasePlanMaximumBytes {
		return nil, fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	return encoded, nil
}

func appendReleasePlanJSON(dst []byte, p ReleasePlan) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, p.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, p.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, p.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, p.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDate, p.Date)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, p.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLayout, p.Layout)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSpec, p.Spec)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldToolchain, p.Toolchain)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVulnDB, p.VulnDB)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldEvidence, p.Evidence)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTools, p.Tools)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldToolCount, p.ToolCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSeedRef, p.SeedRef)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSeed, p.Seed)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateReleasePlanIdentity(p ReleasePlan) error {
	if err := validatePlanScalars(p); err != nil {
		return err
	}
	if err := p.Layout.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Spec.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return validatePlanCrossIdentity(p)
}

func validatePlanScalars(p ReleasePlan) error {
	if err := p.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Date.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := ValidateReleaseIDIdentity(p.ReleaseID, p.Product, p.Version, p.Commit); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return validateRequiredReleaseSeed(p.Seed, ErrFmtReleasePlan)
}

func validatePlanCrossIdentity(p ReleasePlan) error {
	if p.Layout.Product != p.Product || p.Spec.Product != p.Product {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	if p.Layout.Version.String() != p.Version.String() || p.Spec.Version.String() != p.Version.String() {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	if p.Layout.ReleaseID.String() != p.ReleaseID.String() || p.Layout.Date.String() != p.Date.String() {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	if p.Layout.Commit.String() != p.Commit.String() {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	if !policyCommitMatchesPlan(p.Spec.Policy.CommitStamp, p.Commit) {
		return fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	return nil
}

func validateReleasePlanFacts(p ReleasePlan) error {
	if err := p.SeedRef.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Toolchain.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.VulnDB.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	if err := p.Evidence.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return nil
}

func validateReleasePlanTools(p ReleasePlan) error {
	if err := (ToolProvenanceSet{
		Tools:     p.Tools,
		ToolCount: p.ToolCount,
	}.Validate()); err != nil {
		return wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return nil
}

func validateRequiredReleaseSeed(seed GarbleSeed, errFmt string) error {
	if err := seed.Validate(); err != nil {
		return wrapReleaseContract(errFmt, err)
	}
	if seed.IsRandom() {
		return fmt.Errorf(errFmt, core.ErrReleaseContract)
	}
	return nil
}

func policyCommitMatchesPlan(stamp BuildCommitStamp, commit core.BuildCommit) bool {
	return stamp.IsZero() || stamp.Commit.String() == commit.String()
}

// Field order is storage-only; Validate owns deploy identity and target rules.
type DeployPlan struct {
	Version        core.ProductVersion `json:"version"`
	ReleaseID      ReleaseID           `json:"release_id"`
	ManifestSHA256 core.SHA256Hex      `json:"manifest_sha256"`
	AttemptID      UploadAttemptID     `json:"upload_attempt_id"`
	Layout         ReleaseRootLayout   `json:"layout"`
	Targets        []UploadTarget      `json:"targets"`
	Manifest       Manifest            `json:"manifest"`
	TargetCount    uint32              `json:"target_count"`
	Schema         core.SchemaID       `json:"schema"`
	Product        core.Product        `json:"product"`
}

func (p DeployPlan) Validate() error {
	if p.Schema != core.SchemaReleaseDeployPlan {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	if err := validateDeployPlanIdentity(p); err != nil {
		return err
	}
	return validateDeployTargets(p)
}

func (p DeployPlan) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return appendDeployPlanJSON(nil, p)
}

func (p DeployPlan) Canonical(dst []byte) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return appendDeployPlanJSON(dst, p)
}

func (p DeployPlan) SigningSchema() core.SchemaID {
	return p.Schema
}

func appendDeployPlanJSON(dst []byte, p DeployPlan) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, p.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldProduct, p.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, p.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, p.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifestSHA256, p.ManifestSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUploadAttemptID, p.AttemptID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldLayout, p.Layout)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTargets, p.Targets)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTargetCount, p.TargetCount)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldManifest, p.Manifest)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func validateDeployPlanIdentity(p DeployPlan) error {
	if err := p.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.Manifest.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.ManifestSHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.AttemptID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if err := p.Layout.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	return validateDeployPlanCrossIdentity(p)
}

func validateDeployPlanCrossIdentity(p DeployPlan) error {
	if p.Manifest.Product != p.Product || p.Layout.Product != p.Product {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	if p.Manifest.Version.String() != p.Version.String() || p.Layout.Version.String() != p.Version.String() {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	if p.Manifest.ReleaseID.String() != p.ReleaseID.String() || p.Layout.ReleaseID.String() != p.ReleaseID.String() {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	if p.Manifest.Date.String() != p.Layout.Date.String() {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	canonical, err := p.Manifest.Canonical(nil)
	if err != nil {
		return wrapReleaseContract(ErrFmtDeployPlan, err)
	}
	if core.NewSHA256Hex(sha256.Sum256(canonical)) != p.ManifestSHA256 {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	return nil
}

func validateDeployTargets(plan DeployPlan) error {
	if err := (core.CollectionCardinality{
		Length:          len(plan.Targets),
		DeclaredCount:   plan.TargetCount,
		Minimum:         1,
		Maximum:         core.CollectionMaximumDefault,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	for index, target := range plan.Targets {
		if err := target.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtDeployPlan, err)
		}
		if target.AttemptID != plan.AttemptID {
			return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
		}
		artifact, found := manifestArtifactByName(plan.Manifest, target.Artifact)
		if !found {
			return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
		}
		if err := validateUploadBinding(UploadBindingInput{
			Product: plan.Product, ReleaseID: plan.ReleaseID, ManifestSHA256: plan.ManifestSHA256,
			Artifact: artifact.Name, ArtifactSHA256: artifact.SHA256, ArtifactSize: artifact.Size,
			Provider: target.Provider, Bucket: target.Bucket, Object: target.Object, AttemptID: target.AttemptID,
		}, target.Binding, ErrFmtDeployPlan); err != nil {
			return err
		}
		if index > 0 && deployTargetKey(plan.Targets[index-1]) >= deployTargetKey(target) {
			return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
		}
	}
	return validateDeployTargetCoverage(plan.Targets, plan.Manifest)
}

func manifestArtifactByName(manifest Manifest, name ArtifactName) (Artifact, bool) {
	for _, artifact := range manifest.Artifacts {
		if artifact.Name == name {
			return artifact, true
		}
	}
	return Artifact{}, false
}

func deployTargetKey(target UploadTarget) string {
	return target.Provider.String() + "/" + target.Artifact.String()
}

func validateDeployTargetCoverage(targets []UploadTarget, manifest Manifest) error {
	for _, target := range targets {
		if !manifestHasUploadTarget(manifest, target) {
			return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
		}
	}
	for _, target := range targets {
		if !providerTargetsEveryManifestArtifact(targets, target.Provider, manifest) {
			return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
		}
	}
	return nil
}

func manifestHasUploadTarget(manifest Manifest, target UploadTarget) bool {
	for _, artifact := range manifest.Artifacts {
		if artifact.Name == target.Artifact {
			return true
		}
	}
	return false
}

func providerTargetsEveryManifestArtifact(targets []UploadTarget, provider core.StorageProvider, manifest Manifest) bool {
	for _, artifact := range manifest.Artifacts {
		if !hasProviderUploadTarget(targets, provider, artifact.Name) {
			return false
		}
	}
	return true
}

func hasProviderUploadTarget(targets []UploadTarget, provider core.StorageProvider, artifact ArtifactName) bool {
	for _, target := range targets {
		if target.Provider == provider && target.Artifact == artifact {
			return true
		}
	}
	return false
}
