import { Outlet, useLocation } from "react-router-dom";
import { AppShell } from "../components/layout/AppShell";

const titlesByPath: Record<string, string> = {
  "/": "Start",
  "/profile": "Profile",
  "/chat": "Chat",
  "/connections": "Connections",
  "/users": "Users",
  "/provider-configs": "Provider Configs",
};

export function DashboardLayout() {
  const location = useLocation();
  const title =
    location.pathname.startsWith("/chat/")
      ? "Chat"
      : titlesByPath[location.pathname] ?? "Datalk";

  return (
    <AppShell title={title}>
      <Outlet />
    </AppShell>
  );
}
