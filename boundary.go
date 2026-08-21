package composition

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BoundaryFinding struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Rule  string `json:"rule"`
	Value string `json:"value"`
}

var executableSuffixes = map[string]bool{
	".c": true, ".cc": true, ".cpp": true, ".go": true, ".h": true, ".hpp": true,
	".js": true, ".mjs": true, ".rs": true, ".sh": true, ".ts": true, ".tsx": true,
	".yaml": true, ".yml": true,
}

var declarativeFiles = map[string]bool{
	"Cargo.lock": true, "Cargo.toml": true, "go.mod": true, "go.sum": true,
	"package.json": true, "plugin.json": true, "pnpm-lock.yaml": true, "release.json": true,
	"settings.json": true, UnitManifestFile: true,
}

var skippedBoundaryTrees = map[string]bool{
	".git": true, ".task": true, "bin": true, "dist": true, "evidence": true,
	"node_modules": true, "target": true,
}

var siblingSourceTokens = []string{
	"soksak-" + "contracts", "soksak-" + "kits", "soksak-" + "plugins",
	"soksak-" + "sidecars", "wails-" + "services",
}

func CheckRepositoryBoundary(root string) ([]BoundaryFinding, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("repository boundary root must be absolute")
	}
	var findings []BoundaryFinding
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != root && skippedBoundaryTrees[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if info.Mode()&os.ModeSymlink != 0 {
			findings = append(findings, BoundaryFinding{Path: relative, Rule: "no-symlink", Value: path})
			return nil
		}
		if declarativeFiles[info.Name()] || !executableSuffixes[filepath.Ext(info.Name())] && info.Name() != "Dockerfile" && !strings.HasPrefix(info.Name(), "Taskfile") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			text := filepath.ToSlash(scanner.Text())
			for _, token := range siblingSourceTokens {
				if strings.Contains(text, token) {
					findings = append(findings, BoundaryFinding{Path: relative, Line: line, Rule: "no-sibling-source", Value: token})
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Value < findings[j].Value
	})
	return findings, nil
}
