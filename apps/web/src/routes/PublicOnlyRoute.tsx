import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../auth/AuthProvider";

export function PublicOnlyRoute() {
  const { isAuthenticated } = useAuth();
  const location = useLocation();

  if (isAuthenticated) {
    if (location.pathname === "/setup") {
      return <Navigate to="/connections?onboarding=1" replace />;
    }

    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
