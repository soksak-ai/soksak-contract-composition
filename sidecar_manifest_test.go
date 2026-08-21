package composition

import "testing"

func TestSidecarManifestOwnsIdentityInterfaceAndProcess(t *testing.T) {
	body := []byte(`{"spec":"soksak-spec-sidecar@0.0.1","id":"soksak-sidecar-terminal-vt100","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-terminal","version":"0.0.1"},"process":"dist/soksak-sidecar-terminal-vt100"}`)
	manifest, err := ParseSidecarManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Spec != SidecarManifestSpec || manifest.Process == "" {
		t.Fatalf("manifest=%+v", manifest)
	}
}

func TestSidecarManifestRejectsLegacyAndUnsafeShapes(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`{"id":"s","version":"0.0.1","interface":{"id":"i","version":"0.0.1"},"process":"dist/s"}`),
		[]byte(`{"spec":"soksak-spec-sidecar@0.0.1","id":"s","version":"0.0.1","interface":{"id":"i","version":"0.0.1"},"process":"../s"}`),
	} {
		if _, err := ParseSidecarManifest(body); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
