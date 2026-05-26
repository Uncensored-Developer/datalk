import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppProviders } from "../../providers/AppProviders";
import { DashboardLayout } from "../../routes/DashboardLayout";
import { ProtectedRoute } from "../../routes/ProtectedRoute";
import { ConnectionsPage } from "./ConnectionsPage";

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

const warehouseConnection = {
  id: 10,
  name: "Warehouse",
  database: "postgres",
  user_id: 1,
  is_enabled: true,
  metadata: {
    include_namespaces: ["public"],
    exclude_namespaces: ["information_schema"],
    include_tables_by_namespace: { public: ["orders"] },
    exclude_tables_by_namespace: {},
    include_views: true,
    include_indexes: true,
    include_foreign_keys: false,
    include_comments: false,
  },
};

function renderConnectionsRoute() {
  const router = createMemoryRouter(
    [
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <DashboardLayout />,
            children: [{ path: "/connections", element: <ConnectionsPage /> }],
          },
        ],
      },
    ],
    { initialEntries: ["/connections"] },
  );

  render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("ConnectionsPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("lists member-visible connections without admin actions", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(memberSession));
    vi.spyOn(window, "fetch").mockResolvedValueOnce(
      jsonResponse([warehouseConnection]),
    );

    renderConnectionsRoute();

    expect(await screen.findByText("Warehouse")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create connection" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Edit Warehouse" })).not.toBeInTheDocument();
  });

  it("creates a connection with structured metadata", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    const fetchMock = vi
      .spyOn(window, "fetch")
      .mockResolvedValueOnce(jsonResponse([]))
      .mockResolvedValueOnce(jsonResponse(warehouseConnection, { status: 201 }))
      .mockResolvedValueOnce(jsonResponse([]));

    renderConnectionsRoute();

    const createButtons = await screen.findAllByRole("button", {
      name: "Create connection",
    });
    await userEvent.click(createButtons[0]);
    await userEvent.type(screen.getByLabelText("Name"), "Warehouse");
    await userEvent.type(screen.getByLabelText("DSN"), "postgres://user:pass@localhost/db");
    await userEvent.type(screen.getByLabelText("Include namespaces"), "public, analytics");
    await userEvent.type(screen.getByLabelText("Exclude namespaces"), "information_schema");
    await userEvent.type(
      screen.getByLabelText("Include tables by namespace"),
      "public: orders, customers",
    );
    await userEvent.click(screen.getByLabelText("Include views"));
    await userEvent.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/connections",
        expect.objectContaining({ method: "POST" }),
      );
    });
    const createCall = fetchMock.mock.calls.find(
      ([url, init]) => url === "/api/connections" && init?.method === "POST",
    );
    expect(JSON.parse(String(createCall?.[1]?.body))).toEqual({
      name: "Warehouse",
      database: "postgres",
      dsn: "postgres://user:pass@localhost/db",
      metadata: {
        include_namespaces: ["public", "analytics"],
        exclude_namespaces: ["information_schema"],
        include_tables_by_namespace: {
          public: ["orders", "customers"],
        },
        exclude_tables_by_namespace: {},
        include_views: true,
        include_indexes: false,
        include_foreign_keys: false,
        include_comments: false,
      },
    });
  });

  it("refreshes a connection schema", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(memberSession));
    const fetchMock = vi
      .spyOn(window, "fetch")
      .mockResolvedValueOnce(jsonResponse([warehouseConnection]))
      .mockResolvedValueOnce(
        jsonResponse({ connection_id: 10, status: "accepted" }, { status: 202 }),
      );

    renderConnectionsRoute();

    await userEvent.click(
      await screen.findByRole("button", {
        name: "Refresh schema for Warehouse",
      }),
    );

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/connections/10/schema-snapshot/refresh",
        expect.objectContaining({ method: "POST" }),
      );
    });
  });

  it("edits, grants access, and deletes through admin actions", async () => {
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    const fetchMock = vi
      .spyOn(window, "fetch")
      .mockResolvedValueOnce(jsonResponse([warehouseConnection]))
      .mockResolvedValueOnce(jsonResponse({ ...warehouseConnection, name: "Warehouse Primary" }))
      .mockResolvedValueOnce(jsonResponse([warehouseConnection]))
      .mockResolvedValueOnce(
        jsonResponse([
          {
            id: 2,
            email: "analyst@example.com",
            name: "Analyst",
            role: "member",
            is_active: true,
            must_change_password: false,
          },
        ]),
      )
      .mockResolvedValueOnce(
        jsonResponse(
          {
            user_id: 2,
            connection_id: 10,
            can_query: true,
            allow_writes: false,
            can_manage: false,
          },
          { status: 201 },
        ),
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(jsonResponse([]));

    renderConnectionsRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Edit Warehouse" }));
    await userEvent.clear(screen.getByLabelText("Name"));
    await userEvent.type(screen.getByLabelText("Name"), "Warehouse Primary");
    await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/connections/10",
        expect.objectContaining({
          method: "PUT",
          body: expect.stringContaining("Warehouse Primary"),
        }),
      );
    });

    await userEvent.click(await screen.findByRole("button", { name: "Grant access for Warehouse" }));
    await userEvent.click(await screen.findByLabelText("User"));
    await userEvent.click(await screen.findByRole("option", { name: "Analyst (analyst@example.com)" }));
    await userEvent.click(screen.getByRole("button", { name: "Grant access" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/connections/10/access",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            user_id: 2,
            can_query: true,
            allow_writes: false,
            can_manage: false,
          }),
        }),
      );
    });

    await userEvent.click(await screen.findByRole("button", { name: "Delete Warehouse" }));
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/connections/10",
        expect.objectContaining({ method: "DELETE" }),
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
