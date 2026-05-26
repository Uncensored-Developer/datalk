import { createBrowserRouter } from "react-router-dom";
import { LoginPage } from "./pages/auth/LoginPage";
import { SetupPage } from "./pages/auth/SetupPage";
import { OverviewPage } from "./pages/dashboard/OverviewPage";
import { PlaceholderPage } from "./pages/dashboard/PlaceholderPage";
import { ProfilePage } from "./pages/dashboard/ProfilePage";
import { DashboardLayout } from "./routes/DashboardLayout";
import { ProtectedRoute } from "./routes/ProtectedRoute";
import { PublicOnlyRoute } from "./routes/PublicOnlyRoute";

export const router = createBrowserRouter([
  {
    element: <PublicOnlyRoute />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/setup", element: <SetupPage /> },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <DashboardLayout />,
        children: [
          { path: "/", element: <OverviewPage /> },
          { path: "/profile", element: <ProfilePage /> },
          { path: "/chat", element: <PlaceholderPage title="Chat" /> },
          {
            path: "/connections",
            element: <PlaceholderPage title="Connections" />,
          },
          { path: "/users", element: <PlaceholderPage title="Users" /> },
          {
            path: "/provider-configs",
            element: <PlaceholderPage title="Provider Configs" />,
          },
          { path: "*", element: <OverviewPage /> },
        ],
      },
    ],
  },
]);
