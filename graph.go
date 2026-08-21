package composition

import (
	"fmt"
	"sort"
	"strings"
)

type NodeStatus string

const (
	Resolved NodeStatus = "resolved"
	Rejected NodeStatus = "rejected"
	Disabled NodeStatus = "disabled"
)

type GraphNode struct {
	UnitRef
	Mode        UnitMode   `json:"mode"`
	InstallPath string     `json:"installPath"`
	Status      NodeStatus `json:"status"`
}

type EdgeKind string

const (
	DependencyEdge EdgeKind = "dependency"
	BindingEdge    EdgeKind = "binding"
)

type GraphEdge struct {
	From        UnitRef      `json:"from"`
	To          UnitRef      `json:"to"`
	Kind        EdgeKind     `json:"kind"`
	Requirement string       `json:"requirement,omitempty"`
	Contract    *ContractRef `json:"contract,omitempty"`
}

type GraphIssue struct {
	Unit    UnitRef `json:"unit"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
}

type Graph struct {
	Nodes  []GraphNode  `json:"nodes"`
	Edges  []GraphEdge  `json:"edges"`
	Issues []GraphIssue `json:"issues"`
}

func Resolve(settings Settings, manifests map[string]UnitManifest) (Graph, error) {
	if err := ValidateSettings(settings); err != nil {
		return Graph{}, err
	}
	installations := make(map[string]Installation, len(settings.Installations))
	for _, installation := range settings.Installations {
		installations[installation.UnitRef.Key()] = installation
	}
	graph := Graph{Nodes: make([]GraphNode, 0, len(settings.Installations)), Edges: []GraphEdge{}, Issues: []GraphIssue{}}
	status := make(map[string]NodeStatus, len(settings.Installations))
	for _, installation := range settings.Installations {
		current := Resolved
		if !installation.Enabled {
			current = Disabled
		}
		manifest, found := manifests[installation.UnitRef.Key()]
		if installation.Enabled && !found {
			current = Rejected
			graph.Issues = append(graph.Issues, issue(installation.UnitRef, "manifest-missing", "unit manifest is missing"))
		} else if installation.Enabled {
			if err := ValidateUnitManifest(manifest); err != nil {
				current = Rejected
				graph.Issues = append(graph.Issues, issue(installation.UnitRef, "manifest-invalid", err.Error()))
			} else if manifest.UnitRef != installation.UnitRef {
				current = Rejected
				graph.Issues = append(graph.Issues, issue(installation.UnitRef, "manifest-identity", "unit manifest identity does not match settings"))
			}
		}
		status[installation.UnitRef.Key()] = current
		graph.Nodes = append(graph.Nodes, GraphNode{UnitRef: installation.UnitRef, Mode: installation.Mode, InstallPath: installation.InstallPath, Status: current})
	}
	bindingIndex := map[string]Binding{}
	for index, binding := range settings.Bindings {
		if err := validateRef(binding.Consumer); err != nil {
			return Graph{}, fmt.Errorf("composition binding %d consumer: %w", index, err)
		}
		if err := validateRef(binding.Provider); err != nil {
			return Graph{}, fmt.Errorf("composition binding %d provider: %w", index, err)
		}
		if !requirementPattern.MatchString(binding.Requirement) {
			return Graph{}, fmt.Errorf("composition binding %d: invalid requirement", index)
		}
		key := binding.Consumer.Key() + ":" + binding.Requirement
		if _, duplicate := bindingIndex[key]; duplicate {
			return Graph{}, fmt.Errorf("composition binding %d: duplicate %s", index, key)
		}
		bindingIndex[key] = binding
	}
	for _, installation := range settings.Installations {
		key := installation.UnitRef.Key()
		if status[key] != Resolved {
			continue
		}
		manifest := manifests[key]
		for _, dependency := range manifest.Dependencies {
			dependencyInstall, found := installations[dependency.Key()]
			if !found {
				reject(&graph, status, installation.UnitRef, "dependency-missing", "dependency is not installed: "+dependency.Key())
				continue
			}
			graph.Edges = append(graph.Edges, GraphEdge{From: installation.UnitRef, To: dependency, Kind: DependencyEdge})
			if !dependencyInstall.Enabled || status[dependency.Key()] != Resolved {
				reject(&graph, status, installation.UnitRef, "dependency-unavailable", "dependency is disabled or rejected: "+dependency.Key())
			}
		}
		for _, requirement := range manifest.Consumes {
			binding, found := bindingIndex[key+":"+requirement.Name]
			if !found {
				reject(&graph, status, installation.UnitRef, "binding-missing", "binding is missing for requirement "+requirement.Name)
				continue
			}
			providerInstall, installed := installations[binding.Provider.Key()]
			if !installed {
				reject(&graph, status, installation.UnitRef, "binding-provider-missing", "binding provider is not installed: "+binding.Provider.Key())
				continue
			}
			contractCopy := requirement.Contract
			graph.Edges = append(graph.Edges, GraphEdge{From: installation.UnitRef, To: binding.Provider, Kind: BindingEdge, Requirement: requirement.Name, Contract: &contractCopy})
			if !providerInstall.Enabled || status[binding.Provider.Key()] != Resolved {
				reject(&graph, status, installation.UnitRef, "binding-provider-unavailable", "binding provider is disabled or rejected: "+binding.Provider.Key())
				continue
			}
			if !implements(manifests[binding.Provider.Key()], requirement.Contract) {
				reject(&graph, status, installation.UnitRef, "binding-contract", "binding provider does not implement "+requirement.Contract.ID+"@"+requirement.Contract.Version)
			}
		}
	}
	markCycles(&graph, status)
	propagateRejectedDependencies(&graph, status)
	for index := range graph.Nodes {
		graph.Nodes[index].Status = status[graph.Nodes[index].UnitRef.Key()]
	}
	sortGraph(&graph)
	return graph, nil
}

func issue(unit UnitRef, code, message string) GraphIssue {
	return GraphIssue{Unit: unit, Code: code, Message: message}
}

func reject(graph *Graph, status map[string]NodeStatus, unit UnitRef, code, message string) {
	status[unit.Key()] = Rejected
	graph.Issues = append(graph.Issues, issue(unit, code, message))
}

func implements(manifest UnitManifest, wanted ContractRef) bool {
	for _, provided := range manifest.Implements {
		if provided == wanted {
			return true
		}
	}
	return false
}

func markCycles(graph *Graph, status map[string]NodeStatus) {
	adjacency := map[string][]string{}
	refs := map[string]UnitRef{}
	for _, node := range graph.Nodes {
		refs[node.UnitRef.Key()] = node.UnitRef
	}
	for _, edge := range graph.Edges {
		if edge.Kind == DependencyEdge {
			adjacency[edge.From.Key()] = append(adjacency[edge.From.Key()], edge.To.Key())
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var stack []string
	var visit func(string)
	visit = func(key string) {
		if visited[key] {
			return
		}
		if visiting[key] {
			start := 0
			for index, value := range stack {
				if value == key {
					start = index
					break
				}
			}
			for _, member := range stack[start:] {
				reject(graph, status, refs[member], "dependency-cycle", "dependency cycle detected")
			}
			return
		}
		visiting[key] = true
		stack = append(stack, key)
		for _, next := range adjacency[key] {
			visit(next)
		}
		stack = stack[:len(stack)-1]
		visiting[key] = false
		visited[key] = true
	}
	for key := range refs {
		visit(key)
	}
}

func propagateRejectedDependencies(graph *Graph, status map[string]NodeStatus) {
	changed := true
	for changed {
		changed = false
		for _, edge := range graph.Edges {
			if edge.Kind != DependencyEdge || status[edge.From.Key()] != Resolved || status[edge.To.Key()] == Resolved {
				continue
			}
			reject(graph, status, edge.From, "dependency-rejected", "dependency is not resolved: "+edge.To.Key())
			changed = true
		}
	}
}

func sortGraph(graph *Graph) {
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].UnitRef.Key() < graph.Nodes[j].UnitRef.Key() })
	sort.Slice(graph.Edges, func(i, j int) bool {
		a := graph.Edges[i].From.Key() + ":" + string(graph.Edges[i].Kind) + ":" + graph.Edges[i].Requirement + ":" + graph.Edges[i].To.Key()
		b := graph.Edges[j].From.Key() + ":" + string(graph.Edges[j].Kind) + ":" + graph.Edges[j].Requirement + ":" + graph.Edges[j].To.Key()
		return a < b
	})
	sort.Slice(graph.Issues, func(i, j int) bool {
		a := graph.Issues[i].Unit.Key() + ":" + graph.Issues[i].Code + ":" + graph.Issues[i].Message
		b := graph.Issues[j].Unit.Key() + ":" + graph.Issues[j].Code + ":" + graph.Issues[j].Message
		return strings.Compare(a, b) < 0
	})
}
