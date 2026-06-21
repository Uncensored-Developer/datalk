import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import KeyOutlinedIcon from "@mui/icons-material/KeyOutlined";
import ModelTrainingOutlinedIcon from "@mui/icons-material/ModelTrainingOutlined";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControlLabel from "@mui/material/FormControlLabel";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { EmptyState } from "../../components/common/EmptyState";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import { SecretTextField } from "../../components/common/SecretTextField";
import type {
  ChatModel,
  Provider,
  ProviderConfig,
  ProviderConfigTestResponse,
} from "../../types/api";
import { errorMessage } from "../../utils/errors";

const knownProviders: Array<{
  provider: Provider;
  displayName: string;
  defaultBaseUrl: string;
  requiresApiKey: boolean;
}> = [
  {
    provider: "openai",
    displayName: "OpenAI",
    defaultBaseUrl: "https://api.openai.com",
    requiresApiKey: true,
  },
  {
    provider: "anthropic",
    displayName: "Anthropic",
    defaultBaseUrl: "https://api.anthropic.com",
    requiresApiKey: true,
  },
  {
    provider: "gemini",
    displayName: "Gemini",
    defaultBaseUrl: "https://generativelanguage.googleapis.com",
    requiresApiKey: true,
  },
  {
    provider: "ollama",
    displayName: "Ollama",
    defaultBaseUrl: "http://localhost:11434",
    requiresApiKey: false,
  },
];

type ProviderConfigForm = {
  display_name: string;
  api_key: string;
  base_url: string;
  is_enabled: boolean;
  metadata: string;
};

export function ProviderConfigsPage() {
  const { apiClient } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const isOnboarding = searchParams.get("onboarding") === "1";
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null);

  const configsQuery = useQuery({
    queryKey: ["provider-configs"],
    queryFn: () => apiClient.get<ProviderConfig[]>("/chat/provider-configs"),
  });

  const modelsQuery = useQuery({
    queryKey: ["chat-models"],
    queryFn: () => apiClient.get<ChatModel[]>("/chat/models"),
    retry: false,
  });

  if (configsQuery.isLoading) {
    return <LoadingState label="Loading provider configs" />;
  }

  if (configsQuery.isError) {
    return (
      <ErrorState
        title="Could not load provider configs"
        message={errorMessage(configsQuery.error)}
        onRetry={() => void configsQuery.refetch()}
      />
    );
  }

  const configs = configsQuery.data ?? [];
  const configsByProvider = new Map(configs.map((config) => [config.provider, config]));

  return (
    <Stack spacing={3}>
      <Stack spacing={0.5}>
        <Typography variant="h1">Provider Configs</Typography>
        <Typography color="text.secondary">
          Configure LLM providers used for model listing and chat.
        </Typography>
      </Stack>

      {isOnboarding ? (
        <Alert
          severity="info"
          action={
            configs.length > 0 ? (
              <Button
                color="inherit"
                onClick={() => navigate("/chat", { replace: true })}
                size="small"
              >
                Go to chat
              </Button>
            ) : null
          }
        >
          Configure at least one enabled provider so Datalk can list models and answer chat messages.
        </Alert>
      ) : null}

      <Grid container spacing={2}>
        {knownProviders.map((providerInfo) => (
          <Grid key={providerInfo.provider} size={{ xs: 12, md: 6 }}>
            <ProviderCard
              config={configsByProvider.get(providerInfo.provider)}
              providerInfo={providerInfo}
              onEdit={() => setEditingProvider(providerInfo.provider)}
            />
          </Grid>
        ))}
      </Grid>

      <ModelsPanel
        models={modelsQuery.data ?? []}
        error={modelsQuery.error}
        isError={modelsQuery.isError}
        isLoading={modelsQuery.isLoading}
        onRetry={() => void modelsQuery.refetch()}
      />

      <ProviderConfigDialog
        config={editingProvider ? configsByProvider.get(editingProvider) : undefined}
        provider={editingProvider}
        open={Boolean(editingProvider)}
        onClose={() => setEditingProvider(null)}
        onSaved={() => {
          if (isOnboarding) {
            navigate("/chat", { replace: true });
          }
        }}
      />
    </Stack>
  );
}

function ProviderCard({
  config,
  providerInfo,
  onEdit,
}: {
  config?: ProviderConfig;
  providerInfo: (typeof knownProviders)[number];
  onEdit: () => void;
}) {
  return (
    <Paper variant="outlined" sx={{ p: 2, height: "100%" }}>
      <Stack spacing={2}>
        <Stack
          direction="row"
          spacing={1}
          alignItems="flex-start"
          justifyContent="space-between"
        >
          <Stack spacing={0.5} sx={{ minWidth: 0 }}>
            <Typography fontWeight={800}>{config?.display_name ?? providerInfo.displayName}</Typography>
            <Typography color="text.secondary" variant="body2" noWrap>
              {config?.base_url || providerInfo.defaultBaseUrl}
            </Typography>
          </Stack>
          <Tooltip title="Edit provider config">
            <IconButton
              aria-label={`Edit ${providerInfo.displayName}`}
              onClick={onEdit}
            >
              <EditOutlinedIcon />
            </IconButton>
          </Tooltip>
        </Stack>

        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
          {config ? (
            <Chip label="configured" color="success" size="small" />
          ) : (
            <Chip label="not configured" size="small" />
          )}
          {config?.is_enabled ? (
            <Chip label="enabled" color="primary" size="small" />
          ) : (
            <Chip label="disabled" size="small" />
          )}
          {config?.has_api_key ? (
            <Chip icon={<KeyOutlinedIcon />} label="API key set" size="small" />
          ) : providerInfo.requiresApiKey ? (
            <Chip label="API key required" color="warning" size="small" />
          ) : (
            <Chip label="API key optional" size="small" />
          )}
        </Stack>

        <Typography color="text.secondary" variant="body2">
          {providerInfo.requiresApiKey
            ? "Remote provider. API key is required when creating the config."
            : "Local provider. API key can be omitted."}
        </Typography>
      </Stack>
    </Paper>
  );
}

function ModelsPanel({
  models,
  error,
  isError,
  isLoading,
  onRetry,
}: {
  models: ChatModel[];
  error: unknown;
  isError: boolean;
  isLoading: boolean;
  onRetry: () => void;
}) {
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Stack spacing={2}>
        <Stack direction="row" spacing={1} alignItems="center">
          <ModelTrainingOutlinedIcon color="primary" />
          <Stack spacing={0.25}>
            <Typography component="h2" variant="h2">
              Available models
            </Typography>
            <Typography color="text.secondary" variant="body2">
              Models are loaded from currently enabled provider configs.
            </Typography>
          </Stack>
        </Stack>

        {isLoading ? <LoadingState label="Loading models" /> : null}
        {isError ? (
          <Alert severity="warning" action={<Button onClick={onRetry}>Retry</Button>}>
            {errorMessage(error)}
          </Alert>
        ) : null}
        {!isLoading && !isError && models.length === 0 ? (
          <EmptyState
            title="No models available"
            description="Enable and configure a provider to list models."
          />
        ) : null}
        {!isLoading && !isError && models.length > 0 ? (
          <Grid container spacing={1.5}>
            {models.map((model) => (
              <Grid key={model.id} size={{ xs: 12, md: 6 }}>
                <Paper variant="outlined" sx={{ p: 1.5 }}>
                  <Stack spacing={1}>
                    <Stack
                      direction="row"
                      spacing={1}
                      alignItems="center"
                      justifyContent="space-between"
                    >
                      <Typography fontWeight={800}>{model.display_name}</Typography>
                      <Chip label={model.provider} size="small" />
                    </Stack>
                    <Typography color="text.secondary" variant="body2">
                      {model.id}
                    </Typography>
                    {model.description ? (
                      <Typography color="text.secondary" variant="body2">
                        {model.description}
                      </Typography>
                    ) : null}
                    <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                      {model.is_enabled ? (
                        <Chip label="enabled" size="small" color="success" />
                      ) : (
                        <Chip label="disabled" size="small" />
                      )}
                      {model.capabilities.supports_structured_output ? (
                        <Chip label="structured output" size="small" />
                      ) : null}
                      {model.capabilities.supports_system_instructions ? (
                        <Chip label="system instructions" size="small" />
                      ) : null}
                      {model.capabilities.supports_tool_calling ? (
                        <Chip label="tool calling" size="small" />
                      ) : null}
                    </Stack>
                  </Stack>
                </Paper>
              </Grid>
            ))}
          </Grid>
        ) : null}
      </Stack>
    </Paper>
  );
}

function ProviderConfigDialog({
  config,
  provider,
  open,
  onClose,
  onSaved,
}: {
  config?: ProviderConfig;
  provider: Provider | null;
  open: boolean;
  onClose: () => void;
  onSaved?: (config: ProviderConfig) => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const providerInfo = knownProviders.find((item) => item.provider === provider);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    control,
    formState: { errors },
    getValues,
    handleSubmit,
    register,
    reset,
  } = useForm<ProviderConfigForm>({
    values: {
      display_name: config?.display_name ?? providerInfo?.displayName ?? "",
      api_key: "",
      base_url: config?.base_url ?? providerInfo?.defaultBaseUrl ?? "",
      is_enabled: config?.is_enabled ?? true,
      metadata: JSON.stringify(config?.metadata ?? {}, null, 2),
    },
  });
  const [lastTestSignature, setLastTestSignature] = useState<string | null>(null);
  const [testMessage, setTestMessage] = useState<string | null>(null);
  const watchedBaseURL = useWatch({ control, name: "base_url" });
  const watchedAPIKey = useWatch({ control, name: "api_key" });

  const mutation = useMutation({
    mutationFn: (values: ProviderConfigForm) => {
      if (!provider) {
        throw new Error("provider is required");
      }

      const payload = providerPayloadFromForm(values, true);

      return apiClient.put<ProviderConfig>(
        `/chat/provider-configs/${provider}`,
        payload,
      );
    },
    onSuccess(savedConfig) {
      setSubmitError(null);
      reset();
      onClose();
      onSaved?.(savedConfig);
      void queryClient.invalidateQueries({ queryKey: ["provider-configs"] });
      void queryClient.invalidateQueries({ queryKey: ["chat-models"] });
    },
    onError(error) {
      setSubmitError(errorMessage(error));
    },
  });

  const testMutation = useMutation({
    mutationFn: async (values: ProviderConfigForm) => {
      if (!provider) {
        throw new Error("provider is required");
      }
      const payload = providerPayloadFromForm(values, false);
      const response = await apiClient.post<ProviderConfigTestResponse>(
        `/chat/provider-configs/${provider}/test`,
        payload,
      );
      return {
        modelCount: response.model_count,
        signature: providerTestSignature(provider, values),
      };
    },
    onSuccess(result) {
      setSubmitError(null);
      setLastTestSignature(result.signature);
      setTestMessage(`Provider test succeeded. ${result.modelCount} models found.`);
    },
    onError(error) {
      setLastTestSignature(null);
      setTestMessage(null);
      setSubmitError(errorMessage(error));
    },
  });

  const close = () => {
    if (!mutation.isPending && !testMutation.isPending) {
      setSubmitError(null);
      setLastTestSignature(null);
      setTestMessage(null);
      reset();
      onClose();
    }
  };

  const onSubmit = handleSubmit((values) => {
    setSubmitError(null);
    setTestMessage(null);
    try {
      parseMetadata(values.metadata);
    } catch (error) {
      setSubmitError(errorMessage(error));
      return;
    }
    if (provider && requiresProviderConfigTest(config, provider, values)) {
      const signature = providerTestSignature(provider, values);
      if (signature !== lastTestSignature) {
        setSubmitError("Test the current provider credentials before saving.");
        return;
      }
    }
    mutation.mutate(values);
  });

  const currentProviderValues = {
    ...getValues(),
    api_key: watchedAPIKey ?? "",
    base_url: watchedBaseURL ?? "",
  };
  const testMatchesCurrentCredentials =
    provider &&
    lastTestSignature === providerTestSignature(provider, currentProviderValues);

  return (
    <Dialog fullWidth maxWidth="sm" open={open} onClose={close}>
      <Box component="form" onSubmit={onSubmit}>
        <DialogTitle>{providerInfo?.displayName ?? "Provider"} config</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            {config?.has_api_key ? (
              <Alert severity="info">
                An API key is already stored. Leave the API key field blank to
                preserve it.
              </Alert>
            ) : null}
            <TextField
              autoFocus
              label="Display name"
              error={Boolean(errors.display_name)}
              helperText={errors.display_name?.message}
              fullWidth
              {...register("display_name", {
                required: "Display name is required",
              })}
            />
            <TextField
              label="Base URL"
              error={Boolean(errors.base_url)}
              helperText={errors.base_url?.message}
              fullWidth
              {...register("base_url", { required: "Base URL is required" })}
            />
            <SecretTextField
              label="API key"
              error={Boolean(errors.api_key)}
              helperText={
                errors.api_key?.message ??
                (providerInfo?.requiresApiKey && !config
                  ? "Required for this provider."
                  : "Optional on update; blank preserves the stored key.")
              }
              fullWidth
              {...register("api_key", {
                validate: (value) =>
                  !providerInfo?.requiresApiKey || config || value.trim()
                    ? true
                    : "API key is required when creating this provider",
              })}
            />
            <Controller
              control={control}
              name="is_enabled"
              render={({ field }) => (
                <FormControlLabel
                  control={
                    <Switch
                      checked={field.value}
                      onChange={(event) => field.onChange(event.target.checked)}
                    />
                  }
                  label="Enabled"
                />
              )}
            />
            <TextField
              label="Metadata JSON"
              multiline
              minRows={5}
              fullWidth
              {...register("metadata")}
            />
            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} alignItems={{ xs: "stretch", sm: "center" }}>
              <Button
                disabled={testMutation.isPending || mutation.isPending}
                onClick={() => testMutation.mutate(getValues())}
                startIcon={testMutation.isPending ? <CircularProgress color="inherit" size={16} /> : undefined}
                variant="outlined"
              >
                Test provider
              </Button>
              {testMessage && testMatchesCurrentCredentials ? (
                <Alert severity="success" sx={{ py: 0 }}>
                  {testMessage}
                </Alert>
              ) : null}
            </Stack>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button
            disabled={mutation.isPending || testMutation.isPending}
            startIcon={mutation.isPending ? <CircularProgress color="inherit" size={16} /> : undefined}
            type="submit"
            variant="contained"
          >
            Save
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}

function parseMetadata(value: string) {
  if (!value.trim()) {
    return {};
  }

  const parsed = JSON.parse(value) as unknown;
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("Metadata must be a JSON object");
  }

  return parsed as Record<string, unknown>;
}

function providerPayloadFromForm(values: ProviderConfigForm, omitBlankAPIKey: boolean) {
  const metadata = parseMetadata(values.metadata);
  const payload: {
    display_name: string;
    api_key?: string;
    base_url: string;
    is_enabled: boolean;
    metadata: Record<string, unknown>;
  } = {
    display_name: values.display_name.trim(),
    base_url: values.base_url.trim(),
    is_enabled: values.is_enabled,
    metadata,
  };

  const apiKey = values.api_key.trim();
  if (apiKey || !omitBlankAPIKey) {
    payload.api_key = apiKey;
  }

  return payload;
}

function requiresProviderConfigTest(
  config: ProviderConfig | undefined,
  provider: Provider,
  values: ProviderConfigForm,
) {
  if (!config) {
    return true;
  }

  return Boolean(values.api_key.trim()) || values.base_url.trim() !== (config.base_url ?? "") || !provider;
}

function providerTestSignature(provider: Provider, values: ProviderConfigForm) {
  return JSON.stringify({
    provider,
    api_key: values.api_key.trim(),
    base_url: values.base_url.trim(),
  });
}
