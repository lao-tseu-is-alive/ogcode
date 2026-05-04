# Plan Mode Feature — Ogcode

## 1. Overview

Ogcode currently operates in a single **Build Mode** — you launch `ogcode`, and you're dropped straight into a chat view where the agent can read, write, edit, and execute code. 

This feature introduces a **Plan Mode** — a new launch mode (`ogcode plan`) where the user lands in a **planning-first** environment. There is **no direct chat/code execution**. Instead, the user describes their problem/requirement, and ogcode understands the full source code context and plans with the user through conversation. Once satisfied, the user clicks a CTA to break the plan into **tasks** (like Jira/Kanban tickets), and each task is then executed in parallel using git branches, raising PRs back to main upon completion.

---

## 2. Key Concepts

### 2.1 Launch Modes

| Command | Mode | Description |
|---------|------|-------------|
| `ogcode` | Build Mode (existing) | Direct chat with agent, full tool access, write code immediately |
| `ogcode plan` | Plan Mode (new) | Planning-first, no code writing, locked plan → task breakdown → parallel execution |

### 2.2 Plan

A **Plan** is a document (conversation thread) where the user and ogcode collaboratively discuss a problem/requirement. Key properties:

- **No code generation** — The `PlanAgent` has read-only tools only (read, glob, grep, bash for inspection). No write, no edit.
- **Immutable once locked** — Once the "Break into Tasks" CTA is clicked, the plan is locked. No further editing or chatting.
- **Preserved forever** — Plans are never deleted. They serve as living documentation.
- **Referenceable** — Every task created from a plan holds a native link back to the plan document.

### 2.3 Task

A **Task** is a unit of work derived from a plan. Key properties:

- **Auto-generated** — Ogcode breaks down the locked plan into tasks based on dependencies, effort, and complexity.
- **Dependency-aware** — Tasks are ordered; dependencies between tasks are tracked.
- **Effort & Complexity tagged** — Each task has `effort` (S/M/L/XL) and `complexity` (low/medium/high) ratings.
- **Status tracking** — Each task has a status: `pending` → `in_progress` → `completed` / `failed`.
- **Git branch isolation** — Each task gets its own git branch (`task/<task-id>-<slug>`).
- **Auto PR** — On completion, a PR is automatically raised to the main branch, referencing the originating task and plan.
- **Live execution view** — User can jump into any task's execution view to see real-time agent activity.

### 2.4 Task Breakdown Agent (Future vs Now)

- **Now**: The default `BuildAgent` handles the breakdown — it reads the locked plan conversation and produces a structured task list with dependencies, effort, and complexity.
- **Future**: A dedicated `BreakdownAgent` (separate LLM) will handle this. The architecture supports this via the agent system.

---

## 3. Architecture

### 3.1 New CLI Command

```
ogcode plan          # Launch Plan Mode server on port 8080
ogcode plan -p 3000  # Launch Plan Mode on custom port
```

The `plan` subcommand starts the same web server, but with a **mode flag** that changes the default agent, available tools, and the web UI landing page.

### 3.2 Database Schema Changes

#### 3.2.1 `plan` table

```sql
CREATE TABLE IF NOT EXISTS plan (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    directory TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',       -- 'open' | 'locked'
    model TEXT NOT NULL DEFAULT '',
    compaction_summary TEXT NOT NULL DEFAULT '',
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plan_project ON plan(project_id);
CREATE INDEX IF NOT EXISTS idx_plan_status ON plan(status);
```

A **Plan** reuses the existing `message` and `part` tables for its conversation — a `plan` is like a session but with planning semantics. The `plan.id` is referenced by tasks.

#### 3.2.2 `task` table

```sql
CREATE TABLE IF NOT EXISTS task (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES plan(id),
    session_id TEXT REFERENCES session(id),      -- linked execution session
    parent_task_id TEXT REFERENCES task(id),       -- for hierarchical tasks
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    effort TEXT NOT NULL DEFAULT 'M',              -- 'S' | 'M' | 'L' | 'XL'
    complexity TEXT NOT NULL DEFAULT 'medium',     -- 'low' | 'medium' | 'high'
    status TEXT NOT NULL DEFAULT 'pending',        -- 'pending' | 'in_progress' | 'completed' | 'failed'
    dependencies TEXT NOT NULL DEFAULT '[]',        -- JSON array of task IDs this task depends on
    branch_name TEXT NOT NULL DEFAULT '',           -- git branch for this task
    pr_url TEXT NOT NULL DEFAULT '',               -- PR URL once raised
    pr_number INTEGER,                             -- PR number
    order_index INTEGER NOT NULL DEFAULT 0,        -- execution order
    time_created INTEGER NOT NULL,
    time_updated INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_plan ON task(plan_id);
CREATE INDEX IF NOT EXISTS idx_task_status ON task(status);
CREATE INDEX IF NOT EXISTS idx_task_session ON task(session_id);
```

#### 3.2.3 Migration File

```
internal/db/006_plan_task.sql  -- Up migration for the above
```

### 3.3 New Go Packages

#### 3.3.1 `internal/plan/` — Plan Store & Logic

```go
// internal/plan/plan.go
type Plan struct {
    ID                string
    ProjectID         string
    Directory         string
    Title             string
    Status            string  // "open" | "locked"
    Model             string
    CompactionSummary string
    CreatedAt         int64
    UpdatedAt         int64
}

// internal/plan/store.go
type Store struct {
    db *db.DB
}

func NewStore(database *db.DB) *Store
func (s *Store) Create(plan *Plan) error
func (s *Store) Get(id string) (*Plan, error)
func (s *Store) List(directory string) ([]*Plan, error)
func (s *Store) Update(plan *Plan) error
func (s *Store) Lock(id string) error  // Sets status = 'locked', no further edits
```

Plans have their own message/part storage (reusing the existing `message` / `part` tables with `session_id` replaced by concept of `plan_id` OR we use a `thread_type` discriminator).

**Decision**: Reuse the existing `message` and `part` tables by adding a `thread_type` column (`session` or `plan`), OR create separate `plan_message` / `plan_part` tables. The cleanest approach is to **add a `thread_type` column** to the `message` table so plans and sessions share the same message infrastructure but are distinguished by type.

```sql
-- Migration addition
ALTER TABLE message ADD COLUMN thread_type TEXT NOT NULL DEFAULT 'session';
ALTER TABLE part ADD COLUMN thread_type TEXT NOT NULL DEFAULT 'session';
```

> **Alternative considered**: Use `session_id` for both and add a virtual session for each plan. This is simpler and avoids schema changes to message/part tables. A "plan" session is just a session with `session_type = 'plan'`. This is the **recommended approach** — see Section 3.3.2.

**Recommended approach**: Add `session_type` column to the `session` table:

```sql
ALTER TABLE session ADD COLUMN session_type TEXT NOT NULL DEFAULT 'build';
-- 'build' = regular coding session, 'plan' = planning session
```

This way, plans reuse all existing message/part/message infrastructure without any schema changes to those tables. The `plan` table then serves as metadata (locked status, link to tasks) on top of a "plan-type" session.

#### 3.3.2 `internal/task/` — Task Store & Logic

```go
// internal/task/task.go
type Task struct {
    ID            string
    PlanID        string
    SessionID     *string    // nil until execution starts
    ParentTaskID  *string
    Title         string
    Description   string
    Effort        string     // S, M, L, XL
    Complexity    string    // low, medium, high
    Status        string     // pending, in_progress, completed, failed
    Dependencies  []string   // task IDs
    BranchName    string
    PRURL         string
    PRNumber      *int
    OrderIndex    int
    CreatedAt     int64
    UpdatedAt     int64
}

// internal/task/store.go
type Store struct {
    db *db.DB
}

func NewStore(database *db.DB) *Store
func (s *Store) Create(task *Task) error
func (s *Store) Get(id string) (*Task, error)
func (s *Store) ListByPlan(planID string) ([]*Task, error)
func (s *Store) Update(task *Task) error
func (s *Store) UpdateStatus(id string, status string) error
func (s *Store) GetReadyTasks(planID string) ([]*Task, error)  // tasks whose deps are completed
```

### 3.4 Plan Agent

A new agent definition — the **Plan Agent** — with read-only tools:

```go
// internal/agent/agent.go — addition
var PlanAgent = Agent{
    ID:          "plan",
    Name:        "Plan",
    Description: "Planning agent — reads and understands code, plans changes but never writes",
    Tools:       []string{"bash", "read", "glob", "grep"},  // NO write, edit
    System: `You are a planning agent. Your job is to understand the user's problem and codebase,
then collaboratively plan a solution. You MUST NOT write or modify any code. You can:
- Read files to understand the codebase
- Search and navigate the project structure
- Discuss architecture, approach, and trade-offs with the user
- Produce detailed implementation plans

When the user is satisfied with the plan, they will break it into tasks for execution.
Be thorough: understand existing patterns, identify affected files, consider edge cases.`,
}
```

### 3.5 Task Breakdown Flow

When the user clicks "Break into Tasks" CTA:

1. **Lock the plan** — Set `plan.status = 'locked'`, prevent further messages.
2. **Send breakdown prompt** — The locked plan conversation is sent to the agent with a special system prompt instructing it to produce a structured task breakdown.
3. **Parse task breakdown** — The agent returns structured JSON (or markdown with a defined schema) listing tasks with:
   - Title
   - Description
   - Dependencies (references to other task indices)
   - Effort (S/M/L/XL)
   - Complexity (low/medium/high)
   - Suggested order
4. **Create task records** — Store each task in the `task` table, linked to the plan.
5. **Create git branches** — For each task, create a branch `task/<task-id>-<slug>` from main.
6. **Display Kanban board** — Frontend shows the task board with columns for each status.

### 3.6 Task Execution Flow

When a task moves to execution:

1. **Create execution session** — A new session is created for the task (type = 'build'), linked to the task's git branch.
2. **Inject plan context** — The task's description + plan summary are injected as the first system message.
3. **Agent executes** — The BuildAgent works on the task in its isolated branch.
4. **User can observe** — The task execution view shows the agent's real-time activity (tool calls, file changes, streaming output).
5. **Auto PR on completion** — When the agent signals task completion:
   - Push the branch to origin
   - Create a PR via `gh pr create` (or Git API)
   - Update `task.pr_url`, `task.pr_number`, `task.status = 'completed'`

### 3.7 API Endpoints

#### Plan Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/plans` | List all plans for directory |
| `POST` | `/api/plans` | Create a new plan |
| `GET` | `/api/plans/:id` | Get plan by ID |
| `PATCH` | `/api/plans/:id` | Update plan (title, model) |
| `DELETE` | `/api/plans/:id` | Delete a plan |
| `POST` | `/api/plans/:id/lock` | Lock a plan (break-into-tasks CTA) |
| `POST` | `/api/plans/:id/prompt` | Send a message in plan conversation |
| `GET` | `/api/plans/:id/message` | Get plan messages |

#### Task Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/plans/:planId/tasks` | List all tasks for a plan |
| `POST` | `/api/plans/:planId/tasks` | Create tasks (from breakdown) |
| `GET` | `/api/tasks/:id` | Get task by ID |
| `PATCH` | `/api/tasks/:id` | Update task (status, assignee) |
| `POST` | `/api/tasks/:id/start` | Start task execution (creates session + branch) |
| `POST` | `/api/tasks/:id/complete` | Mark task complete + raise PR |

### 3.8 Server Changes

The `Server` struct needs new stores:

```go
type Server struct {
    // ... existing fields ...
    planStore   *plan.Store
    taskStore   *task.Store
}
```

The `Start()` method initializes both stores, and `routes()` adds the new endpoints.

### 3.9 Frontend Changes

#### 3.9.1 Plan Mode Landing Page

Instead of the session list + chat view, Plan Mode shows:

1. **Plan List View** — List of all plans (open + locked) for the project
2. **Create New Plan** — Button to start a new plan
3. **Plan Conversation View** — Open plan → interactive conversation with PlanAgent
4. **Break into Tasks CTA** — Appears when the plan feels ready (or user triggers)

#### 3.9.2 Kanban / Task Board

Once a plan is broken into tasks:

1. **Kanban columns** — `Pending` | `In Progress` | `Completed` | `Failed`
2. **Task cards** — Title, effort badge, complexity badge, dependency indicator
3. **Click to expand** — See task description, linked plan, and current status
4. **Jump into execution** — "View Execution" button on in-progress tasks

#### 3.9.3 Task Execution View

When jumping into a task:

1. Reuses the existing **session chat view** with the agent working on the task
2. Shows a header/breadcrumb: `Plan > Task #3: "Add auth middleware" > Execution`
3. Real-time streaming of agent activity (tool calls, file changes)
4. Back button to return to the task board

---

## 4. Data Flow

```
User types: ogcode plan
                    │
                    ▼
          ┌─────────────────┐
          │  Plan Mode Server │
          │  (PlanAgent only) │
          └─────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
   ┌────▼─────┐          ┌─────▼──────┐
   │ Plan List │          │ New Plan   │
   │ View      │          │ Creation  │
   └───────────┘          └────────────┘
                                │
                    ┌───────────▼───────────┐
                    │  Plan Conversation     │
                    │  (PlanAgent, read-only) │
                    └───────────┬────────────┘
                                │
                    [Break into Tasks CTA]
                                │
                    ┌───────────▼───────────┐
                    │  Plan Locked           │
                    │  ↓                     │
                    │  Agent Breakdown       │
                    │  ↓                     │
                    │  Tasks Created         │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼───────────┐
                    │  Kanban Task Board     │
                    │  Pending → In Progress │
                    │  → Completed           │
                    └───────────┬────────────┘
                                │
                    ┌───────────▼───────────┐
                    │  Task Execution View   │
                    │  (BuildAgent, full tools)│
                    │  - Git branch per task  │
                    │  - Auto PR on complete  │
                    └─────────────────────────┘
```

---

## 5. Plan-to-Task Linking (Traceability)

Every task stores a `plan_id` foreign key. This enables:

- **Back-tracing**: From any task, query its `plan_id` → load the original plan + messages
- **Forward-tracing**: From any plan, query all its tasks via `task.plan_id`
- **Referencing in PRs**: The auto-generated PR body includes:
  ```
  Plan: #<plan-id> — <plan-title>
  Task: #<task-id> — <task-title>
  ```
- **UI navigation**: Clickable links between plan ↔ task ↔ PR

Plans are **never deleted** — they remain accessible even after all tasks are completed, serving as permanent documentation.

---

## 6. Git Branch Strategy

```
main
  ├── task/abc123-add-auth-middleware
  ├── task/def456-update-user-model
  └── task/ghi789-write-auth-tests
```

- Each task creates a branch from `main` (or from a parent task's branch if there's a dependency).
- On task completion: push branch → create PR to `main`.
- If task A depends on task B, task A's branch is rebased/merged from B's branch before A starts.
- Git operations are performed via the existing `bash` tool in the agent.

---

## 7. CLI Entry Point

### `internal/cli/root.go` changes

```go
var planCmd = &cobra.Command{
    Use:   "plan",
    Short: "Start ogcode in Plan Mode",
    RunE: func(cmd *cobra.Command, args []string) error {
        dir, _ := os.Getwd()
        srv := server.New(port, dir, server.ModePlan)  // mode flag
        return srv.Start()
    },
}
```

The server distinguishes between `ModeBuild` (default) and `ModePlan` which changes:
1. Default agent (PlanAgent vs BuildAgent)
2. Available tools (read-only vs full)
3. Frontend routing (plan UI vs chat UI)
4. Available API endpoints

---

## 8. Implementation Phases

### Phase 1: Foundation (Backend)
1. Database migration (`006_plan_task.sql`)
2. `internal/plan/` — Plan model, store
3. `internal/task/` — Task model, store
4. PlanAgent definition in `internal/agent/agent.go`
5. Server mode concept (`ModeBuild` / `ModePlan`)
6. CLI `plan` subcommand
7. API routes for plans and tasks

### Phase 2: Plan Conversation
8. Plan message handling (reuse session message/parts with `session_type = 'plan'`)
9. Plan prompt/loop handling (PlanAgent loop, read-only enforcement)
10. Plan lock mechanism
11. SSE events for plan streaming

### Phase 3: Task Breakdown
12. Task breakdown system prompt for the agent
13. Plan lock → task breakdown flow
14. Task creation from agent response
15. Dependency graph computation
16. Git branch creation per task

### Phase 4: Task Execution
17. Task start — create build session on task branch
18. Inject plan context into task session
19. Real-time execution view (reuse session chat components)
20. Task completion detection
21. Auto PR creation via `gh` CLI or Git API

### Phase 5: Frontend
22. Plan Mode landing page (plan list + create)
23. Plan conversation view
24. "Break into Tasks" CTA button
25. Kanban task board
26. Task execution view (session chat with task context)
27. Breadcrumb navigation (Plan > Tasks > Execution)
28. Plan ↔ Task ← → PR cross-linking in UI

### Phase 6: Polish & Edge Cases
29. Parallel task execution (multiple agents simultaneously)
30. Error handling for git operations (merge conflicts, branch exists)
31. Plan history / archive view
32. Task reordering / manual dependency editing
33. Plan import/export (markdown)
34. Notification system for task state changes

---

## 9. Future Considerations

- **Breakdown Agent**: A dedicated LLM agent for task breakdown, separate from PlanAgent. The `Agent` struct already supports multiple agent types.
- **Multi-Agent Task Assignment**: Tasks can be assigned to different specialized agents. The `task` table already has `session_id` to track which agent runs it.
- **Manual Task Assignment**: Users can manually assign tasks to specific agents from the UI.
- **Task Dependencies UI**: Visual dependency graph in the frontend.
- **Plan Templates**: Save common plan structures as templates.
- **Plan Versioning**: Track plan iterations (v1 locked → v2 new plan for follow-up work).

---

## 10. Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Plan message storage | Add `session_type` column to session table | Reuses all existing message/part infrastructure, minimal schema changes |
| Plan locking | DB-level status change + API enforcement | Simple, reliable, no race conditions |
| Task breakdown agent | Default BuildAgent with special prompt | No new agent needed now; architecture supports dedicated agent later |
| Git branch per task | `task/<id>-<slug>` naming | Clear, traceable, avoids conflicts |
| PR creation | `gh pr create` via bash tool | Leverages existing GitHub CLI, no OAuth flow needed |
| Frontend routing | Mode-based rendering | Plan mode shows planning UI, build mode shows chat UI, from the same server |