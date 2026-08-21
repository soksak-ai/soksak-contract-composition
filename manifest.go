package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	UnitSpec         = "soksak-spec-unit@0.0.1"
	UnitManifestFile = "soksak-unit.json"
)

type ContractRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Requirement struct {
	Name     string      `json:"name"`
	Contract ContractRef `json:"contract"`
}

type Entrypoint struct {
	Role string `json:"role"`
	Name string `json:"name,omitempty"`
	Path string `json:"path"`
}

type ProviderBinding struct {
	Requirement string  `json:"requirement"`
	Provider    UnitRef `json:"provider"`
}

type UnitManifest struct {
	Spec string `json:"spec"`
	UnitRef
	Dependencies []UnitRef         `json:"dependencies"`
	Implements   []ContractRef     `json:"implements"`
	Consumes     []Requirement     `json:"consumes"`
	Bindings     []ProviderBinding `json:"bindings"`
	Entrypoints  []Entrypoint      `json:"entrypoints"`
}

var (
	contractIDPattern  = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")
	requirementPattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")
)

func ParseUnitManifest(data []byte) (UnitManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest UnitManifest
	if err := decoder.Decode(&manifest); err != nil {
		return UnitManifest{}, fmt.Errorf("unit manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return UnitManifest{}, fmt.Errorf("unit manifest: trailing document")
		}
		return UnitManifest{}, fmt.Errorf("unit manifest: trailing data: %w", err)
	}
	if err := ValidateUnitManifest(manifest); err != nil {
		return UnitManifest{}, err
	}
	return manifest, nil
}

func ValidateUnitManifest(manifest UnitManifest) error {
	if manifest.Spec != UnitSpec {
		return fmt.Errorf("unit manifest spec: exact %s required", UnitSpec)
	}
	if err := validateRef(manifest.UnitRef); err != nil {
		return fmt.Errorf("unit manifest: %w", err)
	}
	if manifest.Dependencies == nil || manifest.Implements == nil || manifest.Consumes == nil || manifest.Bindings == nil || manifest.Entrypoints == nil {
		return fmt.Errorf("unit manifest: dependencies, implements, consumes, bindings and entrypoints arrays are required")
	}
	if len(manifest.Entrypoints) == 0 {
		return fmt.Errorf("unit manifest: at least one entrypoint required")
	}
	seenDependencies := map[string]bool{}
	for index, dependency := range manifest.Dependencies {
		if err := validateRef(dependency); err != nil {
			return fmt.Errorf("unit manifest dependency %d: %w", index, err)
		}
		if dependency == manifest.UnitRef {
			return fmt.Errorf("unit manifest dependency %d: self dependency", index)
		}
		if seenDependencies[dependency.Key()] {
			return fmt.Errorf("unit manifest dependency %d: duplicate %s", index, dependency.Key())
		}
		seenDependencies[dependency.Key()] = true
	}
	seenContracts := map[string]bool{}
	for index, provided := range manifest.Implements {
		if err := validateContract(provided); err != nil {
			return fmt.Errorf("unit manifest implements %d: %w", index, err)
		}
		key := provided.ID + "@" + provided.Version
		if seenContracts[key] {
			return fmt.Errorf("unit manifest implements %d: duplicate %s", index, key)
		}
		seenContracts[key] = true
	}
	seenRequirements := map[string]bool{}
	for index, requirement := range manifest.Consumes {
		if !requirementPattern.MatchString(requirement.Name) {
			return fmt.Errorf("unit manifest consumes %d: lowercase requirement name required", index)
		}
		if err := validateContract(requirement.Contract); err != nil {
			return fmt.Errorf("unit manifest consumes %d: %w", index, err)
		}
		if seenRequirements[requirement.Name] {
			return fmt.Errorf("unit manifest consumes %d: duplicate requirement %s", index, requirement.Name)
		}
		seenRequirements[requirement.Name] = true
	}
	seenBindings := map[string]bool{}
	for index, binding := range manifest.Bindings {
		if !requirementPattern.MatchString(binding.Requirement) {
			return fmt.Errorf("unit manifest binding %d: lowercase requirement name required", index)
		}
		if !seenRequirements[binding.Requirement] {
			return fmt.Errorf("unit manifest binding %d: unknown requirement %s", index, binding.Requirement)
		}
		if seenBindings[binding.Requirement] {
			return fmt.Errorf("unit manifest binding %d: duplicate requirement %s", index, binding.Requirement)
		}
		if err := validateRef(binding.Provider); err != nil {
			return fmt.Errorf("unit manifest binding %d provider: %w", index, err)
		}
		seenBindings[binding.Requirement] = true
	}
	for name := range seenRequirements {
		if !seenBindings[name] {
			return fmt.Errorf("unit manifest binding missing for requirement %s", name)
		}
	}
	seenEntrypoints := map[string]bool{}
	for index, entrypoint := range manifest.Entrypoints {
		if !validEntrypointRole(entrypoint.Role) {
			return fmt.Errorf("unit manifest entrypoint %d: invalid role", index)
		}
		if entrypoint.Role == "process" && !unitIDPattern.MatchString(entrypoint.Name) {
			return fmt.Errorf("unit manifest entrypoint %d: process name required", index)
		}
		if entrypoint.Role != "process" && entrypoint.Name != "" {
			return fmt.Errorf("unit manifest entrypoint %d: name allowed only for process", index)
		}
		if err := safeRelativePath(entrypoint.Path, "entrypoint path"); err != nil {
			return fmt.Errorf("unit manifest entrypoint %d: %w", index, err)
		}
		key := entrypoint.Role + ":" + entrypoint.Name + ":" + entrypoint.Path
		if seenEntrypoints[key] {
			return fmt.Errorf("unit manifest entrypoint %d: duplicate", index)
		}
		seenEntrypoints[key] = true
	}
	return nil
}

func InitialBindings(manifest UnitManifest) ([]Binding, error) {
	if err := ValidateUnitManifest(manifest); err != nil {
		return nil, err
	}
	bindings := make([]Binding, 0, len(manifest.Bindings))
	for _, declared := range manifest.Bindings {
		bindings = append(bindings, Binding{Consumer: manifest.UnitRef, Requirement: declared.Requirement, Provider: declared.Provider})
	}
	return bindings, nil
}

func validateContract(ref ContractRef) error {
	if !contractIDPattern.MatchString(ref.ID) {
		return fmt.Errorf("contract id: lowercase id required")
	}
	if !versionPattern.MatchString(ref.Version) {
		return fmt.Errorf("contract version: exact semantic version required")
	}
	return nil
}

func validEntrypointRole(role string) bool {
	switch role {
	case "plugin", "process", "library", "package", "contract", "spec", "service":
		return true
	}
	return false
}

func safeRelativePath(path, label string) error {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s: safe relative path required", label)
	}
	return nil
}
