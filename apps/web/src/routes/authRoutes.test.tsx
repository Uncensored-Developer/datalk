import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppProviders } from "../providers/AppProviders";
import { DashboardLayout } from "./DashboardLayout";
import { ProtectedRoute } from "./ProtectedRoute";
import { PublicOnlyRoute } from "./PublicOnlyRoute";
import { LoginPage } from "../pages/auth/LoginPage";
import { OverviewPage } from "../pages/dashboard/OverviewPage";
import { ProfilePage } from "../pages/dashboard/ProfilePage";

function renderRouter(initialEntries: string[]) {
  const router = createMemoryRouter(
    [
      {
        element: <PublicOnlyRoute />,
        children: [{ path: "/login", element: <LoginPage /> }],
      },
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <DashboardLayout />,
            children: [
              { path: "/", element: <OverviewPage /> },
              { path: "/profile", element: <ProfilePage /> },
            ],
          },
        ],
      },
    ],
    { initialEntries },
  );

  render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("auth routes", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("redirects protected routes to login without a session", async () => {
    renderRouter(["/"]);

    expect(await screen.findByRole("heading", { name: "Sign in" })).toBeInTheDocument();
  });

  it("stores the returned session after login", async () => {
    vi.spyOn(window, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          user: {
            id: 1,
            email: "root@example.com",
            name: "Root User",
            role: "owner",
            must_change_password: false,
          },
          tokens: {
            access_token: "access-token",
            refresh_token: "refresh-token",
            expires_at: "2026-05-25T12:00:00Z",
          },
          must_change_password: false,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );

    renderRouter(["/login"]);

    await userEvent.type(screen.getByLabelText("Email"), "root@example.com");
    await userEvent.type(screen.getByLabelText("Password"), "secret");
    await userEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => {
      expect(window.localStorage.getItem("datalk.session")).toContain(
        "access-token",
      );
    });
  });

  it("does not refresh the session when logout receives a 401", async () => {
    window.localStorage.setItem(
      "datalk.session",
      JSON.stringify({
        user: {
          id: 1,
          email: "root@example.com",
          name: "Root User",
          role: "owner",
          must_change_password: false,
        },
        tokens: {
          access_token: "expired-access-token",
          refresh_token: "refresh-token",
          expires_at: "2026-05-25T12:00:00Z",
        },
        must_change_password: false,
      }),
    );
    const fetchMock = vi.spyOn(window, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: "unauthorized" }), {
        status: 401,
        headers: { "Content-Type": "application/json" },
      }),
    );

    renderRouter(["/"]);

    await userEvent.click(await screen.findByRole("button", { name: "Sign out" }));

    await waitFor(() => {
      expect(window.localStorage.getItem("datalk.session")).toBeNull();
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/auth/logout");
  });

  it("loads profile data with the active session instead of a previous user's cache", async () => {
    window.localStorage.setItem(
      "datalk.session",
      JSON.stringify({
        user: {
          id: 2,
          email: "second@example.com",
          name: "Second User",
          role: "member",
          must_change_password: false,
        },
        tokens: {
          access_token: "second-access-token",
          refresh_token: "second-refresh-token",
          expires_at: "2026-05-25T12:00:00Z",
        },
        must_change_password: false,
      }),
    );
    vi.spyOn(window, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: 2,
          email: "second@example.com",
          name: "Second User",
          role: "member",
          is_active: true,
          must_change_password: false,
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );

    renderRouter(["/profile"]);

    expect(await screen.findAllByText("Second User")).not.toHaveLength(0);
    expect(screen.queryByText("Root User")).not.toBeInTheDocument();
  });
});
