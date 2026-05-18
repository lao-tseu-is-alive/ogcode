package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMemoryMD_NoFiles(t *testing.T) {
	dir := t.TempDir()
	got := LoadMemoryMD(dir)
	if got != "" {
		t.Errorf("expected empty string with no MEMORY.md files, got %q", got)
	}
}

func TestLoadMemoryMD_SingleFile(t *testing.T) {
	dir := t.TempDir()
	content := "Important decisions and patterns for this project."
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadMemoryMD(dir)
	if got == "" {
		t.Fatal("expected non-empty result")
	}
	if !contains(got, "<memory-md") {
		t.Errorf("expected <memory-md> tag, got %q", got)
	}
	if !contains(got, content) {
		t.Errorf("expected content %q in result, got %q", content, got)
	}
	if !contains(got, `path="MEMORY.md"`) {
		t.Errorf("expected relative path in result, got %q", got)
	}
}

func TestLoadMemoryMD_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("  \n\n  "), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadMemoryMD(dir)
	if got != "" {
		t.Errorf("expected empty string for whitespace-only MEMORY.md, got %q", got)
	}
}

func TestLoadMemoryMD_Hierarchy(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "project")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	rootContent := "# Root memory\nWe use Go for backend."
	projectContent := "# Project memory\nThis project uses React."

	if err := os.WriteFile(filepath.Join(root, "MEMORY.md"), []byte(rootContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "MEMORY.md"), []byte(projectContent), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadMemoryMD(subdir)
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

func TestLoadMemoryMD_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	// Create a file larger than the max size
	bigContent := make([]byte, maxMemoryMDSize+1000)
	for i := range bigContent {
		bigContent[i] = 'a'
	}
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), bigContent, 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadMemoryMD(dir)
	if got == "" {
		t.Fatal("expected non-empty result")
	}
	// Should be truncated to maxMemoryMDSize
	tagOverhead := len("\n\n<memory-md path=\"MEMORY.md\">\n\n</memory-md>")
	if len(got) > maxMemoryMDSize+tagOverhead {
		t.Errorf("result too long: got %d bytes, expected at most %d", len(got), maxMemoryMDSize+tagOverhead)
	}
}

func TestDiscoverMemoryMDPaths_WalksUp(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create MEMORY.md at root and at subdir
	if err := os.WriteFile(filepath.Join(root, "MEMORY.md"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "MEMORY.md"), []byte("subdir"), 0644); err != nil {
		t.Fatal(err)
	}

	paths := discoverMemoryMDPaths(subdir)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	// Root-to-leaf order: root first, subdir second
	if filepath.Base(paths[0]) != "MEMORY.md" || filepath.Dir(paths[0]) != root {
		t.Errorf("expected first path in root, got %s", paths[0])
	}
	if filepath.Dir(paths[1]) != subdir {
		t.Errorf("expected second path in subdir, got %s", paths[1])
	}
}

func TestDiscoverMemoryMDPaths_NoFiles(t *testing.T) {
	dir := t.TempDir()
	paths := discoverMemoryMDPaths(dir)
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %v", paths)
	}
}