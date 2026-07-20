package core

import (
	"encoding/json"
	"fmt"
)

// DoctrinePackageLayer is a package's compiler-owned dependency role.
// Inspected repositories declare this typed value themselves; general-purpose
// doctrine tools must never carry product or package-name registries.
type DoctrinePackageLayer uint8

const (
	DoctrinePackageLayerDeclarationIdentifier      = "doctrinePackageLayer"
	DoctrinePackageCapabilityDeclarationIdentifier = "doctrinePackageCapability"
)

const (
	DoctrinePackageLayerUnknown DoctrinePackageLayer = iota
	DoctrinePackageLayerCore
	DoctrinePackageLayerPrimitive
	DoctrinePackageLayerFoundation
	DoctrinePackageLayerComponent
	DoctrinePackageLayerOrchestrator
	DoctrinePackageLayerTestSupport
	DoctrinePackageLayerShell
	doctrinePackageLayerLimit
)

// DefaultDoctrinePackageLayer returns the least-privilege role assigned when a
// package does not explicitly declare a stronger architectural role. Keeping
// the default in Foundation makes omission a compiler-owned contract instead
// of a linter-local naming convention.
func DefaultDoctrinePackageLayer() DoctrinePackageLayer {
	return DoctrinePackageLayerComponent
}

const (
	DoctrinePackageLayerCoreToken         = "core"
	DoctrinePackageLayerPrimitiveToken    = "primitive"
	DoctrinePackageLayerFoundationToken   = "foundation"
	DoctrinePackageLayerComponentToken    = "component"
	DoctrinePackageLayerOrchestratorToken = "orchestrator"
	DoctrinePackageLayerTestSupportToken  = "test_support"
	DoctrinePackageLayerShellToken        = "shell"
)

func doctrinePackageLayerTokens() [doctrinePackageLayerLimit]string {
	return [...]string{
		DoctrinePackageLayerCore:         DoctrinePackageLayerCoreToken,
		DoctrinePackageLayerPrimitive:    DoctrinePackageLayerPrimitiveToken,
		DoctrinePackageLayerFoundation:   DoctrinePackageLayerFoundationToken,
		DoctrinePackageLayerComponent:    DoctrinePackageLayerComponentToken,
		DoctrinePackageLayerOrchestrator: DoctrinePackageLayerOrchestratorToken,
		DoctrinePackageLayerTestSupport:  DoctrinePackageLayerTestSupportToken,
		DoctrinePackageLayerShell:        DoctrinePackageLayerShellToken,
	}
}

func doctrinePackageLayerGoIdentifiers() [doctrinePackageLayerLimit]string {
	return [...]string{
		DoctrinePackageLayerCore:         "DoctrinePackageLayerCore",
		DoctrinePackageLayerPrimitive:    "DoctrinePackageLayerPrimitive",
		DoctrinePackageLayerFoundation:   "DoctrinePackageLayerFoundation",
		DoctrinePackageLayerComponent:    "DoctrinePackageLayerComponent",
		DoctrinePackageLayerOrchestrator: "DoctrinePackageLayerOrchestrator",
		DoctrinePackageLayerTestSupport:  "DoctrinePackageLayerTestSupport",
		DoctrinePackageLayerShell:        "DoctrinePackageLayerShell",
	}
}

func (l DoctrinePackageLayer) String() string {
	if !l.IsValid() {
		return ""
	}
	return doctrinePackageLayerTokens()[l]
}

func (l DoctrinePackageLayer) GoIdentifier() string {
	if !l.IsValid() {
		return ""
	}
	return doctrinePackageLayerGoIdentifiers()[l]
}

func (l DoctrinePackageLayer) IsValid() bool {
	return l > DoctrinePackageLayerUnknown && l < doctrinePackageLayerLimit && doctrinePackageLayerTokens()[l] != ""
}

func (l DoctrinePackageLayer) Validate() error {
	if !l.IsValid() {
		return fmt.Errorf(ErrFmtDoctrineLayer, ErrDoctrineContract)
	}
	return nil
}

func ParseDoctrinePackageLayerGoIdentifier(identifier string) (DoctrinePackageLayer, error) {
	for layer := DoctrinePackageLayerCore; layer < doctrinePackageLayerLimit; layer++ {
		if layer.GoIdentifier() == identifier {
			return layer, nil
		}
	}
	return DoctrinePackageLayerUnknown, fmt.Errorf(ErrFmtDoctrineLayer, ErrDoctrineContract)
}

func ParseDoctrinePackageLayer(token string) (DoctrinePackageLayer, error) {
	for layer := DoctrinePackageLayerCore; layer < doctrinePackageLayerLimit; layer++ {
		if layer.String() == token {
			return layer, nil
		}
	}
	return DoctrinePackageLayerUnknown, fmt.Errorf(ErrFmtDoctrineLayer, ErrDoctrineContract)
}

func (l DoctrinePackageLayer) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(l.String())
}

func (l *DoctrinePackageLayer) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtDoctrineLayer, ErrDoctrineContract)
	}
	parsed, err := ParseDoctrinePackageLayer(token)
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

func DoctrineImportAllowed(src DoctrinePackageLayer, dst DoctrinePackageLayer) bool {
	switch src {
	case DoctrinePackageLayerCore:
		return false
	case DoctrinePackageLayerPrimitive:
		return dst == DoctrinePackageLayerCore
	case DoctrinePackageLayerFoundation:
		return dst == DoctrinePackageLayerCore || dst == DoctrinePackageLayerPrimitive
	case DoctrinePackageLayerComponent:
		return dst == DoctrinePackageLayerCore || dst == DoctrinePackageLayerPrimitive || dst == DoctrinePackageLayerFoundation
	case DoctrinePackageLayerOrchestrator:
		return dst == DoctrinePackageLayerCore || dst == DoctrinePackageLayerPrimitive || dst == DoctrinePackageLayerFoundation || dst == DoctrinePackageLayerComponent || dst == DoctrinePackageLayerOrchestrator
	case DoctrinePackageLayerTestSupport:
		return dst != DoctrinePackageLayerShell && dst.IsValid()
	case DoctrinePackageLayerShell:
		return dst.IsValid()
	default:
		return false
	}
}

// DoctrinePackageCapability is an explicitly declared privilege owned by one
// inspected package. Capabilities are independent enum values rather than a
// bit mask so source analyzers can reject duplicates and unknown values.
type DoctrinePackageCapability uint8

const (
	DoctrinePackageCapabilityUnknown DoctrinePackageCapability = iota
	DoctrinePackageCapabilityProcessExecution
	DoctrinePackageCapabilityTestSupport
	doctrinePackageCapabilityLimit
)

const (
	DoctrinePackageCapabilityProcessExecutionToken = "process_execution"
	DoctrinePackageCapabilityTestSupportToken      = "test_support"
)

func doctrinePackageCapabilityTokens() [doctrinePackageCapabilityLimit]string {
	return [...]string{
		DoctrinePackageCapabilityProcessExecution: DoctrinePackageCapabilityProcessExecutionToken,
		DoctrinePackageCapabilityTestSupport:      DoctrinePackageCapabilityTestSupportToken,
	}
}

func doctrinePackageCapabilityGoIdentifiers() [doctrinePackageCapabilityLimit]string {
	return [...]string{
		DoctrinePackageCapabilityProcessExecution: "DoctrinePackageCapabilityProcessExecution",
		DoctrinePackageCapabilityTestSupport:      "DoctrinePackageCapabilityTestSupport",
	}
}

func (c DoctrinePackageCapability) GoIdentifier() string {
	if !c.IsValid() {
		return ""
	}
	return doctrinePackageCapabilityGoIdentifiers()[c]
}

func (c DoctrinePackageCapability) String() string {
	if !c.IsValid() {
		return ""
	}
	return doctrinePackageCapabilityTokens()[c]
}

func (c DoctrinePackageCapability) IsValid() bool {
	return c > DoctrinePackageCapabilityUnknown && c < doctrinePackageCapabilityLimit && doctrinePackageCapabilityGoIdentifiers()[c] != ""
}

func (c DoctrinePackageCapability) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtDoctrineCapability, ErrDoctrineContract)
	}
	return nil
}

func ParseDoctrinePackageCapabilityGoIdentifier(identifier string) (DoctrinePackageCapability, error) {
	for capability := DoctrinePackageCapabilityProcessExecution; capability < doctrinePackageCapabilityLimit; capability++ {
		if capability.GoIdentifier() == identifier {
			return capability, nil
		}
	}
	return DoctrinePackageCapabilityUnknown, fmt.Errorf(ErrFmtDoctrineCapability, ErrDoctrineContract)
}

func ParseDoctrinePackageCapability(token string) (DoctrinePackageCapability, error) {
	for capability := DoctrinePackageCapabilityProcessExecution; capability < doctrinePackageCapabilityLimit; capability++ {
		if capability.String() == token {
			return capability, nil
		}
	}
	return DoctrinePackageCapabilityUnknown, fmt.Errorf(ErrFmtDoctrineCapability, ErrDoctrineContract)
}

func (c DoctrinePackageCapability) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func (c *DoctrinePackageCapability) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtDoctrineCapability, ErrDoctrineContract)
	}
	parsed, err := ParseDoctrinePackageCapability(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
