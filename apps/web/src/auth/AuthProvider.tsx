import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiClient } from "../api/client";
import { localSessionStore, type SessionStore } from "./sessionStore";
import type { AuthSession, User } from "../types/api";

type AuthContextValue = {
  session: AuthSession | null;
  user: User | null;
  isAuthenticated: boolean;
  apiClient: ApiClient;
  setSession: (session: AuthSession) => void;
  clearSession: () => void;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient();
  const [session, setStoredSession] = useState<AuthSession | null>(() =>
    localSessionStore.getSession(),
  );

  const storeSession = useCallback(
    (nextSession: AuthSession) => {
      const previousSession = localSessionStore.getSession();
      localSessionStore.setSession(nextSession);
      setStoredSession(nextSession);

      if (previousSession?.user.id !== nextSession.user.id) {
        queryClient.clear();
      }
    },
    [queryClient],
  );

  const dropSession = useCallback(() => {
    localSessionStore.clearSession();
    setStoredSession(null);
    queryClient.clear();
  }, [queryClient]);

  const sessionStore = useMemo<SessionStore>(
    () => ({
      getSession: localSessionStore.getSession,
      setSession(nextSession) {
        storeSession(nextSession);
      },
      clearSession() {
        dropSession();
      },
    }),
    [dropSession, storeSession],
  );

  const apiClient = useMemo(
    () => new ApiClient({ sessionStore }),
    [sessionStore],
  );

  const setSession = useCallback(
    (nextSession: AuthSession) => {
      storeSession(nextSession);
    },
    [storeSession],
  );

  const clearSession = useCallback(() => {
    dropSession();
  }, [dropSession]);

  const logout = useCallback(async () => {
    const refreshToken = localSessionStore.getSession()?.tokens.refresh_token;
    try {
      if (refreshToken) {
        await apiClient.post<void>(
          "/auth/logout",
          {
            refresh_token: refreshToken,
          },
          {
            refreshOnUnauthorized: false,
          },
        );
      }
    } catch {
      // Local sign-out must complete even if the server cannot revoke the token.
    } finally {
      dropSession();
    }
  }, [apiClient, dropSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      user: session?.user ?? null,
      isAuthenticated: Boolean(session),
      apiClient,
      setSession,
      clearSession,
      logout,
    }),
    [apiClient, clearSession, logout, session, setSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return value;
}
