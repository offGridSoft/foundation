package core

import (
	"encoding/json"
	"fmt"
)

type Product uint8

const (
	ProductUnknown Product = iota
	ProductWitness
	ProductBug
)

const (
	ProductTokenWitness = "witness"
	ProductTokenBug     = "bug"
)

func productNames() [ProductBug + 1]string {
	return [...]string{
		ProductWitness: ProductTokenWitness,
		ProductBug:     ProductTokenBug,
	}
}

func (p Product) String() string {
	if p.IsValid() {
		return productNames()[p]
	}
	return ""
}

func (p Product) IsValid() bool {
	return p > ProductUnknown && int(p) < len(productNames()) && productNames()[p] != ""
}

func (p Product) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtProduct, ErrFoundationContract)
	}
	return nil
}

func ParseProduct(token string) (Product, error) {
	for product := ProductWitness; int(product) < len(productNames()); product++ {
		if productNames()[product] == token {
			return product, nil
		}
	}
	return ProductUnknown, fmt.Errorf(ErrFmtProduct, ErrFoundationContract)
}

func (p Product) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *Product) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtProduct, ErrFoundationContract)
	}
	parsed, err := ParseProduct(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
