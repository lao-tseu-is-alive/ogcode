package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prasenjeet-symon/ogcode/internal/git"
	"github.com/prasenjeet-symon/ogcode/internal/id"
	"github.com/prasenjeet-symon/ogcode/internal/plan"
	"github.com/prasenjeet-symon/ogcode/internal/session"
	"github.com/prasenjeet-symon/ogcode/internal/task"
)

const taskExecutionTimeout = 30 * time.Minute

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")
	tasks, err := s.taskStore.ListByPlan(planID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []*task.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleCreateTasks(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")

	p, err := s.planStore.Get(planID)
	if err != nil || p == nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}

	var input struct {
		Tasks []struct {
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Effort       string   `json:"effort"`
			Complexity   string   `json:"complexity"`
			Dependencies []string `json:"dependencies"`
			OrderIndex   int      `json:"orderIndex"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var created []*task.Task
	now := task.Now()
	for _, t := range input.Tasks {
		effort := t.Effort
		if effort == "" {
			effort = task.EffortM
		}
		complexity := t.Complexity
		if complexity == "" {
			complexity = task.ComplexityMedium
		}

		newTask := &task.Task{
			ID:           string(id.NewTaskID()),
			PlanID:       planID,
			Title:        t.Title,
			Description:  t.Description,
			Effort:       effort,
			Complexity:   complexity,
			Status:       task.StatusPending,
			Dependencies: t.Dependencies,
			OrderIndex:   t.OrderIndex,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.taskStore.Create(newTask); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		created = append(created, newTask)
	}

	s.bus.Publish("task.created", map[string]any{"planId": planID, "count": len(created)})
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	var update struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Effort      *string `json:"effort"`
		Complexity  *string `json:"complexity"`
		Status      *string `json:"status"`
		BranchName  *string `json:"branchName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if update.Title != nil {
		t.Title = *update.Title
	}
	if update.Description != nil {
		t.Description = *update.Description
	}
	if update.Effort != nil {
		switch *update.Effort {
		case task.EffortS, task.EffortM, task.EffortL, task.EffortXL:
			t.Effort = *update.Effort
		default:
			http.Error(w, "invalid effort: "+*update.Effort, http.StatusBadRequest)
			return
		}
	}
	if update.Complexity != nil {
		switch *update.Complexity {
		case task.ComplexityLow, task.ComplexityMedium, task.ComplexityHigh:
			t.Complexity = *update.Complexity
		default:
			http.Error(w, "invalid complexity: "+*update.Complexity, http.StatusBadRequest)
			return
		}
	}
	if update.Status != nil {
		newStatus := *update.Status
		validTransitions := map[string][]string{
			task.StatusPending:    {task.StatusInProgress},
			task.StatusInProgress: {task.StatusCompleted, task.StatusFailed},
			task.StatusCompleted:  {},
			task.StatusFailed:     {task.StatusPending},
		}
		allowed, ok := validTransitions[t.Status]
		if !ok {
			http.Error(w, "invalid current status: "+t.Status, http.StatusBadRequest)
			return
		}
		valid := false
		for _, s := range allowed {
			if s == newStatus {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, fmt.Sprintf("cannot transition from %s to %s", t.Status, newStatus), http.StatusBadRequest)
			return
		}
		t.Status = newStatus
	}
	if update.BranchName != nil {
		t.BranchName = *update.BranchName
	}
	t.UpdatedAt = task.Now()

	if err := s.taskStore.Update(t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.bus.Publish("task.updated", t)
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleStartTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if t.SessionID != nil {
		http.Error(w, "task already started", http.StatusBadRequest)
		return
	}

	p, _ := s.planStore.Get(t.PlanID)
	if p == nil {
		http.Error(w, "plan not found", http.StatusNotFound)
		return
	}
	if p.Status != plan.StatusLocked {
		http.Error(w, "plan is not locked", http.StatusBadRequest)
		return
	}

	for _, depID := range t.Dependencies {
		dep, err := s.taskStore.Get(depID)
		if err != nil {
			http.Error(w, "dependency not found", http.StatusBadRequest)
			return
		}
		if dep != nil && dep.Status != task.StatusCompleted {
			http.Error(w, fmt.Sprintf("dependency %s not completed (status: %s)", depID, dep.Status), http.StatusBadRequest)
			return
		}
	}

	if err := s.executeTask(t, p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// executeTask creates a worktree, session, and starts an agent loop for the task.
// It uses an atomic DB claim to prevent double-starts under concurrent auto-start.
func (s *Server) executeTask(t *task.Task, p *plan.Plan) error {
	// In-memory fast guard (reduces DB load in the common case)
	if t.SessionID != nil {
		return fmt.Errorf("task already has a session")
	}
	if t.Status != task.StatusPending {
		return fmt.Errorf("task is not pending (status: %s)", t.Status)
	}

	// Resolve the base branch: if this task depends on exactly one predecessor,
	// branch from that task's branch so PRs stack correctly.
	// Read the dependency branch name from the DB before acquiring gitMu so we
	// don't hold the lock longer than necessary.
	baseBranch := ""
	if len(t.Dependencies) > 0 {
		dep, depErr := s.taskStore.Get(t.Dependencies[0])
		if depErr != nil {
			return fmt.Errorf("get dependency task: %w", depErr)
		}
		if dep != nil && dep.BranchName != "" {
			baseBranch = dep.BranchName
		}
	}

	// Serialize repo-level git operations. git worktree add/prune/branch are
	// not safe under concurrent access — they write to .git/worktrees/ and
	// .git/config without holding git's own lock for the full sequence.
	s.gitMu.Lock()
	// If we have a dependency branch, make sure it exists locally while we
	// hold the lock (before the prior task's cleanup goroutine can delete it).
	if baseBranch != "" {
		if err := git.EnsureLocalBranch(s.dir, baseBranch); err != nil {
			slog.Warn("could not ensure local dependency branch, falling back to HEAD",
				"branch", baseBranch, "err", err)
			baseBranch = ""
		}
	}
	slug := git.Slugify(t.Title)
	wt, wtErr := git.CreateTaskWorktree(s.dir, t.ID, slug, baseBranch)
	s.gitMu.Unlock()

	if wtErr != nil {
		return fmt.Errorf("create worktree: %w", wtErr)
	}

	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   s.dir,
		Directory:   wt.Path,
		Title:       "Task: " + t.Title,
		Model:       "",
		SessionType: "build",
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if p.Model != "" {
		sess.Model = p.Model
	}
	if err := s.store.Create(sess); err != nil {
		s.gitMu.Lock()
		_ = git.RemoveTaskWorktreeKeepBranch(s.dir, wt.BranchName)
		s.gitMu.Unlock()
		return fmt.Errorf("create session: %w", err)
	}

	// Atomic DB claim — only one goroutine wins when auto-start races with itself.
	claimed, err := s.taskStore.Claim(t.ID, string(sess.ID), wt.BranchName, wt.Path)
	if err != nil {
		s.gitMu.Lock()
		_ = git.RemoveTaskWorktreeKeepBranch(s.dir, wt.BranchName)
		s.gitMu.Unlock()
		_ = s.store.Delete(sess.ID)
		return fmt.Errorf("claim task: %w", err)
	}
	if !claimed {
		s.gitMu.Lock()
		_ = git.RemoveTaskWorktreeKeepBranch(s.dir, wt.BranchName)
		s.gitMu.Unlock()
		_ = s.store.Delete(sess.ID)
		slog.Info("task already claimed by another goroutine, skipping", "task", t.ID)
		return fmt.Errorf("task already started")
	}

	// Update in-memory struct to reflect DB state
	sessionIDStr := string(sess.ID)
	t.SessionID = &sessionIDStr
	t.BranchName = wt.BranchName
	t.WorktreePath = wt.Path
	t.Status = task.StatusInProgress
	t.UpdatedAt = task.Now()

	promptContent := fmt.Sprintf(
		"**Task**: %s\n\n"+
			"**Branch**: `%s`\n\n"+
			"---\n\n%s\n\n---\n\n"+
			"You are already checked out on branch `%s` in the task worktree. "+
			"Implement the task, then commit ALL your changes before finishing:\n"+
			"```\n"+
			"git add -A && git commit -m 'your descriptive message'\n"+
			"```\n\n"+
			"Focus only on what this task requires.",
		t.Title, wt.BranchName, t.Description, wt.BranchName)

	userMsg := &session.MessageInfo{
		ID:        session.NewMessageID(),
		SessionID: sess.ID,
		Role:      session.RoleUser,
		CreatedAt: session.Now(),
	}
	if err := s.store.CreateMessage(userMsg); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	textData, _ := json.Marshal(session.TextPartData{Text: promptContent})
	userPart := &session.Part{
		ID:        session.NewPartID(),
		MessageID: userMsg.ID,
		SessionID: sess.ID,
		Type:      session.PartText,
		Data:      textData,
		CreatedAt: session.Now(),
		UpdatedAt: session.Now(),
	}
	if err := s.store.CreatePart(userPart); err != nil {
		return fmt.Errorf("create part: %w", err)
	}

	s.bus.Publish("message.updated", userMsg)

	// 30-minute timeout per task; user abort uses context.Canceled,
	// timeout uses context.DeadlineExceeded — handled separately below.
	ctx, cancel := context.WithTimeout(context.Background(), taskExecutionTimeout)

	s.mu.Lock()
	if old, ok := s.running[sess.ID]; ok {
		old()
	}
	s.nextToken++
	token := s.nextToken
	s.running[sess.ID] = cancel
	s.runningToken[sess.ID] = token
	s.mu.Unlock()

	taskSnapshot := *t
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("task agent loop panicked", "task", taskSnapshot.ID, "panic", r)
				func() {
					defer func() { _ = recover() }()
					s.autoFailTask(&taskSnapshot)
				}()
			}
		}()
		defer func() {
			s.mu.Lock()
			if s.runningToken[sess.ID] == token {
				delete(s.running, sess.ID)
				delete(s.runningToken, sess.ID)
			}
			s.mu.Unlock()
			cancel()
		}()

		loopErr := s.loopRunner.RunLoop(ctx, sess.ID, "build")

		switch ctx.Err() {
		case context.Canceled:
			slog.Info("task agent loop cancelled by user", "task", taskSnapshot.ID)
			return
		case context.DeadlineExceeded:
			slog.Warn("task agent loop timed out", "task", taskSnapshot.ID, "timeout", taskExecutionTimeout)
			s.autoFailTask(&taskSnapshot)
			return
		}

		if loopErr != nil {
			slog.Error("task agent loop error", "task", taskSnapshot.ID, "err", loopErr)
			s.autoFailTask(&taskSnapshot)
		} else {
			s.autoCompleteTask(&taskSnapshot)
		}
	}()

	s.bus.Publish("task.started", t)
	return nil
}

// completeTaskWithPR handles the post-completion steps shared by autoCompleteTask
// and handleCompleteTask: auto-commit any leftover changes, push the branch,
// open a PR via gh, and clean up the worktree.
// Any reason a PR was not created is stored in t.PRError so the UI can surface it.
func (s *Server) completeTaskWithPR(t *task.Task) {
	// Auto-commit any changes the agent left uncommitted.
	if t.WorktreePath != "" {
		msg := fmt.Sprintf("chore: complete task %q", t.Title)
		if err := git.CommitAllChanges(t.WorktreePath, msg); err != nil {
			slog.Warn("auto-commit before push", "task", t.ID, "err", err)
		}
	}

	// Bound all network operations (push + PR creation) to 3 minutes total.
	// Without a deadline, a slow or unresponsive GitHub can stall this goroutine
	// indefinitely and block dependent task auto-start.
	const prNetworkTimeout = 3 * time.Minute
	prCtx, prCancel := context.WithTimeout(context.Background(), prNetworkTimeout)
	defer prCancel()

	prErr := ""
	pushed := false
	if t.BranchName == "" {
		prErr = "no branch name assigned to task"
	} else {
		var pushErr error
		pushed, pushErr = git.PushBranch(prCtx, s.dir, t.BranchName)
		switch {
		case pushErr != nil:
			prErr = fmt.Sprintf("push failed: %s", pushErr.Error())
			slog.Error("push branch", "task", t.ID, "branch", t.BranchName, "err", pushErr)
		case !pushed:
			prErr = "no remote configured — add a GitHub remote (git remote add origin <url>) to enable automatic PR creation"
			slog.Warn("push skipped: no remote configured", "task", t.ID, "branch", t.BranchName)
		}
	}

	if pushed {
		// For dependent tasks, target the dependency's branch so the PR stacks.
		prBase := ""
		if len(t.Dependencies) > 0 {
			if dep, depErr := s.taskStore.Get(t.Dependencies[0]); depErr == nil && dep != nil {
				prBase = dep.BranchName
			}
		}
		prTitle := t.Title
		prBody := fmt.Sprintf("## Description\n\n%s\n\n---\n\nCo-authored-by: ogcode <ogcode@local>", t.Description)
		pr, err := git.CreatePR(prCtx, s.dir, t.BranchName, prTitle, prBody, prBase)
		if err != nil {
			prErr = fmt.Sprintf("PR creation failed: %s", err.Error())
			slog.Warn("create PR failed", "task", t.ID, "err", err)
		} else if pr != nil {
			t.PRURL = pr.URL
			t.PRNumber = &pr.Number
		}
	}

	// Persist PR outcome (URL on success, error reason on failure).
	t.PRError = prErr
	t.UpdatedAt = task.Now()
	if err := s.taskStore.Update(t); err != nil {
		slog.Error("save PR info", "task", t.ID, "err", err)
	}

	// Remove worktree. Keep the branch if not pushed (work stays accessible locally).
	// Run in a goroutine to avoid blocking the completion path, but serialize
	// through gitMu so it doesn't race with concurrent worktree-add calls.
	if t.WorktreePath != "" {
		branchName := t.BranchName
		go func() {
			s.gitMu.Lock()
			defer s.gitMu.Unlock()
			var err error
			if pushed {
				err = git.RemoveTaskWorktree(s.dir, branchName)
			} else {
				err = git.RemoveTaskWorktreeKeepBranch(s.dir, branchName)
			}
			if err != nil {
				slog.Warn("remove worktree", "task", t.ID, "err", err)
			}
		}()
	}
}

// autoStartDependentTasks finds all pending tasks whose dependencies are all
// completed and starts them automatically. Called after a task completes.
func (s *Server) autoStartDependentTasks(planID string) {
	ready, err := s.taskStore.GetReadyTasks(planID)
	if err != nil {
		slog.Error("get ready tasks for auto-start", "plan", planID, "err", err)
		return
	}
	if len(ready) == 0 {
		return
	}

	p, err := s.planStore.Get(planID)
	if err != nil || p == nil || p.Status != plan.StatusLocked {
		return
	}

	slog.Info("auto-starting dependent tasks", "plan", planID, "count", len(ready))
	for _, t := range ready {
		taskCopy := t
		if err := s.executeTask(taskCopy, p); err != nil {
			slog.Error("auto-start task failed", "task", t.ID, "title", t.Title, "err", err)
		}
	}
}

func (s *Server) autoCompleteTask(t *task.Task) {
	if err := s.taskStore.UpdateStatus(t.ID, task.StatusCompleted); err != nil {
		slog.Error("auto-complete: update task status", "task", t.ID, "err", err)
		return
	}
	t.Status = task.StatusCompleted
	t.UpdatedAt = task.Now()

	// Start dependent tasks BEFORE removing the worktree. This ensures
	// that dependent tasks can still find this branch as a local ref when
	// they call EnsureLocalBranch inside executeTask — even when no remote
	// is configured. completeTaskWithPR's async removal goroutine will then
	// block on gitMu until after the dependent's CreateTaskWorktree finishes.
	s.autoStartDependentTasks(t.PlanID)

	s.completeTaskWithPR(t)

	fresh, err := s.taskStore.Get(t.ID)
	if err == nil && fresh != nil {
		t = fresh
	}
	s.bus.Publish("task.completed", t)

	go s.tryArchivePlan(t.PlanID)
}

func (s *Server) autoFailTask(t *task.Task) {
	if err := s.taskStore.UpdateStatus(t.ID, task.StatusFailed); err != nil {
		slog.Error("auto-fail: update task status", "task", t.ID, "err", err)
		return
	}
	t.Status = task.StatusFailed
	t.UpdatedAt = task.Now()

	if t.WorktreePath != "" {
		go func() {
			if err := git.RemoveTaskWorktreeKeepBranch(s.dir, t.BranchName); err != nil {
				slog.Warn("auto-fail: remove worktree", "task", t.ID, "err", err)
			}
		}()
	}

	fresh, err := s.taskStore.Get(t.ID)
	if err == nil && fresh != nil {
		t = fresh
	}
	s.bus.Publish("task.failed", t)
}

func (s *Server) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if t.Status != task.StatusInProgress {
		http.Error(w, fmt.Sprintf("task is %s, not in_progress", t.Status), http.StatusBadRequest)
		return
	}

	if err := s.taskStore.UpdateStatus(taskID, task.StatusCompleted); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Status = task.StatusCompleted
	t.UpdatedAt = task.Now()

	s.completeTaskWithPR(t)

	fresh, dbErr := s.taskStore.Get(t.ID)
	if dbErr == nil && fresh != nil {
		t = fresh
	}
	s.bus.Publish("task.completed", t)
	writeJSON(w, http.StatusOK, t)

	go s.autoStartDependentTasks(t.PlanID)
	go s.tryArchivePlan(t.PlanID)
}

func (s *Server) handleFailTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if t.Status != task.StatusInProgress {
		http.Error(w, fmt.Sprintf("task is %s, not in_progress", t.Status), http.StatusBadRequest)
		return
	}

	if err := s.taskStore.UpdateStatus(taskID, task.StatusFailed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Status = task.StatusFailed
	t.UpdatedAt = task.Now()

	if t.WorktreePath != "" {
		go func() {
			if err := git.RemoveTaskWorktreeKeepBranch(s.dir, t.BranchName); err != nil {
				slog.Warn("fail: remove worktree", "task", t.ID, "err", err)
			}
		}()
	}

	fresh, dbErr := s.taskStore.Get(t.ID)
	if dbErr == nil && fresh != nil {
		t = fresh
	}
	s.bus.Publish("task.failed", t)
	writeJSON(w, http.StatusOK, t)
}

// handleRetryTask performs a clean retry of a failed task: deletes the stale
// branch, resets all runtime fields in the DB, then re-executes immediately if
// the plan is locked and all dependencies are still completed.
func (s *Server) handleRetryTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	t, err := s.taskStore.Get(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	if t.Status != task.StatusFailed {
		http.Error(w, fmt.Sprintf("task is %s, not failed", t.Status), http.StatusBadRequest)
		return
	}

	// Remove the stale branch so the next execution starts from a clean slate.
	if t.BranchName != "" {
		if err := git.DeleteBranch(s.dir, t.BranchName); err != nil {
			slog.Warn("retry: delete stale branch", "task", t.ID, "branch", t.BranchName, "err", err)
		}
	}

	if err := s.taskStore.ResetFailed(t.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Status = task.StatusPending
	t.SessionID = nil
	t.BranchName = ""
	t.WorktreePath = ""
	t.PRURL = ""
	t.PRNumber = nil

	p, _ := s.planStore.Get(t.PlanID)
	if p != nil && p.Status == plan.StatusLocked {
		depsOK := true
		for _, depID := range t.Dependencies {
			dep, _ := s.taskStore.Get(depID)
			if dep == nil || dep.Status != task.StatusCompleted {
				depsOK = false
				break
			}
		}
		if depsOK {
			if err := s.executeTask(t, p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	fresh, _ := s.taskStore.Get(t.ID)
	if fresh != nil {
		t = fresh
	}
	s.bus.Publish("task.retried", t)
	writeJSON(w, http.StatusOK, t)
}
