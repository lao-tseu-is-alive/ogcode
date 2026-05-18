package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_MemoryMDSection_AlwaysPresent(t *testing.T) {
	agent := BuildAgent
	dir := "/tmp/test"

	// Case 1: No MEMORY.md content — section should still appear
	prompt := buildSystemPrompt(agent, dir, false, "", "", nil)
	if !strings.Contains(prompt, "## MEMORY.md — Project Long-Term Memory") {
		t.Error("expected MEMORY.md section to appear even when memoryMDContent is empty")
	}
	if !strings.Contains(prompt, "No MEMORY.md file was found") {
		t.Error("expected 'No MEMORY.md file was found' message when memoryMDContent is empty")
	}

	// Case 2: With MEMORY.md content — section should appear with file content indicator
	memContent := "\n\n<memory-md path=\"MEMORY.md\">\n# Project Notes\nSome facts.\n</memory-md>"
	prompt = buildSystemPrompt(agent, dir, false, "", memContent, nil)
	if !strings.Contains(prompt, "## MEMORY.md — Project Long-Term Memory") {
		t.Error("expected MEMORY.md section to appear when memoryMDContent is present")
	}
	if !strings.Contains(prompt, "The content above in the <memory-md> tag") {
		t.Error("expected content indicator when memoryMDContent is present")
	}
	if !strings.Contains(prompt, memContent) {
		t.Error("expected memoryMDContent to be included in prompt")
	}
	if strings.Contains(prompt, "No MEMORY.md file was found") {
		t.Error("did not expect 'No MEMORY.md file was found' when memoryMDContent is present")
	}
}

func TestBuildSystemPrompt_MemoryMDSection_ContainsPurposeSection(t *testing.T) {
	agent := BuildAgent
	dir := "/tmp/test"

	prompt := buildSystemPrompt(agent, dir, false, "", "", nil)

	// Verify key sections are always present
	for _, sub := range []string{
		"### Purpose",
		"### What belongs in MEMORY.md",
		"### What does NOT belong in MEMORY.md",
		"### How it differs from AGENT.md and agentic memory",
		"### How to maintain MEMORY.md",
	} {
		if !strings.Contains(prompt, sub) {
			t.Errorf("expected section %q in prompt when no MEMORY.md exists", sub)
		}
	}
}