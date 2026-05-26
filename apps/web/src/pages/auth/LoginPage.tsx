import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { SecretTextField } from "../../components/common/SecretTextField";
import type { AuthSession, SetupStatus } from "../../types/api";
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
  const [checkingSetup, setCheckingSetup] = useState(true);
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register,
  } = useForm<LoginForm>({
    defaultValues: { email: "", password: "" },
  });

  useEffect(() => {
    let isActive = true;

    apiClient
      .get<SetupStatus>("/auth/setup/status", { auth: false })
      .then((status) => {
        if (isActive && status.setup_required) {
          navigate("/setup", { replace: true });
        }
      })
      .catch(() => {
        // Keep login available if the status probe fails.
      })
      .finally(() => {
        if (isActive) {
          setCheckingSetup(false);
        }
      });

    return () => {
      isActive = false;
    };
  }, [apiClient, navigate]);

  const onSubmit = handleSubmit(async (values) => {
    setSubmitError(null);
    try {
      const session = await apiClient.post<AuthSession>("/auth/login", values, {
        auth: false,
      });
      setSession(session);
      if (session.user.must_change_password || session.must_change_password) {
        navigate("/profile#change-password", { replace: true });
        return;
      }

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
        <SecretTextField
          autoComplete="current-password"
          error={Boolean(errors.password)}
          helperText={errors.password?.message}
          label="Password"
          fullWidth
          {...register("password", { required: "Password is required" })}
        />
        <Button
          disabled={checkingSetup || isSubmitting}
          startIcon={isSubmitting ? <CircularProgress color="inherit" size={16} /> : undefined}
          type="submit"
          variant="contained"
          fullWidth
        >
          {checkingSetup ? "Checking setup" : "Sign in"}
        </Button>
      </Stack>
    </AuthLayout>
  );
}
