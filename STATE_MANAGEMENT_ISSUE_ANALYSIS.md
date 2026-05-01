# State Management Issue Analysis - First Message Not Visible

## Problem Summary
When a user sends their first message in a new chat session, the response is not visible immediately. However, after navigating to another session and coming back, the message appears. This suggests a state synchronization issue.

## Root Cause Analysis

### Issue Location: `web/src/pages/home.tsx` (Line 42-55)

The problem is in the `startSession` function:

```typescript
const startSession = async (content: string) => {
  if (submitting()) return;
  const trimmed = content.trim();
  if (!trimmed) return;
  setSubmitting(true);
  try {
    const sess = await session.newSession();                    // ← Creates new session
    await sendPrompt(sess.id, trimmed, session.selectedModel()); // ← Sends message
    navigate(`/session/${sess.id}`);                            // ← Navigates to chat page
  } catch (e) {
    console.error('start session failed:', e);
    setSubmitting(false);
  }
};
```

### The Race Condition

1. **Navigation happens BEFORE message fetch**: The `navigate()` call is made immediately after `sendPrompt()` completes, but before the chat page has a chance to fetch and display messages.

2. **State not initialized**: When the chat page (`session.tsx`) loads, it calls:
   ```typescript
   createEffect(on(() => params.id, (id) => {
     if (id) {
       session.selectSession(id);  // This loads messages from the server
     }
   }));
   ```

3. **Timing Issue**: The component might mount and the effect might run, but if the initial message hasn't been persisted to the server yet, or if the polling hasn't started, the message won't appear until:
   - The polling interval ticks (every 1500ms)
   - The user navigates away and back (which triggers `selectSession` again)

### Secondary Issue: Initial Message Not in Cache

In `session.tsx` line 220:
```typescript
const haveCache = sameSession && messagesRaw().length > 0;
```

When entering a NEW session (not the same session), `messagesRaw()` is empty because:
- The session was just created
- No messages have been fetched yet
- The optimistic message from home.tsx is not carried over

## Why It Works After Navigation

1. User navigates away from the session
2. `selectSession` is called with a different ID
3. User comes back to the original session
4. `selectSession` is called again with `haveCache = false` (because previous session was cleared)
5. This triggers a fresh `getMessages()` call
6. Messages are now visible because polling has completed and the message is persisted

## Solution

### Fix 1: Ensure Message Fetch Before Displaying Chat (RECOMMENDED)

Modify `home.tsx` to fetch messages before navigating:

```typescript
const startSession = async (content: string) => {
  if (submitting()) return;
  const trimmed = content.trim();
  if (!trimmed) return;
  setSubmitting(true);
  try {
    const sess = await session.newSession();
    await sendPrompt(sess.id, trimmed, session.selectedModel());
    
    // ← ADD: Wait for the initial message to be persisted and fetched
    const msgs = await getMessages(sess.id);
    // Manually update the session context with initial messages
    // This ensures the chat page will have cached messages
    
    navigate(`/session/${sess.id}`);
  } catch (e) {
    console.error('start session failed:', e);
    setSubmitting(false);
  }
};
```

### Fix 2: Initialize Polling Immediately on Chat Page Load

Modify `session.tsx` to start polling immediately when a session is selected:

```typescript
async function selectSession(id: string) {
  const current = activeSession();
  const sameSession = current?.id === id;
  const haveCache = sameSession && messagesRaw().length > 0;

  let session = sessions().find((s) => s.id === id);
  if (!session) {
    session = current?.id === id
      ? current
      : { id, projectId: '', directory: server.directory(), title: 'Loading...', createdAt: Date.now(), updatedAt: Date.now() };
  }
  setActiveSession(session);

  if (!haveCache) {
    setMessages([]);
  }
  try {
    const msgs = await getMessages(id);
    setMessages(msgs);
    
    // ← ADD: Check if we should start polling (last message is incomplete)
    const last = msgs[msgs.length - 1];
    if (last && last.info.role === 'assistant' && !last.info.finish && !last.info.error) {
      startPolling(id);
    }
  } catch (e) {
    console.error('load messages failed:', e);
  }
}
```

### Fix 3: Invalidate Cache on New Session (COMPLEMENTARY)

Modify the cache logic in `session.tsx`:

```typescript
async function selectSession(id: string) {
  const current = activeSession();
  const sameSession = current?.id === id;
  // Always clear cache for newly created sessions (they won't have messages yet)
  const isNewSession = !sessions().find((s) => s.id === id);
  const haveCache = sameSession && !isNewSession && messagesRaw().length > 0;
  // ... rest of the function
}
```

## Recommended Implementation

Implement **Fix 1 + Fix 2** together:

1. **In home.tsx**: Fetch messages before navigating to ensure the session context is populated
2. **In session.tsx**: Auto-start polling if the last message is incomplete

This ensures:
- Messages are visible immediately when the chat page loads
- Polling is automatically initiated if needed
- No orphaned "loading" states
- Better UX with instant feedback

## Files to Modify

1. `/web/src/pages/home.tsx` - Add message fetch before navigation
2. `/web/src/context/session.tsx` - Add polling auto-start logic
