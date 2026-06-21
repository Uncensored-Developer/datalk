import { apiBaseUrl } from "../config";
import type { SessionStore } from "../auth/sessionStore";
import type { ApiErrorShape, AuthSession } from "../types/api";

type ApiClientOptions = {
  baseUrl?: string;
  sessionStore?: SessionStore;
};

type RequestOptions = Omit<RequestInit, "body"> & {
  auth?: boolean;
  body?: unknown;
  refreshOnUnauthorized?: boolean;
};

type EventStreamHandlers<TProgress> = {
  onProgress?: (event: TProgress) => void;
};

export class ApiError extends Error {
  readonly status: number;
  readonly payload: unknown;

  constructor(status: number, message: string, payload: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly sessionStore?: SessionStore;
  private refreshPromise: Promise<AuthSession | null> | null = null;

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? apiBaseUrl;
    this.sessionStore = options.sessionStore;
  }

  async get<T>(path: string, options: RequestOptions = {}) {
    return this.request<T>(path, { ...options, method: "GET" });
  }

  async post<T>(path: string, body?: unknown, options: RequestOptions = {}) {
    return this.request<T>(path, { ...options, method: "POST", body });
  }

  async put<T>(path: string, body?: unknown, options: RequestOptions = {}) {
    return this.request<T>(path, { ...options, method: "PUT", body });
  }

  async delete<T>(path: string, options: RequestOptions = {}) {
    return this.request<T>(path, { ...options, method: "DELETE" });
  }

  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    return this.fetchJson<T>(path, options, true);
  }

  async postEventStream<TProgress, TFinal>(
    path: string,
    body: unknown,
    handlers: EventStreamHandlers<TProgress> = {},
  ): Promise<TFinal> {
    return this.fetchEventStream<TProgress, TFinal>(
      path,
      { method: "POST", body },
      handlers,
      true,
    );
  }

  private async fetchJson<T>(
    path: string,
    options: RequestOptions,
    allowRefresh: boolean,
  ): Promise<T> {
    const response = await fetch(this.urlFor(path), {
      ...options,
      headers: this.headersFor(options),
      body: serializeBody(options.body),
    });

    if (
      response.status === 401 &&
      allowRefresh &&
      options.auth !== false &&
      options.refreshOnUnauthorized !== false
    ) {
      const refreshed = await this.refreshSession();
      if (refreshed) {
        return this.fetchJson<T>(path, options, false);
      }
    }

    if (!response.ok) {
      throw await parseApiError(response);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return (await response.json()) as T;
  }

  private async fetchEventStream<TProgress, TFinal>(
    path: string,
    options: RequestOptions,
    handlers: EventStreamHandlers<TProgress>,
    allowRefresh: boolean,
  ): Promise<TFinal> {
    const response = await fetch(this.urlFor(path), {
      ...options,
      headers: this.headersFor(options),
      body: serializeBody(options.body),
    });

    if (
      response.status === 401 &&
      allowRefresh &&
      options.auth !== false &&
      options.refreshOnUnauthorized !== false
    ) {
      const refreshed = await this.refreshSession();
      if (refreshed) {
        return this.fetchEventStream<TProgress, TFinal>(path, options, handlers, false);
      }
    }

    if (!response.ok) {
      throw await parseApiError(response);
    }

    if (!response.body) {
      throw new ApiError(0, "streaming responses are not supported", null);
    }

    return readEventStream<TProgress, TFinal>(response.body, handlers);
  }

  private async refreshSession() {
    const refreshToken = this.sessionStore?.getSession()?.tokens.refresh_token;
    if (!refreshToken) {
      return null;
    }

    this.refreshPromise ??= this.post<AuthSession>(
      "/auth/refresh",
      { refresh_token: refreshToken },
      { auth: false },
    )
      .then((session) => {
        this.sessionStore?.setSession(session);
        return session;
      })
      .catch(() => {
        this.sessionStore?.clearSession();
        return null;
      })
      .finally(() => {
        this.refreshPromise = null;
      });

    return this.refreshPromise;
  }

  private headersFor(options: RequestOptions) {
    const headers = new Headers(options.headers);

    if (
      options.body !== undefined &&
      !(options.body instanceof FormData) &&
      !headers.has("Content-Type")
    ) {
      headers.set("Content-Type", "application/json");
    }

    const accessToken = this.sessionStore?.getSession()?.tokens.access_token;
    if (options.auth !== false && accessToken && !headers.has("Authorization")) {
      headers.set("Authorization", `Bearer ${accessToken}`);
    }

    return headers;
  }

  private urlFor(path: string) {
    if (/^https?:\/\//.test(path)) {
      return path;
    }

    return `${this.baseUrl}${path.startsWith("/") ? path : `/${path}`}`;
  }
}

function serializeBody(body: unknown) {
  if (body === undefined || body instanceof FormData) {
    return body as BodyInit | undefined;
  }

  return JSON.stringify(body);
}

async function parseApiError(response: Response) {
  const payload = await parseResponsePayload(response);
  const message =
    typeof payload === "object" &&
    payload !== null &&
    "error" in payload &&
    typeof (payload as ApiErrorShape).error === "string"
      ? (payload as ApiErrorShape).error
      : response.statusText || "request failed";

  return new ApiError(response.status, message, payload);
}

async function parseResponsePayload(response: Response) {
  const contentType = response.headers.get("Content-Type") ?? "";
  if (!contentType.includes("application/json")) {
    return response.text();
  }

  try {
    return await response.json();
  } catch {
    return null;
  }
}

async function readEventStream<TProgress, TFinal>(
  body: ReadableStream<Uint8Array>,
  handlers: EventStreamHandlers<TProgress>,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    let separatorIndex = buffer.indexOf("\n\n");
    while (separatorIndex >= 0) {
      const rawEvent = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      const final = handleEventStreamBlock<TProgress, TFinal>(rawEvent, handlers);
      if (final !== undefined) {
        return final;
      }
      separatorIndex = buffer.indexOf("\n\n");
    }
  }

  buffer += decoder.decode();
  if (buffer.trim()) {
    const final = handleEventStreamBlock<TProgress, TFinal>(buffer, handlers);
    if (final !== undefined) {
      return final;
    }
  }

  throw new ApiError(500, "stream ended before final response", null);
}

function handleEventStreamBlock<TProgress, TFinal>(
  rawEvent: string,
  handlers: EventStreamHandlers<TProgress>,
) {
  const lines = rawEvent.split(/\r?\n/);
  const event = lines
    .find((line) => line.startsWith("event:"))
    ?.slice("event:".length)
    .trim();
  const data = lines
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice("data:".length).trimStart())
    .join("\n");

  if (!event || !data) {
    return undefined;
  }

  const payload = JSON.parse(data) as unknown;
  if (event === "progress") {
    handlers.onProgress?.(payload as TProgress);
    return undefined;
  }
  if (event === "final") {
    return payload as TFinal;
  }
  if (event === "error") {
    const message =
      typeof payload === "object" &&
      payload !== null &&
      "error" in payload &&
      typeof (payload as ApiErrorShape).error === "string"
        ? (payload as ApiErrorShape).error
        : "request failed";
    const status =
      typeof payload === "object" &&
      payload !== null &&
      "status" in payload &&
      typeof (payload as { status?: unknown }).status === "number"
        ? (payload as { status: number }).status
        : 500;
    throw new ApiError(status, message, payload);
  }

  return undefined;
}
