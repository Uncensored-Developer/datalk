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
