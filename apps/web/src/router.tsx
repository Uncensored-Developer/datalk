import { createBrowserRouter } from "react-router-dom";
import { LoginPage } from "./pages/auth/LoginPage";
import { SetupPage } from "./pages/auth/SetupPage";
import { OverviewPage } from "./pages/dashboard/OverviewPage";
import { PlaceholderPage } from "./pages/dashboard/PlaceholderPage";
import { ProfilePage } from "./pages/dashboard/ProfilePage";
import { ConnectionsPage } from "./pages/connections/ConnectionsPage";
import { ProviderConfigsPage } from "./pages/providerConfigs/ProviderConfigsPage";
import { UsersPage } from "./pages/users/UsersPage";
import { AdminRoute } from "./routes/AdminRoute";
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
            element: <ConnectionsPage />,
          },
          {
            element: <AdminRoute />,
            children: [{ path: "/users", element: <UsersPage /> }],
          },
          {
            element: <AdminRoute />,
            children: [
              {
                path: "/provider-configs",
                element: <ProviderConfigsPage />,
              },
            ],
          },
          { path: "*", element: <OverviewPage /> },
        ],
      },
    ],
  },
]);
