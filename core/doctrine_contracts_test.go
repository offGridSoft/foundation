package core

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDoctrinePackageLayerContractTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		layer      DoctrinePackageLayer
		identifier string
		valid      bool
	}{
		{name: "core", layer: DoctrinePackageLayerCore, identifier: "DoctrinePackageLayerCore", valid: true},
		{name: "primitive", layer: DoctrinePackageLayerPrimitive, identifier: "DoctrinePackageLayerPrimitive", valid: true},
		{name: "foundation", layer: DoctrinePackageLayerFoundation, identifier: "DoctrinePackageLayerFoundation", valid: true},
		{name: "component", layer: DoctrinePackageLayerComponent, identifier: "DoctrinePackageLayerComponent", valid: true},
		{name: "orchestrator", layer: DoctrinePackageLayerOrchestrator, identifier: "DoctrinePackageLayerOrchestrator", valid: true},
		{name: "test support", layer: DoctrinePackageLayerTestSupport, identifier: "DoctrinePackageLayerTestSupport", valid: true},
		{name: "shell", layer: DoctrinePackageLayerShell, identifier: "DoctrinePackageLayerShell", valid: true},
		{name: "zero", layer: DoctrinePackageLayerUnknown},
		{name: "overflow", layer: doctrinePackageLayerLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.layer.IsValid(); got != tc.valid {
				t.Fatalf("IsValid() = %v, want %v", got, tc.valid)
			}
			err := tc.layer.Validate()
			if tc.valid {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				parsed, parseErr := ParseDoctrinePackageLayerGoIdentifier(tc.identifier)
				if parseErr != nil || parsed != tc.layer {
					t.Fatalf("ParseDoctrinePackageLayerGoIdentifier() = (%v, %v)", parsed, parseErr)
				}
				encoded, marshalErr := json.Marshal(tc.layer)
				if marshalErr != nil {
					t.Fatalf("Marshal() error = %v", marshalErr)
				}
				var decoded DoctrinePackageLayer
				if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil || decoded != tc.layer {
					t.Fatalf("Unmarshal() = (%v, %v), want (%v, nil)", decoded, unmarshalErr, tc.layer)
				}
				return
			}
			if !errors.Is(err, ErrDoctrineContract) {
				t.Fatalf("Validate() error = %v, want ErrDoctrineContract", err)
			}
		})
	}
	if _, err := ParseDoctrinePackageLayerGoIdentifier("DoctrinePackageLayerPeachfuzz"); !errors.Is(err, ErrDoctrineContract) {
		t.Fatalf("lookalike identifier error = %v, want ErrDoctrineContract", err)
	}
}

func TestDefaultDoctrinePackageLayerIsValidLeastPrivilege(t *testing.T) {
	t.Parallel()

	got := DefaultDoctrinePackageLayer()
	if err := got.Validate(); err != nil {
		t.Fatalf("DefaultDoctrinePackageLayer().Validate() error = %v", err)
	}
	if got != DoctrinePackageLayerComponent {
		t.Fatalf("DefaultDoctrinePackageLayer() = %v, want %v", got, DoctrinePackageLayerComponent)
	}
}

func TestDoctrinePackageCapabilityContractTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		capability DoctrinePackageCapability
		valid      bool
	}{
		{name: "process execution", capability: DoctrinePackageCapabilityProcessExecution, valid: true},
		{name: "test support", capability: DoctrinePackageCapabilityTestSupport, valid: true},
		{name: "zero", capability: DoctrinePackageCapabilityUnknown},
		{name: "overflow", capability: doctrinePackageCapabilityLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.capability.Validate()
			if tc.valid {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				parsed, parseErr := ParseDoctrinePackageCapabilityGoIdentifier(tc.capability.GoIdentifier())
				if parseErr != nil || parsed != tc.capability {
					t.Fatalf("ParseDoctrinePackageCapabilityGoIdentifier() = (%v, %v)", parsed, parseErr)
				}
				encoded, marshalErr := json.Marshal(tc.capability)
				if marshalErr != nil {
					t.Fatalf("Marshal() error = %v", marshalErr)
				}
				var decoded DoctrinePackageCapability
				if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil || decoded != tc.capability {
					t.Fatalf("Unmarshal() = (%v, %v), want (%v, nil)", decoded, unmarshalErr, tc.capability)
				}
				return
			}
			if !errors.Is(err, ErrDoctrineContract) {
				t.Fatalf("Validate() error = %v, want ErrDoctrineContract", err)
			}
		})
	}
	if _, err := ParseDoctrinePackageCapabilityGoIdentifier("DoctrinePackageCapabilityExec"); !errors.Is(err, ErrDoctrineContract) {
		t.Fatalf("lookalike identifier error = %v, want ErrDoctrineContract", err)
	}
}

func TestDoctrineImportAllowedTable(t *testing.T) {
	t.Parallel()
	layers := []DoctrinePackageLayer{
		DoctrinePackageLayerCore,
		DoctrinePackageLayerPrimitive,
		DoctrinePackageLayerFoundation,
		DoctrinePackageLayerComponent,
		DoctrinePackageLayerOrchestrator,
		DoctrinePackageLayerTestSupport,
		DoctrinePackageLayerShell,
	}
	for _, src := range layers {
		for _, dst := range layers {
			got := DoctrineImportAllowed(src, dst)
			if src == DoctrinePackageLayerCore && got {
				t.Fatalf("core unexpectedly imports %s", dst)
			}
			if src == DoctrinePackageLayerShell && !got {
				t.Fatalf("shell unexpectedly rejects %s", dst)
			}
			if src == DoctrinePackageLayerTestSupport && dst == DoctrinePackageLayerShell && got {
				t.Fatalf("test support unexpectedly imports shell")
			}
			if src == DoctrinePackageLayerComponent && dst == DoctrinePackageLayerTestSupport && got {
				t.Fatalf("component unexpectedly imports test support")
			}
		}
	}
}
