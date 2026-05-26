import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppProviders } from "../../providers/AppProviders";
import { AdminRoute } from "../../routes/AdminRoute";
import { DashboardLayout } from "../../routes/DashboardLayout";
import { ProtectedRoute } from "../../routes/ProtectedRoute";
import { UsersPage } from "./UsersPage";

const ownerSession = {
  user: {
    id: 1,
    email: "owner@example.com",
    name: "Owner User",
    role: "owner",
    must_change_password: false,
  },
  tokens: {
    access_token: "owner-access-token",
    refresh_token: "owner-refresh-token",
    expires_at: "2026-05-25T12:00:00Z",
  },
  must_change_password: false,
};

const memberSession = {
  ...ownerSession,
  user: {
    ...ownerSession.user,
    id: 2,
    email: "member@example.com",
    name: "Member User",
    role: "member",
  },
};

function renderUsersRoute() {
  const router = createMemoryRouter(
    [
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <DashboardLayout />,
            children: [
              {
                element: <AdminRoute />,
                children: [{ path: "/users", element: <UsersPage /> }],
              },
            ],
          },
        ],
      },
    ],
    { initialEntries: ["/users"] },
  );

  render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("UsersPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("restricts the route for non-admin users", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(memberSession));

    renderUsersRoute();

    expect(
      await screen.findByRole("heading", { name: "Access restricted" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create user" })).not.toBeInTheDocument();
  });

  it("lists users for admins", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    vi.spyOn(window, "fetch").mockResolvedValueOnce(
      jsonResponse([
        {
          id: 1,
          email: "owner@example.com",
          name: "Owner User",
          role: "owner",
          is_active: true,
          must_change_password: false,
        },
        {
          id: 3,
          email: "analyst@example.com",
          name: "Analyst User",
          role: "member",
          is_active: false,
          must_change_password: true,
        },
      ]),
    );

    renderUsersRoute();

    expect(await screen.findByText("Analyst User")).toBeInTheDocument();
    expect(screen.getAllByText("Owner User")).not.toHaveLength(0);
    expect(screen.getByText("analyst@example.com")).toBeInTheDocument();
  });

  it("creates a user through the admin API", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    const fetchMock = vi
      .spyOn(window, "fetch")
      .mockResolvedValueOnce(jsonResponse([]))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            id: 4,
            email: "new@example.com",
            name: "New User",
            role: "member",
            is_active: true,
            must_change_password: true,
          },
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(jsonResponse([]));

    renderUsersRoute();

    const createButtons = await screen.findAllByRole("button", {
      name: "Create user",
    });
    await userEvent.click(createButtons[0]);
    await userEvent.type(screen.getByLabelText("Name"), "New User");
    await userEvent.type(screen.getByLabelText("Email"), "new@example.com");
    await userEvent.type(screen.getByLabelText("Temporary password"), "temporary");
    await userEvent.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/users",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            name: "New User",
            email: "new@example.com",
            password: "temporary",
            role: "member",
          }),
        }),
      );
    });
  });

  it("updates only editable user fields", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    const fetchMock = vi
      .spyOn(window, "fetch")
      .mockResolvedValueOnce(
        jsonResponse([
          {
            id: 3,
            email: "analyst@example.com",
            name: "Analyst User",
            role: "member",
            is_active: true,
            must_change_password: false,
          },
        ]),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: 3,
          email: "analyst@example.com",
          name: "Updated Analyst",
          role: "admin",
          is_active: false,
          must_change_password: false,
        }),
      )
      .mockResolvedValueOnce(jsonResponse([]));

    renderUsersRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Edit Analyst User" }));
    await userEvent.clear(screen.getByLabelText("Name"));
    await userEvent.type(screen.getByLabelText("Name"), "Updated Analyst");
    await userEvent.click(screen.getByLabelText("Role"));
    await userEvent.click(await screen.findByRole("option", { name: "admin" }));
    await userEvent.click(screen.getByLabelText("Active user"));
    await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/users/3",
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({
            name: "Updated Analyst",
            role: "admin",
            is_active: false,
          }),
        }),
      );
    });
  });
});

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}
