package callgraph

// NodeKind classifies what kind of code entity a node represents.
// Kinds are language-agnostic — the same kind covers equivalent constructs
// across programming languages (Go, Python, TypeScript, Java, Rust, C/C++, etc.).
type NodeKind string

const (
	// ── Callable entities ────────────────────────────────────────────────────
	KindFunction    NodeKind = "function"    // standalone callable: Go func, Python def, JS function, Rust fn, C function
	KindMethod      NodeKind = "method"      // instance-bound callable: Go method, Python instance method, Java/C# method
	KindConstructor NodeKind = "constructor" // instance creation: Python __init__, Java/C# constructor, Rust new(), Swift init
	KindInit        NodeKind = "init"        // module/package initializer: Go init(), Python module-level setup, static initializers

	// ── Structural entities ───────────────────────────────────────────────────
	KindType      NodeKind = "type"      // composite data type: Go struct, Python/Java/TS class, Rust struct/enum-with-data, Swift struct, record
	KindInterface NodeKind = "interface" // behavioral contract: Go interface, Rust trait, Java/C# interface, Python ABC/Protocol, Swift protocol, TS interface
	KindEnum      NodeKind = "enum"      // pure named-value set: Java enum, TS enum, Python Enum, C/C++ enum, Swift enum (no associated values)

	// ── Value entities ────────────────────────────────────────────────────────
	KindConst    NodeKind = "const"    // named constant: Go const, Rust const/static, Java static final, JS/TS const, C #define/constexpr
	KindVariable NodeKind = "variable" // module/global mutable state: Go package var, Python global, JS module-level let, Rust static mut

	// ── Organizational entities ───────────────────────────────────────────────
	KindModule NodeKind = "module" // organizational unit: Go package, Python module, JS/TS ES module, Rust mod, C++ namespace, Java package
	KindMacro  NodeKind = "macro"  // code-generating metaprogramming: Rust macro, C/C++ macro, Lisp macro, Elixir macro
)

// CallNode represents any code symbol in the knowledge graph.
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

// CallType classifies the relationship between two nodes in the graph.
// Named CallType for historical reasons; it covers all relationship types,
// not just function calls — including type, structural, and dependency edges.
type CallType string

const (
	// ── Call relationships (function/method → function/method) ───────────────
	CallDirect    CallType = "direct"    // explicit synchronous call: A()
	CallDynamic   CallType = "dynamic"   // via function pointer, stored closure, or first-class function
	CallInterface CallType = "interface" // via interface/virtual/dynamic dispatch
	CallCallback  CallType = "callback"  // passed as a callback or higher-order argument
	CallAsync     CallType = "async"     // concurrent/async invocation: goroutine spawn, JS await, Python asyncio, thread

	// ── Type / structural relationships ──────────────────────────────────────
	CallImplements   CallType = "implements"   // type → interface/trait/protocol/ABC it satisfies
	CallExtends      CallType = "extends"      // type → parent type (class inheritance, struct embedding, mixin)
	CallOverrides    CallType = "overrides"    // method → parent method it overrides/shadows
	CallInstantiates CallType = "instantiates" // function/method → type it constructs (new, make, literal, factory)
	CallContains     CallType = "contains"     // type → type of a field/property it holds (composition)
	CallAliases      CallType = "aliases"      // type → type it is an alias/typedef for

	// ── Dependency relationships ──────────────────────────────────────────────
	CallImports   CallType = "imports"   // module → module it imports or depends on
	CallUses      CallType = "uses"      // function/method → type or const it references without instantiating
	CallReads     CallType = "reads"     // function/method → module/global variable it reads
	CallWrites    CallType = "writes"    // function/method → module/global variable it writes or mutates
	CallDecorates CallType = "decorates" // function/class → entity it wraps or transforms (Python/TS decorator)
	CallThrows    CallType = "throws"    // function/method → exception/error type it can raise
)

// CallEdge represents a directed relationship between two nodes in the graph.
type CallEdge struct {
	ID        int64    `json:"id"`
	Directory string   `json:"directory"`
	CallerID  int64    `json:"callerId"`
	CalleeID  int64    `json:"calleeId"`
	CallType  CallType `json:"callType"`
	CreatedAt int64    `json:"createdAt"`
}

// ValidNodeKind reports whether k is a known node kind.
func ValidNodeKind(k NodeKind) bool {
	switch k {
	case KindFunction, KindMethod, KindConstructor, KindInit,
		KindType, KindInterface, KindEnum,
		KindConst, KindVariable,
		KindModule, KindMacro:
		return true
	}
	return false
}

// ValidCallType reports whether t is a known call/relationship type.
func ValidCallType(t CallType) bool {
	switch t {
	case CallDirect, CallDynamic, CallInterface, CallCallback, CallAsync,
		CallImplements, CallExtends, CallOverrides, CallInstantiates, CallContains, CallAliases,
		CallImports, CallUses, CallReads, CallWrites, CallDecorates, CallThrows:
		return true
	}
	return false
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