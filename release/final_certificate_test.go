package release

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestFinalCertificateEvidenceHostileTable(t *testing.T) {
	t.Parallel()
	valid := validFinalCertificateEvidence(t)

	cases := []struct {
		value  FinalCertificateEvidence
		commit core.BuildCommit
		name   string
		want   bool
	}{
		{name: "certified exact commit", value: valid, commit: mustCommit(t)},
		{name: "candidate accepted before publication", value: CandidateFinalCertificateEvidence(), commit: mustCommit(t)},
		{name: "zero evidence", commit: mustCommit(t), want: true},
		{name: "candidate cannot carry a digest", value: FinalCertificateEvidence{
			Qualification: ReleaseQualificationCandidate, SHA256: mustSHA256(t, "a"),
		}, commit: mustCommit(t), want: true},
		{name: "certified missing reference", value: func() FinalCertificateEvidence {
			value := valid
			value.Reference = EvidenceRef{}
			return value
		}(), commit: mustCommit(t), want: true},
		{name: "certified wrong commit", value: valid, commit: mustOtherCommit(t), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.value.ValidateForCommit(tc.commit)
			if !tc.want && err != nil {
				t.Fatalf("ValidateForCommit() error = %v", err)
			}
			if tc.want && !errors.Is(err, core.ErrReleaseContract) {
				t.Fatalf("ValidateForCommit() error = %v, want %v", err, core.ErrReleaseContract)
			}
		})
	}
}

func TestFinalCertificatePublicationRejectsCandidate(t *testing.T) {
	t.Parallel()
	commit := mustCommit(t)
	if err := CandidateFinalCertificateEvidence().RequireCertified(commit); !errors.Is(err, core.ErrReleaseContract) {
		t.Fatalf("RequireCertified(candidate) error = %v, want %v", err, core.ErrReleaseContract)
	}
	if err := validFinalCertificateEvidence(t).RequireCertified(commit); err != nil {
		t.Fatalf("RequireCertified(certified) error = %v", err)
	}
}
