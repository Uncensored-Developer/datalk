import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AppProviders } from "../../providers/AppProviders";
import { DashboardLayout } from "../../routes/DashboardLayout";
import { ProtectedRoute } from "../../routes/ProtectedRoute";
import type { MessageListItem, SendMessageResponse } from "../../types/api";
import { ChatPage } from "./ChatPage";

const session = {
  user: {
    id: 1,
    email: "user@example.com",
    name: "User",
    role: "member",
    must_change_password: false,
  },
  tokens: {
    access_token: "access-token",
    refresh_token: "refresh-token",
    expires_at: "2026-05-25T12:00:00Z",
  },
  must_change_password: false,
};

const connection = {
  id: 10,
  name: "Warehouse",
  database: "postgres",
  user_id: 1,
  is_enabled: true,
  metadata: {},
};

const model = {
  id: "openai:gpt-5.2",
  provider: "openai",
  display_name: "GPT 5.2",
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
};

const localModel = {
  id: "ollama:llama3.2",
  provider: "ollama",
  display_name: "Llama 3.2",
  is_enabled: true,
  capabilities: {
    supports_tool_calling: false,
    supports_structured_output: false,
    supports_streaming: false,
    supports_system_instructions: true,
    supports_vision: false,
    max_context_tokens: 128000,
    max_output_tokens: 8192,
  },
};

const conversation = {
  id: 100,
  user_id: 1,
  connection_id: 10,
  title: "Revenue questions",
  created_at: "2026-05-25T12:00:00Z",
  updated_at: "2026-05-25T12:00:00Z",
};

const messageItems: MessageListItem[] = [
  {
    message: {
      id: 1000,
      conversation_id: 100,
      role: "user",
      content: "How many users?",
      status: "completed",
      created_at: "2026-05-25T12:00:00Z",
    },
  },
  {
    message: {
      id: 1001,
      conversation_id: 100,
      role: "assistant",
      content: "SELECT count(*) FROM users;",
      provider: "openai",
      model: "openai:gpt-5.2",
      status: "completed",
      created_at: "2026-05-25T12:00:03Z",
    },
    execution: {
      message_id: 1001,
      connection_id: 10,
      database_kind: "postgres",
      generated_sql: "SELECT count(*) FROM users;",
      normalized_sql: "select count(*) from users;",
      result: {
        columns: [{ name: "count", data_type: "bigint" }],
        rows: [{ count: 42 }],
        row_count: 1,
        truncated: false,
        kind: "scalar",
      },
      execution_latency_ms: 120,
      executed_at: "2026-05-25T12:00:03Z",
    },
    retrieval: {
      message_id: 1001,
      snapshot_id: 20,
      query_text: "How many users?",
      retrieved_at: "2026-05-25T12:00:02Z",
    },
  },
];

function renderChatRoute(initialPath = "/chat") {
  const router = createMemoryRouter(
    [
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <DashboardLayout />,
            children: [
              { path: "/chat", element: <ChatPage /> },
              { path: "/chat/:conversationID", element: <ChatPage /> },
            ],
          },
        ],
      },
    ],
    { initialEntries: [initialPath] },
  );

  render(
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>,
  );
}

describe("ChatPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
    window.localStorage.setItem("datalk.session", JSON.stringify(session));
    vi.restoreAllMocks();
  });

  it("loads conversations and renders messages with SQL results", async () => {
    mockChatApi();

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);

    expect(await screen.findByText("How many users?")).toBeInTheDocument();
    expect(screen.getAllByText("SELECT count(*) FROM users;")).not.toHaveLength(0);
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.queryByText("Retrieval snapshot 20 at 2026-05-25T12:00:02Z")).not.toBeInTheDocument();
  });

  it("creates a conversation", async () => {
    const fetchMock = mockChatApi({ conversations: [] });

    renderChatRoute();

    await userEvent.click(await screen.findByRole("button", { name: "New conversation" }));
    await userEvent.type(screen.getByLabelText("Title"), "Revenue questions");
    await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Create" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/conversations",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ connection_id: 10, title: "Revenue questions" }),
        }),
      );
    });
  });

  it("sends a message with the selected model provider", async () => {
    const fetchMock = mockChatApi({ messages: [] });

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);
    await userEvent.type(await screen.findByLabelText("Message"), "How many users?");
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/conversations/100/messages",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            content: "How many users?",
            provider: "openai",
            model: "openai:gpt-5.2",
            require_natural_response: true,
          }),
        }),
      );
    });
  });

  it("remembers the natural response toggle across conversations", async () => {
    const fetchMock = mockChatApi({ messages: [] });

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);
    await userEvent.click(await screen.findByRole("button", { name: "Turn natural response off" }));
    await userEvent.type(await screen.findByLabelText("Message"), "How many users?");
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/conversations/100/messages",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            content: "How many users?",
            provider: "openai",
            model: "openai:gpt-5.2",
            require_natural_response: false,
          }),
        }),
      );
    });
    expect(window.localStorage.getItem("datalk.chat.requireNaturalResponse")).toBe("false");
  });

  it("displays a natural response from the send response in chunks", async () => {
    mockChatApi({
      messages: [],
      sendResponse: {
        conversation,
        user_message: messageItems[0].message,
        assistant_message: {
          ...messageItems[1].message,
          content: "Counts users.",
          natural_response: "There are 42 users.",
        },
        execution: messageItems[1].execution,
        retrieval: messageItems[1].retrieval,
      },
    });

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);
    await userEvent.type(await screen.findByLabelText("Message"), "How many users?");
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(await screen.findByText("There are 42 users.")).toBeInTheDocument();
    expect(screen.queryByText("Counts users.")).not.toBeInTheDocument();
  });

  it("uses and updates the last successful chat model", async () => {
    window.localStorage.setItem("datalk.chat.lastModel", "ollama:llama3.2");
    const fetchMock = mockChatApi({ messages: [], models: [model, localModel] });

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);
    await userEvent.type(await screen.findByLabelText("Message"), "How many users?");
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/conversations/100/messages",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            content: "How many users?",
            provider: "ollama",
            model: "ollama:llama3.2",
            require_natural_response: true,
          }),
        }),
      );
    });
    expect(window.localStorage.getItem("datalk.chat.lastModel")).toBe(
      "ollama:llama3.2",
    );
  });

  it("shows natural responses first and reveals SQL details on toggle", async () => {
    mockChatApi({
      messages: [
        messageItems[0],
        {
          ...messageItems[1],
          message: {
            ...messageItems[1].message,
            content: "Counts users.",
            natural_response: "There are 42 users.",
          },
        },
      ],
    });

    renderChatRoute();

    await userEvent.click((await screen.findAllByText("Revenue questions"))[0]);

    expect(await screen.findByText("There are 42 users.")).toBeInTheDocument();
    expect(screen.queryByText("Counts users.")).not.toBeInTheDocument();
    expect(screen.queryByText("42")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Show SQL and results" }));

    expect(await screen.findByText("Counts users.")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("deletes a conversation", async () => {
    const fetchMock = mockChatApi();

    renderChatRoute();

    await userEvent.click(await screen.findByRole("button", { name: "Delete Revenue questions" }));
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/chat/conversations/100",
        expect.objectContaining({ method: "DELETE" }),
      );
    });
  });

  it("surfaces conversation list load failures", async () => {
    vi.spyOn(window, "fetch").mockImplementation(async (input) => {
      const path = requestPath(input);

      if (path === "/connections") {
        return jsonResponse([connection]);
      }
      if (path === "/chat/models") {
        return jsonResponse([model]);
      }
      if (path.startsWith("/chat/conversations?")) {
        return jsonResponse({ error: "conversation service unavailable" }, { status: 503 });
      }

      return jsonResponse({ error: `Unhandled ${path}` }, { status: 500 });
    });

    renderChatRoute();

    expect(await screen.findAllByText("conversation service unavailable")).not.toHaveLength(0);
    expect(screen.queryByText("No conversations")).not.toBeInTheDocument();
  });

  it("surfaces message list load failures", async () => {
    vi.spyOn(window, "fetch").mockImplementation(async (input, init) => {
      const path = requestPath(input);
      const method = init?.method ?? "GET";

      if (method === "GET" && path === "/connections") {
        return jsonResponse([connection]);
      }
      if (method === "GET" && path === "/chat/models") {
        return jsonResponse([model]);
      }
      if (method === "GET" && path.startsWith("/chat/conversations?")) {
        return jsonResponse([conversation]);
      }
      if (method === "GET" && path === "/chat/conversations/100") {
        return jsonResponse(conversation);
      }
      if (method === "GET" && path === "/chat/conversations/100/messages?limit=50&offset=0") {
        return jsonResponse({ error: "messages unavailable" }, { status: 503 });
      }

      return jsonResponse({ error: `Unhandled ${method} ${path}` }, { status: 500 });
    });

    renderChatRoute("/chat/100");

    expect(await screen.findByText("messages unavailable")).toBeInTheDocument();
    expect(screen.queryByText("No messages yet")).not.toBeInTheDocument();
  });
});

function mockChatApi({
  conversations = [conversation],
  messages = messageItems,
  models = [model],
  sendResponse = {
    conversation,
    user_message: messageItems[0].message,
    assistant_message: messageItems[1].message,
    execution: messageItems[1].execution,
    retrieval: messageItems[1].retrieval,
  },
}: {
  conversations?: typeof conversation[];
  messages?: typeof messageItems;
  models?: Array<typeof model>;
  sendResponse?: SendMessageResponse;
} = {}) {
  return vi.spyOn(window, "fetch").mockImplementation(async (input, init) => {
    const rawPath =
      typeof input === "string"
        ? input
        : input instanceof Request
          ? input.url
          : input.toString();
    const path = rawPath.replace(/^\/api/, "");
    const method = init?.method ?? "GET";

    if (method === "GET" && path === "/connections") {
      return jsonResponse([connection]);
    }
    if (method === "GET" && path === "/chat/models") {
      return jsonResponse(models);
    }
    if (method === "GET" && path.startsWith("/chat/conversations?")) {
      return jsonResponse(conversations);
    }
    if (method === "GET" && path === "/chat/conversations/100") {
      return jsonResponse(conversation);
    }
    if (method === "GET" && path === "/chat/conversations/100/messages?limit=50&offset=0") {
      return jsonResponse(messages);
    }
    if (method === "POST" && path === "/chat/conversations") {
      return jsonResponse(conversation, { status: 201 });
    }
    if (method === "POST" && path === "/chat/conversations/100/messages") {
      return jsonResponse(sendResponse);
    }
    if (method === "DELETE" && path === "/chat/conversations/100") {
      return new Response(null, { status: 204 });
    }

    return jsonResponse({ error: `Unhandled ${method} ${path}` }, { status: 500 });
  });
}

function requestPath(input: RequestInfo | URL) {
  const rawPath =
    typeof input === "string"
      ? input
      : input instanceof Request
        ? input.url
        : input.toString();

  return rawPath.replace(/^\/api/, "");
}

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}
