package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type ReleaseQualification uint8

const (
	ReleaseQualificationUnknown ReleaseQualification = iota
	ReleaseQualificationCandidate
	ReleaseQualificationCertified
)

const (
	releaseQualificationTokenCandidate = "candidate"
	releaseQualificationTokenCertified = "certified"
)

func releaseQualificationNames() [ReleaseQualificationCertified + 1]string {
	return [...]string{
		ReleaseQualificationCandidate: releaseQualificationTokenCandidate,
		ReleaseQualificationCertified: releaseQualificationTokenCertified,
	}
}

func (q ReleaseQualification) String() string {
	if q.IsValid() {
		return releaseQualificationNames()[q]
	}
	return ""
}

func (q ReleaseQualification) IsValid() bool {
	return q > ReleaseQualificationUnknown && int(q) < len(releaseQualificationNames()) &&
		releaseQualificationNames()[q] != ""
}

func (q ReleaseQualification) Validate() error {
	if !q.IsValid() {
		return fmt.Errorf(ErrFmtReleaseQualification, core.ErrReleaseContract)
	}
	return nil
}

func ParseReleaseQualification(token string) (ReleaseQualification, error) {
	for qualification := ReleaseQualificationCandidate; qualification <= ReleaseQualificationCertified; qualification++ {
		if qualification.String() == token {
			return qualification, nil
		}
	}
	return ReleaseQualificationUnknown, fmt.Errorf(ErrFmtReleaseQualification, core.ErrReleaseContract)
}

func (q ReleaseQualification) MarshalJSON() ([]byte, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(q.String())
}

func (q *ReleaseQualification) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtReleaseQualification, core.ErrReleaseContract)
	}
	parsed, err := ParseReleaseQualification(token)
	if err != nil {
		return err
	}
	*q = parsed
	return nil
}

// FinalCertificateEvidence is a closed candidate/certified sum type. Candidate
// evidence exists only to exercise an unfinished release pipeline; it carries
// no certificate identity. Certified evidence binds the exact public
// final_certificate.json bytes to the exact release subject commit.
type FinalCertificateEvidence struct {
	Reference     EvidenceRef          `json:"reference"`
	SHA256        core.SHA256Hex       `json:"sha256"`
	SubjectCommit core.BuildCommit     `json:"subject_commit"`
	Qualification ReleaseQualification `json:"qualification"`
}

func CandidateFinalCertificateEvidence() FinalCertificateEvidence {
	return FinalCertificateEvidence{Qualification: ReleaseQualificationCandidate}
}

func BuildCertifiedFinalCertificateEvidence(
	sha256 core.SHA256Hex,
	subjectCommit core.BuildCommit,
) (FinalCertificateEvidence, error) {
	reference, err := ParseEvidenceRef(FinalCertificateFileName)
	if err != nil {
		return FinalCertificateEvidence{}, err
	}
	evidence := FinalCertificateEvidence{
		Reference: reference, SHA256: sha256, SubjectCommit: subjectCommit,
		Qualification: ReleaseQualificationCertified,
	}
	if err := evidence.Validate(); err != nil {
		return FinalCertificateEvidence{}, err
	}
	return evidence, nil
}

func (e FinalCertificateEvidence) Validate() error {
	if err := e.Qualification.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtFinalCertificateEvidence, err)
	}
	if e.Qualification == ReleaseQualificationCandidate {
		return e.validateCandidate()
	}
	return e.validateCertified()
}

func (e FinalCertificateEvidence) validateCandidate() error {
	if e.Reference != (EvidenceRef{}) || e.SHA256 != (core.SHA256Hex{}) ||
		e.SubjectCommit != (core.BuildCommit{}) {
		return fmt.Errorf(ErrFmtFinalCertificateEvidence, core.ErrReleaseContract)
	}
	return nil
}

func (e FinalCertificateEvidence) validateCertified() error {
	if err := e.Reference.Validate(); err != nil || e.Reference.String() != FinalCertificateFileName {
		return fmt.Errorf(ErrFmtFinalCertificateEvidence, core.ErrReleaseContract)
	}
	if err := e.SHA256.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtFinalCertificateEvidence, err)
	}
	if err := e.SubjectCommit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtFinalCertificateEvidence, err)
	}
	return nil
}

func (e FinalCertificateEvidence) ValidateForCommit(commit core.BuildCommit) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if err := commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtFinalCertificateEvidence, err)
	}
	if e.Qualification == ReleaseQualificationCertified && e.SubjectCommit != commit {
		return fmt.Errorf(ErrFmtFinalCertificateEvidence, core.ErrReleaseContract)
	}
	return nil
}

func (e FinalCertificateEvidence) RequireCertified(commit core.BuildCommit) error {
	if err := e.ValidateForCommit(commit); err != nil {
		return err
	}
	if e.Qualification != ReleaseQualificationCertified {
		return fmt.Errorf(ErrFmtFinalCertificateEvidence, core.ErrReleaseContract)
	}
	return nil
}

func (e FinalCertificateEvidence) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, jsonFieldQualification, e.Qualification)
	if e.Qualification == ReleaseQualificationCertified {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReference, e.Reference)
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSHA256, e.SHA256)
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSubjectCommit, e.SubjectCommit)
	}
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}
