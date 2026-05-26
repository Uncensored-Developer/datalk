import AddOutlinedIcon from "@mui/icons-material/AddOutlined";
import DeleteOutlineOutlinedIcon from "@mui/icons-material/DeleteOutlineOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import KeyOutlinedIcon from "@mui/icons-material/KeyOutlined";
import RefreshOutlinedIcon from "@mui/icons-material/RefreshOutlined";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useAuth } from "../../auth/AuthProvider";
import { ConfirmDialog } from "../../components/common/ConfirmDialog";
import { EmptyState } from "../../components/common/EmptyState";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import type {
  Connection,
  ConnectionAccessGrant,
  ConnectionMetadata,
  DatabaseKind,
  SchemaRefreshResponse,
  User,
} from "../../types/api";
import { errorMessage } from "../../utils/errors";

const databaseKinds: DatabaseKind[] = ["postgres", "mysql", "cql"];

type ConnectionFormValues = {
  name: string;
  database: DatabaseKind;
  dsn: string;
  is_enabled: boolean;
  include_namespaces: string;
  exclude_namespaces: string;
  include_tables_by_namespace: string;
  exclude_tables_by_namespace: string;
  include_views: boolean;
  include_indexes: boolean;
  include_foreign_keys: boolean;
  include_comments: boolean;
};

type AccessFormValues = {
  user_id: string;
  can_query: boolean;
  allow_writes: boolean;
  can_manage: boolean;
};

export function ConnectionsPage() {
  const { apiClient, user } = useAuth();
  const queryClient = useQueryClient();
  const isAdmin = user?.role === "owner" || user?.role === "admin";
  const [createOpen, setCreateOpen] = useState(false);
  const [editingConnection, setEditingConnection] = useState<Connection | null>(null);
  const [grantConnection, setGrantConnection] = useState<Connection | null>(null);
  const [deletingConnection, setDeletingConnection] = useState<Connection | null>(null);
  const [refreshMessage, setRefreshMessage] = useState<string | null>(null);

  const connectionsQuery = useQuery({
    queryKey: ["connections"],
    queryFn: () => apiClient.get<Connection[]>("/connections"),
  });

  const refreshMutation = useMutation({
    mutationFn: (connection: Connection) =>
      apiClient.post<SchemaRefreshResponse>(
        `/connections/${connection.id}/schema-snapshot/refresh`,
      ),
    onSuccess(response) {
      setRefreshMessage(`Schema refresh accepted for connection ${response.connection_id}.`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (connection: Connection) =>
      apiClient.delete<void>(`/connections/${connection.id}`),
    onSuccess() {
      setDeletingConnection(null);
      void queryClient.invalidateQueries({ queryKey: ["connections"] });
    },
  });

  if (connectionsQuery.isLoading) {
    return <LoadingState label="Loading connections" />;
  }

  if (connectionsQuery.isError) {
    return (
      <ErrorState
        title="Could not load connections"
        message={errorMessage(connectionsQuery.error)}
        onRetry={() => void connectionsQuery.refetch()}
      />
    );
  }

  const connections = connectionsQuery.data ?? [];

  return (
    <Stack spacing={3}>
      <Stack
        direction={{ xs: "column", sm: "row" }}
        spacing={2}
        alignItems={{ xs: "stretch", sm: "center" }}
        justifyContent="space-between"
      >
        <Stack spacing={0.5}>
          <Typography variant="h1">Connections</Typography>
          <Typography color="text.secondary">
            Manage database connections and schema refreshes.
          </Typography>
        </Stack>
        {isAdmin ? (
          <Button
            startIcon={<AddOutlinedIcon />}
            variant="contained"
            onClick={() => setCreateOpen(true)}
          >
            Create connection
          </Button>
        ) : null}
      </Stack>

      {refreshMessage ? (
        <Alert severity="success" onClose={() => setRefreshMessage(null)}>
          {refreshMessage}
        </Alert>
      ) : null}
      {refreshMutation.isError ? (
        <Alert severity="error">{errorMessage(refreshMutation.error)}</Alert>
      ) : null}
      {deleteMutation.isError ? (
        <Alert severity="error">{errorMessage(deleteMutation.error)}</Alert>
      ) : null}

      {connections.length === 0 ? (
        <EmptyState
          title="No connections found"
          description={
            isAdmin
              ? "Create a connection to make database schemas available."
              : "Ask an admin to grant access to a connection."
          }
          action={
            isAdmin ? (
              <Button variant="contained" onClick={() => setCreateOpen(true)}>
                Create connection
              </Button>
            ) : null
          }
        />
      ) : (
        <Grid container spacing={2}>
          {connections.map((connection) => (
            <Grid key={connection.id} size={{ xs: 12, lg: 6 }}>
              <ConnectionCard
                connection={connection}
                isAdmin={isAdmin}
                isRefreshing={
                  refreshMutation.isPending &&
                  refreshMutation.variables?.id === connection.id
                }
                onDelete={() => setDeletingConnection(connection)}
                onEdit={() => setEditingConnection(connection)}
                onGrantAccess={() => setGrantConnection(connection)}
                onRefresh={() => refreshMutation.mutate(connection)}
              />
            </Grid>
          ))}
        </Grid>
      )}

      <ConnectionDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      <ConnectionDialog
        connection={editingConnection}
        open={Boolean(editingConnection)}
        onClose={() => setEditingConnection(null)}
      />
      <AccessGrantDialog
        connection={grantConnection}
        open={Boolean(grantConnection)}
        onClose={() => setGrantConnection(null)}
      />
      <ConfirmDialog
        open={Boolean(deletingConnection)}
        title="Delete connection"
        description={`Delete ${deletingConnection?.name ?? "this connection"}? This cannot be undone.`}
        confirmLabel="Delete"
        destructive
        onCancel={() => setDeletingConnection(null)}
        onConfirm={() => {
          if (deletingConnection) {
            deleteMutation.mutate(deletingConnection);
          }
        }}
      />
    </Stack>
  );
}

function ConnectionCard({
  connection,
  isAdmin,
  isRefreshing,
  onDelete,
  onEdit,
  onGrantAccess,
  onRefresh,
}: {
  connection: Connection;
  isAdmin: boolean;
  isRefreshing: boolean;
  onDelete: () => void;
  onEdit: () => void;
  onGrantAccess: () => void;
  onRefresh: () => void;
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
            <Typography fontWeight={800} noWrap>
              {connection.name}
            </Typography>
            <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
              <Chip label={connection.database} size="small" color="primary" />
              {connection.is_enabled ? (
                <Chip label="enabled" size="small" color="success" />
              ) : (
                <Chip label="disabled" size="small" color="error" />
              )}
            </Stack>
          </Stack>
          <Stack direction="row" spacing={0.5}>
            <Tooltip title="Refresh schema">
              <span>
                <IconButton
                  aria-label={`Refresh schema for ${connection.name}`}
                  disabled={isRefreshing}
                  onClick={onRefresh}
                >
                  <RefreshOutlinedIcon />
                </IconButton>
              </span>
            </Tooltip>
            {isAdmin ? (
              <>
                <Tooltip title="Grant access">
                  <IconButton
                    aria-label={`Grant access for ${connection.name}`}
                    onClick={onGrantAccess}
                  >
                    <KeyOutlinedIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Edit connection">
                  <IconButton
                    aria-label={`Edit ${connection.name}`}
                    onClick={onEdit}
                  >
                    <EditOutlinedIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Delete connection">
                  <IconButton
                    aria-label={`Delete ${connection.name}`}
                    color="error"
                    onClick={onDelete}
                  >
                    <DeleteOutlineOutlinedIcon />
                  </IconButton>
                </Tooltip>
              </>
            ) : null}
          </Stack>
        </Stack>

        <MetadataSummary metadata={connection.metadata} />
      </Stack>
    </Paper>
  );
}

function MetadataSummary({ metadata }: { metadata?: ConnectionMetadata | null }) {
  if (!metadata) {
    return (
      <Typography color="text.secondary" variant="body2">
        No metadata filters configured.
      </Typography>
    );
  }

  const namespaceText = [
    metadata.include_namespaces?.length
      ? `include: ${metadata.include_namespaces.join(", ")}`
      : null,
    metadata.exclude_namespaces?.length
      ? `exclude: ${metadata.exclude_namespaces.join(", ")}`
      : null,
  ]
    .filter(Boolean)
    .join(" | ");

  return (
    <Stack spacing={1}>
      <Typography color="text.secondary" variant="body2">
        {namespaceText || "All namespaces"}
      </Typography>
      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
        {metadata.include_views ? <Chip label="views" size="small" /> : null}
        {metadata.include_indexes ? <Chip label="indexes" size="small" /> : null}
        {metadata.include_foreign_keys ? (
          <Chip label="foreign keys" size="small" />
        ) : null}
        {metadata.include_comments ? <Chip label="comments" size="small" /> : null}
      </Stack>
    </Stack>
  );
}

function ConnectionDialog({
  connection,
  open,
  onClose,
}: {
  connection?: Connection | null;
  open: boolean;
  onClose: () => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const isEdit = Boolean(connection);
  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<ConnectionFormValues>({
    values: formValuesFromConnection(connection),
  });

  const mutation = useMutation({
    mutationFn: (values: ConnectionFormValues) => {
      const payload = payloadFromConnectionForm(values, isEdit);
      return isEdit
        ? apiClient.put<Connection>(`/connections/${connection?.id}`, payload)
        : apiClient.post<Connection>("/connections", payload);
    },
    onSuccess() {
      setSubmitError(null);
      reset();
      onClose();
      void queryClient.invalidateQueries({ queryKey: ["connections"] });
    },
    onError(error) {
      setSubmitError(errorMessage(error));
    },
  });

  const close = () => {
    if (!mutation.isPending) {
      setSubmitError(null);
      reset();
      onClose();
    }
  };

  return (
    <Dialog fullWidth maxWidth="md" open={open} onClose={close}>
      <Box component="form" onSubmit={handleSubmit((values) => mutation.mutate(values))}>
        <DialogTitle>{isEdit ? "Edit connection" : "Create connection"}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, sm: 6 }}>
                <TextField
                  autoFocus
                  label="Name"
                  error={Boolean(errors.name)}
                  helperText={errors.name?.message}
                  fullWidth
                  {...register("name", { required: "Name is required" })}
                />
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <Controller
                  control={control}
                  name="database"
                  render={({ field }) => (
                    <FormControl fullWidth>
                      <InputLabel id="database-label">Database</InputLabel>
                      <Select {...field} label="Database" labelId="database-label">
                        {databaseKinds.map((kind) => (
                          <MenuItem key={kind} value={kind}>
                            {kind}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  )}
                />
              </Grid>
              <Grid size={{ xs: 12 }}>
                <TextField
                  label="DSN"
                  type="password"
                  fullWidth
                  error={Boolean(errors.dsn)}
                  helperText={
                    errors.dsn?.message ??
                    (isEdit
                      ? "Leave blank to preserve the stored DSN."
                      : "DSNs are encrypted server-side and never returned.")
                  }
                  {...register("dsn", {
                    validate: (value) =>
                      isEdit || value.trim() ? true : "DSN is required",
                  })}
                />
              </Grid>
              {isEdit ? (
                <Grid size={{ xs: 12 }}>
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
                </Grid>
              ) : null}
            </Grid>

            <MetadataFields control={control} register={register} />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button disabled={mutation.isPending} type="submit" variant="contained">
            {isEdit ? "Save" : "Create"}
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}

function MetadataFields({
  control,
  register,
}: {
  control: ReturnType<typeof useForm<ConnectionFormValues>>["control"];
  register: ReturnType<typeof useForm<ConnectionFormValues>>["register"];
}) {
  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Stack spacing={2}>
        <Stack spacing={0.5}>
          <Typography component="h2" variant="h2">
            Metadata filters
          </Typography>
          <Typography color="text.secondary" variant="body2">
            Use comma-separated values. Table maps use one namespace per line:
            `public: users, orders`.
          </Typography>
        </Stack>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <TextField
              label="Include namespaces"
              fullWidth
              {...register("include_namespaces")}
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <TextField
              label="Exclude namespaces"
              fullWidth
              {...register("exclude_namespaces")}
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <TextField
              label="Include tables by namespace"
              multiline
              minRows={3}
              fullWidth
              {...register("include_tables_by_namespace")}
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <TextField
              label="Exclude tables by namespace"
              multiline
              minRows={3}
              fullWidth
              {...register("exclude_tables_by_namespace")}
            />
          </Grid>
          {[
            ["include_views", "Include views"],
            ["include_indexes", "Include indexes"],
            ["include_foreign_keys", "Include foreign keys"],
            ["include_comments", "Include comments"],
          ].map(([name, label]) => (
            <Grid key={name} size={{ xs: 12, sm: 6 }}>
              <Controller
                control={control}
                name={name as keyof ConnectionFormValues}
                render={({ field }) => (
                  <FormControlLabel
                    control={
                      <Switch
                        checked={Boolean(field.value)}
                        onChange={(event) => field.onChange(event.target.checked)}
                      />
                    }
                    label={label}
                  />
                )}
              />
            </Grid>
          ))}
        </Grid>
      </Stack>
    </Paper>
  );
}

function AccessGrantDialog({
  connection,
  open,
  onClose,
}: {
  connection: Connection | null;
  open: boolean;
  onClose: () => void;
}) {
  const { apiClient } = useAuth();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    control,
    formState: { errors },
    handleSubmit,
    reset,
  } = useForm<AccessFormValues>({
    defaultValues: {
      user_id: "",
      can_query: true,
      allow_writes: false,
      can_manage: false,
    },
  });

  const usersQuery = useQuery({
    queryKey: ["users", "connection-access"],
    queryFn: () => apiClient.get<User[]>("/users"),
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: (values: AccessFormValues) =>
      apiClient.post<ConnectionAccessGrant>(
        `/connections/${connection?.id}/access`,
        {
          user_id: Number(values.user_id),
          can_query: values.can_query,
          allow_writes: values.allow_writes,
          can_manage: values.can_manage,
        },
      ),
    onSuccess() {
      setSubmitError(null);
      reset();
      onClose();
    },
    onError(error) {
      setSubmitError(errorMessage(error));
    },
  });

  const close = () => {
    if (!mutation.isPending) {
      setSubmitError(null);
      reset();
      onClose();
    }
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={open} onClose={close}>
      <Box component="form" onSubmit={handleSubmit((values) => mutation.mutate(values))}>
        <DialogTitle>Grant connection access</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Typography color="text.secondary" variant="body2">
              {connection?.name}
            </Typography>
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            {usersQuery.isError ? (
              <Alert severity="error">{errorMessage(usersQuery.error)}</Alert>
            ) : null}
            <Controller
              control={control}
              name="user_id"
              rules={{ required: "User is required" }}
              render={({ field }) => (
                <FormControl fullWidth error={Boolean(errors.user_id)}>
                  <InputLabel id="grant-user-label">User</InputLabel>
                  <Select
                    {...field}
                    label="User"
                    labelId="grant-user-label"
                    disabled={usersQuery.isLoading}
                  >
                    {(usersQuery.data ?? []).map((user) => (
                      <MenuItem key={user.id} value={String(user.id)}>
                        {user.name} ({user.email})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            />
            {[
              ["can_query", "Can query"],
              ["allow_writes", "Allow writes"],
              ["can_manage", "Can manage"],
            ].map(([name, label]) => (
              <Controller
                key={name}
                control={control}
                name={name as keyof AccessFormValues}
                render={({ field }) => (
                  <FormControlLabel
                    control={
                      <Switch
                        checked={Boolean(field.value)}
                        onChange={(event) => field.onChange(event.target.checked)}
                      />
                    }
                    label={label}
                  />
                )}
              />
            ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button disabled={mutation.isPending} type="submit" variant="contained">
            Grant access
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}

function formValuesFromConnection(
  connection?: Connection | null,
): ConnectionFormValues {
  const metadata = connection?.metadata;
  return {
    name: connection?.name ?? "",
    database: connection?.database ?? "postgres",
    dsn: "",
    is_enabled: connection?.is_enabled ?? true,
    include_namespaces: formatList(metadata?.include_namespaces),
    exclude_namespaces: formatList(metadata?.exclude_namespaces),
    include_tables_by_namespace: formatMap(metadata?.include_tables_by_namespace),
    exclude_tables_by_namespace: formatMap(metadata?.exclude_tables_by_namespace),
    include_views: Boolean(metadata?.include_views),
    include_indexes: Boolean(metadata?.include_indexes),
    include_foreign_keys: Boolean(metadata?.include_foreign_keys),
    include_comments: Boolean(metadata?.include_comments),
  };
}

function payloadFromConnectionForm(values: ConnectionFormValues, isEdit: boolean) {
  const payload: {
    name: string;
    database: DatabaseKind;
    dsn?: string;
    is_enabled?: boolean;
    metadata: ConnectionMetadata;
  } = {
    name: values.name.trim(),
    database: values.database,
    metadata: {
      include_namespaces: parseList(values.include_namespaces),
      exclude_namespaces: parseList(values.exclude_namespaces),
      include_tables_by_namespace: parseMap(values.include_tables_by_namespace),
      exclude_tables_by_namespace: parseMap(values.exclude_tables_by_namespace),
      include_views: values.include_views,
      include_indexes: values.include_indexes,
      include_foreign_keys: values.include_foreign_keys,
      include_comments: values.include_comments,
    },
  };

  const dsn = values.dsn.trim();
  if (dsn) {
    payload.dsn = dsn;
  }
  if (isEdit) {
    payload.is_enabled = values.is_enabled;
  }

  return payload;
}

function parseList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatList(value?: string[] | null) {
  return value?.join(", ") ?? "";
}

function parseMap(value: string) {
  return value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .reduce<Record<string, string[]>>((acc, line) => {
      const [namespace, tables] = line.split(":");
      if (!namespace || !tables) {
        return acc;
      }
      acc[namespace.trim()] = parseList(tables);
      return acc;
    }, {});
}

function formatMap(value?: Record<string, string[]> | null) {
  if (!value) {
    return "";
  }

  return Object.entries(value)
    .map(([namespace, tables]) => `${namespace}: ${tables.join(", ")}`)
    .join("\n");
}
