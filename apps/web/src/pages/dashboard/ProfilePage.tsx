import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Divider from "@mui/material/Divider";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { ErrorState } from "../../components/common/ErrorState";
import { LoadingState } from "../../components/common/LoadingState";
import { SecretTextField } from "../../components/common/SecretTextField";
import { useAuth } from "../../auth/AuthProvider";
import type { User } from "../../types/api";
import { errorMessage } from "../../utils/errors";

type PasswordForm = {
  current_password: string;
  new_password: string;
};

export function ProfilePage() {
  const { apiClient, session, setSession } = useAuth();
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    formState: { errors },
    handleSubmit,
    register,
    reset,
  } = useForm<PasswordForm>({
    defaultValues: { current_password: "", new_password: "" },
  });

  const meQuery = useQuery({
    queryKey: ["me", session?.user.id, session?.tokens.access_token],
    queryFn: () => apiClient.get<User>("/users/me"),
    enabled: Boolean(session),
  });

  const passwordMutation = useMutation({
    mutationFn: (values: PasswordForm) =>
      apiClient.post<User>("/users/me/password", values),
    onSuccess(user) {
      if (session) {
        setSession({ ...session, user });
      }
      setSuccessMessage("Password changed");
      setSubmitError(null);
      reset();
    },
    onError(error) {
      setSuccessMessage(null);
      setSubmitError(errorMessage(error));
    },
  });

  const onSubmit = handleSubmit((values) => {
    passwordMutation.mutate(values);
  });

  if (meQuery.isLoading) {
    return <LoadingState label="Loading profile" />;
  }

  if (meQuery.isError) {
    return (
      <ErrorState
        message={errorMessage(meQuery.error)}
        onRetry={() => void meQuery.refetch()}
        title="Could not load profile"
      />
    );
  }

  const user = meQuery.data;

  return (
    <Stack spacing={3} sx={{ maxWidth: 760 }}>
      <Stack spacing={0.5}>
        <Typography variant="h1">Profile</Typography>
        <Typography color="text.secondary">
          Manage your signed-in account and password.
        </Typography>
      </Stack>

      <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
        <Stack spacing={2}>
          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={1}
            alignItems={{ xs: "flex-start", sm: "center" }}
            justifyContent="space-between"
          >
            <Stack spacing={0.25}>
              <Typography fontWeight={800}>{user?.name}</Typography>
              <Typography color="text.secondary" variant="body2">
                {user?.email}
              </Typography>
            </Stack>
            <Stack direction="row" spacing={1}>
              <Chip label={user?.role} size="small" color="primary" />
              {user?.is_active === false ? (
                <Chip label="inactive" size="small" color="error" />
              ) : null}
            </Stack>
          </Stack>

          <Divider />

          <Stack
            component="form"
            id="change-password"
            spacing={2}
            onSubmit={onSubmit}
            noValidate
          >
            <Stack spacing={0.5}>
              <Typography component="h2" variant="h2">
                Change password
              </Typography>
              <Typography color="text.secondary" variant="body2">
                Enter your current password and a new password.
              </Typography>
            </Stack>
            {successMessage ? <Alert severity="success">{successMessage}</Alert> : null}
            {submitError ? <Alert severity="error">{submitError}</Alert> : null}
            <SecretTextField
              autoComplete="current-password"
              error={Boolean(errors.current_password)}
              helperText={errors.current_password?.message}
              label="Current password"
              fullWidth
              {...register("current_password", {
                required: "Current password is required",
              })}
            />
            <SecretTextField
              autoComplete="new-password"
              error={Boolean(errors.new_password)}
              helperText={errors.new_password?.message}
              label="New password"
              fullWidth
              {...register("new_password", {
                required: "New password is required",
              })}
            />
            <Button
              disabled={passwordMutation.isPending}
              type="submit"
              variant="contained"
              sx={{ alignSelf: { sm: "flex-start" } }}
            >
              Change password
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Stack>
  );
}
