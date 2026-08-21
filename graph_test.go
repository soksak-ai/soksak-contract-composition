package composition

import "testing"

func TestGraphKeepsKindsSeparateAndRejectsDisabledProvider(t *testing.T) {
	plugin := Plugin{PluginRef: PluginRef{ID: "p", Version: "0.0.1"}, Enabled: true, InstallPath: "/p", Manifest: "plugin.json", Source: pathSource("/p")}
	sidecar := Sidecar{SidecarRef: SidecarRef{ID: "s", Version: "0.0.1"}, Enabled: false, InstallPath: "/s", Manifest: "sidecar.json", Source: pathSource("/s")}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Plugins: []Plugin{plugin}, Sidecars: []Sidecar{sidecar}, Kits: []Kit{}, Bindings: []Binding{{Consumer: Endpoint{Plugin: &plugin.PluginRef}, Requirement: "state", Provider: Endpoint{Sidecar: &sidecar.SidecarRef}}}}
	graph, err := Resolve(settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Plugins) != 1 || len(graph.Sidecars) != 1 || len(graph.Kits) != 0 {
		t.Fatalf("graph=%+v", graph)
	}
	if graph.Plugins[0].Status != Rejected || graph.Sidecars[0].Status != Disabled {
		t.Fatalf("graph=%+v", graph)
	}
}
func TestGraphDoesNotInferBindings(t *testing.T) {
	plugin := Plugin{PluginRef: PluginRef{ID: "p", Version: "0.0.1"}, Enabled: true, InstallPath: "/p", Manifest: "plugin.json", Source: pathSource("/p")}
	sidecar := Sidecar{SidecarRef: SidecarRef{ID: "s", Version: "0.0.1"}, Enabled: true, InstallPath: "/s", Manifest: "sidecar.json", Source: pathSource("/s")}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Plugins: []Plugin{plugin}, Sidecars: []Sidecar{sidecar}, Kits: []Kit{}, Bindings: []Binding{}}
	graph, err := Resolve(settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Bindings) != 0 || graph.Plugins[0].Status != Resolved {
		t.Fatalf("graph=%+v", graph)
	}
}
