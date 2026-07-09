package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
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
	Product   core.Product        `json:"product"`
}

func (p ReleasePlan) Validate() error {
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
	return core.AppendCanonicalJSON(dst, p)
}

func (p ReleasePlan) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldProduct, p.Product)
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
	Layout         ReleaseRootLayout   `json:"layout"`
	Targets        []UploadTarget      `json:"targets"`
	Manifest       Manifest            `json:"manifest"`
	TargetCount    uint32              `json:"target_count"`
	Product        core.Product        `json:"product"`
}

func (p DeployPlan) Validate() error {
	if err := validateDeployPlanIdentity(p); err != nil {
		return err
	}
	return validateDeployTargets(p.Targets, p.TargetCount)
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
	return nil
}

func validateDeployTargets(targets []UploadTarget, count uint32) error {
	if count == 0 || int(count) != len(targets) {
		return fmt.Errorf(ErrFmtDeployPlan, core.ErrReleaseContract)
	}
	seen := core.NewUniqueStringSet(len(targets))
	for _, target := range targets {
		if err := target.Validate(); err != nil {
			return wrapReleaseContract(ErrFmtDeployPlan, err)
		}
		if err := seen.Add(deployTargetKey(target)); err != nil {
			return wrapReleaseContract(ErrFmtDeployPlan, err)
		}
	}
	return nil
}

func deployTargetKey(target UploadTarget) string {
	return target.Provider.String() + "/" + target.Bucket.String() + "/" + target.Prefix.String()
}
