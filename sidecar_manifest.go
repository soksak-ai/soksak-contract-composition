package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const SidecarManifestSpec = "soksak-spec-sidecar@0.0.1"

type SidecarInterface struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type SidecarManifest struct {
	Spec      string           `json:"spec"`
	ID        string           `json:"id"`
	Version   string           `json:"version"`
	Interface SidecarInterface `json:"interface"`
	Process   string           `json:"process"`
	Library   []string         `json:"library,omitempty"`
}

func ParseSidecarManifest(body []byte) (SidecarManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var manifest SidecarManifest
	if err := decoder.Decode(&manifest); err != nil {
		return SidecarManifest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return SidecarManifest{}, fmt.Errorf("sidecar manifest has trailing data")
	}
	if manifest.Spec != SidecarManifestSpec || !idPattern.MatchString(manifest.ID) || !versionPattern.MatchString(manifest.Version) || !idPattern.MatchString(manifest.Interface.ID) || !versionPattern.MatchString(manifest.Interface.Version) {
		return SidecarManifest{}, fmt.Errorf("invalid sidecar manifest identity or interface")
	}
	if !safeManifestPath(manifest.Process) {
		return SidecarManifest{}, fmt.Errorf("sidecar process must be a safe relative path")
	}
	for _, library := range manifest.Library {
		if !safeManifestPath(library) {
			return SidecarManifest{}, fmt.Errorf("sidecar library must be a safe relative path")
		}
	}
	return manifest, nil
}

func safeManifestPath(value string) bool {
	return value != "" && !filepath.IsAbs(value) && filepath.Clean(value) == value && value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}
