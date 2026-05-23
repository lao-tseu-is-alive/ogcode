package agent

import (
	"strings"
	"testing"
)

func TestCallGraphPrompt_BuildRole(t *testing.T) {
	prompt := callGraphPrompt("build")

	// Build role should include post-mutation sync section
	if !strings.Contains(prompt, "Post-mutation call graph sync") {
		t.Error("expected build role to include post-mutation sync section")
	}
	if !strings.Contains(prompt, "When to build the call graph") {
		t.Error("expected build role to include 'When to build' section")
	}
	if !strings.Contains(prompt, "Using search instead of grep") {
		t.Error("expected build role to include search guidance section")
	}
	if !strings.Contains(prompt, "Populating the doc field") {
		t.Error("expected build role to include doc field section")
	}
	if !strings.Contains(prompt, "Call graph completeness invariant") {
		t.Error("expected build role to include completeness invariant")
	}
}

func TestCallGraphPrompt_PlanRole(t *testing.T) {
	prompt := callGraphPrompt("plan")

	// Plan role should NOT include post-mutation sync section
	if strings.Contains(prompt, "Post-mutation call graph sync") {
		t.Error("did not expect plan role to include post-mutation sync section")
	}
	if !strings.Contains(prompt, "When to build the call graph") {
		t.Error("expected plan role to include 'When to build' section")
	}
	if !strings.Contains(prompt, "Using search instead of grep") {
		t.Error("expected plan role to include search guidance section")
	}
	if !strings.Contains(prompt, "Populating the doc field") {
		t.Error("expected plan role to include doc field section")
	}
	if !strings.Contains(prompt, "Call graph completeness invariant") {
		t.Error("expected plan role to include completeness invariant")
	}
	// Plan role should mention planning, not implementation
	if !strings.Contains(prompt, "planning modifications") {
		t.Error("expected plan role to mention 'planning modifications'")
	}
}

func TestMemoryMDPrompt_CanWrite(t *testing.T) {
	prompt := memoryMDPrompt(true)

	if !strings.Contains(prompt, "### How to maintain MEMORY.md") {
		t.Error("expected 'How to maintain' heading when canWriteFiles=true")
	}
	if !strings.Contains(prompt, "Use the edit tool for targeted updates") {
		t.Error("expected edit tool mention when canWriteFiles=true")
	}
	if !strings.Contains(prompt, "Use the write tool only") {
		t.Error("expected write tool mention when canWriteFiles=true")
	}
	if strings.Contains(prompt, "you are a read-only agent") {
		t.Error("did not expect read-only wording when canWriteFiles=true")
	}
}

func TestMemoryMDPrompt_ReadOnly(t *testing.T) {
	prompt := memoryMDPrompt(false)

	if !strings.Contains(prompt, "### How to use MEMORY.md") {
		t.Error("expected 'How to use' heading when canWriteFiles=false")
	}
	if strings.Contains(prompt, "Use the edit tool") {
		t.Error("did not expect edit tool mention when canWriteFiles=false")
	}
	if strings.Contains(prompt, "Use the write tool") {
		t.Error("did not expect write tool mention when canWriteFiles=false")
	}
	if !strings.Contains(prompt, "you are a read-only agent") {
		t.Error("expected read-only wording when canWriteFiles=false")
	}
}

func TestMemoryMDPrompt_CommonSections(t *testing.T) {
	// Both variants should include these common sections
	for _, canWrite := range []bool{true, false} {
		prompt := memoryMDPrompt(canWrite)
		for _, sub := range []string{
			"### Purpose",
			"### What belongs in MEMORY.md",
			"### What does NOT belong in MEMORY.md",
			"### How it differs from AGENT.md and agentic memory",
		} {
			if !strings.Contains(prompt, sub) {
				t.Errorf("expected section %q in prompt (canWrite=%v)", sub, canWrite)
			}
		}
	}
}

func TestMarkdownCapabilitiesPrompt(t *testing.T) {
	prompt := markdownCapabilitiesPrompt()
	if !strings.Contains(prompt, "Mermaid diagrams") {
		t.Error("expected Mermaid mention in markdown capabilities prompt")
	}
	if !strings.Contains(prompt, "LaTeX math") {
		t.Error("expected LaTeX mention in markdown capabilities prompt")
	}
}

func TestParallelToolCallsPrompt(t *testing.T) {
	prompt := parallelToolCallsPrompt()
	if !strings.Contains(prompt, "Parallel tool calls") {
		t.Error("expected 'Parallel tool calls' heading")
	}
	if !strings.Contains(prompt, "independent") {
		t.Error("expected 'independent' mention in parallel tool calls prompt")
	}
}

func TestNoPackageManagerDirsPrompt(t *testing.T) {
	prompt := noPackageManagerDirsPrompt()
	if !strings.Contains(prompt, "node_modules") {
		t.Error("expected 'node_modules' mention in no-package-manager-dirs prompt")
	}
	if !strings.Contains(prompt, "third-party code") {
		t.Error("expected 'third-party code' mention")
	}
}

func TestProjectNotesPrompt_CanWrite(t *testing.T) {
	prompt := projectNotesPrompt(true)

	if !strings.Contains(prompt, "## Project notes") {
		t.Error("expected 'Project notes' heading when canWriteFiles=true")
	}
	if !strings.Contains(prompt, ".ogcode/notes/*.md") {
		t.Error("expected '.ogcode/notes/*.md' mention when canWriteFiles=true")
	}
	if !strings.Contains(prompt, "managed exclusively by the NoteAgent") {
		t.Error("expected NoteAgent restriction when canWriteFiles=true")
	}
	if !strings.Contains(prompt, "Do not create, modify, or delete any files in .ogcode/notes/") {
		t.Error("expected explicit read-only restriction for notes dir when canWriteFiles=true")
	}
	if !strings.Contains(prompt, "You may only read notes") {
		t.Error("expected read-only permission wording when canWriteFiles=true")
	}
}

func TestProjectNotesPrompt_ReadOnly(t *testing.T) {
	prompt := projectNotesPrompt(false)

	if !strings.Contains(prompt, "## Project notes") {
		t.Error("expected 'Project notes' heading when canWriteFiles=false")
	}
	if !strings.Contains(prompt, ".ogcode/notes/*.md") {
		t.Error("expected '.ogcode/notes/*.md' mention when canWriteFiles=false")
	}
	// Read-only agents should NOT see the NoteAgent restriction since they can't write anyway
	if strings.Contains(prompt, "managed exclusively by the NoteAgent") {
		t.Error("did not expect NoteAgent restriction when canWriteFiles=false")
	}
	if strings.Contains(prompt, "Do not create, modify, or delete") {
		t.Error("did not expect write restriction wording when canWriteFiles=false (they can't write at all)")
	}
}

func TestProjectNotesPrompt_CommonSections(t *testing.T) {
	// Both variants should include common guidance
	for _, canWrite := range []bool{true, false} {
		prompt := projectNotesPrompt(canWrite)
		for _, sub := range []string{
			"## Project notes",
			".ogcode/notes/",
			"don't repeat what is already documented",
		} {
			if !strings.Contains(prompt, sub) {
				t.Errorf("expected %q in projectNotesPrompt (canWrite=%v)", sub, canWrite)
			}
		}
	}
}

func TestBuildAgent_HasExpectedTools(t *testing.T) {
	if !BuildAgent.HasTool("write") {
		t.Error("BuildAgent should have write tool")
	}
	if !BuildAgent.HasTool("edit") {
		t.Error("BuildAgent should have edit tool")
	}
	if !BuildAgent.HasTool("callgraph") {
		t.Error("BuildAgent should have callgraph tool")
	}
	if !BuildAgent.HasTool("memory_recall") {
		t.Error("BuildAgent should have memory_recall tool")
	}
}

func TestPlanAgent_HasExpectedTools(t *testing.T) {
	if PlanAgent.HasTool("write") {
		t.Error("PlanAgent should not have write tool")
	}
	if PlanAgent.HasTool("edit") {
		t.Error("PlanAgent should not have edit tool")
	}
	if !PlanAgent.HasTool("callgraph") {
		t.Error("PlanAgent should have callgraph tool")
	}
	if !PlanAgent.HasTool("memory_recall") {
		t.Error("PlanAgent should have memory_recall tool")
	}
	if !PlanAgent.HasTool("read") {
		t.Error("PlanAgent should have read tool")
	}
}

func TestNoteAgent_HasExpectedTools(t *testing.T) {
	if NoteAgent.HasTool("write") {
		t.Error("NoteAgent should not have write tool")
	}
	if NoteAgent.HasTool("edit") {
		t.Error("NoteAgent should not have edit tool")
	}
	if NoteAgent.HasTool("callgraph") {
		t.Error("NoteAgent should not have callgraph tool")
	}
	if NoteAgent.HasTool("memory_recall") {
		t.Error("NoteAgent should not have memory_recall tool (single-iteration agent)")
	}
}

func TestBreakdownAgent_HasExpectedTools(t *testing.T) {
	if BreakdownAgent.HasTool("write") {
		t.Error("BreakdownAgent should not have write tool")
	}
	if BreakdownAgent.HasTool("edit") {
		t.Error("BreakdownAgent should not have edit tool")
	}
	if !BreakdownAgent.HasTool("callgraph") {
		t.Error("BreakdownAgent should have callgraph tool")
	}
	if BreakdownAgent.HasTool("memory_recall") {
		t.Error("BreakdownAgent should not have memory_recall tool (single-iteration agent)")
	}
	if !BreakdownAgent.HasTool("submit_task_breakdown") {
		t.Error("BreakdownAgent should have submit_task_breakdown tool")
	}
}

func TestBuildAgent_SystemPrompt_ContainsSharedSections(t *testing.T) {
	// Verify that BuildAgent's system prompt includes the shared prompt sections
	if !strings.Contains(BuildAgent.System, "Parallel tool calls") {
		t.Error("BuildAgent system prompt should reference parallel tool calls section")
	}
	if !strings.Contains(BuildAgent.System, "Error recovery") {
		t.Error("BuildAgent system prompt should include error recovery section")
	}
	if !strings.Contains(BuildAgent.System, "Project notes") {
		t.Error("BuildAgent system prompt should mention project notes")
	}
	// BuildAgent has write/edit tools, so its project notes section must include
	// the read-only restriction for .ogcode/notes/
	if !strings.Contains(BuildAgent.System, "managed exclusively by the NoteAgent") {
		t.Error("BuildAgent system prompt should include NoteAgent restriction for notes directory")
	}
	if !strings.Contains(BuildAgent.System, "Do not create, modify, or delete any files in .ogcode/notes/") {
		t.Error("BuildAgent system prompt should include read-only restriction for notes directory")
	}
}

func TestBreakdownAgent_SystemPrompt_ContainsCallGraphAndNotes(t *testing.T) {
	// Verify BreakdownAgent mentions project notes and call graph
	if !strings.Contains(BreakdownAgent.System, "Read project notes") {
		t.Error("BreakdownAgent should mention reading project notes")
	}
	if !strings.Contains(BreakdownAgent.System, "callgraph") {
		t.Error("BreakdownAgent should mention the callgraph tool")
	}
	if !strings.Contains(BreakdownAgent.System, "Verify with the call graph") {
		t.Error("BreakdownAgent should have call graph verification step")
	}
	// BreakdownAgent should include the shared call graph prompt section (plan variant)
	// These sections are injected dynamically by buildSystemPrompt when callGraphEnabled=true.
	builtPrompt := buildSystemPrompt(BreakdownAgent, "/tmp/test", false, true, "", "", nil)
	if !strings.Contains(builtPrompt, "When to build the call graph") {
		t.Error("BreakdownAgent should include 'When to build the call graph' section from shared prompt")
	}
	if !strings.Contains(builtPrompt, "Using search instead of grep") {
		t.Error("BreakdownAgent should include search guidance from shared call graph prompt")
	}
	if !strings.Contains(builtPrompt, "Populating the doc field") {
		t.Error("BreakdownAgent should include doc field guidance from shared call graph prompt")
	}
	// BreakdownAgent should NOT include post-mutation sync (it's a read-only agent)
	if strings.Contains(BreakdownAgent.System, "Post-mutation call graph sync") {
		t.Error("BreakdownAgent should NOT include post-mutation sync section (read-only agent)")
	}
}

func TestGetAgent(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"plan", "plan"},
		{"breakdown", "breakdown"},
		{"note", "note"},
		{"build", "build"},
		{"unknown", "build"}, // default
		{"", "build"},         // default
	}
	for _, tc := range tests {
		agent := GetAgent(tc.name)
		if agent.ID != tc.expected {
			t.Errorf("GetAgent(%q) = %q, want %q", tc.name, agent.ID, tc.expected)
		}
	}
}