package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewestVSIX(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"docs-agent-0.3.4.vsix", "docs-agent-0.3.5.vsix"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := newestVSIX(directory)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "docs-agent-0.3.5.vsix" {
		t.Fatalf("newest VSIX = %s", got)
	}
}

func TestNewestVSIXRequiresArtifact(t *testing.T) {
	if _, err := newestVSIX(t.TempDir()); err == nil {
		t.Fatal("expected missing VSIX error")
	}
}
