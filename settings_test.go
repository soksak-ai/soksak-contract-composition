package composition

import (
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"
const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testUnit(kind UnitKind, id string) UnitRef {
	return UnitRef{Kind: kind, ID: id, Version: "0.0.1"}
}

func TestSettingsRecordInstalledAndDevelopmentUnits(t *testing.T) {
	settings := Settings{
		Spec: SettingsSpec, Generation: 1,
		Installations: []Installation{
			{
				UnitRef: testUnit(Plugin, "terminal-view"), Mode: Installed,
				InstallPath: "/opt/soksak/terminal-view", Manifest: "soksak-unit.json",
				Source: Source{Type: ArchiveSource, URL: "https://example.invalid/terminal-view.tar.gz", SHA256: testDigest, Repository: "https://github.com/example/terminal-view", Commit: testCommit},
			},
			{
				UnitRef: testUnit(Sidecar, "terminal-state"), Mode: Installed,
				InstallPath: "/opt/soksak/terminal-state", Manifest: "soksak-unit.json",
				Source: Source{Type: GitSource, URL: "https://github.com/example/terminal-state", Commit: testCommit},
			},
		},
		Plugins: []PluginSelection{{Plugin: testUnit(Plugin, "terminal-view"), Enabled: true}},
	}
	if err := ValidateSettings(settings); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsSupportRuntimeUnitKinds(t *testing.T) {
	for _, kind := range []UnitKind{Plugin, Sidecar, Kit} {
		settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{{
			UnitRef: testUnit(kind, "unit-"+string(kind)), Mode: Development,
			InstallPath: "/work/unit-" + string(kind), Manifest: "soksak-unit.json",
			Source: Source{Type: PathSource, Path: "/work/unit-" + string(kind)},
		}}, Plugins: []PluginSelection{}}
		if kind == Plugin {
			settings.Plugins = []PluginSelection{{Plugin: settings.Installations[0].UnitRef, Enabled: true}}
		}
		if err := ValidateSettings(settings); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
	}
	for _, kind := range []UnitKind{"contract", "spec", "service"} {
		settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{{
			UnitRef: testUnit(kind, "unit-"+string(kind)), Mode: Development,
			InstallPath: "/work/unit-" + string(kind), Manifest: "soksak-unit.json",
			Source: Source{Type: PathSource, Path: "/work/unit-" + string(kind)},
		}}}
		if err := ValidateSettings(settings); err == nil {
			t.Errorf("accepted runtime unit kind %s", kind)
		}
	}
}

func TestOnlyPluginsAreActivationRoots(t *testing.T) {
	plugin := testUnit(Plugin, "view")
	sidecar := testUnit(Sidecar, "backend")
	settings := Settings{
		Spec: SettingsSpec, Generation: 1,
		Installations: []Installation{
			{UnitRef: plugin, Mode: Development, InstallPath: "/work/view", Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "/work/view"}},
			{UnitRef: sidecar, Mode: Development, InstallPath: "/work/backend", Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "/work/backend"}},
		},
		Plugins: []PluginSelection{{Plugin: plugin, Enabled: true}},
	}
	if err := ValidateSettings(settings); err != nil {
		t.Fatal(err)
	}
	settings.Plugins = []PluginSelection{{Plugin: sidecar, Enabled: true}}
	if err := ValidateSettings(settings); err == nil {
		t.Fatal("sidecar was accepted as an activation root")
	}
}

func TestDevelopmentModeBlocksUpdater(t *testing.T) {
	installed := Installation{UnitRef: testUnit(Plugin, "installed"), Mode: Installed}
	development := Installation{UnitRef: testUnit(Plugin, "development"), Mode: Development}
	if decision := UpdatePolicy(installed); !decision.Allowed || decision.Status != UpdateAllowed {
		t.Fatalf("installed update policy = %+v", decision)
	}
	if decision := UpdatePolicy(development); decision.Allowed || decision.Status != UpdateDevelopmentUnit {
		t.Fatalf("development update policy = %+v", decision)
	}
}

func TestSettingsRejectRelativePathsAndUnpinnedSources(t *testing.T) {
	cases := []Installation{
		{UnitRef: testUnit(Plugin, "relative"), Mode: Development, InstallPath: "relative", Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "relative"}},
		{UnitRef: testUnit(Sidecar, "git"), Mode: Installed, InstallPath: "/opt/git", Manifest: "soksak-unit.json", Source: Source{Type: GitSource, URL: "https://github.com/example/git", Commit: "main"}},
		{UnitRef: testUnit(Kit, "archive"), Mode: Installed, InstallPath: "/opt/archive", Manifest: "soksak-unit.json", Source: Source{Type: ArchiveSource, URL: "https://example.invalid/a.zip"}},
		{UnitRef: testUnit(Plugin, "path"), Mode: Development, InstallPath: "/work/path", Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "relative"}},
	}
	for _, install := range cases {
		plugins := []PluginSelection{}
		if install.Kind == Plugin {
			plugins = []PluginSelection{{Plugin: install.UnitRef, Enabled: true}}
		}
		if err := ValidateSettings(Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{install}, Plugins: plugins}); err == nil {
			t.Errorf("accepted invalid installation: %+v", install)
		}
	}
}

func TestModeChangesPreserveAcquisitionSource(t *testing.T) {
	for _, source := range []Source{
		{Type: GitSource, URL: "https://github.com/example/demo", Commit: testCommit},
		{Type: ArchiveSource, URL: "https://example.invalid/demo.tgz", SHA256: testDigest, Repository: "https://github.com/example/demo", Commit: testCommit},
		{Type: PathSource, Path: "/work/demo"},
	} {
		for _, mode := range []UnitMode{Installed, Development} {
			install := Installation{
				UnitRef: testUnit(Plugin, "demo"), Mode: mode, InstallPath: "/work/demo",
				Manifest: "soksak-unit.json", Source: source,
			}
			if err := ValidateSettings(Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{install}, Plugins: []PluginSelection{{Plugin: install.UnitRef, Enabled: true}}}); err != nil {
				t.Errorf("%s %s: %v", source.Type, mode, err)
			}
		}
	}
}

func TestSettingsJSONIsStrictAndExactVersioned(t *testing.T) {
	unknown := `{"spec":"soksak-spec-composition@0.0.1","generation":1,"installations":[],"plugins":[],"bindings":[],"fallback":true}`
	if _, err := ParseSettings([]byte(unknown)); err == nil {
		t.Fatal("settings accepted an unknown fallback field")
	}
	rangeVersion := `{"spec":"soksak-spec-composition@0.0.1","generation":1,"installations":[{"kind":"plugin","id":"demo","version":"^0.0.1","mode":"development","installPath":"/work/demo","manifest":"soksak-unit.json","source":{"type":"path","path":"/work/demo"}}],"plugins":[],"bindings":[]}`
	if _, err := ParseSettings([]byte(rangeVersion)); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("range version error = %v", err)
	}
}
