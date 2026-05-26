import { Outlet, useLocation } from "react-router-dom";
import { AppShell } from "../components/layout/AppShell";

const titlesByPath: Record<string, string> = {
  "/": "Overview",
  "/profile": "Profile",
  "/chat": "Chat",
  "/connections": "Connections",
  "/users": "Users",
  "/provider-configs": "Provider Configs",
};

export function DashboardLayout() {
  const location = useLocation();

  return (
    <AppShell title={titlesByPath[location.pathname] ?? "Datalk"}>
      <Outlet />
    </AppShell>
  );
}
