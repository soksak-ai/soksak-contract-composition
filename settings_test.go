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
				UnitRef: testUnit(Plugin, "terminal-view"), Mode: Installed, Enabled: true,
				InstallPath: "/opt/soksak/terminal-view", Manifest: "soksak-unit.json",
				Source: Source{Type: ArchiveSource, URL: "https://example.invalid/terminal-view.tar.gz", SHA256: testDigest},
			},
			{
				UnitRef: testUnit(Sidecar, "terminal-state"), Mode: Installed, Enabled: true,
				InstallPath: "/opt/soksak/terminal-state", Manifest: "soksak-unit.json",
				Source: Source{Type: GitSource, URL: "https://github.com/example/terminal-state", Commit: testCommit},
			},
			{
				UnitRef: testUnit(Contract, "terminal-contract"), Mode: Development, Enabled: true,
				InstallPath: "/work/terminal-contract", Manifest: "soksak-unit.json",
				Source: Source{Type: PathSource, Path: "/work/terminal-contract"},
			},
		},
	}
	if err := ValidateSettings(settings); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsSupportEveryUnitKind(t *testing.T) {
	for _, kind := range []UnitKind{Plugin, Sidecar, Kit, Contract, Spec, Service} {
		settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{{
			UnitRef: testUnit(kind, "unit-"+string(kind)), Mode: Development, Enabled: true,
			InstallPath: "/work/unit-" + string(kind), Manifest: "soksak-unit.json",
			Source: Source{Type: PathSource, Path: "/work/unit-" + string(kind)},
		}}}
		if err := ValidateSettings(settings); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
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
		{UnitRef: testUnit(Contract, "path"), Mode: Development, InstallPath: "/work/path", Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "relative"}},
	}
	for _, install := range cases {
		if err := ValidateSettings(Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{install}}); err == nil {
			t.Errorf("accepted invalid installation: %+v", install)
		}
	}
}

func TestModeAndSourceMustAgree(t *testing.T) {
	developmentArchive := Installation{
		UnitRef: testUnit(Plugin, "dev-archive"), Mode: Development, InstallPath: "/work/dev-archive",
		Manifest: "soksak-unit.json", Source: Source{Type: ArchiveSource, URL: "https://example.invalid/a.tgz", SHA256: testDigest},
	}
	installedPath := Installation{
		UnitRef: testUnit(Plugin, "installed-path"), Mode: Installed, InstallPath: "/work/installed-path",
		Manifest: "soksak-unit.json", Source: Source{Type: PathSource, Path: "/work/installed-path"},
	}
	for _, install := range []Installation{developmentArchive, installedPath} {
		if err := ValidateSettings(Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{install}}); err == nil {
			t.Errorf("accepted mismatched mode and source: %+v", install)
		}
	}
}

func TestSettingsJSONIsStrictAndExactVersioned(t *testing.T) {
	unknown := `{"spec":"soksak-spec-composition@0.0.1","generation":1,"installations":[],"bindings":[],"fallback":true}`
	if _, err := ParseSettings([]byte(unknown)); err == nil {
		t.Fatal("settings accepted an unknown fallback field")
	}
	rangeVersion := `{"spec":"soksak-spec-composition@0.0.1","generation":1,"installations":[{"kind":"plugin","id":"demo","version":"^0.0.1","mode":"development","enabled":true,"installPath":"/work/demo","manifest":"soksak-unit.json","source":{"type":"path","path":"/work/demo"}}],"bindings":[]}`
	if _, err := ParseSettings([]byte(rangeVersion)); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("range version error = %v", err)
	}
}
