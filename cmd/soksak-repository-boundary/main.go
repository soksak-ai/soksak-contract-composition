package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	composition "github.com/soksak-ai/soksak-contract-composition"
)

func main() {
	root := "."
	if len(os.Args) == 2 {
		root = os.Args[1]
	} else if len(os.Args) > 2 {
		fmt.Fprintln(os.Stderr, "usage: soksak-repository-boundary [root]")
		os.Exit(2)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	findings, err := composition.CheckRepositoryBoundary(absolute)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(findings) == 0 {
		return
	}
	encoder := json.NewEncoder(os.Stderr)
	for _, finding := range findings {
		_ = encoder.Encode(finding)
	}
	os.Exit(1)
}
