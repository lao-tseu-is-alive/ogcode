package callgraph

// NodeKind represents what kind of code entity a CallNode is.
type NodeKind string

const (
	KindFunction NodeKind = "function"
	KindMethod   NodeKind = "method"
	KindInit     NodeKind = "init"
)

// CallNode represents a function or method in the call graph.
type CallNode struct {
	ID        int64    `json:"id"`
	Directory string   `json:"directory"`
	Package   string   `json:"package"`
	Symbol    string   `json:"symbol"`
	FilePath  string   `json:"filePath"`
	Line      int      `json:"line"`
	Kind      NodeKind `json:"kind"`
	Signature string   `json:"signature,omitempty"`
	Doc       string   `json:"doc,omitempty"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

// CallType represents the type of a call relationship.
type CallType string

const (
	CallDirect   CallType = "direct"
	CallDynamic  CallType = "dynamic"
	CallInterface CallType = "interface"
	CallCallback CallType = "callback"
)

// CallEdge represents a caller→callee relationship in the call graph.
type CallEdge struct {
	ID        int64    `json:"id"`
	Directory string   `json:"directory"`
	CallerID  int64    `json:"callerId"`
	CalleeID  int64    `json:"calleeId"`
	CallType  CallType `json:"callType"`
	CreatedAt int64    `json:"createdAt"`
}

// CalleeOfResult holds a call edge along with the callee's info,
// returned by queries like "what does X call?".
type CalleeOfResult struct {
	Edge   CallEdge  `json:"edge"`
	Callee CallNode  `json:"callee"`
}

// CallerOfResult holds a call edge along with the caller's info,
// returned by queries like "who calls X?".
type CallerOfResult struct {
	Edge   CallEdge  `json:"edge"`
	Caller CallNode  `json:"caller"`
}