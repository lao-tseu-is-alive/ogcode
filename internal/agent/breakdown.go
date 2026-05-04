package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ogcode/ogcode/internal/session"
)

// TaskDefinition represents a single task parsed from the breakdown agent's JSON output.
type TaskDefinition struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Dependencies []int  `json:"dependencies"`
	Effort       string `json:"effort"`
	Complexity   string `json:"complexity"`
	OrderIndex   int    `json:"orderIndex"`
}

// BreakdownPrompt constructs the user message for the breakdown agent from the plan conversation.
// archivePaths contains filesystem paths to previously completed plan markdown files for this
// project. Only the paths are mentioned — the agent reads them via its file tools if needed.
func BreakdownPrompt(messages []*session.MessageWithParts, archivePaths []string) string {
	var b strings.Builder

	if len(archivePaths) > 0 {
		b.WriteString("Previously completed plans for this project are archived at:\n\n")
		for _, path := range archivePaths {
			b.WriteString(fmt.Sprintf("- %s\n", path))
		}
		b.WriteString("\nRead them if you need context about past decisions or what has already been implemented.\n\n")
	}

	b.WriteString("Below is the full conversation from a planning session. Analyze it and produce a structured task breakdown as a JSON array.\n\n")
	b.WriteString("--- BEGIN PLAN CONVERSATION ---\n\n")

	for _, msg := range messages {
		role := string(msg.Info.Role)
		if msg.Info.Agent != "" {
			role = fmt.Sprintf("%s (%s)", role, msg.Info.Agent)
		}
		b.WriteString(fmt.Sprintf("[%s]:\n", role))

		for _, part := range msg.Parts {
			switch part.Type {
			case session.PartText:
				var data session.TextPartData
				if err := json.Unmarshal(part.Data, &data); err == nil {
					b.WriteString(data.Text)
					b.WriteString("\n")
				}
			case session.PartTool:
				var data session.ToolPartData
				if err := json.Unmarshal(part.Data, &data); err == nil {
					b.WriteString(fmt.Sprintf("[Tool: %s", data.Tool))
					if data.State.Title != nil {
						b.WriteString(fmt.Sprintf(" — %s", *data.State.Title))
					}
					b.WriteString("]")
					if data.State.Output != nil {
						output := *data.State.Output
						if len(output) > 500 {
							output = output[:500] + "..."
						}
						b.WriteString(fmt.Sprintf("\nOutput: %s", output))
					}
					b.WriteString("\n")
				}
			case session.PartReasoning:
				var data session.ReasoningPartData
				if err := json.Unmarshal(part.Data, &data); err == nil {
					b.WriteString(fmt.Sprintf("[Reasoning: %s]\n", data.Text))
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("--- END PLAN CONVERSATION ---\n\n")
	b.WriteString("DEPENDENCY RULE: Each task may depend on AT MOST ONE other task. " +
		"Design strictly linear chains (A→B→C). " +
		"Fan-in graphs (A,B→C) are not allowed. " +
		"If a task logically requires work from multiple predecessors, consolidate those predecessors into a single task first.\n\n")
	b.WriteString("Now call the submit_task_breakdown tool with the complete task breakdown array. Do NOT output raw JSON — use the tool.")

	return b.String()
}

// extractJSONArray finds the first '[' in s and returns the substring up to and
// including its balanced closing ']', respecting JSON string escaping. This
// avoids the over-match problem of a greedy `\[.*\]` regex, which would extend
// to the last ']' in the entire string and produce invalid JSON when the LLM
// appends trailing text containing brackets.
func extractJSONArray(s string) string {
	start := strings.IndexByte(s, '[')
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// ParseTasks extracts and parses the task definitions from the breakdown agent's response text.
// It handles cases where the LLM wraps the JSON in markdown code fences or adds preamble.
func ParseTasks(text string) ([]TaskDefinition, error) {
	cleaned := strings.TrimSpace(text)

	// Remove ```json ... ``` wrapping
	if strings.HasPrefix(cleaned, "```") {
		idx := strings.Index(cleaned[3:], "\n")
		if idx >= 0 {
			cleaned = cleaned[3+idx+1:]
		}
		closingIdx := strings.LastIndex(cleaned, "```")
		if closingIdx >= 0 {
			cleaned = cleaned[:closingIdx]
		}
		cleaned = strings.TrimSpace(cleaned)
	}

	// Find the JSON array in the text
	match := extractJSONArray(cleaned)
	if match == "" {
		return nil, fmt.Errorf("no JSON array found in breakdown response")
	}

	// LLMs sometimes emit invalid JSON escape sequences (e.g. \` inside strings).
	// Strip backslashes before characters that aren't valid JSON escapes.
	match = sanitizeInvalidEscapes(match)

	var tasks []TaskDefinition
	if err := json.Unmarshal([]byte(match), &tasks); err != nil {
		return nil, fmt.Errorf("parse task definitions: %w", err)
	}

	// Validate and set defaults
	for i := range tasks {
		if tasks[i].Effort == "" {
			tasks[i].Effort = "M"
		}
		if tasks[i].Complexity == "" {
			tasks[i].Complexity = "medium"
		}
		if tasks[i].OrderIndex == 0 && i > 0 {
			tasks[i].OrderIndex = i
		}
		if tasks[i].Dependencies == nil {
			tasks[i].Dependencies = []int{}
		}
	}

	return tasks, nil
}

// sanitizeInvalidEscapes removes backslash escapes before characters that are
// not valid JSON escape sequences. LLMs sometimes emit strings like \` or \'
// inside JSON values, which Go's json.Unmarshal rejects.
func sanitizeInvalidEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			switch next {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', 'u':
				b.WriteByte('\\')
				b.WriteByte(next)
			default:
				// Drop the backslash — e.g. \` becomes `
				b.WriteByte(next)
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}