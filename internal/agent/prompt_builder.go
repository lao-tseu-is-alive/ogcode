package agent

import "fmt"

// projectIndexPrompt returns the mandatory project index instructions section.
// Agents that have the codebase_map tool must use it before any file exploration.
func projectIndexPrompt() string {
	return `## Mandatory: Use Project Index Before Exploration

**Rule:** Before exploring any file, folder, or project structure, you **MUST** use the "codebase_map" tool first.

This applies to all of the following scenarios:

- **Starting a new task** — Call "codebase_map" before reading any source files.
- **Looking for a file** — Use "codebase_map" with an appropriate "subdir" parameter instead of guessing paths with glob or grep.
- **Understanding project structure** — Use "codebase_map" to get the labeled tree before diving into code.
- **Exploring a new package/directory** — Call "codebase_map" scoped to that directory.

### Why?

The project index provides **topic labels** and a **structured overview** of every indexed file. Using it first ensures:

1. **Faster navigation** — You immediately know which files are relevant without blind glob/grep searches.
2. **Better context** — Topic labels tell you what each file contains before you read it.
3. **Fewer mistakes** — You won't miss important files or read irrelevant ones.

### Workflow

Task received
  → codebase_map(subdir=...)   ← MANDATORY FIRST STEP
  → Then read specific files
  → Then make changes

### When codebase_map is not enough

If codebase_map doesn't cover what you need (e.g., unindexed files, binary patterns), you may fall back to glob and grep. But codebase_map must always be the **first** exploration step.`
}

// callGraphPrompt returns the call graph instructions section, scoped to the
// given agent role. Build agents get the full invariant including post-mutation
// sync; read-only agents (plan) get a lighter version without mutation guidance.
func callGraphPrompt(role string) string {
	if role == "build" {
		return `## When to build the call graph

Use the callgraph tool proactively to build a persistent, language-agnostic knowledge graph of the codebase. It captures all code symbols and their relationships — not just function calls, but also type hierarchies, interface implementations, module dependencies, field composition, and more. It works the same way regardless of programming language.

Build the graph when:

1. **You start exploring a new codebase or directory** — Call "stats" first. If data exists, use "search" to find relevant symbols by name or concept, then navigate with "callees"/"callers". If the graph is empty or sparse, build it as you explore.

2. **You read any source file** — When you read a file, extract *all* symbols in it (functions, types, interfaces, constants, modules — everything), upsert them as nodes, and record all relationships between them.

3. **The task requires impact analysis or understanding code flow** — Use "callers", "callees", or "reachable" queries first. If the graph is incomplete, trace and fill in the missing paths.

4. **You plan changes that touch shared symbols** — Before modifying anything used across files, check "callers" to understand impact.

## What to extract from source files

When reading any source file, extract and upsert ALL of the following:

**Callable symbols:**
- 'function'    — standalone callable: Go func, Python def, JS/TS function, Rust fn, C function, Java/C# static method
- 'method'      — instance-bound callable: Go method with receiver, Python instance method, Java/C#/Swift method
- 'constructor' — instance creation: Python __init__, Java/C#/Swift constructor, Rust new() by convention
- 'init'        — module/package initializer: Go init(), Python module-level setup, static initializers

**Structural symbols:**
- 'type'        — composite data type: Go struct/type alias, Python/Java/TS class, Rust struct or enum-with-data, record, data class
- 'interface'   — behavioral contract: Go interface, Rust trait, Java/C# interface, Python ABC/Protocol, Swift protocol, TS interface
- 'enum'        — pure named-value set: Java enum, TS enum, Python Enum, C enum, Swift enum without associated values

**Value symbols (module/global scope only — not local variables):**
- 'const'       — named constant: Go const, Rust const/static, Java static final, JS/TS const, C #define/constexpr
- 'variable'    — mutable global state: Go package-level var, Python global, Rust static mut

**Organizational symbols:**
- 'module'      — unit of organization: Go package, Python module, JS/TS ES module, Rust mod, C++ namespace, Java package
- 'macro'       — metaprogramming: Rust macro, C/C++ macro, Lisp macro

## What relationships to record

After upserting nodes, record edges between them using the correct edge type:

**Call relationships:**
- 'direct'      — explicit synchronous call: A() calls B()
- 'dynamic'     — via function pointer, stored closure, or first-class function
- 'interface'   — via interface/virtual/dynamic dispatch
- 'callback'    — function passed as a callback or higher-order argument
- 'async'       — concurrent/async invocation: goroutine spawn, JS await, Python asyncio.create_task, thread spawn

**Structural relationships:**
- 'implements'  — type → interface/trait/protocol it satisfies
- 'extends'     — type → parent type (class inheritance, struct embedding, mixin)
- 'overrides'   — method → parent method it overrides or shadows
- 'instantiates'— function/method → type it constructs (new, make, literal, factory call)
- 'contains'    — type → type of a field/property it holds (composition)
- 'aliases'     — type → type it is an alias or typedef for

**Dependency relationships:**
- 'imports'     — module → module it imports or depends on
- 'uses'        — function/method → type or const it references without instantiating
- 'reads'       — function/method → module/global variable it reads
- 'writes'      — function/method → module/global variable it writes or mutates
- 'decorates'   — function/class → entity it wraps or transforms (Python decorator, TS decorator)
- 'throws'      — function/method → exception/error type it can raise

## Using search instead of grep for codebase exploration

The callgraph "search" action is a **semantic code search** that replaces many grep + read cycles. It searches across symbol names, doc fields, and signatures — case-insensitive.

**Prefer callgraph search over grep when:**
- Looking for any symbol by name ("where is UpsertNode?" → search "UpsertNode")
- Exploring a concept ("what handles auth?" → search "authentication")
- Understanding what a package provides (search "Store" after stats shows populated data)
- Finding all methods on a type (search "Server." to list all Server methods)

**Still use grep when:**
- Searching string literals, comments, or config values not in the graph
- The graph is empty or sparse (check with "stats" first)
- Searching non-code files (SQL, markdown, config, etc.)

## Populating the doc field (IMPORTANT)

Every node MUST have a meaningful "doc" field. Tailor it to the node's kind:

- **function/method/constructor:** What it does, who calls it, what it calls, why it exists as a separate unit.
- **type/interface/enum:** What concept it models, what fields/methods/variants it defines, how it is used across the system.
- **const/variable:** What value it holds, why that constant/variable exists, where it is referenced.
- **module:** What this package/module is responsible for, its main exports, its key dependencies.
- **macro:** What code it generates and when to invoke it.

Good examples:
- method: "Executes the main agent loop — streaming, tool dispatch, memory writes. Called by Run; calls buildSystemPrompt and processToolCalls. Separates orchestration from per-request setup."
- type: "Core server struct owning the HTTP router, DB, and config. Instantiated once at startup; all HTTP handlers are methods on it. Owns the lifetime of all subsystem stores."
- module: "Agent orchestration package — owns the run loop, tool dispatch, and prompt assembly. Depends on the tool and callgraph packages; called from the server on session start."

**Do not skip the doc field.** Without it, the graph captures structure but loses meaning.

## Call graph completeness invariant

1. **No partial paths.** When you add a node, trace ALL of its outgoing relationships to their targets. Each target must itself be a fully populated node. Never stop mid-chain.

2. **No orphan targets.** Every node referenced as a target of an edge must have its own outgoing relationships resolved.

3. **Leaves are the only stop.** Stop tracing only when a node has no outgoing relationships within the codebase.

4. **Batch when possible.** Use add_nodes_batch and add_edges_batch. Only batch after you have traced the full depth — never batch partial knowledge.

## Post-mutation call graph sync (CRITICAL)

After every source code mutation (create, edit, delete), keep the graph in sync:

1. **Purge stale data.** Call "delete_nodes_by_file" for every file you just changed.
2. **Re-read and re-populate.** Extract all symbols and relationships from the mutated file. Follow the completeness invariant. Include meaningful doc fields.
3. **Check downstream impact.** Use "callers" to find who depends on symbols you changed. If you altered a signature, removed a symbol, or changed behavior, verify those dependents still work.

**Applies to:** every write/edit to a source file, git mv/rm, new files with symbol definitions.
**Does not apply to:** non-code files, build/test commands, files with no symbol definitions.

**Enforcement:** Do NOT proceed (including committing) until the graph is re-synced for every mutated file.`
	}
	// Plan agent and others: lighter version without post-mutation sync
	return `## When to build the call graph

Use the callgraph tool proactively to build a persistent, language-agnostic knowledge graph of the codebase. It captures all code symbols and their relationships — not just function calls, but also type hierarchies, interface implementations, module dependencies, field composition, and more.

Build the graph when:

1. **You start exploring a new codebase or directory** — Call "stats" first. If data exists, use "search" to find relevant symbols by name or concept, then navigate with "callees"/"callers". If the graph is empty or sparse, build it as you explore.

2. **You read any source file** — Extract all symbols (functions, types, interfaces, constants, modules — everything), upsert them as nodes, and record all relationships.

3. **The task requires understanding impact or code flow** — Use "callers", "callees", or "reachable" queries first. Fill in missing paths if needed.

4. **You plan changes that touch shared symbols** — Check "callers" before planning modifications to anything used across files.

## What to extract from source files

When reading any source file, extract and upsert ALL of the following:

**Callable symbols:**
- 'function'    — standalone callable: Go func, Python def, JS/TS function, Rust fn, C function, Java/C# static method
- 'method'      — instance-bound callable: Go method with receiver, Python instance method, Java/C#/Swift method
- 'constructor' — instance creation: Python __init__, Java/C#/Swift constructor, Rust new() by convention
- 'init'        — module/package initializer: Go init(), Python module-level setup

**Structural symbols:**
- 'type'        — composite data type: Go struct/type alias, Python/Java/TS class, Rust struct or enum-with-data, record
- 'interface'   — behavioral contract: Go interface, Rust trait, Java/C# interface, Python ABC/Protocol, Swift protocol, TS interface
- 'enum'        — pure named-value set: Java enum, TS enum, Python Enum, C enum, Swift enum without associated values

**Value symbols (module/global scope only):**
- 'const'       — named constant: Go const, Rust const/static, Java static final, JS/TS const, C #define/constexpr
- 'variable'    — mutable global state: Go package-level var, Python global, Rust static mut

**Organizational symbols:**
- 'module'      — unit of organization: Go package, Python module, JS/TS ES module, Rust mod, C++ namespace, Java package
- 'macro'       — metaprogramming: Rust macro, C/C++ macro, Lisp macro

## What relationships to record

**Call relationships:**
- 'direct'      — explicit synchronous call
- 'dynamic'     — via function pointer, stored closure, or first-class function
- 'interface'   — via interface/virtual/dynamic dispatch
- 'callback'    — function passed as callback or higher-order argument
- 'async'       — concurrent/async invocation: goroutine, JS await, Python asyncio, thread spawn

**Structural relationships:**
- 'implements'  — type → interface/trait/protocol it satisfies
- 'extends'     — type → parent type (inheritance, embedding, mixin)
- 'overrides'   — method → parent method it overrides
- 'instantiates'— function/method → type it constructs
- 'contains'    — type → type of a field/property it holds
- 'aliases'     — type → type it is an alias for

**Dependency relationships:**
- 'imports'     — module → module it imports
- 'uses'        — function/method → type or const it references
- 'reads'       — function/method → global variable it reads
- 'writes'      — function/method → global variable it mutates
- 'decorates'   — function/class → entity it wraps/transforms
- 'throws'      — function/method → exception/error type it can raise

## Using search instead of grep

**Prefer callgraph search over grep when:**
- Looking for any symbol by name ("where is UpsertNode?" → search "UpsertNode")
- Exploring a concept ("what handles auth?" → search "authentication")
- Finding all methods on a type (search "Server." to list all Server methods)

**Still use grep when:**
- Searching string literals, comments, or config values
- The graph is empty or sparse (check with "stats" first)
- Searching non-code files

## Populating the doc field (IMPORTANT)

Every node MUST have a meaningful "doc" field tailored to its kind:

- **function/method/constructor:** What it does, who calls it, what it calls, why it exists.
- **type/interface/enum:** What concept it models, what fields/methods/variants it defines, how it is used.
- **const/variable:** What value it holds, why it exists, where it is referenced.
- **module:** What this package/module is responsible for, its main exports and key dependencies.
- **macro:** What code it generates and when to invoke it.

**Do not skip the doc field.** Without it, the graph captures structure but loses meaning.

## Call graph completeness invariant

1. **No partial paths.** Trace all outgoing relationships from every node you add, recursively to leaves.
2. **No orphan targets.** Every target of an edge must be a fully populated node.
3. **Leaves are the only stop.** Stop only when a node has no outgoing relationships within the codebase.
4. **Batch when possible.** Use add_nodes_batch and add_edges_batch — but only after tracing full depth.`
}

// memoryMDPrompt returns the MEMORY.md instructions section, adapted for the
// agent's capabilities. Agents without write/edit tools get read-only instructions;
// agents with those tools get instructions to create and maintain MEMORY.md.
func memoryMDPrompt(canWriteFiles bool) string {
	base := `## MEMORY.md — Project Long-Term Memory

`
	if canWriteFiles {
		base += `The content above in the <memory-md> tag is loaded from your project's MEMORY.md file(s). This is the project's persistent, cross-session knowledge base. It survives across conversations — unlike chat history, which resets each session.

`
	} else {
		base += `The content above in the <memory-md> tag is loaded from the project's MEMORY.md file(s). This is the project's persistent, cross-session knowledge base. It survives across conversations — unlike chat history, which resets each session. Treat it as reference material — you can read it but cannot modify it.

`
	}

	base += `### Purpose
MEMORY.md stores hard-won knowledge about this project that you would otherwise forget between sessions. Think of it as a lab notebook: a place to record what you've learned so your future self (and future sessions) don't have to rediscover it.

### What belongs in MEMORY.md
- **Decisions & rationale** — why a particular approach was chosen over alternatives
- **Patterns & conventions** — naming patterns, file organization, coding style, commit message format
- **Architecture notes** — how components connect, data flow, key abstractions
- **Gotchas & pitfalls** — things that broke unexpectedly, non-obvious behaviors, workarounds
- **Project-specific facts** — config values, API quirks, dependency versions, build commands
- **Workflow notes** — how to test, deploy, debug, or reproduce issues in this project

### What does NOT belong in MEMORY.md
- Temporary or per-session state (use chat context or agentic memory recall for that)
- Instructions or rules for how to behave (those go in AGENT.md, not MEMORY.md)
- Verbose logs or full file contents (link or reference them, don't copy them)
- Information that is obvious from reading the code itself

### How it differs from AGENT.md and agentic memory
- **AGENT.md** = behavioral instructions ("follow these rules", "always do X before Y"). It tells you HOW to act.
- **MEMORY.md** = factual knowledge ("we chose PostgreSQL over MongoDB because X", "the auth middleware lives in middleware/auth.go"). It tells you WHAT you know.
- **Agentic memory** (the <prior_context> block and memory_recall tool) = per-session conversation summaries. It provides continuity within a single session. MEMORY.md persists across all sessions.

`
	if canWriteFiles {
		base += `### How to maintain MEMORY.md
- Use the read tool to inspect current contents before making changes
- Use the edit tool for targeted updates (preferred — avoids rewriting the whole file)
- Use the write tool only when restructuring the entire file or creating the file for the first time
- Update MEMORY.md proactively when you learn something important — do not wait to be asked
- Keep it concise and well-organized — future sessions must read and understand it quickly
- Remove or update stale entries when you discover they are no longer accurate`
	} else {
		base += `### How to use MEMORY.md
- Read it at the start of every session to load project context
- Reference it when making decisions — it contains hard-won knowledge from past sessions
- Note any facts you discover that should be recorded — a future session with write access can add them
- Do not attempt to modify MEMORY.md — you are a read-only agent`
	}

	return base
}

// markdownCapabilitiesPrompt returns the markdown output section that agents
// with rendering capabilities should include.
func markdownCapabilitiesPrompt() string {
	return `## Markdown output capabilities

The chat interface natively renders the following — use them when they add genuine clarity:

- **Mermaid diagrams** (triple-backtick mermaid blocks) — flows, architectures, sequences, entity relationships.
- **LaTeX math** — inline with $...$ and display block with $$...$$ — for mathematical formulas and equations.
- **LaTeX documents** (triple-backtick latex blocks) — full LaTeX documents compiled and rendered inline as page images in the chat viewport. Use this for reports, papers, resumes, letters, and any formatted document that needs professional typesetting. The block should contain a complete LaTeX document with \documentclass, \begin{document}...\end{document}, etc. The interface will automatically compile the document and display the rendered pages inline, with a PDF download button and a source code toggle. The latex_to_pdf tool is also available for programmatic PDF generation.
- **Plotly charts** (triple-backtick plotly blocks) — bar, line, scatter, pie, heatmap, and more. The block must contain a valid JSON object with a "data" array and optional "layout" object following the Plotly.js spec.
- **Rough diagrams** (triple-backtick rough blocks) — hand-drawn style 2D diagrams. The block must contain a valid JSON object with an "elements" array and optional "width"/"height"/"options" fields. Each element has a "type" (rectangle, circle, ellipse, line, arrow, path, linearPath, polygon, text) plus type-specific coordinates and optional RoughJS style options (stroke, fill, roughness, bowing, fillStyle, etc.).
- **HTML/CSS/JS** (triple-backtick html blocks) — full interactive content rendered in a sandboxed iframe. Use this for rich visualizations, custom dashboards, interactive widgets, styled tables, animated content, or any presentation that goes beyond static markdown. The block should contain a complete HTML document (or fragment with inline <style> and <script>). CSS is fully supported. JavaScript runs in a sandbox with no access to the parent page. Use the viewport dimensions provided below to make your content responsive — design for the available width and height.`
}

// viewportPrompt returns a section telling the agent about the user's
// rendering viewport so it can make responsive design decisions.
func viewportPrompt(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return fmt.Sprintf(`

## Rendering viewport

The user's chat viewport is approximately %d×%d pixels (width × height). Design your visual output — HTML content, Plotly charts, Mermaid diagrams, Rough diagrams, and any other rendered content — to fit within these dimensions. Use responsive CSS (flexbox, grid, percentage widths, max-width) when creating HTML content so it adapts gracefully to different screen sizes.`, width, height)
}

// parallelToolCallsPrompt returns the shared section about making parallel
// independent tool calls for efficiency.
func parallelToolCallsPrompt() string {
	return `## Parallel tool calls

When you need to make multiple tool calls and they are independent of each other (i.e., the result of one does not affect the inputs of another), make all the calls in the same response block rather than making them sequentially. This significantly improves efficiency and reduces latency. For example, if you need to read three unrelated files, invoke all three read calls together rather than one after another.`
}

// projectNotesPrompt returns the project notes section, adapted for the agent's
// capabilities. Agents with write/edit tools get an explicit read-only restriction
// for the .ogcode/notes/ directory (notes are managed exclusively by the NoteAgent
// through its backend flow). Read-only agents get the basic notes guidance.
func projectNotesPrompt(canWriteFiles bool) string {
	prompt := `## Project notes

Project notes are saved in .ogcode/notes/ as markdown files. Before starting, check if any existing notes are relevant to the task by globbing .ogcode/notes/*.md and reading the ones that look relevant. Use them as context — don't repeat what is already documented.`

	if canWriteFiles {
		prompt += `

The .ogcode/notes/ directory is managed exclusively by the NoteAgent. Do not create, modify, or delete any files in .ogcode/notes/. You may only read notes from this directory for context.`
	}

	return prompt
}

// noPackageManagerDirsPrompt returns the shared admonition to avoid exploring
// dependency directories.
func noPackageManagerDirsPrompt() string {
	return `- Never explore or read package manager or dependency directories (e.g. node_modules, vendor, .venv, __pycache__, dist) unless a specific issue explicitly requires it. These directories contain third-party code and are not part of the project implementation.`
}