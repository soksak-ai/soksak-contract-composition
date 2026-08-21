package composition

import (
	"errors"
	"testing"
)

func emptySettings(generation uint64) Settings {
	return Settings{Spec: SettingsSpec, Generation: generation, Installations: []Installation{}, Plugins: []PluginSelection{}, Bindings: []Binding{}}
}

func TestReplaceRequiresCASAndAdvancesOneGeneration(t *testing.T) {
	next, change, err := Replace(emptySettings(4), emptySettings(5), 4)
	if err != nil {
		t.Fatal(err)
	}
	if next.Generation != 5 || change != (Change{PreviousGeneration: 4, Generation: 5}) {
		t.Fatalf("change = %+v", change)
	}
}

func TestReplaceRejectsConcurrentAndSkippedWrites(t *testing.T) {
	if _, _, err := Replace(emptySettings(4), emptySettings(5), 3); err == nil {
		t.Fatal("stale generation was accepted")
	} else {
		var conflict ErrGenerationConflict
		if !errors.As(err, &conflict) {
			t.Fatalf("error = %T %v", err, err)
		}
	}
	if _, _, err := Replace(emptySettings(4), emptySettings(6), 4); err == nil {
		t.Fatal("generation skip was accepted")
	}
}

func TestCompositionChangeHasOneStableEventName(t *testing.T) {
	if ChangeEvent != "composition.changed" {
		t.Fatalf("event = %q", ChangeEvent)
	}
}

func TestInitializeCreatesOnlyGenerationOne(t *testing.T) {
	settings, change, err := Initialize(Settings{Spec: SettingsSpec, Generation: 1, Installations: []Installation{}, Plugins: []PluginSelection{}, Bindings: []Binding{}})
	if err != nil {
		t.Fatal(err)
	}
	if settings.Generation != 1 || change != (Change{PreviousGeneration: 0, Generation: 1}) {
		t.Fatalf("change = %+v", change)
	}
	if _, _, err := Initialize(emptySettings(2)); err == nil {
		t.Fatal("generation two was accepted as initialization")
	}
}
