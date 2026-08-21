package composition

type Status string

const (
	Resolved Status = "resolved"
	Disabled Status = "disabled"
	Rejected Status = "rejected"
)

type PluginState struct {
	Plugin Plugin   `json:"plugin"`
	Status Status   `json:"status"`
	Issues []string `json:"issues"`
}
type SidecarState struct {
	Sidecar Sidecar  `json:"sidecar"`
	Status  Status   `json:"status"`
	Issues  []string `json:"issues"`
}
type KitState struct {
	Kit    Kit      `json:"kit"`
	Status Status   `json:"status"`
	Issues []string `json:"issues"`
}
type Graph struct {
	Plugins  []PluginState  `json:"plugins"`
	Sidecars []SidecarState `json:"sidecars"`
	Kits     []KitState     `json:"kits"`
	Bindings []Binding      `json:"bindings"`
}

func Resolve(settings Settings) (Graph, error) {
	if err := ValidateSettings(settings); err != nil {
		return Graph{}, err
	}
	graph := Graph{Plugins: []PluginState{}, Sidecars: []SidecarState{}, Kits: []KitState{}, Bindings: append([]Binding(nil), settings.Bindings...)}
	enabled := map[string]bool{}
	for _, value := range settings.Sidecars {
		status := Resolved
		if !value.Enabled {
			status = Disabled
		}
		graph.Sidecars = append(graph.Sidecars, SidecarState{Sidecar: value, Status: status, Issues: []string{}})
		enabled["sidecar:"+value.ID+"@"+value.Version] = value.Enabled
	}
	for _, value := range settings.Kits {
		status := Resolved
		if !value.Enabled {
			status = Disabled
		}
		graph.Kits = append(graph.Kits, KitState{Kit: value, Status: status, Issues: []string{}})
		enabled["kit:"+value.ID+"@"+value.Version] = value.Enabled
	}
	for _, value := range settings.Plugins {
		status := Resolved
		issues := []string{}
		if !value.Enabled {
			status = Disabled
		}
		if status == Resolved {
			for _, binding := range settings.Bindings {
				if binding.Consumer.Plugin != nil && *binding.Consumer.Plugin == value.PluginRef {
					provider, _ := endpointKey(binding.Provider)
					if active, known := enabled[provider]; !known || !active {
						status = Rejected
						issues = append(issues, "binding provider is absent or disabled: "+provider)
					}
				}
			}
		}
		graph.Plugins = append(graph.Plugins, PluginState{Plugin: value, Status: status, Issues: issues})
	}
	return graph, nil
}
