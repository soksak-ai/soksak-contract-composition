package composition

import (
	"strings"
	"testing"
)

func contract(id string) ContractRef {
	return ContractRef{ID: id, Version: "0.0.1"}
}

func unitManifest(unit UnitRef) UnitManifest {
	return UnitManifest{
		Spec: UnitSpec, UnitRef: unit,
		Dependencies: []UnitRef{}, Implements: []ContractRef{}, Consumes: []Requirement{},
		Entrypoints: []Entrypoint{{Role: "package", Path: "package.json"}},
	}
}

func devInstall(unit UnitRef, enabled bool) Installation {
	path := "/work/" + unit.ID
	return Installation{
		UnitRef: unit, Mode: Development, InstallPath: path,
		Manifest: UnitManifestFile, Source: Source{Type: PathSource, Path: path},
	}
}

func selections(units ...UnitRef) []PluginSelection {
	result := []PluginSelection{}
	for _, unit := range units {
		if unit.Kind == Plugin {
			result = append(result, PluginSelection{Plugin: unit, Enabled: true})
		}
	}
	return result
}

func TestResolveBuildsExplicitCrossKindEdges(t *testing.T) {
	view := testUnit(Plugin, "terminal-view")
	pty := testUnit(Sidecar, "pty-owner")
	state := testUnit(Sidecar, "terminal-state")
	kit := testUnit(Kit, "terminal-runtime")
	ptyContract := contract("soksak-spec-sidecar-pty")
	stateContract := contract("soksak-spec-sidecar-terminal")
	settings := Settings{
		Spec:          SettingsSpec,
		Generation:    1,
		Installations: []Installation{devInstall(view, true), devInstall(pty, true), devInstall(state, true), devInstall(kit, true)},
		Plugins:       selections(view),
		Bindings: []Binding{
			{Consumer: view, Requirement: "pty", Provider: pty},
			{Consumer: view, Requirement: "state", Provider: state},
		},
	}
	viewManifest := unitManifest(view)
	viewManifest.Dependencies = []UnitRef{kit}
	viewManifest.Consumes = []Requirement{{Name: "pty", Contract: ptyContract}, {Name: "state", Contract: stateContract}}
	ptyManifest := unitManifest(pty)
	ptyManifest.Implements = []ContractRef{ptyContract}
	stateManifest := unitManifest(state)
	stateManifest.Implements = []ContractRef{stateContract}
	graph, err := Resolve(settings, map[string]UnitManifest{
		view.Key(): viewManifest, pty.Key(): ptyManifest, state.Key(): stateManifest, kit.Key(): unitManifest(kit),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 4 || len(graph.Edges) != 3 || len(graph.Issues) != 0 {
		t.Fatalf("graph = %+v", graph)
	}
	for _, node := range graph.Nodes {
		if node.Status != Resolved {
			t.Errorf("%s status = %s", node.UnitRef.Key(), node.Status)
		}
	}
}

func TestResolveRejectsOnlyTheConsumerWithAMissingBinding(t *testing.T) {
	consumer := testUnit(Plugin, "consumer")
	unrelated := testUnit(Kit, "unrelated")
	wanted := contract("soksak-spec-sidecar-demo")
	consumerManifest := unitManifest(consumer)
	consumerManifest.Consumes = []Requirement{{Name: "backend", Contract: wanted}}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{devInstall(consumer, true), devInstall(unrelated, true)}, Plugins: selections(consumer)}
	graph, err := Resolve(settings, map[string]UnitManifest{consumer.Key(): consumerManifest, unrelated.Key(): unitManifest(unrelated)})
	if err != nil {
		t.Fatal(err)
	}
	if statusOf(t, graph, consumer) != Rejected {
		t.Fatal("consumer was not rejected")
	}
	if statusOf(t, graph, unrelated) != Resolved {
		t.Fatal("unrelated unit was affected")
	}
	if !issueContains(graph, consumer, "binding") {
		t.Fatalf("issues = %+v", graph.Issues)
	}
}

func TestResolveRejectsAContractMismatchWithoutFallback(t *testing.T) {
	consumer := testUnit(Plugin, "consumer")
	provider := testUnit(Sidecar, "provider")
	wanted := contract("soksak-spec-sidecar-wanted")
	consumerManifest := unitManifest(consumer)
	consumerManifest.Consumes = []Requirement{{Name: "backend", Contract: wanted}}
	providerManifest := unitManifest(provider)
	providerManifest.Implements = []ContractRef{contract("soksak-spec-sidecar-other")}
	settings := Settings{
		Spec:          SettingsSpec,
		Generation:    1,
		Installations: []Installation{devInstall(consumer, true), devInstall(provider, true)},
		Plugins:       selections(consumer),
		Bindings:      []Binding{{Consumer: consumer, Requirement: "backend", Provider: provider}},
	}
	graph, err := Resolve(settings, map[string]UnitManifest{consumer.Key(): consumerManifest, provider.Key(): providerManifest})
	if err != nil {
		t.Fatal(err)
	}
	if statusOf(t, graph, consumer) != Rejected || statusOf(t, graph, provider) != Resolved {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
	if !issueContains(graph, consumer, "does not implement") {
		t.Fatalf("issues = %+v", graph.Issues)
	}
}

func TestDisabledPluginLeavesSharedDependenciesResolvedButInactive(t *testing.T) {
	consumer := testUnit(Plugin, "consumer")
	dependency := testUnit(Kit, "dependency")
	consumerManifest := unitManifest(consumer)
	consumerManifest.Dependencies = []UnitRef{dependency}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{devInstall(consumer, false), devInstall(dependency, false)}, Plugins: []PluginSelection{{Plugin: consumer, Enabled: false}}}
	graph, err := Resolve(settings, map[string]UnitManifest{consumer.Key(): consumerManifest, dependency.Key(): unitManifest(dependency)})
	if err != nil {
		t.Fatal(err)
	}
	if statusOf(t, graph, consumer) != Disabled || statusOf(t, graph, dependency) != Resolved {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
	if activeOf(t, graph, consumer) || activeOf(t, graph, dependency) {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
}

func TestActivePluginsShareOneExactDependencyNode(t *testing.T) {
	one := testUnit(Plugin, "one")
	two := testUnit(Plugin, "two")
	shared := testUnit(Kit, "shared")
	oneManifest := unitManifest(one)
	oneManifest.Dependencies = []UnitRef{shared}
	twoManifest := unitManifest(two)
	twoManifest.Dependencies = []UnitRef{shared}
	settings := Settings{
		Spec: SettingsSpec, Generation: 1,
		Installations: []Installation{devInstall(one, true), devInstall(two, true), devInstall(shared, true)},
		Plugins:       selections(one, two), Bindings: []Binding{},
	}
	graph, err := Resolve(settings, map[string]UnitManifest{one.Key(): oneManifest, two.Key(): twoManifest, shared.Key(): unitManifest(shared)})
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 3 || len(graph.Edges) != 2 {
		t.Fatalf("graph = %+v", graph)
	}
	if !activeOf(t, graph, one) || !activeOf(t, graph, two) || !activeOf(t, graph, shared) {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
}

func TestSharedDependencyActivityIsDerivedFromPluginRoots(t *testing.T) {
	one := testUnit(Plugin, "one")
	two := testUnit(Plugin, "two")
	shared := testUnit(Sidecar, "shared")
	oneManifest := unitManifest(one)
	oneManifest.Dependencies = []UnitRef{shared}
	twoManifest := unitManifest(two)
	twoManifest.Dependencies = []UnitRef{shared}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{devInstall(one, true), devInstall(two, true), devInstall(shared, true)}, Plugins: []PluginSelection{{Plugin: one, Enabled: false}, {Plugin: two, Enabled: true}}, Bindings: []Binding{}}
	graph, err := Resolve(settings, map[string]UnitManifest{one.Key(): oneManifest, two.Key(): twoManifest, shared.Key(): unitManifest(shared)})
	if err != nil {
		t.Fatal(err)
	}
	if activeOf(t, graph, one) || !activeOf(t, graph, two) || !activeOf(t, graph, shared) {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
	settings.Plugins[1].Enabled = false
	graph, err = Resolve(settings, map[string]UnitManifest{one.Key(): oneManifest, two.Key(): twoManifest, shared.Key(): unitManifest(shared)})
	if err != nil {
		t.Fatal(err)
	}
	if activeOf(t, graph, shared) {
		t.Fatalf("shared dependency remained active: %+v", graph.Nodes)
	}
}

func TestResolveRejectsCycleMembersAndKeepsUnrelatedUnits(t *testing.T) {
	a := testUnit(Kit, "a")
	b := testUnit(Kit, "b")
	c := testUnit(Kit, "c")
	aManifest := unitManifest(a)
	aManifest.Dependencies = []UnitRef{b}
	bManifest := unitManifest(b)
	bManifest.Dependencies = []UnitRef{a}
	settings := Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{devInstall(a, true), devInstall(b, true), devInstall(c, true)}, Plugins: []PluginSelection{}}
	graph, err := Resolve(settings, map[string]UnitManifest{a.Key(): aManifest, b.Key(): bManifest, c.Key(): unitManifest(c)})
	if err != nil {
		t.Fatal(err)
	}
	if statusOf(t, graph, a) != Rejected || statusOf(t, graph, b) != Rejected || statusOf(t, graph, c) != Resolved {
		t.Fatalf("nodes = %+v", graph.Nodes)
	}
	if !issueContains(graph, a, "cycle") || !issueContains(graph, b, "cycle") {
		t.Fatalf("issues = %+v", graph.Issues)
	}
}

func TestUnitManifestJSONRejectsRangesAndUnknownFields(t *testing.T) {
	rangeDependency := `{"spec":"soksak-spec-unit@0.0.1","kind":"kit","id":"demo","version":"0.0.1","dependencies":[{"kind":"kit","id":"dep","version":"^0.0.1"}],"implements":[],"consumes":[],"entrypoints":[{"role":"package","path":"package.json"}]}`
	if _, err := ParseUnitManifest([]byte(rangeDependency)); err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("range dependency error = %v", err)
	}
	unknown := `{"spec":"soksak-spec-unit@0.0.1","kind":"kit","id":"demo","version":"0.0.1","dependencies":[],"implements":[],"consumes":[],"entrypoints":[{"role":"package","path":"package.json"}],"fallback":true}`
	if _, err := ParseUnitManifest([]byte(unknown)); err == nil {
		t.Fatal("manifest accepted fallback")
	}
}

func statusOf(t *testing.T, graph Graph, unit UnitRef) NodeStatus {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.UnitRef == unit {
			return node.Status
		}
	}
	t.Fatalf("node not found: %s", unit.Key())
	return ""
}

func issueContains(graph Graph, unit UnitRef, text string) bool {
	for _, issue := range graph.Issues {
		if issue.Unit == unit && strings.Contains(issue.Message, text) {
			return true
		}
	}
	return false
}

func activeOf(t *testing.T, graph Graph, unit UnitRef) bool {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.UnitRef == unit {
			return node.Active
		}
	}
	t.Fatalf("node not found: %s", unit.Key())
	return false
}
