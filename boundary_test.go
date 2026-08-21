package composition

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBoundaryFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBoundaryRejectsSiblingSourceUseInTestsAndTasks(t *testing.T) {
	root := t.TempDir()
	writeBoundaryFile(t, root, "provider_test.go", "package provider\nconst source = \"../soksak-"+"sidecars/other\"\n")
	writeBoundaryFile(t, root, "Taskfile.yml", "cmds: [cd ../soksak-"+"plugins/other && npm test]\n")
	findings, err := CheckRepositoryBoundary(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestBoundaryRejectsSplitPathConstruction(t *testing.T) {
	root := t.TempDir()
	writeBoundaryFile(t, root, "gate_test.go", "package gate\nvar root = filepath.Join(\"..\", \"soksak-"+"sidecars\", name)\n")
	findings, err := CheckRepositoryBoundary(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Rule != "no-sibling-source" {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestBoundaryAllowsDeclarativeDependenciesAndContractReferences(t *testing.T) {
	root := t.TempDir()
	writeBoundaryFile(t, root, "package.json", "{\"dependencies\":{\"kit\":\"file:../../soksak-"+"kits/kit\"}}")
	writeBoundaryFile(t, root, "provider.go", "package provider\nconst contract = \"soksak-spec-sidecar-terminal@0.0.1\"\n")
	findings, err := CheckRepositoryBoundary(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestBoundaryRejectsSourceSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "source.go")
	if err := os.WriteFile(target, []byte("package source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "source.go")); err != nil {
		t.Fatal(err)
	}
	findings, err := CheckRepositoryBoundary(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Rule != "no-symlink" {
		t.Fatalf("findings = %+v", findings)
	}
}
