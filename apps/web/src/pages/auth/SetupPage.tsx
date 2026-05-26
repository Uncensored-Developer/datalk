import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import { SecretTextField } from "../../components/common/SecretTextField";
import type { AuthSession } from "../../types/api";
import { errorMessage } from "../../utils/errors";
import { AuthLayout } from "./AuthLayout";

type SetupForm = {
  name: string;
  email: string;
  password: string;
};

export function SetupPage() {
  const { apiClient, setSession } = useAuth();
  const navigate = useNavigate();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [setupUnavailable, setSetupUnavailable] = useState(false);
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<SetupForm>({
    defaultValues: { name: "", email: "", password: "" },
  });

  const onSubmit = handleSubmit(async (values) => {
    setSubmitError(null);
    setSetupUnavailable(false);
    try {
      const session = await apiClient.post<AuthSession>("/auth/setup", values, {
        auth: false,
      });
      setSession(session);
      navigate("/connections?onboarding=1", { replace: true });
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        setSetupUnavailable(true);
      }
      setSubmitError(errorMessage(error));
    }
  });

  return (
    <AuthLayout
      title="Set up Datalk"
      subtitle="Create the first owner account for this installation."
    >
      <Stack component="form" spacing={2} onSubmit={onSubmit} noValidate>
        {submitError ? (
          <Alert
            severity={setupUnavailable ? "info" : "error"}
            action={
              setupUnavailable ? (
                <Button component={RouterLink} to="/login" size="small">
                  Sign in
                </Button>
              ) : null
            }
          >
            {submitError}
          </Alert>
        ) : null}
        <TextField
          autoComplete="name"
          autoFocus
          error={Boolean(errors.name)}
          helperText={errors.name?.message}
          label="Name"
          fullWidth
          {...register("name", { required: "Name is required" })}
        />
        <TextField
          autoComplete="email"
          error={Boolean(errors.email)}
          helperText={errors.email?.message}
          label="Email"
          type="email"
          fullWidth
          {...register("email", { required: "Email is required" })}
        />
        <SecretTextField
          autoComplete="new-password"
          error={Boolean(errors.password)}
          helperText={errors.password?.message}
          label="Password"
          fullWidth
          {...register("password", { required: "Password is required" })}
        />
        <Button
          disabled={isSubmitting}
          startIcon={isSubmitting ? <CircularProgress color="inherit" size={16} /> : undefined}
          type="submit"
          variant="contained"
          fullWidth
        >
          Create owner account
        </Button>
        <Typography color="text.secondary" variant="body2">
          Already set up?{" "}
          <Link component={RouterLink} to="/login">
            Sign in
          </Link>
        </Typography>
      </Stack>
    </AuthLayout>
  );
}
