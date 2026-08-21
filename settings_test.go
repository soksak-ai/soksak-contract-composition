package composition

import "testing"

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func pathSource(path string) Source { return Source{Type: PathSource, Path: path} }

func TestSettingsExposePluginsSidecarsAndKitsSeparately(t *testing.T) {
	settings := Settings{Spec: SettingsSpec, Generation: 1, Plugins: []Plugin{{PluginRef: PluginRef{ID: "view", Version: "0.0.1"}, Enabled: true, Development: true, InstallPath: "/work/view", Manifest: "plugin.json", Source: pathSource("/work/view")}}, Sidecars: []Sidecar{{SidecarRef: SidecarRef{ID: "state", Version: "0.0.1"}, Enabled: true, Development: false, InstallPath: "/opt/state", Manifest: "release/sidecar.json", Source: Source{Type: GitSource, URL: "https://github.com/example/state", Commit: testCommit}}}, Kits: []Kit{}, Bindings: []Binding{}}
	if err := ValidateSettings(settings); err != nil {
		t.Fatal(err)
	}
}
func TestDevelopmentIsBooleanAndIndependentFromSource(t *testing.T) {
	for _, development := range []bool{false, true} {
		value := Plugin{PluginRef: PluginRef{ID: "view", Version: "0.0.1"}, Enabled: true, Development: development, InstallPath: "/work/view", Manifest: "plugin.json", Source: pathSource("/source/view")}
		settings := Settings{Spec: SettingsSpec, Generation: 1, Plugins: []Plugin{value}, Sidecars: []Sidecar{}, Kits: []Kit{}, Bindings: []Binding{}}
		if err := ValidateSettings(settings); err != nil {
			t.Fatal(err)
		}
		decision := PluginUpdatePolicy(value)
		if decision.Allowed == development {
			t.Fatalf("decision=%+v", decision)
		}
	}
}
func TestEachKindHasItsOwnDevelopmentAndEnabledState(t *testing.T) {
	plugin := Plugin{PluginRef: PluginRef{ID: "p", Version: "0.0.1"}, Development: true, InstallPath: "/p", Manifest: "plugin.json", Source: pathSource("/p")}
	sidecar := Sidecar{SidecarRef: SidecarRef{ID: "s", Version: "0.0.1"}, Development: true, InstallPath: "/s", Manifest: "sidecar.json", Source: pathSource("/s")}
	kit := Kit{KitRef: KitRef{ID: "k", Version: "0.0.1"}, Development: true, InstallPath: "/k", Manifest: "package.json", Source: pathSource("/k")}
	if PluginUpdatePolicy(plugin).Allowed || SidecarUpdatePolicy(sidecar).Allowed || KitUpdatePolicy(kit).Allowed {
		t.Fatal("development component was updateable")
	}
}
func TestSettingsRejectDuplicateVersionsAndAmbiguousBindingEndpoints(t *testing.T) {
	one := Plugin{PluginRef: PluginRef{ID: "p", Version: "0.0.1"}, InstallPath: "/one", Manifest: "plugin.json", Source: pathSource("/one")}
	two := one
	two.Version = "0.0.2"
	two.InstallPath = "/two"
	settings := Settings{Spec: SettingsSpec, Generation: 1, Plugins: []Plugin{one, two}, Sidecars: []Sidecar{}, Kits: []Kit{}, Bindings: []Binding{}}
	if err := ValidateSettings(settings); err == nil {
		t.Fatal("two plugin versions were accepted")
	}
	settings.Plugins = []Plugin{one}
	settings.Bindings = []Binding{{Consumer: Endpoint{Plugin: &one.PluginRef, Sidecar: &SidecarRef{ID: "s", Version: "0.0.1"}}, Requirement: "state", Provider: Endpoint{Sidecar: &SidecarRef{ID: "s", Version: "0.0.1"}}}}
	if err := ValidateSettings(settings); err == nil {
		t.Fatal("ambiguous consumer was accepted")
	}
}
