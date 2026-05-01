package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/ogcode/ogcode/internal/bus"
	"github.com/ogcode/ogcode/internal/id"
	"github.com/ogcode/ogcode/internal/mcp"
	"github.com/ogcode/ogcode/internal/memory"
	"github.com/ogcode/ogcode/internal/provider"
	"github.com/ogcode/ogcode/internal/session"
	"github.com/ogcode/ogcode/internal/tool"
)

// LoopRunner orchestrates the agent loop for a session.
type LoopRunner struct {
	Store           *session.Store
	Bus             *bus.Bus
	Registry        *provider.Registry
	DefaultProvider provider.Provider
	Tools           *tool.Registry
	Dir             string
	MaxSteps        int
	Memory          *memory.Memory
	MCP             *mcp.Client
}

// RunLoop executes the core agent loop: prompt -> stream -> tools -> loop back.
func (lr *LoopRunner) RunLoop(ctx context.Context, sessionID session.SessionID, agentName string) error {
	agent := GetAgent(agentName)
	if lr.MaxSteps == 0 {
		lr.MaxSteps = 1000
	}

	// Always notify the frontend when the loop exits, regardless of reason.
	// Without this, any early return (DB error, stream error, panic recovery)
	// leaves the client in a permanently-stuck loading state.
	exitReason := "error"
	defer func() {
		lr.Bus.Publish("loop.done", map[string]string{
			"sessionId": string(sessionID),
			"reason":    exitReason,
		})
	}()

	// Resolve provider based on session's model
	sess, err := lr.Store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	// Load AGENT.md files from session directory
	workDir := lr.Dir
	if sess != nil && sess.Directory != "" {
		workDir = sess.Directory
	}
	agentMDContent := LoadAgentMD(workDir)

	var p provider.Provider
	var modelID string
	if sess != nil && sess.Model != "" {
		p = lr.Registry.ResolveProvider(sess.Model)
		modelID = sess.Model
	}
	if p == nil {
		p = lr.DefaultProvider
	}

	// Restore compaction summary from a previous turn (persisted in the session row).
	// Applied on every step so all future LLM calls stay within the context window.
	compactionSummary := ""
	if sess != nil {
		compactionSummary = sess.CompactionSummary
	}

	slog.Info("agent loop starting", "session", sessionID, "agent", agent.ID, "model", modelID)

	// Agentic memory: read graph context before the loop
	var memoryText string
	if lr.Memory != nil && lr.Memory.Enabled() {
		graphText := lr.Memory.ReadMemory(ctx, string(sessionID))
		if graphText != "" {
			memoryText = graphText
		}
		// If not the first turn, also recall relevant context
		messages, _ := lr.Store.GetMessages(sessionID, "", 1000)
		if len(messages) > 1 {
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Info.Role == session.RoleUser {
					var userText string
					for _, p := range messages[i].Parts {
						if p.Type == session.PartText {
							var data session.TextPartData
							if json.Unmarshal(p.Data, &data) == nil && data.Text != "" {
								userText = data.Text
							}
						}
					}
					if userText != "" {
						recallText := lr.Memory.RecallMemory(ctx, string(sessionID), userText)
						if recallText != "" {
							if memoryText != "" {
								memoryText += "\n\n### Relevant Context\n" + recallText
							} else {
								memoryText = recallText
							}
						}
					}
					break
				}
			}
		}
	}

	for step := 1; step <= lr.MaxSteps; step++ {
		if step == lr.MaxSteps {
			slog.Warn("agent loop reached MaxSteps limit", "session", sessionID, "maxSteps", lr.MaxSteps)
		}
		slog.Info("agent loop step", "session", sessionID, "step", step)

		// Check for context cancellation at the start of each loop iteration
		if ctx.Err() != nil {
			slog.Info("agent loop cancelled", "session", sessionID, "step", step)
			exitReason = "aborted"
			return ctx.Err()
		}

		// Load all messages for this session (retry on DB contention)
		var messages []*session.MessageWithParts
		for dbAttempt := 0; dbAttempt < 3; dbAttempt++ {
			messages, err = lr.Store.GetMessages(sessionID, "", 1000)
			if err == nil {
				break
			}
			slog.Warn("get messages failed, retrying", "session", sessionID, "attempt", dbAttempt+1, "err", err)
			if dbAttempt < 2 {
				time.Sleep(time.Duration(dbAttempt+1) * 500 * time.Millisecond)
			}
		}
		if err != nil {
			return fmt.Errorf("load messages: %w", err)
		}

		// Check if we should continue: last assistant finished means done
		if shouldBreak(messages) {
			last := messages[len(messages)-1]
			finish := "stop"
			if last.Info.Finish != nil {
				finish = *last.Info.Finish
			}
			slog.Info("agent loop breaking", "session", sessionID, "reason", "last assistant finished", "finish", finish, "totalMessages", len(messages))
			exitReason = finish
			if lr.Memory != nil && lr.Memory.Enabled() {
				lr.writeMemory(ctx, sessionID)
			}
			return nil
		}

		// Create new assistant message
		assistantID := session.MessageID(id.NewMessageID())
		assistantMsg := &session.MessageInfo{
			ID:        assistantID,
			SessionID: sessionID,
			Role:      session.RoleAssistant,
			Agent:     agent.ID,
			ParentID:  lastUserMessageID(messages),
			CreatedAt: session.Now(),
		}
		if err := lr.Store.CreateMessage(assistantMsg); err != nil {
			return fmt.Errorf("create assistant message: %w", err)
		}

		// Resolve tools for this agent
		agentTools := lr.Tools.ForAgent(agent.Tools)
		providerTools := make([]provider.ToolDefinition, 0, len(agentTools))
		for _, t := range agentTools {
			providerTools = append(providerTools, provider.ToolDefinition{
				Name:        t.ID(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			})
		}

		// Add MCP tools if available
		if lr.MCP != nil {
			for name, td := range lr.MCP.Tools() {
				providerTools = append(providerTools, provider.ToolDefinition{
					Name:        name,
					Description: td.Description,
					Parameters:  td.InputSchema,
				})
			}
		}
		slog.Info("resolved tools", "count", len(providerTools))

		// Build system prompt
		var mcpTools map[string]mcp.ToolDef
		if lr.MCP != nil {
			mcpTools = lr.MCP.Tools()
		}
		system := buildSystemPrompt(agent, lr.Dir, lr.Memory != nil && lr.Memory.Enabled(), agentMDContent, mcpTools)

		// Convert messages to provider format (with memory context filtering)
		modelMessages := toProviderMessages(messages, memoryText)

		// If a compaction summary exists (from this turn or a prior one), trim the
		// provider messages to the recent window so the same context limit isn't hit
		// again, and inject the summary into the system prompt.
		systemPrompts := []string{system}
		if compactionSummary != "" {
			modelMessages = trimToRecent(modelMessages, 12)
			systemPrompts = append(systemPrompts, compactionSummary)
		}

		// Stream from LLM with retry for transient errors
		streamReq := provider.StreamRequest{
			Model:    modelID,
			System:   systemPrompts,
			Messages: modelMessages,
			Tools:    providerTools,
			Abort:    ctx,
		}
		slog.Info("calling LLM", "session", sessionID, "step", step, "model", modelID, "messages", len(modelMessages))

		var streamCh <-chan provider.StreamEvent
		var streamErr error
		compactionCount := 0
		const maxCompactions = 2
		const maxRetries = 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			streamCh, streamErr = p.StreamChat(ctx, streamReq)
			if streamErr == nil {
				break
			}
			slog.Warn("stream chat attempt failed", "session", sessionID, "attempt", attempt, "err", streamErr)
			// Context length exceeded: summarize old history with the LLM and retry
			if compactionCount < maxCompactions && isContextLengthError(streamErr) {
				before := len(streamReq.Messages)
				slog.Info("context length exceeded, using LLM to compact history", "session", sessionID, "messages", before)
				summaryAddendum, compactedMsgs := lr.llmCompact(ctx, p, modelID, streamReq.Messages)
				streamReq.Messages = compactedMsgs
				if summaryAddendum != "" {
					streamReq.System = append(streamReq.System, summaryAddendum)
					// Persist so future steps and future turns reuse the same summary.
					// Use the column-specific updater to avoid overwriting concurrent
					// changes to other session fields (title, model, etc.).
					compactionSummary = summaryAddendum
					if err := lr.Store.UpdateCompactionSummary(sessionID, summaryAddendum); err != nil {
						slog.Error("persist compaction summary", "err", err)
					}
				}
				compactionCount++
				slog.Info("llm-compacted context", "session", sessionID, "before", before, "after", len(streamReq.Messages))
				lr.Bus.Publish("loop.compacted", map[string]string{"sessionId": string(sessionID)})
				attempt-- // don't count compaction as a retry attempt
				continue
			}
			if attempt < maxRetries && isTransientError(streamErr) {
				backoff := time.Duration(attempt*attempt) * time.Second
				slog.Info("retrying stream chat", "session", sessionID, "backoff", backoff)
				select {
				case <-ctx.Done():
					exitReason = "aborted"
					return ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			// Non-transient or exhausted retries
			errStr := streamErr.Error()
			assistantMsg.Error = &errStr
			finish := "error"
			assistantMsg.Finish = &finish
			lr.Store.UpdateMessage(assistantMsg)
			lr.Bus.Publish("message.updated", assistantMsg)
			return fmt.Errorf("stream chat: %w", streamErr)
		}
		// Guard: loop exhausted all attempts via continue without returning (edge case)
		if streamErr != nil {
			errStr := streamErr.Error()
			assistantMsg.Error = &errStr
			finish := "error"
			assistantMsg.Finish = &finish
			lr.Store.UpdateMessage(assistantMsg)
			lr.Bus.Publish("message.updated", assistantMsg)
			return fmt.Errorf("stream chat: %w", streamErr)
		}

		// Process stream events
		var currentText strings.Builder
		var currentReasoning strings.Builder
		var pendingToolCalls []pendingToolCall
		var finishReason string
		var streamUsage *provider.TokenUsage

		// Streaming text part: created on first delta and flushed to DB periodically
		// so the UI shows live output instead of waiting for the full response.
		var streamTextPart *session.Part
		var lastTextFlush time.Time
		const textFlushInterval = 300 * time.Millisecond

		// Streaming reasoning part: same pattern — accumulate deltas into one part
		// instead of creating a new Part per thinking chunk.
		var streamReasoningPart *session.Part
		var lastReasoningFlush time.Time
		const reasoningFlushInterval = 1 * time.Second

		flushReasoningPart := func(final bool) {
			if currentReasoning.Len() == 0 || streamReasoningPart == nil {
				return
			}
			if !final && time.Since(lastReasoningFlush) < reasoningFlushInterval {
				return
			}
			text := currentReasoning.String()
			// Truncate extremely long reasoning to avoid flooding DB and UI
			const maxReasoningLen = 50_000
			if len(text) > maxReasoningLen {
				text = text[:maxReasoningLen] + "\n... (truncated)"
			}
			reasonData, _ := json.Marshal(session.ReasoningPartData{Text: text})
			streamReasoningPart.Data = reasonData
			streamReasoningPart.UpdatedAt = session.Now()
			lr.Store.UpdatePart(streamReasoningPart)
			lastReasoningFlush = time.Now()
			lr.Bus.Publish("message.part.updated", map[string]string{
				"sessionId": string(sessionID),
				"partId":    string(streamReasoningPart.ID),
			})
		}

		flushTextPart := func(final bool) {
			if currentText.Len() == 0 || streamTextPart == nil {
				return
			}
			if !final && time.Since(lastTextFlush) < textFlushInterval {
				return
			}
			textData, _ := json.Marshal(session.TextPartData{Text: currentText.String()})
			streamTextPart.Data = textData
			streamTextPart.UpdatedAt = session.Now()
			lr.Store.UpdatePart(streamTextPart)
			lastTextFlush = time.Now()
			lr.Bus.Publish("message.part.updated", map[string]string{
				"sessionId": string(sessionID),
				"partId":    string(streamTextPart.ID),
			})
		}

		for evt := range streamCh {
			// Check for context cancellation while processing stream events
			if ctx.Err() != nil {
				slog.Info("agent loop cancelled during stream processing", "session", sessionID)
				flushTextPart(true)
				flushReasoningPart(true)
				// Mark the assistant message as aborted
				abortedReason := "aborted"
				assistantMsg.Finish = &abortedReason
				lr.Store.UpdateMessage(assistantMsg)
				lr.Bus.Publish("message.updated", assistantMsg)
				exitReason = "aborted"
				return ctx.Err()
			}

			switch evt.Type {
			case provider.EventTextDelta:
				currentText.WriteString(evt.Text)
				if streamTextPart == nil {
					// Create the part in DB on first delta so the client can see text arriving
					textData, _ := json.Marshal(session.TextPartData{Text: currentText.String()})
					p := &session.Part{
						ID:        session.PartID(id.NewPartID()),
						MessageID: assistantID,
						SessionID: sessionID,
						Type:      session.PartText,
						Data:      textData,
						CreatedAt: session.Now(),
						UpdatedAt: session.Now(),
					}
					if err := lr.Store.CreatePart(p); err != nil {
						slog.Error("create streaming text part", "err", err)
					} else {
						streamTextPart = p
						lastTextFlush = time.Now()
						lr.Bus.Publish("message.part.updated", map[string]string{
							"sessionId": string(sessionID),
							"partId":    string(p.ID),
						})
					}
				} else {
					flushTextPart(false)
				}

			case provider.EventUsage:
				if evt.Usage != nil {
					u := *evt.Usage
					streamUsage = &u
				}

			case provider.EventToolCallStart:
				tc := pendingToolCall{
					CallID: evt.ToolCallID,
					Name:   evt.ToolName,
					Input:  evt.ToolInput,
				}
				pendingToolCalls = append(pendingToolCalls, tc)

				// Create tool part
					toolInput := evt.ToolInput
					if len(toolInput) == 0 {
						toolInput = json.RawMessage("{}")
					}
					toolData, _ := json.Marshal(session.ToolPartData{
						Tool:   evt.ToolName,
						CallID: evt.ToolCallID,
						State: session.ToolState{
							Status: session.ToolPending,
							Input:  toolInput,
						},
					})
				toolPart := &session.Part{
					ID:        session.PartID(id.NewPartID()),
					MessageID: assistantID,
					SessionID: sessionID,
					Type:      session.PartTool,
					Data:      toolData,
					CreatedAt: session.Now(),
					UpdatedAt: session.Now(),
				}
				if err := lr.Store.CreatePart(toolPart); err != nil {
					slog.Error("create tool part", "err", err)
				}
				tc.PartID = toolPart.ID
				pendingToolCalls[len(pendingToolCalls)-1] = tc

				lr.Bus.Publish("message.part.updated", map[string]string{
					"sessionId": string(sessionID),
					"partId":    string(toolPart.ID),
				})

			case provider.EventToolCallDelta:
				// Accumulate tool input deltas
				for i := range pendingToolCalls {
					if pendingToolCalls[i].CallID == evt.ToolCallID {
						pendingToolCalls[i].Input = append(pendingToolCalls[i].Input, evt.ToolInput...)
						break
					}
				}

			case provider.EventToolCallEnd:
				// Finalize tool input
				for i := range pendingToolCalls {
					if pendingToolCalls[i].CallID == evt.ToolCallID {
						pendingToolCalls[i].Ready = true
						break
					}
				}

			case provider.EventReasoning:
				currentReasoning.WriteString(evt.Text)
				if streamReasoningPart == nil {
					text := currentReasoning.String()
					const maxReasoningLen = 50_000
					if len(text) > maxReasoningLen {
						text = text[:maxReasoningLen] + "\n... (truncated)"
					}
					reasonData, _ := json.Marshal(session.ReasoningPartData{Text: text})
					p := &session.Part{
						ID:        session.PartID(id.NewPartID()),
						MessageID: assistantID,
						SessionID: sessionID,
						Type:      session.PartReasoning,
						Data:      reasonData,
						CreatedAt: session.Now(),
						UpdatedAt: session.Now(),
					}
					if err := lr.Store.CreatePart(p); err != nil {
						slog.Error("create streaming reasoning part", "err", err)
					} else {
						streamReasoningPart = p
						lastReasoningFlush = time.Now()
						lr.Bus.Publish("message.part.updated", map[string]string{
							"sessionId": string(sessionID),
							"partId":    string(p.ID),
						})
					}
				} else {
					flushReasoningPart(false)
				}

			case provider.EventFinish:
				if evt.FinishReason != nil {
					finishReason = *evt.FinishReason
				}

			case provider.EventError:
				errStr := evt.Error
				assistantMsg.Error = &errStr
				finish := "error"
				assistantMsg.Finish = &finish
				lr.Store.UpdateMessage(assistantMsg)
				lr.Bus.Publish("message.updated", assistantMsg)
				return fmt.Errorf("stream error: %s", evt.Error)
			}
		}

		// Finalize text part: flush any remaining buffered text to DB.
		// If no streaming part was created yet (e.g. model returned text without deltas),
		// create the part now.
		if currentText.Len() > 0 {
			if streamTextPart != nil {
				flushTextPart(true)
			} else {
				textData, _ := json.Marshal(session.TextPartData{Text: currentText.String()})
				textPart := &session.Part{
					ID:        session.PartID(id.NewPartID()),
					MessageID: assistantID,
					SessionID: sessionID,
					Type:      session.PartText,
					Data:      textData,
					CreatedAt: session.Now(),
					UpdatedAt: session.Now(),
				}
				if err := lr.Store.CreatePart(textPart); err != nil {
					slog.Error("create text part", "err", err)
				}
			}
		}

		// Finalize reasoning part: flush any remaining buffered reasoning to DB.
		if currentReasoning.Len() > 0 {
			if streamReasoningPart != nil {
				flushReasoningPart(true)
			} else {
				text := currentReasoning.String()
				const maxReasoningLen = 50_000
				if len(text) > maxReasoningLen {
					text = text[:maxReasoningLen] + "\n... (truncated)"
				}
				reasonData, _ := json.Marshal(session.ReasoningPartData{Text: text})
				reasonPart := &session.Part{
					ID:        session.PartID(id.NewPartID()),
					MessageID: assistantID,
					SessionID: sessionID,
					Type:      session.PartReasoning,
					Data:      reasonData,
					CreatedAt: session.Now(),
					UpdatedAt: session.Now(),
				}
				if err := lr.Store.CreatePart(reasonPart); err != nil {
					slog.Error("create reasoning part", "err", err)
				}
			}
		}

		// Mark all pending tool calls as ready when the stream is done.
		// Some providers send finish_reason "stop" alongside tool calls, so we
		// must mark them ready regardless of the reason — otherwise they stay in
		// "pending" forever and the frontend thinks tools are still running.
		if len(pendingToolCalls) > 0 {
			for i := range pendingToolCalls {
				pendingToolCalls[i].Ready = true
			}
		}

			// Detect stream interruption: if the channel closed without a finish event
		// and without any error event, the stream was likely interrupted (network,
		// timeout, etc.). Do NOT default to "stop" — that silently kills long loops.
		if finishReason == "" {
			if currentText.Len() > 0 || len(pendingToolCalls) > 0 {
				// We received content but no finish signal — stream was interrupted
				slog.Warn("stream ended without finish_reason, treating as error", "session", sessionID, "textLen", currentText.Len(), "toolCalls", len(pendingToolCalls))
				finishReason = "error"
				errStr := "stream interrupted: LLM connection closed without finish signal"
				assistantMsg.Error = &errStr
			} else {
				// No content and no finish — likely a connection failure
				slog.Warn("stream ended without content or finish_reason", "session", sessionID)
				finishReason = "error"
				errStr := "stream interrupted: no content received"
				assistantMsg.Error = &errStr
			}
		}
		assistantMsg.Finish = &finishReason
		if streamUsage != nil {
			tc := session.TokenCounts{
				Input:      streamUsage.InputTokens,
				Output:     streamUsage.OutputTokens,
				Reasoning:  streamUsage.ReasoningTokens,
				CacheRead:  streamUsage.CacheReadTokens,
				CacheWrite: streamUsage.CacheWriteTokens,
				Total:      streamUsage.InputTokens + streamUsage.OutputTokens,
			}
			assistantMsg.Tokens = &tc
		}
		if err := lr.Store.UpdateMessage(assistantMsg); err != nil {
			slog.Error("update message finish", "err", err)
		}
		lr.Bus.Publish("message.updated", assistantMsg)

		// Execute ready tool calls
		toolCallsExecuted := false
		for _, tc := range pendingToolCalls {
			if !tc.Ready {
				continue
			}
			toolCallsExecuted = true

			// Check for context cancellation before each tool execution
			if ctx.Err() != nil {
				slog.Info("agent loop cancelled before tool execution", "session", sessionID, "tool", tc.Name)
				// Mark any remaining pending/running tool parts as cancelled
				break
			}


			// Mark tool part as running
			part, _ := lr.Store.GetPart(tc.PartID)
			if part != nil {
				var toolData session.ToolPartData
				json.Unmarshal(part.Data, &toolData)
				toolData.State = session.ToolState{
					Status: session.ToolRunning,
					Input:  toolData.State.Input,
					Title:  &tc.Name,
					Time: session.ToolTime{
						Start: session.Now(),
					},
				}
				updatedData, _ := json.Marshal(toolData)
				part.Data = updatedData
				part.UpdatedAt = session.Now()
				lr.Store.UpdatePart(part)
				lr.Bus.Publish("message.part.updated", map[string]string{
					"sessionId": string(sessionID),
					"partId":    string(part.ID),
				})
			}
			result, err := lr.executeTool(ctx, sessionID, assistantID, tc, agent)

			// Update tool part with result
			part, perr := lr.Store.GetPart(tc.PartID)
			if perr != nil {
				slog.Error("get tool part", "err", perr)
				continue
			}

			var toolData session.ToolPartData
			if err := json.Unmarshal(part.Data, &toolData); err != nil {
				slog.Error("unmarshal tool data", "err", err)
				continue
			}

			if err != nil {
				errStr := err.Error()
				toolData.State = session.ToolState{
					Status: session.ToolError,
					Input:  tc.Input,
					Error:  &errStr,
					Title:  &tc.Name,
					Time:   toolData.State.Time,
				}
			} else {
				toolData.State = session.ToolState{
					Status:   session.ToolCompleted,
					Input:    tc.Input,
					Output:   &result.Output,
					Title:    &result.Title,
					Metadata: mustMarshal(result.Metadata),
					Time: session.ToolTime{
						Start: toolData.State.Time.Start,
						End:   session.Now(),
					},
				}
			}

			updatedData, _ := json.Marshal(toolData)
			part.Data = updatedData
			part.UpdatedAt = session.Now()
			if err := lr.Store.UpdatePart(part); err != nil {
				slog.Error("update tool part", "err", err)
			}

			lr.Bus.Publish("message.part.updated", map[string]string{
				"sessionId": string(sessionID),
				"partId":    string(part.ID),
			})
		}

		// If tool calls were executed, always continue the loop regardless of
		// finish_reason — some providers send "stop" even alongside tool calls.
		// Only break when there are no tool calls to feed back.
		if !toolCallsExecuted {
			slog.Info("agent loop complete", "session", sessionID, "steps", step, "reason", finishReason)
			exitReason = finishReason
			if lr.Memory != nil && lr.Memory.Enabled() {
				lr.writeMemory(ctx, sessionID)
			}
			return nil
		}

		// Create a user message with tool results so the next LLM iteration sees them.
		toolResultID := session.MessageID(id.NewMessageID())
		toolResultMsg := &session.MessageInfo{
			ID:        toolResultID,
			SessionID: sessionID,
			Role:      session.RoleUser,
			ParentID:  &assistantID,
			CreatedAt: session.Now(),
		}
		if err := lr.Store.CreateMessage(toolResultMsg); err != nil {
			slog.Error("create tool result message", "err", err)
		} else {
			for _, tc := range pendingToolCalls {
				if !tc.Ready {
					continue
				}
				part, perr := lr.Store.GetPart(tc.PartID)
				if perr != nil {
					continue
				}
				var toolData session.ToolPartData
				if err := json.Unmarshal(part.Data, &toolData); err != nil {
					continue
				}
				resultData, _ := json.Marshal(session.ToolPartData{
					Tool:   toolData.Tool,
					CallID: toolData.CallID,
					State:  toolData.State,
				})
				resultPart := &session.Part{
					ID:        session.PartID(id.NewPartID()),
					MessageID: toolResultID,
					SessionID: sessionID,
					Type:      session.PartTool,
					Data:      resultData,
					CreatedAt: session.Now(),
					UpdatedAt: session.Now(),
				}
				if err := lr.Store.CreatePart(resultPart); err != nil {
					slog.Error("create tool result part", "err", err)
				}
			}
			lr.Bus.Publish("message.updated", toolResultMsg)
		}
	}

	// Reached MaxSteps with tool calls still pending — treat as stop.
	exitReason = "stop"
	if lr.Memory != nil && lr.Memory.Enabled() {
		lr.writeMemory(ctx, sessionID)
	}
	return nil
}

// writeMemory extracts the last conversation turn and persists it via memory_add.
func (lr *LoopRunner) writeMemory(ctx context.Context, sessionID session.SessionID) {
	messages, err := lr.Store.GetMessages(sessionID, "", 1000)
	if err != nil {
		slog.Warn("writeMemory: failed to load messages", "err", err)
		return
	}

	// Find the last user message that has a text part (skip tool-result user messages)
	var userText string
	var userMsgIdx int
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role != session.RoleUser || len(messages[i].Parts) == 0 {
			continue
		}
		hasText := false
		for _, p := range messages[i].Parts {
			if p.Type == session.PartText {
				var data session.TextPartData
				if json.Unmarshal(p.Data, &data) == nil && data.Text != "" {
					userText = data.Text
					hasText = true
				}
			}
		}
		if hasText {
			userMsgIdx = i
			break
		}
	}
	if userText == "" {
		slog.Info("writeMemory: no user text found, skipping")
		return
	}

	// Build response trace from assistant messages after the user message
	responseText := buildTurnResponse(messages, userMsgIdx)
	if responseText == "" {
		slog.Info("writeMemory: no response text, skipping")
		return
	}

	slog.Info("writeMemory: persisting turn", "session", sessionID, "questionLen", len(userText), "responseLen", len(responseText))
	lr.Memory.WriteMemory(ctx, string(sessionID), userText, responseText)
}

// buildTurnResponse serializes all assistant messages after a given user message
// into a structured text trace (tool calls, results, reasoning, text).
func buildTurnResponse(messages []*session.MessageWithParts, userMsgIdx int) string {
	var b strings.Builder
	for i := userMsgIdx + 1; i < len(messages); i++ {
		m := messages[i]
		if m.Info.Role == session.RoleAssistant {
			fmt.Fprintf(&b, "--- Assistant iteration ---\n")
			for _, p := range m.Parts {
				switch p.Type {
				case session.PartText:
					var data session.TextPartData
					if json.Unmarshal(p.Data, &data) == nil && data.Text != "" {
						fmt.Fprintf(&b, "Text: %s\n", data.Text)
					}
				case session.PartTool:
					var data session.ToolPartData
					if json.Unmarshal(p.Data, &data) == nil {
						status := string(data.State.Status)
						fmt.Fprintf(&b, "Tool: %s (%s)\n", data.Tool, status)
						if data.State.Input != nil {
							fmt.Fprintf(&b, "  Input: %s\n", string(data.State.Input))
						}
						if data.State.Output != nil {
							output := *data.State.Output
							if len(output) > 500 {
								output = output[:500] + "..."
							}
							fmt.Fprintf(&b, "  Output: %s\n", output)
						}
						if data.State.Error != nil {
							fmt.Fprintf(&b, "  Error: %s\n", *data.State.Error)
						}
					}
				case session.PartReasoning:
					var data session.ReasoningPartData
					if json.Unmarshal(p.Data, &data) == nil && data.Text != "" {
						fmt.Fprintf(&b, "Reasoning: %s\n", data.Text)
					}
				}
			}
		}
	}
	return b.String()
}

func (lr *LoopRunner) executeTool(ctx context.Context, sessionID session.SessionID, messageID session.MessageID, tc pendingToolCall, a Agent) (tool.Result, error) {
	// Try built-in tools first
	t := lr.Tools.Get(tc.Name)
	if t != nil {
		slog.Info("executing built-in tool", "tool", tc.Name)
		tctx := tool.Context{
			SessionID:  sessionID,
			MessageID:  messageID,
			Agent:      a.ID,
			CallID:     tc.CallID,
			Ctx:        ctx,
			SessionDir: lr.Dir,
		}
		return t.Execute(ctx, tc.Input, tctx)
	}

	// Try MCP tools
	if lr.MCP != nil {
		for name, td := range lr.MCP.Tools() {
			if name == tc.Name {
				slog.Info("executing MCP tool", "tool", name)
				var args map[string]any
				if err := json.Unmarshal(tc.Input, &args); err != nil {
					return tool.Result{}, fmt.Errorf("parse MCP tool input: %w", err)
				}
				output, duration, err := lr.MCP.CallTool(ctx, name, args)
				title := td.Name
				metadata := map[string]any{
					"duration_ms": duration.Milliseconds(),
				}
				if err != nil {
					return tool.Result{Title: title, Output: err.Error(), Metadata: metadata}, err
				}
				return tool.Result{Title: title, Output: output, Metadata: metadata}, nil
			}
		}
	}

	slog.Warn("unknown tool requested", "tool", tc.Name)
	return tool.Result{}, fmt.Errorf("unknown tool: %s", tc.Name)
}

type pendingToolCall struct {
	CallID string
	Name   string
	Input  json.RawMessage
	Ready  bool
	PartID session.PartID
}

func shouldBreak(messages []*session.MessageWithParts) bool {
	if len(messages) == 0 {
		return false
	}
	last := messages[len(messages)-1]
	if last.Info.Role != session.RoleAssistant {
		return false
	}
	if last.Info.Finish != nil {
		f := *last.Info.Finish
		return f == "stop" || f == "error" || f == "aborted"
	}
	return false
}

func lastUserMessageID(messages []*session.MessageWithParts) *session.MessageID {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == session.RoleUser {
			id := messages[i].Info.ID
			return &id
		}
	}
	return nil
}

func toProviderMessages(messages []*session.MessageWithParts, memoryText string) []provider.ModelMessage {
	// When memory is active, filter to only the last user message and everything after.
	// This replaces full history with the compressed <prior_context> block.
	if memoryText != "" {
		// Find the last user message index
		lastUserIdx := -1
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Info.Role == session.RoleUser {
				// Skip tool-result user messages (they have tool parts)
				hasText := false
				for _, p := range messages[i].Parts {
					if p.Type == session.PartText {
						hasText = true
					}
				}
				if hasText {
					lastUserIdx = i
					break
				}
			}
		}

		if lastUserIdx >= 0 {
			// Include everything from the last text-user message onwards
			// plus any preceding tool-result messages (for ongoing tool chains)
			filtered := messages[lastUserIdx:]

			// Prepend <prior_context> to the first user message
			result := convertMessages(filtered)

			// Find the first user message and prepend context
			for i, msg := range result {
				if msg.Role == "user" {
					var content string
					if msg.Content != nil {
						json.Unmarshal(msg.Content, &content)
					}
					content = "<prior_context>\n" + memoryText + "\n</prior_context>\n\n" + content
					result[i].Content, _ = json.Marshal(content)
					break
				}
			}

			return result
		}
	}

	// No memory: truncate to last N messages to stay within context limits.
	// Keep the original user ask and the most recent conversation tail.
	const maxContextMessages = 50
	if len(messages) > maxContextMessages {
		var originalAsk string
		for _, m := range messages {
			if m.Info.Role == session.RoleUser {
				for _, p := range m.Parts {
					if p.Type == session.PartText {
						var data session.TextPartData
						if json.Unmarshal(p.Data, &data) == nil && data.Text != "" {
							originalAsk = data.Text
							break
						}
					}
				}
				if originalAsk != "" {
					break
				}
			}
		}
		recent := messages[len(messages)-maxContextMessages:]
		result := convertMessages(recent)
		// Prepend the original ask so the LLM doesn't lose context
		if originalAsk != "" && len(result) > 0 {
			prefix := "[Earlier conversation truncated. Original request: " + originalAsk + "]\n\n"
			var firstContent string
			if result[0].Content != nil {
				json.Unmarshal(result[0].Content, &firstContent)
			}
			if firstContent != "" {
				result[0].Content, _ = json.Marshal(prefix + firstContent)
			}
		}
		slog.Info("truncated message context", "total", len(messages), "sent", maxContextMessages)
		return result
	}
	return convertMessages(messages)
}

func convertMessages(messages []*session.MessageWithParts) []provider.ModelMessage {
	var result []provider.ModelMessage
	for _, m := range messages {
		// Collect text and tool parts
		var textParts []string
		var toolCallParts []session.ToolPartData
		var toolResultParts []session.ToolPartData

		for _, p := range m.Parts {
			switch p.Type {
			case session.PartText:
				var data session.TextPartData
				json.Unmarshal(p.Data, &data)
				if data.Text != "" {
					textParts = append(textParts, data.Text)
				}
			case session.PartTool:
				var data session.ToolPartData
				json.Unmarshal(p.Data, &data)
				if m.Info.Role == session.RoleAssistant {
					toolCallParts = append(toolCallParts, data)
				} else {
					toolResultParts = append(toolResultParts, data)
				}
			}
		}

		if m.Info.Role == session.RoleAssistant && len(toolCallParts) > 0 {
			// Assistant message with tool calls: emit as a single message with tool_calls array
			type oaiToolCall struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}

			var calls []oaiToolCall
			for _, tc := range toolCallParts {
				call := oaiToolCall{
					ID:   tc.CallID,
					Type: "function",
				}
				call.Function.Name = tc.Tool
				// Arguments must be a JSON string (the OpenAI API expects a string, not a raw object)
				if tc.State.Input != nil {
					call.Function.Arguments = string(tc.State.Input)
				} else {
					call.Function.Arguments = "{}"
				}
				calls = append(calls, call)
			}
			toolCallsJSON, _ := json.Marshal(calls)

			msg := provider.ModelMessage{
				Role:      "assistant",
				ToolCalls: toolCallsJSON,
			}
			if len(textParts) > 0 {
				msg.Content, _ = json.Marshal(strings.Join(textParts, ""))
			}
			result = append(result, msg)
		} else if m.Info.Role == session.RoleUser && len(toolResultParts) > 0 {
			// User message with tool results: emit each as a separate role=tool message
			for _, tr := range toolResultParts {
				output := ""
				if tr.State.Output != nil {
					output = *tr.State.Output
				} else if tr.State.Error != nil {
					output = "Error: " + *tr.State.Error
				}
				content, _ := json.Marshal(output)
				result = append(result, provider.ModelMessage{
					Role:       "tool",
					Content:    content,
					ToolCallID: tr.CallID,
					Name:       tr.Tool,
				})
			}
		} else {
			// Plain text message
			if len(textParts) > 0 {
				content, _ := json.Marshal(strings.Join(textParts, ""))
				result = append(result, provider.ModelMessage{
					Role:    string(m.Info.Role),
					Content: content,
				})
			}
		}
	}
	return result
}

func buildSystemPrompt(a Agent, dir string, memoryEnabled bool, agentMDContent string, mcpTools map[string]mcp.ToolDef) string {
	now := time.Now().Format("Mon Jan 2 15:04:05 MST 2006")
	prompt := fmt.Sprintf(`%s

Working directory: %s
Platform: %s/%s
Current date: %s`, a.System, dir, runtime.GOOS, runtime.GOARCH, now)

	if agentMDContent != "" {
		prompt += agentMDContent
	}

	// Inject MCP skill descriptions (excluding memory tools)
	if len(mcpTools) > 0 {
		prompt += "\n\n## Available Skills\n\n"
		prompt += "You have access to specialized tools via MCP. Use them proactively when relevant:\n\n"

		skillSections := []struct {
			heading string
			tools   []skillDesc
		}{
			{
				heading: "### Codebase Research",
				tools: []skillDesc{
					{name: "answer_codebase", tip: "PREFERRED for grounded, synthesized answers about committed code with file paths and line citations."},
					{name: "query_codebase", tip: "For targeted search across the committed index (semantic, pattern, or hybrid)."},
					{name: "cache_map", tip: "Use first to understand what a source contains before deeper queries."},
				},
			},
			{
				heading: "### Live / Uncommitted Changes",
				tools: []skillDesc{
					{name: "hot_answer", tip: "PREFERRED for Q&A about uncommitted working-tree edits. Use when user asks about 'my changes' or recent local modifications."},
					{name: "hot_query", tip: "For real-time search across uncommitted changes."},
				},
			},
			{
				heading: "### Multi-Repo (Tracks)",
				tools: []skillDesc{
					{name: "track_answer", tip: "For synthesized answers across multiple repos in a Track."},
					{name: "track_query", tip: "For targeted search across all sources in a Track."},
					{name: "hot_track_answer", tip: "For Q&A across live uncommitted changes in a Track."},
					{name: "hot_track_query", tip: "For real-time search across live changes in a Track."},
				},
			},
		}

		for _, section := range skillSections {
			hasTool := false
			for _, s := range section.tools {
				if _, ok := mcpTools[s.name]; ok {
					hasTool = true
					break
				}
			}
			if !hasTool {
				continue
			}
			prompt += section.heading + "\n"
			for _, s := range section.tools {
				if _, ok := mcpTools[s.name]; ok {
					prompt += fmt.Sprintf("- **%s**: %s\n", s.name, s.tip)
				}
			}
			prompt += "\n"
		}
	}

	if memoryEnabled {
		prompt += `

You have access to agentic memory. Prior conversation context is provided in <prior_context> blocks. This is a lossy summary of everything discussed so far — treat it as authoritative for past events. It does NOT contain verbatim code or exact search results.

If you need to probe deeper into past context, use the memory_recall tool with a specific question. Do not rely on <prior_context> for exact details — use it for orientation, then recall for specifics.`
	}

	return prompt
}

type skillDesc struct {
	name string
	tip  string
}

func mustMarshal(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, _ := json.Marshal(v)
	return data
}

// isTransientError returns true for errors that are worth retrying
// (rate limits, timeouts, connection resets, server errors).
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	// Rate limiting
	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "429") {
		return true
	}
	// Server errors
	if strings.Contains(lower, "500") || strings.Contains(lower, "502") || strings.Contains(lower, "503") || strings.Contains(lower, "504") {
		return true
	}
	// Connection-level issues
	if strings.Contains(lower, "connection reset") || strings.Contains(lower, "eof") ||
		strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "refused") || strings.Contains(lower, "temporary") {
		return true
	}
	// Anthropic overloaded
	if strings.Contains(lower, "overloaded") {
		return true
	}
	return false
}

// ensureStartsWithUser trims leading messages until the slice starts with a "user"
// role message, which is required for valid LLM conversation structure.
func ensureStartsWithUser(messages []provider.ModelMessage) []provider.ModelMessage {
	for len(messages) > 0 && messages[0].Role != "user" {
		messages = messages[1:]
	}
	return messages
}

// trimToRecent trims a slice of provider messages to the most recent n entries,
// ensuring the first kept message has role "user" so the conversation is valid.
func trimToRecent(messages []provider.ModelMessage, n int) []provider.ModelMessage {
	if len(messages) <= n {
		return messages
	}
	return ensureStartsWithUser(messages[len(messages)-n:])
}

// isContextLengthError returns true when the provider rejects the request because
// the prompt exceeds the model's maximum context window.
func isContextLengthError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "too long") ||
		strings.Contains(lower, "context length") ||
		strings.Contains(lower, "maximum context") ||
		strings.Contains(lower, "context_length_exceeded") ||
		strings.Contains(lower, "prompt is too long")
}

// llmCompact summarizes the older portion of the conversation using the LLM itself,
// producing a rich technical summary that replaces the trimmed middle section.
// It returns a system-prompt addendum containing the summary and the recent messages
// to keep verbatim. Falls back to mechanical truncation if the LLM call fails.
func (lr *LoopRunner) llmCompact(ctx context.Context, p provider.Provider, modelID string, messages []provider.ModelMessage) (systemAddendum string, compacted []provider.ModelMessage) {
	const keepRecent = 12
	if len(messages) <= keepRecent {
		return "", messages
	}

	oldMessages := messages[:len(messages)-keepRecent]
	recent := messages[len(messages)-keepRecent:]

	// Ensure recent starts with a user message (required for valid conversation structure)
	recent = ensureStartsWithUser(recent)
	if len(recent) == 0 {
		return "", compactMessagesTruncate(messages)
	}

	// Render old messages into readable text for the summarizer
	var history strings.Builder
	for _, m := range oldMessages {
		var content string
		if m.Content != nil {
			json.Unmarshal(m.Content, &content)
		}
		switch {
		case m.Role == "tool":
			if content != "" {
				out := content
				if len(out) > 800 {
					out = out[:800] + "...(truncated)"
				}
				fmt.Fprintf(&history, "[tool result %s]: %s\n\n", m.Name, out)
			}
		case m.ToolCalls != nil:
			fmt.Fprintf(&history, "[assistant tool calls]: %s\n\n", string(m.ToolCalls))
		case content != "":
			fmt.Fprintf(&history, "[%s]: %s\n\n", m.Role, content)
		}
	}

	if history.Len() == 0 {
		return "", recent
	}

	historyText := history.String()
	if len(historyText) > 30_000 {
		historyText = historyText[:30_000] + "\n...(older history truncated)"
	}

	summarizerPrompt := "Summarize the following conversation history concisely. Capture: the original task/goal, " +
		"key decisions made, files modified (with paths), code written or changed, errors encountered " +
		"and how they were resolved, and the current state of work. Be specific about technical details — " +
		"file paths, function names, variable names, commands run. This summary will replace the full " +
		"history so the conversation can continue within the model's context window." +
		"\n\n" + historyText

	// json.Marshal properly encodes the combined string, including any special
	// characters in historyText, producing a valid JSON string value for Content.
	summarizerContent, _ := json.Marshal(summarizerPrompt)

	summaryReq := provider.StreamRequest{
		Model: modelID,
		System: []string{
			"You are a precise technical conversation summarizer. Produce a concise but complete summary " +
				"that preserves all technical details needed to continue the work effectively.",
		},
		Messages: []provider.ModelMessage{
			{Role: "user", Content: summarizerContent},
		},
		Abort: ctx,
	}

	ch, err := p.StreamChat(ctx, summaryReq)
	if err != nil {
		slog.Warn("llm compact: summary call failed, falling back to truncation", "err", err)
		return "", compactMessagesTruncate(messages)
	}

	var summary strings.Builder
	for evt := range ch {
		if evt.Type == provider.EventTextDelta {
			summary.WriteString(evt.Text)
		}
	}

	if summary.Len() == 0 {
		slog.Warn("llm compact: empty summary received, falling back to truncation")
		return "", compactMessagesTruncate(messages)
	}

	addendum := "\n\n## Compacted Conversation History (AI-generated summary)\n\n" +
		"Earlier conversation exchanges were trimmed to fit the context window. " +
		"The following summary is authoritative — treat it as what has already been done:\n\n" +
		summary.String()

	return addendum, recent
}

// compactMessagesTruncate is the mechanical fallback: keeps the original first
// message and the most recent exchanges, discarding the middle verbatim.
func compactMessagesTruncate(messages []provider.ModelMessage) []provider.ModelMessage {
	const keepRecent = 15
	if len(messages) <= keepRecent+1 {
		return messages
	}

	original := messages[0]
	recent := messages[len(messages)-keepRecent:]

	recent = ensureStartsWithUser(recent)
	if len(recent) == 0 {
		return messages
	}

	note := "[Context auto-compacted: earlier conversation omitted. Original request preserved above.] "
	var existingContent string
	if recent[0].Content != nil {
		json.Unmarshal(recent[0].Content, &existingContent)
	}
	annotated := recent[0]
	annotated.Content, _ = json.Marshal(note + existingContent)

	result := make([]provider.ModelMessage, 0, len(recent)+1)
	result = append(result, original)
	result = append(result, annotated)
	result = append(result, recent[1:]...)
	return result
}