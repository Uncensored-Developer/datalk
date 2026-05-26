import AddOutlinedIcon from "@mui/icons-material/AddOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import CircularProgress from "@mui/material/CircularProgress";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
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
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import { EmptyState } from "../../components/common/EmptyState";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import { SecretTextField } from "../../components/common/SecretTextField";
import type { Connection, ConnectionAccessGrant, User, UserRole } from "../../types/api";
import { errorMessage } from "../../utils/errors";

const roles: UserRole[] = ["owner", "admin", "member"];

type CreateUserForm = {
  name: string;
  email: string;
  password: string;
  role: UserRole;
  connection_ids: string[];
  can_query: boolean;
  allow_writes: boolean;
  can_manage: boolean;
};

type EditUserForm = {
  name: string;
  role: UserRole;
  is_active: boolean;
};

export function UsersPage() {
  const { apiClient } = useAuth();
  const [createOpen, setCreateOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editDialogKey, setEditDialogKey] = useState(0);

  const usersQuery = useQuery({
    queryKey: ["users"],
    queryFn: () => apiClient.get<User[]>("/users"),
  });

  if (usersQuery.isLoading) {
    return <LoadingState label="Loading users" />;
  }

  if (usersQuery.isError) {
    const isForbidden =
      usersQuery.error instanceof ApiError && usersQuery.error.status === 403;

    return (
      <Stack spacing={2}>
        <Stack spacing={0.5}>
          <Typography variant="h1">Users</Typography>
          <Typography color="text.secondary">
            Manage who can access this Datalk installation.
          </Typography>
        </Stack>
        <ErrorState
          title={isForbidden ? "Admin access required" : "Could not load users"}
          message={
            isForbidden
              ? "Only owners and admins can list and manage users."
              : errorMessage(usersQuery.error)
          }
          onRetry={() => void usersQuery.refetch()}
        />
      </Stack>
    );
  }

  const users = usersQuery.data ?? [];

  return (
    <Stack spacing={3}>
      <Stack
        direction={{ xs: "column", sm: "row" }}
        spacing={2}
        alignItems={{ xs: "stretch", sm: "center" }}
        justifyContent="space-between"
      >
        <Stack spacing={0.5}>
          <Typography variant="h1">Users</Typography>
          <Typography color="text.secondary">
            Create accounts and update user status or role.
          </Typography>
        </Stack>
        <Button
          startIcon={<AddOutlinedIcon />}
          variant="contained"
          onClick={() => setCreateOpen(true)}
        >
          Create user
        </Button>
      </Stack>

      {users.length === 0 ? (
        <EmptyState
          title="No users found"
          description="Create the first managed user account."
          action={
            <Button variant="contained" onClick={() => setCreateOpen(true)}>
              Create user
            </Button>
          }
        />
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table sx={{ minWidth: 760 }}>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Email</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Password</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.id} hover>
                  <TableCell>
                    <Typography fontWeight={700}>{user.name}</Typography>
                  </TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>
                    <RoleChip role={user.role} />
                  </TableCell>
                  <TableCell>
                    {user.is_active === false ? (
                      <Chip label="inactive" size="small" color="error" />
                    ) : (
                      <Chip label="active" size="small" color="success" />
                    )}
                  </TableCell>
                  <TableCell>
                    {user.must_change_password ? (
                      <Chip label="change required" size="small" color="warning" />
                    ) : (
                      <Chip label="current" size="small" />
                    )}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="Edit user">
                      <IconButton
                        aria-label={`Edit ${user.name}`}
                        onClick={() => {
                          setEditDialogKey((key) => key + 1);
                          setEditingUser(user);
                        }}
                      >
                        <EditOutlinedIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <CreateUserDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      <EditUserDialog
        key={editDialogKey}
        user={editingUser}
        onClose={() => setEditingUser(null)}
      />
    </Stack>
  );
}

function RoleChip({ role }: { role: UserRole }) {
  const color = role === "owner" ? "secondary" : role === "admin" ? "primary" : "default";

  return <Chip label={role} color={color} size="small" />;
}

function CreateUserDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<CreateUserForm>({
    defaultValues: {
      name: "",
      email: "",
      password: "",
      role: "member",
      connection_ids: [],
      can_query: true,
      allow_writes: false,
      can_manage: false,
    },
  });
  const connectionsQuery = useQuery({
    queryKey: ["connections", "create-user-access"],
    queryFn: () => apiClient.get<Connection[]>("/connections"),
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: async (values: CreateUserForm) => {
      const user = await apiClient.post<User>("/users", {
        name: values.name,
        email: values.email,
        password: values.password,
        role: values.role,
      });

      if (values.connection_ids.length > 0) {
        await Promise.all(
          values.connection_ids.map((connectionID) =>
            apiClient.post<ConnectionAccessGrant>(
              `/connections/${connectionID}/access`,
              {
                user_id: user.id,
                can_query: values.can_query,
                allow_writes: values.allow_writes,
                can_manage: values.can_manage,
              },
            ),
          ),
        );
      }

      return user;
    },
    onSuccess() {
      setSubmitError(null);
      reset();
      onClose();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      void queryClient.invalidateQueries({ queryKey: ["connection-access"] });
    },
    onError(error) {
      setSubmitError(errorMessage(error));
    },
  });

  const close = () => {
    if (!createMutation.isPending) {
      setSubmitError(null);
      reset();
      onClose();
    }
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={open} onClose={close}>
      <Box component="form" onSubmit={handleSubmit((values) => createMutation.mutate(values))}>
        <DialogTitle>Create user</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            <TextField
              autoFocus
              label="Name"
              error={Boolean(errors.name)}
              helperText={errors.name?.message}
              fullWidth
              {...register("name", { required: "Name is required" })}
            />
            <TextField
              label="Email"
              type="email"
              error={Boolean(errors.email)}
              helperText={errors.email?.message}
              fullWidth
              {...register("email", { required: "Email is required" })}
            />
            <SecretTextField
              label="Temporary password"
              error={Boolean(errors.password)}
              helperText={errors.password?.message}
              fullWidth
              {...register("password", { required: "Password is required" })}
            />
            <Controller
              control={control}
              name="role"
              render={({ field }) => (
                <FormControl fullWidth>
                  <InputLabel id="create-role-label">Role</InputLabel>
                  <Select {...field} label="Role" labelId="create-role-label">
                    {roles.map((role) => (
                      <MenuItem key={role} value={role}>
                        {role}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            />
            {connectionsQuery.isLoading ? (
              <LoadingState label="Loading connections" />
            ) : null}
            {connectionsQuery.isError ? (
              <Alert severity="error">{errorMessage(connectionsQuery.error)}</Alert>
            ) : null}
            {!connectionsQuery.isLoading &&
            !connectionsQuery.isError &&
            (connectionsQuery.data ?? []).length > 0 ? (
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Stack spacing={2}>
                  <Stack spacing={0.5}>
                    <Typography component="h3" fontWeight={800}>
                      Connection access
                    </Typography>
                    <Typography color="text.secondary" variant="body2">
                      Grant access after the user account is created.
                    </Typography>
                  </Stack>
                  <Controller
                    control={control}
                    name="connection_ids"
                    render={({ field }) => (
                      <FormControl fullWidth>
                        <InputLabel id="create-user-connections-label">
                          Connections
                        </InputLabel>
                        <Select
                          {...field}
                          label="Connections"
                          labelId="create-user-connections-label"
                          multiple
                          renderValue={(selected) =>
                            selected
                              .map((id) => {
                                const connection = connectionsQuery.data?.find(
                                  (item) => String(item.id) === id,
                                );
                                return connection?.name ?? id;
                              })
                              .join(", ")
                          }
                        >
                          {(connectionsQuery.data ?? []).map((connection) => (
                            <MenuItem key={connection.id} value={String(connection.id)}>
                              {connection.name}
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
                      name={name as keyof CreateUserForm}
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
              </Paper>
            ) : null}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button
            disabled={createMutation.isPending}
            startIcon={createMutation.isPending ? <CircularProgress color="inherit" size={16} /> : undefined}
            type="submit"
            variant="contained"
          >
            Create
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}

function EditUserDialog({
  user,
  onClose,
}: {
  user: User | null;
  onClose: () => void;
}) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    control,
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<EditUserForm>({
    defaultValues: {
      name: user?.name ?? "",
      role: user?.role ?? "member",
      is_active: user?.is_active !== false,
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: EditUserForm) =>
      apiClient.put<User>(`/users/${user?.id}`, values),
    onSuccess() {
      setSubmitError(null);
      onClose();
      reset();
      void queryClient.invalidateQueries({ queryKey: ["users"] });
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
    onError(error) {
      setSubmitError(errorMessage(error));
    },
  });

  const close = () => {
    if (!updateMutation.isPending) {
      setSubmitError(null);
      onClose();
    }
  };

  return (
    <Dialog fullWidth maxWidth="sm" open={Boolean(user)} onClose={close}>
      <Box component="form" onSubmit={handleSubmit((values) => updateMutation.mutate(values))}>
        <DialogTitle>Edit user</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            <TextField
              autoFocus
              label="Name"
              error={Boolean(errors.name)}
              helperText={errors.name?.message}
              fullWidth
              {...register("name", { required: "Name is required" })}
            />
            <Controller
              control={control}
              name="role"
              render={({ field }) => (
                <FormControl fullWidth>
                  <InputLabel id="edit-role-label">Role</InputLabel>
                  <Select {...field} label="Role" labelId="edit-role-label">
                    {roles.map((role) => (
                      <MenuItem key={role} value={role}>
                        {role}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              )}
            />
            <Controller
              control={control}
              name="is_active"
              render={({ field }) => (
                <FormControlLabel
                  control={
                    <Switch
                      checked={field.value}
                      onChange={(event) => field.onChange(event.target.checked)}
                    />
                  }
                  label="Active user"
                />
              )}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={close}>Cancel</Button>
          <Button disabled={updateMutation.isPending} type="submit" variant="contained">
            Save
          </Button>
        </DialogActions>
      </Box>
    </Dialog>
  );
}
