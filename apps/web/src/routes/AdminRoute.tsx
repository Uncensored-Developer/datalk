import Alert from "@mui/material/Alert";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthProvider";

export function AdminRoute() {
  const { user } = useAuth();
  const isAdmin = user?.role === "owner" || user?.role === "admin";

  if (!isAdmin) {
    return (
      <Stack spacing={2} sx={{ maxWidth: 720 }}>
        <Typography variant="h1">Access restricted</Typography>
        <Alert severity="warning">
          This area is available only to owners and admins.
        </Alert>
      </Stack>
    );
  }

  return <Outlet />;
}
