package core

import (
	"errors"
	"testing"
)

func TestUpdateNoticeHostileTable(t *testing.T) {
	t.Parallel()
	version, err := ParseProductVersion(FoundationVersion2026)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := BuildUpdateNotice(ProductBug, &version)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mutate func(*UpdateNotice)
		name   string
		accept bool
	}{
		{name: "available", accept: true},
		{name: "available without version", mutate: func(n *UpdateNotice) { n.LatestVersion = nil }},
		{name: "version without availability", mutate: func(n *UpdateNotice) { n.Available = false }},
		{name: "missing product", mutate: func(n *UpdateNotice) { n.Product = ProductUnknown }},
		{name: "empty version", mutate: func(n *UpdateNotice) { n.LatestVersion = &ProductVersion{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			notice := valid
			if tc.mutate != nil {
				tc.mutate(&notice)
			}
			err := notice.Validate()
			if tc.accept && err != nil {
				t.Fatalf("UpdateNotice.Validate() error = %v", err)
			}
			if !tc.accept && !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("UpdateNotice.Validate() error = %v, want foundation contract", err)
			}
		})
	}

	current, err := BuildUpdateNotice(ProductBug, nil)
	if err != nil {
		t.Fatal(err)
	}
	if current.Available || current.LatestVersion != nil {
		t.Fatalf("current notice = %#v", current)
	}
}
