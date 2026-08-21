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

type SourceType string

const (
	ArchiveSource SourceType = "archive"
	GitSource     SourceType = "git"
	PathSource    SourceType = "path"
)

type Source struct {
	Type       SourceType `json:"type"`
	URL        string     `json:"url,omitempty"`
	Repository string     `json:"repository,omitempty"`
	Commit     string     `json:"commit,omitempty"`
	SHA256     string     `json:"sha256,omitempty"`
	Path       string     `json:"path,omitempty"`
}

type PluginRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type SidecarRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type KitRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Plugin struct {
	PluginRef
	Enabled     bool   `json:"enabled"`
	Development bool   `json:"development"`
	InstallPath string `json:"installPath"`
	Manifest    string `json:"manifest"`
	Source      Source `json:"source"`
}

type Sidecar struct {
	SidecarRef
	Enabled     bool   `json:"enabled"`
	Development bool   `json:"development"`
	InstallPath string `json:"installPath"`
	Manifest    string `json:"manifest"`
	Source      Source `json:"source"`
}

type Kit struct {
	KitRef
	Enabled     bool   `json:"enabled"`
	Development bool   `json:"development"`
	InstallPath string `json:"installPath"`
	Manifest    string `json:"manifest"`
	Source      Source `json:"source"`
}

type Endpoint struct {
	Plugin  *PluginRef  `json:"plugin,omitempty"`
	Sidecar *SidecarRef `json:"sidecar,omitempty"`
	Kit     *KitRef     `json:"kit,omitempty"`
}

type Binding struct {
	Consumer    Endpoint `json:"consumer"`
	Requirement string   `json:"requirement"`
	Provider    Endpoint `json:"provider"`
}

type Settings struct {
	Spec       string    `json:"spec"`
	Generation uint64    `json:"generation"`
	Plugins    []Plugin  `json:"plugins"`
	Sidecars   []Sidecar `json:"sidecars"`
	Kits       []Kit     `json:"kits"`
	Bindings   []Binding `json:"bindings"`
}

type UpdateDecision struct {
	Allowed bool   `json:"allowed"`
	Status  string `json:"status"`
}

var idPattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")
var versionPattern = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")
var commitPattern = regexp.MustCompile("^[0-9a-f]{40}$")
var digestPattern = regexp.MustCompile("^[0-9a-f]{64}$")

func ParseSettings(body []byte) (Settings, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var settings Settings
	if err := decoder.Decode(&settings); err != nil {
		return Settings{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Settings{}, fmt.Errorf("settings has trailing data")
	}
	normalize(&settings)
	if err := ValidateSettings(settings); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func ValidateSettings(settings Settings) error {
	if settings.Spec != SettingsSpec || settings.Generation < 1 {
		return fmt.Errorf("invalid composition identity")
	}
	if settings.Plugins == nil || settings.Sidecars == nil || settings.Kits == nil || settings.Bindings == nil {
		return fmt.Errorf("plugins, sidecars, kits and bindings arrays are required")
	}
	versions := map[string]string{}
	for index, value := range settings.Plugins {
		if err := validateRecord("plugin", value.ID, value.Version, value.InstallPath, value.Manifest, value.Source); err != nil {
			return fmt.Errorf("plugin %d: %w", index, err)
		}
		if err := oneVersion(versions, "plugin", value.ID, value.Version); err != nil {
			return err
		}
	}
	for index, value := range settings.Sidecars {
		if err := validateRecord("sidecar", value.ID, value.Version, value.InstallPath, value.Manifest, value.Source); err != nil {
			return fmt.Errorf("sidecar %d: %w", index, err)
		}
		if err := oneVersion(versions, "sidecar", value.ID, value.Version); err != nil {
			return err
		}
	}
	for index, value := range settings.Kits {
		if err := validateRecord("kit", value.ID, value.Version, value.InstallPath, value.Manifest, value.Source); err != nil {
			return fmt.Errorf("kit %d: %w", index, err)
		}
		if err := oneVersion(versions, "kit", value.ID, value.Version); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	for index, binding := range settings.Bindings {
		consumer, err := endpointKey(binding.Consumer)
		if err != nil {
			return fmt.Errorf("binding %d consumer: %w", index, err)
		}
		if _, err := endpointKey(binding.Provider); err != nil {
			return fmt.Errorf("binding %d provider: %w", index, err)
		}
		if !idPattern.MatchString(binding.Requirement) {
			return fmt.Errorf("binding %d has invalid requirement", index)
		}
		key := consumer + ":" + binding.Requirement
		if seen[key] {
			return fmt.Errorf("duplicate binding %s", key)
		}
		seen[key] = true
	}
	return nil
}

func PluginUpdatePolicy(value Plugin) UpdateDecision   { return updatePolicy(value.Development) }
func SidecarUpdatePolicy(value Sidecar) UpdateDecision { return updatePolicy(value.Development) }
func KitUpdatePolicy(value Kit) UpdateDecision         { return updatePolicy(value.Development) }
func updatePolicy(development bool) UpdateDecision {
	if development {
		return UpdateDecision{Status: "development"}
	}
	return UpdateDecision{Allowed: true, Status: "managed"}
}

func normalize(settings *Settings) {
	if settings.Plugins == nil {
		settings.Plugins = []Plugin{}
	}
	if settings.Sidecars == nil {
		settings.Sidecars = []Sidecar{}
	}
	if settings.Kits == nil {
		settings.Kits = []Kit{}
	}
	if settings.Bindings == nil {
		settings.Bindings = []Binding{}
	}
}
func validateRecord(kind, id, version, path, manifest string, source Source) error {
	if !idPattern.MatchString(id) || !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid %s identity", kind)
	}
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("installPath must be absolute")
	}
	if manifest == "" || filepath.IsAbs(manifest) || filepath.Clean(manifest) != manifest || manifest == ".." || strings.HasPrefix(manifest, ".."+string(filepath.Separator)) {
		return fmt.Errorf("manifest must be a safe relative path")
	}
	return validateSource(source)
}
func validateSource(source Source) error {
	switch source.Type {
	case GitSource:
		if source.URL == "" || source.Repository != "" || !commitPattern.MatchString(source.Commit) || source.SHA256 != "" || source.Path != "" {
			return fmt.Errorf("invalid git source")
		}
	case ArchiveSource:
		if source.URL == "" || source.Repository == "" || !commitPattern.MatchString(source.Commit) || !digestPattern.MatchString(source.SHA256) || source.Path != "" {
			return fmt.Errorf("invalid archive source")
		}
	case PathSource:
		if source.Path == "" || !filepath.IsAbs(source.Path) || filepath.Clean(source.Path) != source.Path || source.URL != "" || source.Repository != "" || source.Commit != "" || source.SHA256 != "" {
			return fmt.Errorf("invalid path source")
		}
	default:
		return fmt.Errorf("invalid source type")
	}
	return nil
}
func oneVersion(versions map[string]string, kind, id, version string) error {
	key := kind + ":" + id
	if current, exists := versions[key]; exists {
		return fmt.Errorf("%s already has version %s; version %s conflicts", key, current, version)
	}
	versions[key] = version
	return nil
}
func endpointKey(endpoint Endpoint) (string, error) {
	count := 0
	key := ""
	if endpoint.Plugin != nil {
		count++
		key = "plugin:" + endpoint.Plugin.ID + "@" + endpoint.Plugin.Version
	}
	if endpoint.Sidecar != nil {
		count++
		key = "sidecar:" + endpoint.Sidecar.ID + "@" + endpoint.Sidecar.Version
	}
	if endpoint.Kit != nil {
		count++
		key = "kit:" + endpoint.Kit.ID + "@" + endpoint.Kit.Version
	}
	if count != 1 {
		return "", fmt.Errorf("exactly one plugin, sidecar or kit reference required")
	}
	return key, nil
}
