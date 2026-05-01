package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentMD_NoFiles(t *testing.T) {
	dir := t.TempDir()
	got := LoadAgentMD(dir)
	if got != "" {
		t.Errorf("expected empty string with no AGENT.md files, got %q", got)
	}
}

func TestLoadAgentMD_SingleFile(t *testing.T) {
	dir := t.TempDir()
	content := "Always use Go style comments."
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadAgentMD(dir)
	if got == "" {
		t.Fatal("expected non-empty result")
	}
	if !contains(got, "<agent-md") {
		t.Errorf("expected <agent-md> tag, got %q", got)
	}
	if !contains(got, content) {
		t.Errorf("expected content %q in result, got %q", content, got)
	}
	if !contains(got, `path="AGENT.md"`) {
		t.Errorf("expected relative path in result, got %q", got)
	}
}

func TestLoadAgentMD_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("  \n\n  "), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadAgentMD(dir)
	if got != "" {
		t.Errorf("expected empty string for whitespace-only AGENT.md, got %q", got)
	}
}

func TestLoadAgentMD_Hierarchy(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "project")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	rootContent := "# Root instructions\nUse tabs for indentation."
	projectContent := "# Project instructions\nAlways run tests before committing."

	if err := os.WriteFile(filepath.Join(root, "AGENT.md"), []byte(rootContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "AGENT.md"), []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadAgentMD(subdir)
	if got == "" {
		t.Fatal("expected non-empty result")
	}

	// Root content should appear before project content (root-to-leaf order)
	rootIdx := indexOf(got, rootContent)
	projectIdx := indexOf(got, projectContent)
	if rootIdx == -1 || projectIdx == -1 {
		t.Fatalf("expected both contents in result, got %q", got)
	}
	if rootIdx >= projectIdx {
		t.Errorf("expected root content before project content, root at %d, project at %d", rootIdx, projectIdx)
	}
}

func TestLoadAgentMD_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	// Create a file larger than the max size
	bigContent := make([]byte, maxAgentMDSize+1000)
	for i := range bigContent {
		bigContent[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), bigContent, 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadAgentMD(dir)
	if got == "" {
		t.Fatal("expected non-empty result")
	}
	// Should be truncated to maxAgentMDSize
	tagOverhead := len("\n\n<agent-md path=\"AGENT.md\">\n\n</agent-md>")
	if len(got) > maxAgentMDSize+tagOverhead {
		t.Errorf("result too long: got %d bytes, expected at most %d", len(got), maxAgentMDSize+tagOverhead)
	}
}

func TestDiscoverAgentMDPaths_WalksUp(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create AGENT.md at root and at subdir
	if err := os.WriteFile(filepath.Join(root, "AGENT.md"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "AGENT.md"), []byte("subdir"), 0644); err != nil {
		t.Fatal(err)
	}

	paths := discoverAgentMDPaths(subdir)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	// Root-to-leaf order: root first, subdir second
	if filepath.Base(paths[0]) != "AGENT.md" || filepath.Dir(paths[0]) != root {
		t.Errorf("expected first path in root, got %s", paths[0])
	}
	if filepath.Dir(paths[1]) != subdir {
		t.Errorf("expected second path in subdir, got %s", paths[1])
	}
}

func TestDiscoverAgentMDPaths_NoFiles(t *testing.T) {
	dir := t.TempDir()
	paths := discoverAgentMDPaths(dir)
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %v", paths)
	}
}

// helpers

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}