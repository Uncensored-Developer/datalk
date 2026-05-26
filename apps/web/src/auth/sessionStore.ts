import type { AuthSession } from "../types/api";

const storageKey = "datalk.session";

export type SessionStore = {
  getSession: () => AuthSession | null;
  setSession: (session: AuthSession) => void;
  clearSession: () => void;
};

export const localSessionStore: SessionStore = {
  getSession() {
    if (typeof window === "undefined") {
      return null;
    }

    const rawSession = window.localStorage.getItem(storageKey);
    if (!rawSession) {
      return null;
    }

    try {
      return JSON.parse(rawSession) as AuthSession;
    } catch {
      window.localStorage.removeItem(storageKey);
      return null;
    }
  },
  setSession(session) {
    window.localStorage.setItem(storageKey, JSON.stringify(session));
  },
  clearSession() {
    window.localStorage.removeItem(storageKey);
  },
};
