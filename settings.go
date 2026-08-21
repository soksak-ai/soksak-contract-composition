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
	SettingsSpec = "soksak-spec-composition@0.0.1"
	SettingsFile = "settings.json"
)

type UnitKind string

const (
	Plugin  UnitKind = "plugin"
	Sidecar UnitKind = "sidecar"
	Kit     UnitKind = "kit"
)

type UnitMode string

const (
	Installed   UnitMode = "installed"
	Development UnitMode = "development"
)

type SourceType string

const (
	ArchiveSource SourceType = "archive"
	GitSource     SourceType = "git"
	PathSource    SourceType = "path"
)

type UnitRef struct {
	Kind    UnitKind `json:"kind"`
	ID      string   `json:"id"`
	Version string   `json:"version"`
}

func (ref UnitRef) Key() string { return string(ref.Kind) + ":" + ref.ID + "@" + ref.Version }

type Source struct {
	Type       SourceType `json:"type"`
	URL        string     `json:"url,omitempty"`
	Repository string     `json:"repository,omitempty"`
	Commit     string     `json:"commit,omitempty"`
	SHA256     string     `json:"sha256,omitempty"`
	Path       string     `json:"path,omitempty"`
}

type Installation struct {
	UnitRef
	Mode        UnitMode `json:"mode"`
	InstallPath string   `json:"installPath"`
	Manifest    string   `json:"manifest"`
	Source      Source   `json:"source"`
}

type PluginSelection struct {
	Plugin  UnitRef `json:"plugin"`
	Enabled bool    `json:"enabled"`
}

type Binding struct {
	Consumer    UnitRef `json:"consumer"`
	Requirement string  `json:"requirement"`
	Provider    UnitRef `json:"provider"`
}

type Settings struct {
	Spec          string            `json:"spec"`
	Generation    uint64            `json:"generation"`
	Installations []Installation    `json:"installations"`
	Plugins       []PluginSelection `json:"plugins"`
	Bindings      []Binding         `json:"bindings"`
}

type UpdateStatus string

const (
	UpdateAllowed         UpdateStatus = "allowed"
	UpdateDevelopmentUnit UpdateStatus = "development-unit"
)

type UpdateDecision struct {
	Allowed bool         `json:"allowed"`
	Status  UpdateStatus `json:"status"`
}

var (
	unitIDPattern  = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")
	versionPattern = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")
	commitPattern  = regexp.MustCompile("^[0-9a-f]{40}$")
	digestPattern  = regexp.MustCompile("^[0-9a-f]{64}$")
)

func ParseSettings(data []byte) (Settings, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var settings Settings
	if err := decoder.Decode(&settings); err != nil {
		return Settings{}, fmt.Errorf("composition settings: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return Settings{}, fmt.Errorf("composition settings: trailing document")
		}
		return Settings{}, fmt.Errorf("composition settings: trailing data: %w", err)
	}
	if settings.Installations == nil {
		settings.Installations = []Installation{}
	}
	if settings.Plugins == nil {
		settings.Plugins = []PluginSelection{}
	}
	if settings.Bindings == nil {
		settings.Bindings = []Binding{}
	}
	if err := ValidateSettings(settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func ValidateSettings(settings Settings) error {
	if settings.Spec != SettingsSpec {
		return fmt.Errorf("composition settings spec: exact %s required", SettingsSpec)
	}
	if settings.Generation < 1 {
		return fmt.Errorf("composition settings generation: positive integer required")
	}
	seen := make(map[string]bool, len(settings.Installations))
	plugins := make(map[string]bool)
	for index, installation := range settings.Installations {
		if err := validateInstallation(installation); err != nil {
			return fmt.Errorf("composition settings installation %d: %w", index, err)
		}
		key := installation.UnitRef.Key()
		if seen[key] {
			return fmt.Errorf("composition settings installation %d: duplicate unit %s", index, key)
		}
		seen[key] = true
		if installation.Kind == Plugin {
			plugins[key] = true
		}
	}
	selected := make(map[string]bool, len(settings.Plugins))
	for index, selection := range settings.Plugins {
		if err := validateRef(selection.Plugin); err != nil {
			return fmt.Errorf("composition settings plugin %d: %w", index, err)
		}
		if selection.Plugin.Kind != Plugin {
			return fmt.Errorf("composition settings plugin %d: plugin kind required", index)
		}
		key := selection.Plugin.Key()
		if !plugins[key] {
			return fmt.Errorf("composition settings plugin %d: plugin is not installed: %s", index, key)
		}
		if selected[key] {
			return fmt.Errorf("composition settings plugin %d: duplicate selection %s", index, key)
		}
		selected[key] = true
	}
	for key := range plugins {
		if !selected[key] {
			return fmt.Errorf("composition settings plugin selection missing: %s", key)
		}
	}
	return nil
}

func validateInstallation(installation Installation) error {
	if err := validateRef(installation.UnitRef); err != nil {
		return err
	}
	if installation.Mode != Installed && installation.Mode != Development {
		return fmt.Errorf("mode: installed or development required")
	}
	if err := absoluteCleanPath(installation.InstallPath, "installPath"); err != nil {
		return err
	}
	if installation.Manifest == "" || filepath.IsAbs(installation.Manifest) || filepath.Clean(installation.Manifest) != installation.Manifest || strings.HasPrefix(installation.Manifest, ".."+string(filepath.Separator)) || installation.Manifest == ".." {
		return fmt.Errorf("manifest: safe relative path required")
	}
	return validateSource(installation.Source)
}

func validateRef(ref UnitRef) error {
	if !validKind(ref.Kind) {
		return fmt.Errorf("kind: plugin, sidecar or kit required")
	}
	if !unitIDPattern.MatchString(ref.ID) {
		return fmt.Errorf("id: lowercase unit id required")
	}
	if !versionPattern.MatchString(ref.Version) {
		return fmt.Errorf("version: exact semantic version required")
	}
	return nil
}

func validKind(kind UnitKind) bool {
	switch kind {
	case Plugin, Sidecar, Kit:
		return true
	default:
		return false
	}
}

func validateSource(source Source) error {
	switch source.Type {
	case GitSource:
		if source.URL == "" || source.Repository != "" || !commitPattern.MatchString(source.Commit) || source.SHA256 != "" || source.Path != "" {
			return fmt.Errorf("source: git requires url and exact 40-character commit only")
		}
	case ArchiveSource:
		if source.URL == "" || source.Repository == "" || !commitPattern.MatchString(source.Commit) || !digestPattern.MatchString(source.SHA256) || source.Path != "" {
			return fmt.Errorf("source: archive requires repository, exact commit, asset url and exact SHA-256")
		}
	case PathSource:
		if err := absoluteCleanPath(source.Path, "source.path"); err != nil {
			return err
		}
		if source.URL != "" || source.Repository != "" || source.Commit != "" || source.SHA256 != "" {
			return fmt.Errorf("source: path accepts no archive or git fields")
		}
	default:
		return fmt.Errorf("source.type: archive, git or path required")
	}
	return nil
}

func absoluteCleanPath(path, label string) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("%s: clean absolute path required", label)
	}
	return nil
}

func UpdatePolicy(installation Installation) UpdateDecision {
	if installation.Mode == Development {
		return UpdateDecision{Status: UpdateDevelopmentUnit}
	}
	return UpdateDecision{Allowed: true, Status: UpdateAllowed}
}
