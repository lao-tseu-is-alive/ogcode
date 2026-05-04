package permission

import "github.com/prasenjeet-symon/ogcode/internal/id"

type PermissionID = id.PermissionID

func NewPermissionID() PermissionID { return id.NewPermissionID() }

type Action string

const (
	Allow Action = "allow"
	Deny  Action = "deny"
	Ask   Action = "ask"
)

type Rule struct {
	Permission string `json:"permission"` // tool name or "edit", "*"
	Pattern    string `json:"pattern"`    // glob pattern
	Action     Action `json:"action"`
}

type Ruleset []Rule

type Request struct {
	ID        PermissionID
	SessionID string
	Tool      string
	Input     string
	Patterns  []string
}

// Evaluate checks the ruleset and returns the action for the given tool and path.
func (rs Ruleset) Evaluate(toolName, path string) Action {
	for _, rule := range rs {
		if rule.Permission == "*" || rule.Permission == toolName {
			if rule.Pattern == "*" || matchGlob(rule.Pattern, path) {
				return rule.Action
			}
		}
	}
	return Ask // default: ask the user
}

// DefaultRuleset returns the default permission rules for the build agent.
func DefaultRuleset() Ruleset {
	return Ruleset{
		{Permission: "read", Pattern: "*", Action: Allow},
		{Permission: "glob", Pattern: "*", Action: Allow},
		{Permission: "grep", Pattern: "*", Action: Allow},
		{Permission: "bash", Pattern: "*", Action: Ask},
		{Permission: "write", Pattern: "*", Action: Ask},
		{Permission: "edit", Pattern: "*", Action: Ask},
	}
}

// PendingRequest holds a permission request awaiting user reply.
type PendingRequest struct {
	Request Request
	ReplyCh chan string // "once", "always", "reject"
}

// Manager manages pending permission requests.
type Manager struct {
	pending map[PermissionID]*PendingRequest
}

func NewManager() *Manager {
	return &Manager{pending: make(map[PermissionID]*PendingRequest)}
}

func (m *Manager) Create(req Request) *PendingRequest {
	pr := &PendingRequest{
		Request: req,
		ReplyCh: make(chan string, 1),
	}
	m.pending[req.ID] = pr
	return pr
}

func (m *Manager) Get(id PermissionID) *PendingRequest {
	return m.pending[id]
}

func (m *Manager) Reply(id PermissionID, response string) bool {
	pr := m.pending[id]
	if pr == nil {
		return false
	}
	pr.ReplyCh <- response
	delete(m.pending, id)
	return true
}

// matchGlob does simple glob matching (* matches any sequence).
func matchGlob(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	// Simple implementation: only handle exact match or * wildcard
	if pattern == s {
		return true
	}
	return false
}