package core

import (
	"errors"
	"fmt"
)

const ErrFmtUpdateNotice = "core.UpdateNotice: %w"

type UpdateNotice struct {
	LatestVersion *ProductVersion `json:"latest_version,omitempty"`
	Product       Product         `json:"product"`
	Available     bool            `json:"available"`
}

func BuildUpdateNotice(product Product, latestVersion *ProductVersion) (UpdateNotice, error) {
	notice := UpdateNotice{Product: product, Available: latestVersion != nil}
	if latestVersion != nil {
		version := *latestVersion
		notice.LatestVersion = &version
	}
	if err := notice.Validate(); err != nil {
		return UpdateNotice{}, err
	}
	return notice, nil
}

func (n UpdateNotice) Validate() error {
	if err := n.Product.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUpdateNotice, errors.Join(ErrFoundationContract, err))
	}
	if n.Available != (n.LatestVersion != nil) {
		return fmt.Errorf(ErrFmtUpdateNotice, ErrFoundationContract)
	}
	if n.LatestVersion != nil {
		if err := n.LatestVersion.Validate(); err != nil {
			return fmt.Errorf(ErrFmtUpdateNotice, errors.Join(ErrFoundationContract, err))
		}
	}
	return nil
}
