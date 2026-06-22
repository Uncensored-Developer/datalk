import CloseFullscreenOutlinedIcon from "@mui/icons-material/CloseFullscreenOutlined";
import CodeOutlinedIcon from "@mui/icons-material/CodeOutlined";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import KeyboardArrowRightOutlinedIcon from "@mui/icons-material/KeyboardArrowRightOutlined";
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
import Divider from "@mui/material/Divider";
import FormControl from "@mui/material/FormControl";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
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
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { EmptyState } from "../../components/common/EmptyState";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import type {
  ChatModel,
  Connection,
  Conversation,
  MessageExecution,
  MessageListItem,
  Provider,
  SendMessageProgress,
  SendMessageResponse,
} from "../../types/api";
import { errorMessage } from "../../utils/errors";

type SendForm = {
  content: string;
  model: string;
};

type StartConversationForm = SendForm & {
  connection_id: string;
};

const lastChatModelKey = "datalk.chat.lastModel";
const requireNaturalResponseKey = "datalk.chat.requireNaturalResponse";

export function ChatPage() {
  const { apiClient } = useAuth();
  const navigate = useNavigate();
  const { conversationID } = useParams();
  const [searchParams] = useSearchParams();
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
      <StartConversationPanel
        models={modelsQuery.data ?? []}
        modelsError={modelsQuery.isError ? errorMessage(modelsQuery.error) : null}
        modelsIsLoading={modelsQuery.isLoading}
        defaultConnectionID={searchParams.get("connection_id") ?? ""}
        onCreated={(conversation) => navigate(`/chat/${conversation.id}`)}
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
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const streamTimersRef = useRef<number[]>([]);
  const [streamedNaturalResponses, setStreamedNaturalResponses] = useState<Record<number, string>>({});
  const [pendingProgress, setPendingProgress] = useState<SendMessageProgress | null>(null);
  const lastMessageID = messages.at(-1)?.message.id;

  useEffect(() => {
    if ((lastMessageID || pendingProgress) && typeof messagesEndRef.current?.scrollIntoView === "function") {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [lastMessageID, pendingProgress]);

  useEffect(() => {
    const timers = streamTimersRef.current;
    return () => {
      for (const timer of timers) {
        window.clearInterval(timer);
      }
    };
  }, []);

  if (!conversation) {
    return (
      <EmptyState
        title="Select a conversation"
        description="Choose a conversation from the side navigation or create a new one."
      />
    );
  }

  const streamNaturalResponse = (messageID: number, fullText: string) => {
    const chunks = fullText.match(/\S+\s*/g) ?? [fullText];
    let index = 0;
    setStreamedNaturalResponses((current) => ({ ...current, [messageID]: "" }));

    const timer = window.setInterval(() => {
      index += 1;
      const visibleText = chunks.slice(0, index).join("");
      setStreamedNaturalResponses((current) => ({
        ...current,
        [messageID]: visibleText,
      }));

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
  };

  const handleSendSuccess = (response: SendMessageResponse) => {
    const naturalResponse = response.assistant_message.natural_response?.trim();
    if (!naturalResponse) {
      return false;
    }

    queryClient.setQueryData<MessageListItem[]>(
      ["chat-messages", conversation.id],
      (current = []) => {
        const nextItems: MessageListItem[] = [
          { message: response.user_message, retrieval: response.retrieval },
          {
            message: response.assistant_message,
            execution: response.execution,
          },
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
      }}
    >
      <Stack spacing={0.5} sx={{ pb: 2 }}>
        <Typography variant="h1">{conversationTitle(conversation)}</Typography>
        <Typography color="text.secondary" variant="body2">
          Connection {conversation.connection_id}
        </Typography>
      </Stack>

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
          sx={{ mb: 2 }}
        >
          {modelsError}
        </Alert>
      ) : null}

      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: "auto",
          pr: { xs: 0, sm: 1 },
          pb: 2,
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
        {!isLoading && !messagesError && messages.length === 0 ? (
          <EmptyState
            title="No messages yet"
            description="Send the first question for this conversation."
          />
        ) : null}
        <Stack spacing={2}>
          {messages.map((item) => (
            <MessageItem
              key={item.message.id}
              item={item}
              streamedNaturalResponse={streamedNaturalResponses[item.message.id]}
            />
          ))}
          {pendingProgress ? <PendingAssistantMessage progress={pendingProgress} /> : null}
          <Box ref={messagesEndRef} />
        </Stack>
      </Box>

      <Box
        sx={{
          pt: 2,
          borderTop: "1px solid",
          borderColor: "divider",
          bgcolor: "background.default",
        }}
      >
        <SendMessageForm
          conversationID={conversation.id}
          models={models}
          onProgress={setPendingProgress}
          onSendSuccess={handleSendSuccess}
          onSendSettled={() => setPendingProgress(null)}
        />
      </Box>
    </Box>
  );
}

function StartConversationPanel({
  models,
  modelsError,
  modelsIsLoading,
  defaultConnectionID,
  onCreated,
}: {
  models: ChatModel[];
  modelsError: string | null;
  modelsIsLoading: boolean;
  defaultConnectionID: string;
  onCreated: (conversation: Conversation) => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const connectionsQuery = useQuery({
    queryKey: ["connections"],
    queryFn: () => apiClient.get<Connection[]>("/connections"),
  });
  const connections = Array.isArray(connectionsQuery.data) ? connectionsQuery.data : [];
  const initialConnectionID = connections.some((connection) => String(connection.id) === defaultConnectionID)
    ? defaultConnectionID
    : String(connections[0]?.id ?? "");
  const defaultModel = useDefaultChatModel(models);
  const [requireNaturalResponse, setRequireNaturalResponse] = useNaturalResponsePreference();
  const selectedModelByID = useMemo(
    () => new Map(models.map((model) => [model.id, model])),
    [models],
  );
  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
    setError,
  } = useForm<StartConversationForm>({
    values: {
      connection_id: initialConnectionID,
      content: "",
      model: defaultModel,
    },
  });
  const contentField = register("content", {
    validate: (value) => value.trim() ? true : "Message is required",
  });

  const mutation = useMutation({
    mutationFn: async (values: StartConversationForm) => {
      const selectedModel = selectedModelByID.get(values.model);
      if (!selectedModel) {
        throw new Error("Model is required");
      }

      const conversation = await apiClient.post<Conversation>("/chat/conversations", {
        connection_id: Number(values.connection_id),
        title: null,
      });
      queryClient.setQueryData(["chat-conversation", conversation.id], conversation);

      const response = await apiClient.post<SendMessageResponse>(
        `/chat/conversations/${conversation.id}/messages`,
        {
          content: values.content.trim(),
          provider: selectedModel.provider as Provider,
          model: values.model,
          require_natural_response: requireNaturalResponse,
        },
      );

      queryClient.setQueryData<MessageListItem[]>(["chat-messages", conversation.id], [
        { message: response.user_message, retrieval: response.retrieval },
        {
          message: response.assistant_message,
          execution: response.execution,
        },
      ]);
      queryClient.setQueryData(["chat-conversation", conversation.id], response.conversation);

      try {
        const titledConversation = await apiClient.post<Conversation>(
          `/chat/conversations/${conversation.id}/title/infer`,
          {},
        );
        queryClient.setQueryData(["chat-conversation", conversation.id], titledConversation);
      } catch {
        // Title inference is best-effort after the first successful message.
      }

      return conversation;
    },
    onSuccess(conversation, values) {
      window.localStorage.setItem(lastChatModelKey, values.model);
      window.localStorage.setItem(requireNaturalResponseKey, String(requireNaturalResponse));
      reset({ connection_id: values.connection_id, content: "", model: values.model });
      void queryClient.invalidateQueries({ queryKey: ["chat-conversations"] });
      onCreated(conversation);
    },
    onError(error) {
      setError("content", { message: errorMessage(error) });
    },
  });

  if (modelsIsLoading || connectionsQuery.isLoading) {
    return <LoadingState label="Loading chat" />;
  }

  return (
    <Box
      sx={{
        minHeight: "calc(100vh - 160px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Paper
        component="form"
        onSubmit={handleSubmit((values) => mutation.mutate(values))}
        variant="outlined"
        sx={{ width: "min(760px, 100%)", p: { xs: 2, sm: 3 }, borderRadius: 2 }}
      >
        <Stack spacing={2}>
          <Stack spacing={0.5}>
            <Typography variant="h1">New Chat</Typography>
            <Typography color="text.secondary">
              Ask a question against an available database connection.
            </Typography>
          </Stack>
          {modelsError ? <Alert severity="warning">{modelsError}</Alert> : null}
          {connectionsQuery.isError ? (
            <Alert severity="error">{errorMessage(connectionsQuery.error)}</Alert>
          ) : null}
          <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
            <Controller
              control={control}
              name="connection_id"
              rules={{ required: "Connection is required" }}
              render={({ field }) => (
                <FormControl error={Boolean(errors.connection_id)} fullWidth>
                  <InputLabel id="start-connection-label">Connection</InputLabel>
                  <Select
                    {...field}
                    disabled={connections.length === 0}
                    label="Connection"
                    labelId="start-connection-label"
                  >
                    {connections.map((connection) => (
                      <MenuItem key={connection.id} value={String(connection.id)}>
                        {connection.name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            />
            <Controller
              control={control}
              name="model"
              rules={{ required: "Model is required" }}
              render={({ field }) => (
                <FormControl error={Boolean(errors.model)} fullWidth>
                  <InputLabel id="start-model-label">Model</InputLabel>
                  <Select
                    {...field}
                    disabled={models.length === 0}
                    label="Model"
                    labelId="start-model-label"
                  >
                    {models.map((model) => (
                      <MenuItem key={model.id} value={model.id}>
                        {model.display_name} ({model.id})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            />
          </Stack>
          <TextField
            multiline
            minRows={4}
            error={Boolean(errors.content)}
            helperText={errors.content?.message ?? errors.connection_id?.message ?? errors.model?.message}
            placeholder="Message Datalk"
            fullWidth
            {...contentField}
          />
          <Stack direction="row" alignItems="center" spacing={1}>
            <Tooltip title={requireNaturalResponse ? "Natural response on" : "Natural response off"}>
              <IconButton
                aria-label={
                  requireNaturalResponse ? "Turn natural response off" : "Turn natural response on"
                }
                color={requireNaturalResponse ? "primary" : "default"}
                onClick={() => {
                  const nextValue = !requireNaturalResponse;
                  setRequireNaturalResponse(nextValue);
                  window.localStorage.setItem(requireNaturalResponseKey, String(nextValue));
                }}
              >
                <PsychologyAltOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Box sx={{ flex: 1 }} />
            <Button
              disabled={mutation.isPending || models.length === 0 || connections.length === 0}
              startIcon={mutation.isPending ? <CircularProgress color="inherit" size={16} /> : <SendOutlinedIcon />}
              type="submit"
              variant="contained"
            >
              Send
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Box>
  );
}

function PendingAssistantMessage({ progress }: { progress: SendMessageProgress }) {
  return (
    <Stack direction="row" justifyContent="flex-start" sx={{ width: "100%" }}>
      <Paper
        variant="outlined"
        sx={{
          maxWidth: { xs: "100%", md: "78%" },
          p: 1.5,
          borderRadius: 2,
          bgcolor: "background.paper",
          borderColor: "divider",
        }}
      >
        <Stack alignItems="center" direction="row" spacing={1.25}>
          <CircularProgress size={16} />
          <Typography color="text.secondary" variant="body2">
            {progressMessage(progress)}
          </Typography>
        </Stack>
      </Paper>
    </Stack>
  );
}

function progressMessage(progress: SendMessageProgress) {
  switch (progress.stage) {
    case "retrieving_schema":
      return "Retrieving schema context";
    case "generating_sql":
      return "Generating SQL";
    case "executing_sql":
      return "Executing SQL";
    case "regenerating_sql":
      return "Execution failed, regenerating SQL";
    case "generating_response":
      return "Generating response";
    default:
      return "Working";
  }
}

function MessageItem({
  item,
  streamedNaturalResponse,
}: {
  item: MessageListItem;
  streamedNaturalResponse?: string;
}) {
  const [detailsOpen, setDetailsOpen] = useState(false);
  const isAssistant = item.message.role === "assistant";
  const hasModelInfo = Boolean(item.message.provider || item.message.model);
  const hasNaturalResponse = isAssistant && Boolean(item.message.natural_response);
  const hasHiddenDetails = hasNaturalResponse && Boolean(item.message.content || item.execution);
  const messageText = hasNaturalResponse
    ? streamedNaturalResponse ?? item.message.natural_response
    : item.message.content;

  return (
    <Stack
      direction="row"
      justifyContent={isAssistant ? "flex-start" : "flex-end"}
      sx={{ width: "100%" }}
    >
      <Paper
        variant="outlined"
        sx={{
          p: 1.5,
          borderRadius: 3,
          width: "fit-content",
          maxWidth: { xs: "100%", md: "78%" },
          bgcolor: isAssistant ? "background.paper" : "primary.main",
          color: isAssistant ? "text.primary" : "primary.contrastText",
          borderColor: isAssistant ? "divider" : "primary.main",
          "& .assistant-message-controls": {
            opacity: 0,
            transition: (theme) =>
              theme.transitions.create("opacity", {
                duration: theme.transitions.duration.shortest,
              }),
          },
          "&:hover .assistant-message-controls, &:focus-within .assistant-message-controls": {
            opacity: 1,
          },
        }}
      >
        <Stack spacing={1.5}>
          {isAssistant && (hasModelInfo || hasHiddenDetails) ? (
            <Stack
              direction="row"
              justifyContent="flex-end"
              spacing={0.5}
            >
              {hasModelInfo ? (
                <Box className="assistant-message-controls">
                  <Tooltip
                    title={[
                      item.message.provider ? `Provider: ${item.message.provider}` : null,
                      item.message.model ? `Model: ${item.message.model}` : null,
                    ]
                      .filter(Boolean)
                      .join(" | ")}
                  >
                    <IconButton aria-label="Message model details" size="small">
                      <InfoOutlinedIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Box>
              ) : null}
              {hasHiddenDetails ? (
                <Tooltip title={detailsOpen ? "Hide SQL and results" : "Show SQL and results"}>
                  <IconButton
                    aria-label={detailsOpen ? "Hide SQL and results" : "Show SQL and results"}
                    color={detailsOpen ? "primary" : "default"}
                    onClick={() => setDetailsOpen((open) => !open)}
                    size="small"
                  >
                    <KeyboardArrowRightOutlinedIcon
                      fontSize="small"
                      sx={{
                        transform: detailsOpen ? "rotate(90deg)" : "rotate(0deg)",
                        transition: (theme) =>
                          theme.transitions.create("transform", {
                            duration: theme.transitions.duration.shortest,
                          }),
                      }}
                    />
                  </IconButton>
                </Tooltip>
              ) : null}
            </Stack>
          ) : null}
          <Typography sx={{ whiteSpace: "pre-wrap" }}>{messageText}</Typography>
          {item.message.error_message ? (
            <Alert severity="error">{item.message.error_message}</Alert>
          ) : null}
          {hasNaturalResponse ? (
            <Collapse in={detailsOpen} unmountOnExit>
              <Stack spacing={1.5}>
                <Divider />
                {item.message.content ? (
                  <Typography color="text.secondary" sx={{ whiteSpace: "pre-wrap" }} variant="body2">
                    {item.message.content}
                  </Typography>
                ) : null}
                {item.execution ? <ExecutionPanel execution={item.execution} /> : null}
              </Stack>
            </Collapse>
          ) : item.execution ? (
            <ExecutionPanel execution={item.execution} />
          ) : null}
        </Stack>
      </Paper>
    </Stack>
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
    <Paper variant="outlined" sx={{ p: 1.5, color: "text.primary" }}>
      <Stack spacing={1.5}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" spacing={1}>
          <Box />
          <Stack
            className="assistant-message-controls"
            direction="row"
            spacing={0.5}
            alignItems="center"
          >
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
        </Stack>
        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
          {execution.result.truncated ? (
            <Chip label="truncated" color="warning" size="small" />
          ) : null}
        </Stack>
        <Collapse in={sqlOpen} unmountOnExit>
          <Box
            component="pre"
            sx={{
              m: 0,
              p: 1.5,
              borderRadius: 1,
              bgcolor: "action.hover",
              overflowX: "auto",
              fontSize: 13,
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
      }}
    >
      <Typography color="text.secondary" variant="caption">
        {column.name}
      </Typography>
      <Typography component="div" fontWeight={800} sx={{ mt: 0.5, wordBreak: "break-word" }} variant="h2">
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
    <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: "100%" }}>
      <Table stickyHeader size="small" sx={{ minWidth: 560 }}>
        <TableHead>
          <TableRow>
            {execution.result.columns.map((column) => (
              <TableCell key={column.name}>{column.name}</TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {execution.result.rows.map((row, index) => (
            <TableRow key={index}>
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
  onProgress,
  onSendSuccess,
  onSendSettled,
}: {
  conversationID: number;
  models: ChatModel[];
  onProgress: (progress: SendMessageProgress | null) => void;
  onSendSuccess: (response: SendMessageResponse) => boolean;
  onSendSettled: () => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const defaultModel = useDefaultChatModel(models);
  const [requireNaturalResponse, setRequireNaturalResponse] = useNaturalResponsePreference();
  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
    setError,
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
      const body = {
        content: values.content.trim(),
        provider: selectedModel?.provider as Provider,
        model: values.model,
        require_natural_response: requireNaturalResponse,
      };

      if (typeof ReadableStream === "undefined") {
        return apiClient.post<SendMessageResponse>(
          `/chat/conversations/${conversationID}/messages`,
          body,
        );
      }

      onProgress({ stage: "retrieving_schema" });
      return apiClient.postEventStream<SendMessageProgress, SendMessageResponse>(
        `/chat/conversations/${conversationID}/messages/stream`,
        body,
        { onProgress },
      );
    },
    onSuccess(response, values) {
      window.localStorage.setItem(lastChatModelKey, values.model);
      window.localStorage.setItem(requireNaturalResponseKey, String(requireNaturalResponse));
      reset({ content: "", model: values.model });
      const responseHandled = onSendSuccess(response);
      if (!responseHandled) {
        void queryClient.invalidateQueries({ queryKey: ["chat-messages", conversationID] });
      }
      void queryClient.invalidateQueries({ queryKey: ["chat-conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["chat-conversation", conversationID] });
    },
    onError(error) {
      setError("content", { message: errorMessage(error) });
    },
    onSettled() {
      onSendSettled();
    },
  });

  return (
    <Paper
      component="form"
      onSubmit={handleSubmit((values) => mutation.mutate(values))}
      variant="outlined"
      sx={{
        p: 1,
        borderRadius: 3,
        bgcolor: "background.paper",
        boxShadow: (theme) => theme.shadows[1],
      }}
    >
      <TextField
        multiline
        maxRows={6}
        minRows={1}
        error={Boolean(errors.content)}
        placeholder="Message Datalk"
        fullWidth
        variant="standard"
        slotProps={{
          htmlInput: {
            "aria-label": "Message",
          },
          input: {
            disableUnderline: true,
            sx: {
              px: 1.25,
              py: 1,
              fontSize: 16,
              lineHeight: 1.5,
            },
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
      <Stack direction="row" alignItems="center" spacing={1} sx={{ px: 0.5, pb: 0.25 }}>
        <Controller
          control={control}
          name="model"
          rules={{ required: "Model is required" }}
          render={({ field }) => (
            <FormControl error={Boolean(errors.model)} size="small" sx={{ minWidth: 220 }}>
              <InputLabel id="model-label">Model</InputLabel>
              <Select
                {...field}
                label="Model"
                labelId="model-label"
                disabled={models.length === 0}
                sx={{ borderRadius: 999 }}
              >
                {models.map((model) => (
                  <MenuItem key={model.id} value={model.id}>
                    {model.display_name} ({model.id})
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
        />
        <Tooltip title={requireNaturalResponse ? "Natural response on" : "Natural response off"}>
          <IconButton
            aria-label={
              requireNaturalResponse ? "Turn natural response off" : "Turn natural response on"
            }
            color={requireNaturalResponse ? "primary" : "default"}
            onClick={() => {
              const nextValue = !requireNaturalResponse;
              setRequireNaturalResponse(nextValue);
              window.localStorage.setItem(requireNaturalResponseKey, String(nextValue));
            }}
            size="small"
          >
            <PsychologyAltOutlinedIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Box sx={{ flex: 1 }} />
        <Tooltip title="Send">
          <span>
            <IconButton
              aria-label="Send"
              color="primary"
              disabled={mutation.isPending || models.length === 0}
              type="submit"
              sx={{
                bgcolor: "primary.main",
                color: "primary.contrastText",
                "&:hover": { bgcolor: "primary.dark" },
                "&.Mui-disabled": {
                  bgcolor: "action.disabledBackground",
                  color: "action.disabled",
                },
              }}
            >
              {mutation.isPending ? (
                <CircularProgress color="inherit" size={20} />
              ) : (
                <SendOutlinedIcon />
              )}
            </IconButton>
          </span>
        </Tooltip>
      </Stack>
      {errors.content?.message || errors.model?.message ? (
        <Typography color="error" sx={{ px: 1.25, pt: 0.5 }} variant="caption">
          {errors.content?.message ?? errors.model?.message}
        </Typography>
      ) : null}
    </Paper>
  );
}

function useDefaultChatModel(models: ChatModel[]) {
  return useMemo(() => {
    const storedModel =
      typeof window === "undefined"
        ? null
        : window.localStorage.getItem(lastChatModelKey);

    if (storedModel && models.some((model) => model.id === storedModel)) {
      return storedModel;
    }

    return models[0]?.id ?? "";
  }, [models]);
}

function useNaturalResponsePreference() {
  return useState(() => {
    if (typeof window === "undefined") {
      return true;
    }
    const stored = window.localStorage.getItem(requireNaturalResponseKey);
    if (stored === "false") {
      return false;
    }
    if (stored === "true") {
      return true;
    }
    return true;
  });
}

function conversationTitle(conversation: Conversation) {
  return conversation.title?.trim() || "New Chat";
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
