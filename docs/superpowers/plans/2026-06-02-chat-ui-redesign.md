# Chat UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the chat interface in `ChatPage.tsx` to feel as smooth and polished as Claude or ChatGPT — clean message bubbles, a modern floating compose bar with a non-intrusive model picker, a conversational empty state, and no unnecessary chrome.

**Architecture:** All changes are confined to `apps/web/src/pages/chat/ChatPage.tsx` and one new helper component (`ChatWelcome`). No backend changes. No new dependencies — only MUI components already in the project. Each task is independently shippable.

**Tech Stack:** React 19, MUI v7, react-hook-form, @tanstack/react-query, TypeScript

---

## What's wrong right now (reference for every task)

| Problem | Location in code |
|---|---|
| Big `h1` title + "Connection X" subtitle above messages | `MessagePanel` lines 182–187 |
| Empty state is a bordered `Paper` card with inbox icon | `MessagePanel` line 235–239, `EmptyState` component |
| Both user AND assistant bubbles use `Paper variant="outlined"` — hard borders everywhere | `MessageItem` line 273 |
| Model selector is a full `<Select>` with `<InputLabel>` crammed into the compose box | `SendMessageForm` lines 582–604 |
| Model label shows BOTH display name AND full id: "llama3.2:1b (ollama:llama3.2:1b)" | `MenuItem` line 598–599 |
| Compose box uses `variant="outlined"` Paper — visible box border | `SendMessageForm` line 543 |
| Hard `borderTop` line between messages and compose area | `MessagePanel` line 250 |
| No "AI is thinking" indicator while mutation is pending | Nowhere |
| Asymmetric `pr: 1` only (no `pl`) on the messages scroll area | `MessagePanel` line 211 |

---

## Files

| File | Action | Responsibility |
|---|---|---|
| `apps/web/src/pages/chat/ChatPage.tsx` | Modify | All chat UI — panel, messages, compose bar |

No new files needed. All changes stay in the one file.

---

## Task 1: Remove the conversation header and replace the empty state

**Problem:** The `h1` title + "Connection X" is a web-page header, not a chat header. The bordered `EmptyState` card looks like a form placeholder. Goal: when there are no messages, show a clean centered welcome using the conversation title — no box, no icon.

**Files:**
- Modify: `apps/web/src/pages/chat/ChatPage.tsx` (lines 182–240)

- [ ] **Step 1: Delete the conversation header Stack**

In `MessagePanel`, remove these lines entirely:
```tsx
<Stack spacing={0.5} sx={{ pb: 2 }}>
  <Typography variant="h1">{conversation.title}</Typography>
  <Typography color="text.secondary" variant="body2">
    Connection {conversation.connection_id}
  </Typography>
</Stack>
```

- [ ] **Step 2: Replace the "No messages yet" EmptyState with an inline welcome**

Replace this block:
```tsx
{!isLoading && !messagesError && messages.length === 0 ? (
  <EmptyState
    title="No messages yet"
    description="Send the first question for this conversation."
  />
) : null}
```

With a centered conversational welcome — no card, no border, no icon:
```tsx
{!isLoading && !messagesError && messages.length === 0 ? (
  <Box
    sx={{
      flex: 1,
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      textAlign: "center",
      py: 8,
      gap: 1,
    }}
  >
    <Typography variant="h2" fontWeight={700} sx={{ mb: 0.5 }}>
      {conversation.title}
    </Typography>
    <Typography color="text.secondary" variant="body1">
      Ask anything about your data. I'll write the SQL and show you the results.
    </Typography>
  </Box>
) : null}
```

- [ ] **Step 3: Verify it hot-reloads and looks right**

Open http://localhost:5173, click on a conversation with no messages. You should see just the centered text — no card, no icon, no title at the top.

- [ ] **Step 4: Commit**
```bash
git add apps/web/src/pages/chat/ChatPage.tsx
git commit -m "feat(chat): remove page header, replace empty state with inline welcome"
```

---

## Task 2: Redesign message bubbles

**Problem:** Both user and assistant messages use `Paper variant="outlined"` which creates hard box borders. Assistant messages especially should feel like natural conversational text, not a form card.

**Target:**
- **User message** — right-aligned pill bubble, `primary.main` background, generous border-radius, no outline border.
- **Assistant message** — left-aligned, NO box/border, just text with a subtle AI-branded left margin. The info icon for model details moves to a tiny metadata line *below* the message text.

**Files:**
- Modify: `apps/web/src/pages/chat/ChatPage.tsx` — `MessageItem` function (lines 263–320)

- [ ] **Step 1: Rewrite `MessageItem`**

Replace the entire `MessageItem` function with:

```tsx
function MessageItem({ item }: { item: MessageListItem }) {
  const isAssistant = item.message.role === "assistant";
  const hasModelInfo = Boolean(item.message.provider || item.message.model);

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: isAssistant ? "flex-start" : "flex-end",
        gap: 0.5,
        px: 1,
      }}
    >
      {/* Message bubble */}
      <Box
        sx={
          isAssistant
            ? {
                maxWidth: { xs: "100%", md: "82%" },
                color: "text.primary",
              }
            : {
                maxWidth: { xs: "88%", md: "72%" },
                bgcolor: "primary.main",
                color: "primary.contrastText",
                borderRadius: "18px 18px 4px 18px",
                px: 2,
                py: 1.25,
              }
        }
      >
        <Typography
          sx={{
            whiteSpace: "pre-wrap",
            lineHeight: 1.65,
            fontSize: "0.9375rem",
          }}
        >
          {item.message.content}
        </Typography>
        {item.message.error_message ? (
          <Alert severity="error" sx={{ mt: 1 }}>
            {item.message.error_message}
          </Alert>
        ) : null}
        {item.execution ? <ExecutionPanel execution={item.execution} /> : null}
      </Box>

      {/* Metadata line below assistant message */}
      {isAssistant && hasModelInfo ? (
        <Tooltip
          title={[
            item.message.provider ? `Provider: ${item.message.provider}` : null,
            item.message.model ? `Model: ${item.message.model}` : null,
          ]
            .filter(Boolean)
            .join(" · ")}
        >
          <Typography
            variant="caption"
            color="text.disabled"
            sx={{
              px: 1,
              cursor: "default",
              userSelect: "none",
              "&:hover": { color: "text.secondary" },
              transition: "color 0.15s",
            }}
          >
            {item.message.model ?? item.message.provider}
          </Typography>
        </Tooltip>
      ) : null}
    </Box>
  );
}
```

- [ ] **Step 2: Verify both message types render correctly**

Send a test message. Check:
- User message: right-aligned, blue pill shape, no hard border
- Assistant message: left-aligned, plain text, subtle model caption underneath
- Error messages still show the Alert

- [ ] **Step 3: Commit**
```bash
git add apps/web/src/pages/chat/ChatPage.tsx
git commit -m "feat(chat): redesign message bubbles — pill user, borderless assistant"
```

---

## Task 3: Add a typing / pending indicator

**Problem:** When the AI is generating a response, the UI is completely silent — no feedback at all. Both Claude and ChatGPT show a pulsing "…" or animated indicator.

**Files:**
- Modify: `apps/web/src/pages/chat/ChatPage.tsx` — `MessagePanel` and `SendMessageForm`

**Approach:** Pass `isPending` down from `SendMessageForm` to `MessagePanel` via a lifted state, OR simply add the indicator directly inside `SendMessageForm`'s render (it already knows `mutation.isPending`). The simpler approach: render a "thinking" row *inside the messages scroll area* only when pending. We'll lift the pending state.

- [ ] **Step 1: Lift `isPending` from `SendMessageForm` to `MessagePanel`**

Add `onPendingChange` callback to `SendMessageForm` props and a `isPending` state in `MessagePanel`:

In `MessagePanel`, add:
```tsx
const [isAIResponding, setIsAIResponding] = useState(false);
```

Pass to `SendMessageForm`:
```tsx
<SendMessageForm
  conversationID={conversation.id}
  models={models}
  onPendingChange={setIsAIResponding}
/>
```

In `SendMessageForm` props, add:
```tsx
onPendingChange: (pending: boolean) => void;
```

In `SendMessageForm`, add a `useEffect`:
```tsx
useEffect(() => {
  onPendingChange(mutation.isPending);
}, [mutation.isPending, onPendingChange]);
```

- [ ] **Step 2: Render the typing indicator in the messages scroll area**

After `{messages.map(...)}` and before `<Box ref={messagesEndRef} />`, add:

```tsx
{isAIResponding ? (
  <Box sx={{ display: "flex", alignItems: "flex-start", px: 1 }}>
    <Box
      sx={{
        display: "flex",
        gap: "5px",
        alignItems: "center",
        py: 1.5,
        px: 0.5,
      }}
    >
      {[0, 1, 2].map((i) => (
        <Box
          key={i}
          sx={{
            width: 7,
            height: 7,
            borderRadius: "50%",
            bgcolor: "text.disabled",
            animation: "typingPulse 1.2s ease-in-out infinite",
            animationDelay: `${i * 0.2}s`,
            "@keyframes typingPulse": {
              "0%, 60%, 100%": { opacity: 0.2, transform: "scale(1)" },
              "30%": { opacity: 1, transform: "scale(1.2)" },
            },
          }}
        />
      ))}
    </Box>
  </Box>
) : null}
```

- [ ] **Step 3: Scroll to bottom when indicator appears**

Update the `useEffect` scroll trigger to also fire when `isAIResponding` changes:
```tsx
useEffect(() => {
  if ((lastMessageID || isAIResponding) && typeof messagesEndRef.current?.scrollIntoView === "function") {
    messagesEndRef.current.scrollIntoView({ behavior: "smooth", block: "end" });
  }
}, [lastMessageID, isAIResponding]);
```

- [ ] **Step 4: Verify**

Send a message and watch for the three-dot pulse animation to appear below the last message while the AI responds.

- [ ] **Step 5: Commit**
```bash
git add apps/web/src/pages/chat/ChatPage.tsx
git commit -m "feat(chat): add animated typing indicator while AI is responding"
```

---

## Task 4: Redesign the compose bar

**Problem:** The compose area has a heavy outlined Paper box, a full `<Select>` dropdown with `<InputLabel>` label crammed inside the text area, and shows the full internal model ID in the menu items. It needs to feel like a clean floating input — text area + a minimal model chip + send button.

**Target:** A floating Paper with no visible outline border (shadow only), the model shown as a small unobtrusive `Chip` in the bottom-left (clicking opens a `Menu` instead of a heavy `Select`), and only `model.display_name` shown — no redundant ID.

**Files:**
- Modify: `apps/web/src/pages/chat/ChatPage.tsx` — `SendMessageForm` (lines 475–638)

- [ ] **Step 1: Add `Menu` and `MenuItem` (already imported) — add `useState` for menu anchor**

At the top of `SendMessageForm`, add:
```tsx
const [modelMenuAnchor, setModelMenuAnchor] = useState<HTMLElement | null>(null);
```

- [ ] **Step 2: Rewrite the compose bar JSX**

Replace everything from `return (` to the end of `SendMessageForm` with:

```tsx
  return (
    <Box>
      <Paper
        component="form"
        onSubmit={handleSubmit((values) => mutation.mutate(values))}
        elevation={3}
        sx={{
          borderRadius: 3,
          bgcolor: "background.paper",
          overflow: "hidden",
          border: "1px solid",
          borderColor: "divider",
        }}
      >
        {/* Text input */}
        <TextField
          multiline
          maxRows={8}
          minRows={1}
          error={Boolean(errors.content)}
          placeholder="Message Datalk…"
          fullWidth
          variant="standard"
          slotProps={{
            htmlInput: { "aria-label": "Message" },
            input: {
              disableUnderline: true,
              sx: { px: 2, pt: 1.5, pb: 0.5, fontSize: "0.9375rem", lineHeight: 1.6 },
            },
          }}
          {...contentField}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !mutation.isPending) {
              event.preventDefault();
              void handleSubmit((values) => mutation.mutate(values))();
            }
          }}
        />

        {/* Bottom toolbar */}
        <Stack
          direction="row"
          alignItems="center"
          sx={{ px: 1.5, pb: 1.25, pt: 0.5 }}
        >
          {/* Model chip */}
          <Controller
            control={control}
            name="model"
            rules={{ required: "Model is required" }}
            render={({ field }) => {
              const selected = selectedModelByID.get(field.value);
              return (
                <>
                  <Chip
                    label={selected?.display_name ?? "Select model"}
                    size="small"
                    variant="outlined"
                    onClick={(e) => setModelMenuAnchor(e.currentTarget)}
                    disabled={models.length === 0}
                    sx={{
                      borderRadius: 999,
                      fontSize: "0.75rem",
                      cursor: "pointer",
                      maxWidth: 200,
                    }}
                  />
                  <Menu
                    anchorEl={modelMenuAnchor}
                    open={Boolean(modelMenuAnchor)}
                    onClose={() => setModelMenuAnchor(null)}
                    anchorOrigin={{ vertical: "top", horizontal: "left" }}
                    transformOrigin={{ vertical: "bottom", horizontal: "left" }}
                    slotProps={{ paper: { sx: { minWidth: 220, borderRadius: 2 } } }}
                  >
                    {models.map((model) => (
                      <MenuItem
                        key={model.id}
                        selected={model.id === field.value}
                        onClick={() => {
                          field.onChange(model.id);
                          setModelMenuAnchor(null);
                        }}
                      >
                        <Stack spacing={0.25}>
                          <Typography variant="body2" fontWeight={600}>
                            {model.display_name}
                          </Typography>
                          {model.description ? (
                            <Typography variant="caption" color="text.secondary">
                              {model.description}
                            </Typography>
                          ) : null}
                        </Stack>
                      </MenuItem>
                    ))}
                  </Menu>
                </>
              );
            }}
          />

          <Box sx={{ flex: 1 }} />

          {/* Send button */}
          <Tooltip title={mutation.isPending ? "Responding…" : "Send (Enter)"}>
            <span>
              <IconButton
                aria-label="Send"
                disabled={mutation.isPending || models.length === 0}
                type="submit"
                size="small"
                sx={{
                  bgcolor: "primary.main",
                  color: "primary.contrastText",
                  width: 32,
                  height: 32,
                  "&:hover": { bgcolor: "primary.dark" },
                  "&.Mui-disabled": {
                    bgcolor: "action.disabledBackground",
                    color: "action.disabled",
                  },
                }}
              >
                {mutation.isPending ? (
                  <CircularProgress color="inherit" size={16} />
                ) : (
                  <SendOutlinedIcon sx={{ fontSize: 16 }} />
                )}
              </IconButton>
            </span>
          </Tooltip>
        </Stack>

        {/* Inline error */}
        {errors.content?.message || errors.model?.message ? (
          <Typography color="error" sx={{ px: 2, pb: 1 }} variant="caption">
            {errors.content?.message ?? errors.model?.message}
          </Typography>
        ) : null}
      </Paper>

      {/* Hint */}
      <Typography
        variant="caption"
        color="text.disabled"
        textAlign="center"
        display="block"
        sx={{ mt: 0.75 }}
      >
        Enter to send · Shift+Enter for new line
      </Typography>
    </Box>
  );
```

- [ ] **Step 3: Remove unused imports**

Remove `Select`, `InputLabel`, `FormControl` from the imports at the top of the file (now unused). Keep `Menu` — add it to the imports if it's not already there.

Add `Menu` to MUI imports:
```tsx
import Menu from "@mui/material/Menu";
```

- [ ] **Step 4: Verify compose bar**

Check:
- Clicking the chip opens a menu above it (not a dropdown in the box)
- Only `display_name` shown in both chip and menu (no internal ID)
- Send button is a small circle, tooltip says "Send (Enter)" / "Responding…"
- Shift+Enter adds a newline, Enter sends
- Error shows inline below the toolbar, not above

- [ ] **Step 5: Commit**
```bash
git add apps/web/src/pages/chat/ChatPage.tsx
git commit -m "feat(chat): redesign compose bar — chip model picker, clean layout, keyboard hints"
```

---

## Task 5: Polish — spacing, separator, and scroll

**Problem:** A few rough edges remain: the hard `borderTop` line between messages and the compose area feels rigid; the messages scroll area has asymmetric right-only padding; the overall vertical rhythm feels crowded.

**Files:**
- Modify: `apps/web/src/pages/chat/ChatPage.tsx` — `MessagePanel` layout

- [ ] **Step 1: Remove the hard border-top separator**

Find the compose area wrapper Box:
```tsx
<Box
  sx={{
    pt: 2,
    borderTop: "1px solid",
    borderColor: "divider",
    bgcolor: "background.default",
  }}
>
```

Replace with:
```tsx
<Box sx={{ pt: 1.5 }}>
```

The compose Paper's `elevation={3}` already creates a visual lift — no need for a hard line.

- [ ] **Step 2: Fix messages scroll area padding**

Change the messages scroll area from:
```tsx
sx={{
  flex: 1,
  minHeight: 0,
  overflowY: "auto",
  pr: { xs: 0, sm: 1 },
  pb: 2,
}}
```
To:
```tsx
sx={{
  flex: 1,
  minHeight: 0,
  overflowY: "auto",
  pb: 2,
}}
```
(Symmetric — no right-only padding. The message bubbles themselves have `px: 1` now from Task 2.)

- [ ] **Step 3: Increase spacing between messages**

Change `<Stack spacing={2}>` to `<Stack spacing={3}>` in the messages area for more breathing room between turns.

- [ ] **Step 4: Widen the chat column slightly**

The current `maxWidth: 740` is good for centering but a touch narrow for data tables. Bump to `maxWidth: 800`.

- [ ] **Step 5: Verify overall layout**

Check the full page — no harsh divider line, messages have good spacing, tables don't clip, compose area floats cleanly at the bottom.

- [ ] **Step 6: Commit**
```bash
git add apps/web/src/pages/chat/ChatPage.tsx
git commit -m "feat(chat): polish spacing, remove hard separator, fix symmetric padding"
```

---

## Done — what changed

| Before | After |
|---|---|
| Big `h1` page title + "Connection X" subtitle | Gone — welcome text shown inline in empty state only |
| Bordered card empty state with inbox icon | Centered conversational welcome text |
| Both message types use outlined Paper boxes | User = pill bubble · Assistant = plain text |
| Full `<Select>` dropdown inside the compose box | Small `<Chip>` that opens a `<Menu>` above |
| Model shows full ID: "llama3.2:1b (ollama:llama3.2:1b)" | Only `display_name` shown |
| No AI thinking indicator | Three-dot pulse animation while pending |
| Hard `borderTop` separator | Removed — compose Paper elevation handles it |
| Asymmetric right-only padding on scroll area | Symmetric padding |
