package session

import (
	"encoding/json"
	"time"
)

type Session struct {
	ID                SessionID `json:"id"`
	ProjectID         string    `json:"projectId"`
	Directory         string    `json:"directory"`
	Title             string    `json:"title"`
	Model             string    `json:"model,omitempty"`
	SessionType       string    `json:"sessionType,omitempty"`
	Permission        string    `json:"permission,omitempty"`
	CompactionSummary  string    `json:"compactionSummary,omitempty"`
	MemoryTokensSaved  int       `json:"memoryTokensSaved,omitempty"`
	CreatedAt          int64     `json:"createdAt"`
	UpdatedAt          int64     `json:"updatedAt"`
}

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

type MessageInfo struct {
	ID        MessageID   `json:"id"`
	SessionID SessionID   `json:"sessionId"`
	Role      MessageRole `json:"role"`
	Agent     string      `json:"agent,omitempty"`
	ParentID  *MessageID  `json:"parentId,omitempty"`
	Finish    *string     `json:"finish,omitempty"`
	Cost      float64     `json:"cost,omitempty"`
	Tokens    *TokenCounts `json:"tokens,omitempty"`
	Error     *string     `json:"error,omitempty"`
	CreatedAt int64       `json:"createdAt"`
}

type TokenCounts struct {
	Total      int `json:"total,omitempty"`
	Input      int `json:"input,omitempty"`
	Output     int `json:"output,omitempty"`
	Reasoning  int `json:"reasoning,omitempty"`
	CacheRead  int `json:"cacheRead,omitempty"`
	CacheWrite int `json:"cacheWrite,omitempty"`
}

type MessageWithParts struct {
	Info   MessageInfo `json:"info"`
	Parts  []Part      `json:"parts"`
}

type PartType string

const (
	PartText     PartType = "text"
	PartTool     PartType = "tool"
	PartReasoning PartType = "reasoning"
	PartFile     PartType = "file"
)

type Part struct {
	ID        PartID          `json:"id"`
	MessageID MessageID       `json:"messageId"`
	SessionID SessionID       `json:"sessionId"`
	Type      PartType        `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt int64           `json:"createdAt"`
	UpdatedAt int64           `json:"updatedAt"`
}

type TextPartData struct {
	Text string `json:"text"`
}

type ToolStatus string

const (
	ToolPending   ToolStatus = "pending"
	ToolRunning   ToolStatus = "running"
	ToolCompleted ToolStatus = "completed"
	ToolError     ToolStatus = "error"
)

type ToolPartData struct {
	Tool   string          `json:"tool"`
	CallID string          `json:"callId"`
	State  ToolState       `json:"state"`
}

type ToolState struct {
	Status   ToolStatus      `json:"status"`
	Input    json.RawMessage `json:"input"`
	Output   *string         `json:"output,omitempty"`
	Error    *string         `json:"error,omitempty"`
	Title    *string         `json:"title,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Time     ToolTime        `json:"time"`
}

type ToolTime struct {
	Start int64  `json:"start,omitempty"`
	End   int64  `json:"end,omitempty"`
}

type ReasoningPartData struct {
	Text string `json:"text"`
}

func Now() int64 {
	return time.Now().UnixMilli()
}