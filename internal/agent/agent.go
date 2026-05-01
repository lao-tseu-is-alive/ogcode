package agent

// Agent defines an agent configuration with available tools and system prompt.
type Agent struct {
	ID          string
	Name        string
	Description string
	Tools       []string
	System      string
}

// BuildAgent is the default full-access coding agent.
var BuildAgent = Agent{
	ID:          "build",
	Name:        "Build",
	Description: "Full-access coding agent",
	Tools:       []string{"bash", "read", "write", "edit", "glob", "grep"},
	System:      "You are a coding assistant. Help the user with their coding tasks. You can read, write, and edit files, run commands, and search the codebase.",
}

func (a *Agent) HasTool(toolID string) bool {
	for _, t := range a.Tools {
		if t == toolID {
			return true
		}
	}
	return false
}

// GetAgent returns the agent by name, defaulting to BuildAgent.
func GetAgent(name string) Agent {
	if name == "" || name == "build" {
		return BuildAgent
	}
	return BuildAgent
}