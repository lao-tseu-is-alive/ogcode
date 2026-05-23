package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt_MemoryMDSection_AlwaysPresent(t *testing.T) {
	agent := BuildAgent
	dir := "/tmp/test"

	// Case 1: No MEMORY.md content — section should still appear
	prompt := buildSystemPrompt(agent, dir, false, false, "", "", nil)
	if !strings.Contains(prompt, "## MEMORY.md — Project Long-Term Memory") {
		t.Error("expected MEMORY.md section to appear even when memoryMDContent is empty")
	}
	if !strings.Contains(prompt, "No MEMORY.md file was found") {
		t.Error("expected 'No MEMORY.md file was found' message when memoryMDContent is empty")
	}

	// Case 2: With MEMORY.md content — section should appear with file content indicator
	memContent := "\n\n<memory-md path=\"MEMORY.md\">\n# Project Notes\nSome facts.\n</memory-md>"
	prompt = buildSystemPrompt(agent, dir, false, false, "", memContent, nil)
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

func TestBuildSystemPrompt_MemoryMDSection_RoleAware(t *testing.T) {
	dir := "/tmp/test"

	// BuildAgent has write and edit tools — should get read/write instructions
	buildPrompt := buildSystemPrompt(BuildAgent, dir, false, "", "", nil)
	if !strings.Contains(buildPrompt, "### How to maintain MEMORY.md") {
		t.Error("expected 'How to maintain' heading for BuildAgent (has write tools)")
	}
	if !strings.Contains(buildPrompt, "Use the edit tool for targeted updates") {
		t.Error("expected 'Use the edit tool' instruction for BuildAgent")
	}
	if !strings.Contains(buildPrompt, "create one in the project root directory") {
		t.Error("expected creation prompt when memoryMDContent is empty and agent can write")
	}

	// PlanAgent has no write/edit tools — should get read-only instructions
	planPrompt := buildSystemPrompt(PlanAgent, dir, false, "", "", nil)
	if !strings.Contains(planPrompt, "### How to use MEMORY.md") {
		t.Error("expected 'How to use' heading for PlanAgent (read-only)")
	}
	if strings.Contains(planPrompt, "Use the edit tool") {
		t.Error("did not expect 'Use the edit tool' for read-only PlanAgent")
	}
	if strings.Contains(planPrompt, "create one in the project root directory") {
		t.Error("did not expect creation prompt for read-only PlanAgent")
	}

	// NoteAgent has no write/edit tools — should get read-only instructions
	notePrompt := buildSystemPrompt(NoteAgent, dir, false, "", "", nil)
	if !strings.Contains(notePrompt, "### How to use MEMORY.md") {
		t.Error("expected 'How to use' heading for NoteAgent (read-only)")
	}
	if strings.Contains(notePrompt, "Use the write tool") {
		t.Error("did not expect 'Use the write tool' for read-only NoteAgent")
	}
}

func TestBuildSystemPrompt_MemoryMDSection_WithContent(t *testing.T) {
	dir := "/tmp/test"
	memContent := "\n\n<memory-md path=\"MEMORY.md\">\n# Project Notes\nSome facts.\n</memory-md>"

	// BuildAgent with MEMORY.md content — should show content but NOT creation prompt
	buildPrompt := buildSystemPrompt(BuildAgent, dir, false, "", memContent, nil)
	if !strings.Contains(buildPrompt, "The content above in the <memory-md> tag") {
		t.Error("expected content indicator when memoryMDContent is present for BuildAgent")
	}
	if strings.Contains(buildPrompt, "No MEMORY.md file was found") {
		t.Error("did not expect 'No MEMORY.md file was found' when memoryMDContent is present for BuildAgent")
	}
	if strings.Contains(buildPrompt, "create one in the project root directory") {
		t.Error("did not expect creation prompt when memoryMDContent is present for BuildAgent")
	}

	// PlanAgent with MEMORY.md content — should show read-only version, no creation prompt
	planPrompt := buildSystemPrompt(PlanAgent, dir, false, "", memContent, nil)
	if !strings.Contains(planPrompt, "The content above in the <memory-md> tag") {
		t.Error("expected content indicator when memoryMDContent is present for PlanAgent")
	}
	if strings.Contains(planPrompt, "create one in the project root directory") {
		t.Error("did not expect creation prompt for read-only PlanAgent even with content")
	}
}