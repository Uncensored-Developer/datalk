import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link as RouterLink, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import type { AuthSession } from "../../types/api";
import { errorMessage } from "../../utils/errors";
import { AuthLayout } from "./AuthLayout";

type LoginForm = {
  email: string;
  password: string;
};

type LocationState = {
  from?: {
    pathname?: string;
  };
};

export function LoginPage() {
  const { apiClient, setSession } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<LoginForm>({
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = handleSubmit(async (values) => {
    setSubmitError(null);
    try {
      const session = await apiClient.post<AuthSession>("/auth/login", values, {
        auth: false,
      });
      setSession(session);
      const state = location.state as LocationState | null;
      navigate(state?.from?.pathname ?? "/", { replace: true });
    } catch (error) {
      setSubmitError(errorMessage(error));
    }
  });

  return (
    <AuthLayout
      title="Sign in"
      subtitle="Use your Datalk account to continue."
    >
      <Stack component="form" spacing={2} onSubmit={onSubmit} noValidate>
        {submitError ? <Alert severity="error">{submitError}</Alert> : null}
        <TextField
          autoComplete="email"
          autoFocus
          error={Boolean(errors.email)}
          helperText={errors.email?.message}
          label="Email"
          type="email"
          fullWidth
          {...register("email", { required: "Email is required" })}
        />
        <TextField
          autoComplete="current-password"
          error={Boolean(errors.password)}
          helperText={errors.password?.message}
          label="Password"
          type="password"
          fullWidth
          {...register("password", { required: "Password is required" })}
        />
        <Button disabled={isSubmitting} type="submit" variant="contained" fullWidth>
          Sign in
        </Button>
        <Typography color="text.secondary" variant="body2">
          First time here?{" "}
          <Link component={RouterLink} to="/setup">
            Set up the first owner
          </Link>
        </Typography>
      </Stack>
    </AuthLayout>
  );
}
