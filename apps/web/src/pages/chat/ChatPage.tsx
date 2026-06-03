import CloseFullscreenOutlinedIcon from "@mui/icons-material/CloseFullscreenOutlined";
import CodeOutlinedIcon from "@mui/icons-material/CodeOutlined";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import OpenInFullOutlinedIcon from "@mui/icons-material/OpenInFullOutlined";
import PsychologyAltOutlinedIcon from "@mui/icons-material/PsychologyAltOutlined";
import SendOutlinedIcon from "@mui/icons-material/SendOutlined";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Collapse from "@mui/material/Collapse";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import IconButton from "@mui/material/IconButton";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useParams } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { EmptyState } from "../../components/common/EmptyState";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import type {
  ChatModel,
  Conversation,
  MessageExecution,
  MessageListItem,
  Provider,
  SendMessageResponse,
} from "../../types/api";
import { errorMessage } from "../../utils/errors";

type SendForm = {
  content: string;
  model: string;
};

const lastChatModelKey = "datalk.chat.lastModel";
const requireNaturalResponseKey = "datalk.chat.requireNaturalResponse";

export function ChatPage() {
  const { apiClient } = useAuth();
  const { conversationID } = useParams();
  const parsedConversationID = conversationID ? Number(conversationID) : null;
  const hasValidConversationID =
    parsedConversationID !== null && Number.isInteger(parsedConversationID);

  const modelsQuery = useQuery({
    queryKey: ["chat-models"],
    queryFn: () => apiClient.get<ChatModel[]>("/chat/models"),
    retry: false,
  });
  const conversationQuery = useQuery({
    queryKey: ["chat-conversation", parsedConversationID],
    queryFn: () => apiClient.get<Conversation>(`/chat/conversations/${parsedConversationID}`),
    enabled: hasValidConversationID,
  });
  const messagesQuery = useQuery({
    queryKey: ["chat-messages", parsedConversationID],
    queryFn: () =>
      apiClient.get<MessageListItem[]>(
        `/chat/conversations/${parsedConversationID}/messages?limit=50&offset=0`,
      ),
    enabled: hasValidConversationID,
    retry: false,
  });

  if (!conversationID) {
    return (
      <EmptyState
        title="Select a conversation"
        description="Choose a conversation from the side navigation or create a new one."
      />
    );
  }

  if (!hasValidConversationID) {
    return (
      <ErrorState
        title="Invalid conversation"
        message="Choose a valid conversation from the side navigation."
      />
    );
  }

  if (conversationQuery.isLoading || modelsQuery.isLoading) {
    return <LoadingState label="Loading conversation" />;
  }

  if (conversationQuery.isError) {
    return (
      <ErrorState
        title="Could not load conversation"
        message={errorMessage(conversationQuery.error)}
        onRetry={() => void conversationQuery.refetch()}
      />
    );
  }

  return (
    <MessagePanel
      conversation={conversationQuery.data ?? null}
      messages={messagesQuery.data ?? []}
      isLoading={messagesQuery.isLoading}
      messagesError={messagesQuery.isError ? errorMessage(messagesQuery.error) : null}
      messagesIsFetching={messagesQuery.isFetching}
      models={modelsQuery.data ?? []}
      modelsError={modelsQuery.isError ? errorMessage(modelsQuery.error) : null}
      modelsIsFetching={modelsQuery.isFetching}
      onRetryModels={() => void modelsQuery.refetch()}
    />
  );
}

function MessagePanel({
  conversation,
  messages,
  isLoading,
  messagesError,
  messagesIsFetching,
  models,
  modelsError,
  modelsIsFetching,
  onRetryModels,
}: {
  conversation: Conversation | null;
  messages: MessageListItem[];
  isLoading: boolean;
  messagesError: string | null;
  messagesIsFetching: boolean;
  models: ChatModel[];
  modelsError: string | null;
  modelsIsFetching: boolean;
  onRetryModels: () => void;
}) {
  const queryClient = useQueryClient();
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const streamTimersRef = useRef<number[]>([]);
  const [streamedNaturalResponses, setStreamedNaturalResponses] = useState<Record<number, string>>({});
  const lastMessageID = messages.at(-1)?.message.id;
  const [isPending, setIsPending] = useState(false);
  const [optimisticContent, setOptimisticContent] = useState<string | null>(null);

  /** Always-instant, direct DOM scroll — most reliable across browsers. */
  const scrollToBottom = useCallback(() => {
    const el = scrollContainerRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, []);

  const streamNaturalResponse = useCallback((messageID: number, fullText: string) => {
    const chunks = fullText.match(/\S+\s*/g) ?? [fullText];
    let index = 0;
    setStreamedNaturalResponses((current) => ({ ...current, [messageID]: "" }));
    const timer = window.setInterval(() => {
      index += 1;
      const visible = chunks.slice(0, index).join("");
      setStreamedNaturalResponses((current) => ({ ...current, [messageID]: visible }));
      if (index >= chunks.length) {
        window.clearInterval(timer);
        window.setTimeout(() => {
          setStreamedNaturalResponses((current) => {
            const next = { ...current };
            delete next[messageID];
            return next;
          });
        }, 400);
      }
    }, 45);
    streamTimersRef.current.push(timer);
  }, []);

  // When real messages arrive: clear optimistic state, scroll to bottom.
  useEffect(() => {
    if (!lastMessageID) return;
    setOptimisticContent(null);
    setIsPending(false);
    const id = setTimeout(scrollToBottom, 0);

    // Kick off typewriter for any new assistant message with a natural response.
    const lastMsg = messages.at(-1);
    if (lastMsg?.message.role === "assistant" && lastMsg.message.natural_response) {
      streamNaturalResponse(lastMsg.message.id, lastMsg.message.natural_response);
    }

    return () => clearTimeout(id);
  }, [lastMessageID, scrollToBottom, streamNaturalResponse]); // eslint-disable-line react-hooks/exhaustive-deps

  // Scroll instantly the moment the user's optimistic bubble appears.
  useEffect(() => {
    if (optimisticContent) scrollToBottom();
  }, [optimisticContent, scrollToBottom]);

  // Cleanup timers on unmount
  useEffect(() => {
    const timers = streamTimersRef.current;
    return () => { for (const t of timers) window.clearInterval(t); };
  }, []);

  const handleOptimisticMessage = useCallback((content: string) => {
    setOptimisticContent(content);
    setIsPending(true);
  }, []);

  // Merge real messages with the optimistic user message
  const allMessages: MessageListItem[] = useMemo(() => {
    if (!optimisticContent || !conversation) return messages;
    return [
      ...messages,
      {
        message: {
          id: -1,
          conversation_id: conversation.id,
          role: "user" as const,
          content: optimisticContent,
          status: "pending" as const,
          created_at: new Date().toISOString(),
        },
      },
    ];
  }, [messages, optimisticContent, conversation]);

  if (!conversation) {
    return (
      <EmptyState
        title="Select a conversation"
        description="Choose a conversation from the side navigation or create a new one."
      />
    );
  }

  const handleSendSuccess = (response: SendMessageResponse) => {
    const naturalResponse = response.assistant_message.natural_response?.trim();
    if (!naturalResponse) return false;

    queryClient.setQueryData<MessageListItem[]>(
      ["chat-messages", conversation.id],
      (current = []) => {
        const nextItems: MessageListItem[] = [
          { message: response.user_message, retrieval: response.retrieval },
          { message: response.assistant_message, execution: response.execution },
        ];
        const nextIDs = new Set(nextItems.map((item) => item.message.id));
        return [
          ...current.filter((item) => !nextIDs.has(item.message.id)),
          ...nextItems,
        ];
      },
    );
    streamNaturalResponse(response.assistant_message.id, naturalResponse);
    return true;
  };

  return (
    <Box
      sx={{
        height: { xs: "calc(100vh - 112px)", sm: "calc(100vh - 120px)" },
        display: "flex",
        flexDirection: "column",
        minHeight: 0,
        maxWidth: 800,
        width: "100%",
        mx: "auto",
      }}
    >
      {/* Models error banner */}
      {modelsError ? (
        <Alert
          severity="warning"
          action={
            <Button
              disabled={modelsIsFetching}
              onClick={onRetryModels}
              startIcon={modelsIsFetching ? <CircularProgress color="inherit" size={16} /> : undefined}
            >
              Retry
            </Button>
          }
          sx={{ mb: 1.5 }}
        >
          {modelsError}
        </Alert>
      ) : null}

      {/* Messages scroll area */}
      <Box
        ref={scrollContainerRef}
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: "auto",
          pb: 3,
          display: "flex",
          flexDirection: "column",
          scrollbarWidth: "none",
          "&::-webkit-scrollbar": { display: "none" },
        }}
      >
        {isLoading ? <LoadingState label="Loading messages" /> : null}

        {messagesError ? (
          <Alert
            severity="error"
            action={
              <Button
                color="inherit"
                disabled={messagesIsFetching}
                onClick={() => void queryClient.invalidateQueries({ queryKey: ["chat-messages", conversation.id] })}
                size="small"
                startIcon={messagesIsFetching ? <CircularProgress color="inherit" size={16} /> : undefined}
              >
                Retry
              </Button>
            }
          >
            {messagesError}
          </Alert>
        ) : null}

        {/* Conversational welcome — no card, no border */}
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
              px: 2,
            }}
          >
            <Typography
              variant="h2"
              fontWeight={700}
              sx={{ mb: 1, fontSize: { xs: "1.4rem", sm: "1.75rem" } }}
            >
              {conversation.title}
            </Typography>
            <Typography color="text.secondary" sx={{ maxWidth: 380 }}>
              Ask anything about your data. I'll write the SQL and show you the results.
            </Typography>
          </Box>
        ) : null}

        {/* Message list (real + optimistic) */}
        {allMessages.length > 0 ? (
          <Stack spacing={3} sx={{ pt: 2 }}>
            {allMessages.map((item) => (
              <MessageItem
                key={item.message.id}
                item={item}
                streamedNaturalResponse={streamedNaturalResponses[item.message.id]}
              />
            ))}
          </Stack>
        ) : null}

        {/* Typing indicator — shown while the API is working */}
        {isPending ? (
          <Box sx={{ display: "flex", alignItems: "center", px: 2, pt: 3 }}>
            <Box sx={{ display: "flex", gap: "5px", alignItems: "center" }}>
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
                      "0%, 60%, 100%": { opacity: 0.25, transform: "scale(1)" },
                      "30%": { opacity: 1, transform: "scale(1.25)" },
                    },
                  }}
                />
              ))}
            </Box>
          </Box>
        ) : null}

      </Box>

      {/* Compose bar — no hard border separator */}
      <Box sx={{ pt: 1.5, pb: 0.5 }}>
        <SendMessageForm
          conversationID={conversation.id}
          models={models}
          onOptimisticMessage={handleOptimisticMessage}
          onSendSuccess={handleSendSuccess}
        />
      </Box>
    </Box>
  );
}

function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  const time = date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  if (isToday) return time;

  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  const isYesterday =
    date.getFullYear() === yesterday.getFullYear() &&
    date.getMonth() === yesterday.getMonth() &&
    date.getDate() === yesterday.getDate();

  if (isYesterday) return `Yesterday ${time}`;
  return `${date.toLocaleDateString([], { month: "short", day: "numeric" })} ${time}`;
}

function MessageItem({
  item,
  streamedNaturalResponse,
}: {
  item: MessageListItem;
  streamedNaturalResponse?: string;
}) {
  const isAssistant = item.message.role === "assistant";
  const timestamp = item.message.created_at ? formatTimestamp(item.message.created_at) : null;

  // Prefer natural_response (with optional typewriter effect) over raw content
  const displayText = isAssistant && item.message.natural_response
    ? (streamedNaturalResponse ?? item.message.natural_response)
    : item.message.content;

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: isAssistant ? "flex-start" : "flex-end",
        gap: 0.5,
        px: 2,
        animation: "messageIn 0.22s cubic-bezier(0.25, 0.46, 0.45, 0.94) both",
        "@keyframes messageIn": {
          from: { opacity: 0, transform: "translateY(10px)" },
          to: { opacity: 1, transform: "translateY(0)" },
        },
      }}
    >
      {/* Bubble */}
      <Box
        sx={
          isAssistant
            ? { maxWidth: { xs: "100%", md: "85%" }, color: "text.primary" }
            : {
                maxWidth: { xs: "88%", md: "72%" },
                bgcolor: (theme) =>
                  theme.palette.mode === "dark" ? "#374151" : "#dde4f0",
                color: (theme) =>
                  theme.palette.mode === "dark" ? "#f9fafb" : "#111827",
                borderRadius: "12px 12px 3px 12px",
                px: 2,
                py: 1.25,
              }
        }
      >
        <Typography sx={{ whiteSpace: "pre-wrap", lineHeight: 1.7, fontSize: "0.9375rem" }}>
          {displayText}
        </Typography>
        {item.message.error_message ? (
          <Alert severity="error" sx={{ mt: 1 }}>{item.message.error_message}</Alert>
        ) : null}
        {item.execution ? <ExecutionPanel execution={item.execution} /> : null}
      </Box>

      {/* Permanent timestamp */}
      {timestamp ? (
        <Typography
          variant="caption"
          color="text.disabled"
          sx={{ px: 0.5, fontSize: "0.7rem", userSelect: "none" }}
        >
          {timestamp}
        </Typography>
      ) : null}
    </Box>
  );
}

function ExecutionPanel({ execution }: { execution: MessageExecution }) {
  const [sqlOpen, setSqlOpen] = useState(false);
  const [fullscreenOpen, setFullscreenOpen] = useState(false);
  const isScalarResult =
    execution.result.columns.length === 1 && execution.result.rows.length === 1;
  const executionDetails = [
    `Database: ${execution.database_kind}`,
    `Latency: ${execution.execution_latency_ms} ms`,
    `${execution.result.row_count} row${execution.result.row_count === 1 ? "" : "s"}`,
    `${execution.result.columns.length} column${execution.result.columns.length === 1 ? "" : "s"}`,
  ].join(" | ");

  return (
    <Paper
      variant="outlined"
      sx={{
        mt: 1.5,
        p: 1.5,
        color: "text.primary",
        borderRadius: 2,
        borderColor: "divider",
      }}
    >
      <Stack spacing={1.5}>
        {/* Toolbar */}
        <Stack direction="row" justifyContent="flex-end" alignItems="center" spacing={0.5}>
          <Tooltip title={executionDetails}>
            <IconButton aria-label="Execution details" size="small">
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title={sqlOpen ? "Hide SQL" : "Show SQL"}>
            <IconButton
              aria-label={sqlOpen ? "Hide SQL" : "Show SQL"}
              color={sqlOpen ? "primary" : "default"}
              onClick={() => setSqlOpen((open) => !open)}
              size="small"
            >
              <CodeOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Open table full screen">
            <IconButton
              aria-label="Open table full screen"
              size="small"
              onClick={() => setFullscreenOpen(true)}
            >
              <OpenInFullOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Stack>

        {execution.result.truncated ? (
          <Chip label="truncated" color="warning" size="small" sx={{ alignSelf: "flex-start" }} />
        ) : null}

        <Collapse in={sqlOpen} unmountOnExit>
          <Box
            component="pre"
            sx={{
              m: 0,
              p: 1.5,
              borderRadius: 1.5,
              bgcolor: "action.hover",
              overflowX: "auto",
              fontSize: 13,
              fontFamily: "monospace",
            }}
          >
            {execution.generated_sql}
          </Box>
        </Collapse>

        {isScalarResult ? (
          <ScalarResult execution={execution} />
        ) : (
          <ResultTable execution={execution} />
        )}

        <Dialog fullScreen open={fullscreenOpen} onClose={() => setFullscreenOpen(false)}>
          <DialogTitle sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Typography component="span" fontWeight={800} sx={{ flex: 1 }}>
              Query Results
            </Typography>
            <Tooltip title="Close full screen">
              <IconButton aria-label="Close table full screen" onClick={() => setFullscreenOpen(false)}>
                <CloseFullscreenOutlinedIcon />
              </IconButton>
            </Tooltip>
          </DialogTitle>
          <DialogContent sx={{ height: "100%", minHeight: 0 }}>
            <ResultTable execution={execution} />
          </DialogContent>
        </Dialog>
      </Stack>
    </Paper>
  );
}

function ScalarResult({ execution }: { execution: MessageExecution }) {
  const column = execution.result.columns[0];
  const value = execution.result.rows[0]?.[column.name];

  return (
    <Paper
      variant="outlined"
      sx={{
        px: 2,
        py: 1.75,
        bgcolor: "action.hover",
        borderStyle: "dashed",
        borderRadius: 2,
      }}
    >
      <Typography color="text.secondary" variant="caption">
        {column.name}
      </Typography>
      <Typography
        component="div"
        fontWeight={800}
        sx={{ mt: 0.5, wordBreak: "break-word" }}
        variant="h2"
      >
        {formatCellValue(value)}
      </Typography>
    </Paper>
  );
}

function ResultTable({ execution }: { execution: MessageExecution }) {
  if (execution.result.rows.length === 0) {
    return (
      <Typography color="text.secondary" variant="body2">
        No rows returned.
      </Typography>
    );
  }

  return (
    <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: "100%", borderRadius: 2 }}>
      <Table stickyHeader size="small" sx={{ minWidth: 400 }}>
        <TableHead>
          <TableRow>
            {execution.result.columns.map((column) => (
              <TableCell key={column.name} sx={{ fontWeight: 700 }}>
                {column.name}
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {execution.result.rows.map((row, index) => (
            <TableRow key={index} hover>
              {execution.result.columns.map((column) => (
                <TableCell key={column.name}>
                  {formatCellValue(row[column.name])}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function SendMessageForm({
  conversationID,
  models,
  onOptimisticMessage,
  onSendSuccess,
}: {
  conversationID: number;
  models: ChatModel[];
  onOptimisticMessage: (content: string) => void;
  onSendSuccess: (response: SendMessageResponse) => boolean;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const [modelMenuAnchor, setModelMenuAnchor] = useState<HTMLElement | null>(null);
  const [sendError, setSendError] = useState<string | null>(null);
  const [shaking, setShaking] = useState(false);
  const [requireNaturalResponse, setRequireNaturalResponse] = useState(() => {
    if (typeof window === "undefined") return true;
    const stored = window.localStorage.getItem(requireNaturalResponseKey);
    return stored !== "false";
  });

  const shake = useCallback(() => {
    setShaking(true);
    setTimeout(() => setShaking(false), 400);
  }, []);

  const defaultModel = useMemo(() => {
    const storedModel =
      typeof window === "undefined"
        ? null
        : window.localStorage.getItem(lastChatModelKey);
    if (storedModel && models.some((model) => model.id === storedModel)) {
      return storedModel;
    }
    return models[0]?.id ?? "";
  }, [models]);

  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<SendForm>({
    values: { content: "", model: defaultModel },
  });

  const contentField = register("content", {
    validate: (value) => value.trim() ? true : "Message is required",
  });

  const selectedModelByID = useMemo(
    () => new Map(models.map((model) => [model.id, model])),
    [models],
  );

  const mutation = useMutation({
    mutationFn: (values: SendForm) => {
      const selectedModel = selectedModelByID.get(values.model);
      return apiClient.post<SendMessageResponse>(
        `/chat/conversations/${conversationID}/messages`,
        {
          content: values.content.trim(),
          provider: selectedModel?.provider as Provider,
          model: values.model,
          require_natural_response: requireNaturalResponse,
        },
      );
    },
    onSuccess(response, values) {
      window.localStorage.setItem(lastChatModelKey, values.model);
      window.localStorage.setItem(requireNaturalResponseKey, String(requireNaturalResponse));
      const handled = onSendSuccess(response);
      if (!handled) {
        void queryClient.invalidateQueries({ queryKey: ["chat-messages", conversationID] });
      }
      void queryClient.invalidateQueries({ queryKey: ["chat-conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["chat-conversation", conversationID] });
    },
    onError(error) {
      setSendError(errorMessage(error));
      shake();
    },
  });

  const isPending = mutation.isPending;

  return (
    <Box>
      {/* Input row: text field + send button only */}
      <Paper
        component="form"
        onSubmit={handleSubmit((values) => {
          const content = values.content.trim();
          if (!content) return;
          setSendError(null);
          onOptimisticMessage(content);
          reset({ content: "", model: values.model });
          mutation.mutate(values);
        }, () => shake())}
        elevation={2}
        sx={{
          borderRadius: 1.5,
          bgcolor: "background.paper",
          border: "1px solid",
          borderColor: (errors.content || sendError) ? "error.main" : "divider",
          overflow: "hidden",
          display: "flex",
          alignItems: "flex-end",
          gap: 0,
          transition: "border-color 0.15s, box-shadow 0.15s",
          "&:focus-within": {
            borderColor: (errors.content || sendError) ? "error.main" : "primary.main",
            boxShadow: (theme) =>
              `0 0 0 2px ${(errors.content || sendError) ? theme.palette.error.main : theme.palette.primary.main}22`,
          },
          ...(shaking
            ? {
                animation: "shake 0.35s cubic-bezier(.36,.07,.19,.97) both",
                "@keyframes shake": {
                  "0%, 100%": { transform: "translateX(0)" },
                  "15%": { transform: "translateX(-6px)" },
                  "30%": { transform: "translateX(5px)" },
                  "45%": { transform: "translateX(-4px)" },
                  "60%": { transform: "translateX(3px)" },
                  "75%": { transform: "translateX(-2px)" },
                  "90%": { transform: "translateX(1px)" },
                },
              }
            : {}),
        }}
      >
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
              sx: {
                px: 2,
                py: 1.25,
                fontSize: "0.9375rem",
                lineHeight: 1.65,
              },
            },
          }}
          {...contentField}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !isPending) {
              event.preventDefault();
              void handleSubmit((values) => {
                const content = values.content.trim();
                if (!content) return;
                setSendError(null);
                onOptimisticMessage(content);
                reset({ content: "", model: values.model });
                mutation.mutate(values);
              }, () => shake())();
            }
          }}
        />

        <Box sx={{ pr: 1.5, pb: 1.25, flexShrink: 0 }}>
          <Tooltip title={isPending ? "Responding…" : "Send  ↵"}>
            <span>
              <IconButton
                aria-label="Send"
                disabled={isPending || models.length === 0}
                type="submit"
                size="small"
                sx={{
                  color: "primary.main",
                  "&:hover": { color: "primary.dark", bgcolor: "transparent" },
                  "&.Mui-disabled": { color: "action.disabled" },
                }}
              >
                {isPending ? (
                  <CircularProgress color="inherit" size={18} />
                ) : (
                  <SendOutlinedIcon sx={{ fontSize: 20 }} />
                )}
              </IconButton>
            </span>
          </Tooltip>
        </Box>
      </Paper>

      {/* Below-input row: keyboard hint (left) + model selector (right) */}
      <Stack direction="row" alignItems="center" sx={{ mt: 0.75, px: 0.25 }}>
        <Typography variant="caption" color="text.disabled">
          Enter to send · Shift+Enter for new line
        </Typography>

        <Box sx={{ flex: 1 }} />

        {/* Natural response toggle */}
        <Tooltip title={requireNaturalResponse ? "Natural response on" : "Natural response off"}>
          <IconButton
            size="small"
            color={requireNaturalResponse ? "primary" : "default"}
            onClick={() => {
              const next = !requireNaturalResponse;
              setRequireNaturalResponse(next);
              window.localStorage.setItem(requireNaturalResponseKey, String(next));
            }}
          >
            <PsychologyAltOutlinedIcon fontSize="small" />
          </IconButton>
        </Tooltip>

        {/* Model selector — bottom right, outside the input */}
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
                    fontSize: "0.72rem",
                    cursor: "pointer",
                    maxWidth: 200,
                    height: 24,
                  }}
                />
                <Menu
                  anchorEl={modelMenuAnchor}
                  open={Boolean(modelMenuAnchor)}
                  onClose={() => setModelMenuAnchor(null)}
                  anchorOrigin={{ vertical: "top", horizontal: "right" }}
                  transformOrigin={{ vertical: "bottom", horizontal: "right" }}
                  slotProps={{
                    paper: { sx: { minWidth: 220, borderRadius: 2, mb: 0.5 } },
                  }}
                >
                  {models.map((model) => (
                    <MenuItem
                      key={model.id}
                      selected={model.id === field.value}
                      onClick={() => {
                        field.onChange(model.id);
                        setModelMenuAnchor(null);
                      }}
                      sx={{ borderRadius: 1, mx: 0.5, my: 0.25 }}
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
      </Stack>
    </Box>
  );
}

function formatCellValue(value: unknown) {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}
