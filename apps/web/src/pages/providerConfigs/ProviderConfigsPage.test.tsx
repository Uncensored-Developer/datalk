import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppProviders } from "../../providers/AppProviders";
import { AdminRoute } from "../../routes/AdminRoute";
import { DashboardLayout } from "../../routes/DashboardLayout";
import { ProtectedRoute } from "../../routes/ProtectedRoute";
import { ProviderConfigsPage } from "./ProviderConfigsPage";

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

function renderProviderConfigsRoute() {
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
                children: [
                  {
                    path: "/provider-configs",
                    element: <ProviderConfigsPage />,
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
    { initialEntries: ["/provider-configs"] },
  );

  render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("ProviderConfigsPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.localStorage.setItem("datalk.session", JSON.stringify(ownerSession));
    vi.restoreAllMocks();
  });

  it("renders provider configs and available models", async () => {
    vi.spyOn(window, "fetch").mockImplementation((input) => {
      const url = String(input);
      if (url === "/api/connections" || url.startsWith("/api/chat/conversations")) {
        return Promise.resolve(jsonResponse([]));
      }
      if (url === "/api/chat/provider-configs") {
        return Promise.resolve(jsonResponse([
          {
            id: 1,
            provider: "openai",
            display_name: "OpenAI",
            base_url: "https://api.openai.com",
            is_enabled: true,
            has_api_key: true,
            metadata: {},
            created_at: "2026-05-25T12:00:00Z",
            updated_at: "2026-05-25T12:00:00Z",
          },
        ]));
      }
      if (url === "/api/chat/models") {
        return Promise.resolve(jsonResponse([
          {
            id: "openai:gpt-5.2",
            provider: "openai",
            display_name: "GPT 5.2",
            description: "General model",
            is_enabled: true,
            capabilities: {
              supports_tool_calling: true,
              supports_structured_output: true,
              supports_streaming: false,
              supports_system_instructions: true,
              supports_vision: false,
              max_context_tokens: 128000,
              max_output_tokens: 8192,
            },
          },
        ]));
      }
      return Promise.resolve(jsonResponse({ error: "unexpected request" }, { status: 500 }));
    });

    renderProviderConfigsRoute();

    expect(await screen.findByText("OpenAI")).toBeInTheDocument();
    expect(screen.getByText("API key set")).toBeInTheDocument();
    expect(await screen.findByText("GPT 5.2")).toBeInTheDocument();
  });

  it("requires an API key when creating a remote provider config", async () => {
    const fetchMock = vi.spyOn(window, "fetch").mockImplementation((input) => {
      const url = String(input);
      if (
        url === "/api/connections" ||
        url.startsWith("/api/chat/conversations") ||
        url === "/api/chat/provider-configs" ||
        url === "/api/chat/models"
      ) {
        return Promise.resolve(jsonResponse([]));
      }
      return Promise.resolve(jsonResponse({ error: "unexpected request" }, { status: 500 }));
    });

    renderProviderConfigsRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Edit OpenAI" }));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(
      await screen.findByText("API key is required when creating this provider"),
    ).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalledWith(
      "/api/chat/provider-configs/openai",
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("saves provider config payloads and metadata", async () => {
    const fetchMock = vi.spyOn(window, "fetch").mockImplementation((input, init) => {
      const url = String(input);
      const method = init?.method ?? "GET";
      if (
        url === "/api/connections" ||
        url.startsWith("/api/chat/conversations") ||
        url === "/api/chat/models"
      ) {
        return Promise.resolve(jsonResponse([]));
      }
      if (url === "/api/chat/provider-configs" && method === "GET") {
        return Promise.resolve(jsonResponse([]));
      }
      if (url === "/api/chat/provider-configs/openai/test" && method === "POST") {
        return Promise.resolve(jsonResponse({ ok: true, model_count: 1 }));
      }
      if (url === "/api/chat/provider-configs/openai" && method === "PUT") {
        return Promise.resolve(jsonResponse({
          id: 1,
          provider: "openai",
          display_name: "OpenAI",
          base_url: "https://api.openai.com",
          is_enabled: true,
          has_api_key: true,
          metadata: { project: "analytics" },
          created_at: "2026-05-25T12:00:00Z",
          updated_at: "2026-05-25T12:00:00Z",
        }));
      }
      return Promise.resolve(jsonResponse({ error: "unexpected request" }, { status: 500 }));
    });

    renderProviderConfigsRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Edit OpenAI" }));
    await userEvent.type(screen.getByLabelText("API key"), "sk-test");
    fireEvent.change(screen.getByLabelText("Metadata JSON"), {
      target: { value: '{"project":"analytics"}' },
    });
    await userEvent.click(screen.getByRole("button", { name: "Test provider" }));
    expect(await screen.findByText("Provider test succeeded. 1 models found.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/provider-configs/openai/test",
        expect.objectContaining({ method: "POST" }),
      );
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/provider-configs/openai",
        expect.objectContaining({ method: "PUT" }),
      );
    });
    const saveCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        url === "/api/chat/provider-configs/openai" && init?.method === "PUT",
    );
    expect(JSON.parse(String(saveCall?.[1]?.body))).toEqual({
      display_name: "OpenAI",
      api_key: "sk-test",
      base_url: "https://api.openai.com",
      is_enabled: true,
      metadata: { project: "analytics" },
    });
  });

  it("omits api_key when updating an existing provider with a stored key", async () => {
    const fetchMock = vi.spyOn(window, "fetch").mockImplementation((input, init) => {
      const url = String(input);
      const method = init?.method ?? "GET";
      if (
        url === "/api/connections" ||
        url.startsWith("/api/chat/conversations") ||
        url === "/api/chat/models"
      ) {
        return Promise.resolve(jsonResponse([]));
      }
      if (url === "/api/chat/provider-configs" && method === "GET") {
        return Promise.resolve(jsonResponse([
          {
            id: 1,
            provider: "openai",
            display_name: "OpenAI",
            base_url: "https://api.openai.com",
            is_enabled: true,
            has_api_key: true,
            metadata: {},
            created_at: "2026-05-25T12:00:00Z",
            updated_at: "2026-05-25T12:00:00Z",
          },
        ]));
      }
      if (url === "/api/chat/provider-configs/openai" && method === "PUT") {
        return Promise.resolve(jsonResponse({
          id: 1,
          provider: "openai",
          display_name: "OpenAI",
          base_url: "https://api.openai.com",
          is_enabled: false,
          has_api_key: true,
          metadata: {},
          created_at: "2026-05-25T12:00:00Z",
          updated_at: "2026-05-25T12:00:00Z",
        }));
      }
      return Promise.resolve(jsonResponse({ error: "unexpected request" }, { status: 500 }));
    });

    renderProviderConfigsRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Edit OpenAI" }));
    await userEvent.click(screen.getByRole("switch", { name: "Enabled" }));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/provider-configs/openai",
        expect.objectContaining({ method: "PUT" }),
      );
    });
    const saveCall = fetchMock.mock.calls.find(
      ([url, init]) =>
        url === "/api/chat/provider-configs/openai" && init?.method === "PUT",
    );
    expect(JSON.parse(String(saveCall?.[1]?.body))).toEqual({
      display_name: "OpenAI",
      base_url: "https://api.openai.com",
      is_enabled: false,
      metadata: {},
    });
  });

  it("shows model availability errors", async () => {
    vi.spyOn(window, "fetch").mockImplementation((input) => {
      const url = String(input);
      if (
        url === "/api/connections" ||
        url.startsWith("/api/chat/conversations") ||
        url === "/api/chat/provider-configs"
      ) {
        return Promise.resolve(jsonResponse([]));
      }
      if (url === "/api/chat/models") {
        return Promise.resolve(
          jsonResponse({ error: "provider unavailable" }, { status: 400 }),
        );
      }
      return Promise.resolve(jsonResponse({ error: "unexpected request" }, { status: 500 }));
    });

    renderProviderConfigsRoute();

    expect(await screen.findByText("provider unavailable")).toBeInTheDocument();
  });
});

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}
