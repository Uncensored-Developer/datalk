import CssBaseline from "@mui/material/CssBaseline";
import useMediaQuery from "@mui/material/useMediaQuery";
import { ThemeProvider } from "@mui/material/styles";
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import { createAppTheme } from "../theme";

export type ThemeModePreference = "light" | "dark" | "system";
type ResolvedThemeMode = "light" | "dark";

type ThemeModeContextValue = {
  modePreference: ThemeModePreference;
  resolvedMode: ResolvedThemeMode;
  setModePreference: (mode: ThemeModePreference) => void;
};

const storageKey = "datalk.themeMode";
const ThemeModeContext = createContext<ThemeModeContextValue | null>(null);

export function ThemeModeProvider({ children }: PropsWithChildren) {
  const prefersDark = useMediaQuery("(prefers-color-scheme: dark)");
  const [modePreference, setStoredModePreference] =
    useState<ThemeModePreference>(() => readStoredModePreference());

  const resolvedMode =
    modePreference === "system" ? (prefersDark ? "dark" : "light") : modePreference;

  const theme = useMemo(() => createAppTheme(resolvedMode), [resolvedMode]);

  const setModePreference = (nextMode: ThemeModePreference) => {
    setStoredModePreference(nextMode);
    window.localStorage.setItem(storageKey, nextMode);
  };

  useEffect(() => {
    document.documentElement.dataset.theme = resolvedMode;
  }, [resolvedMode]);

  const value = useMemo<ThemeModeContextValue>(
    () => ({ modePreference, resolvedMode, setModePreference }),
    [modePreference, resolvedMode],
  );

  return (
    <ThemeModeContext.Provider value={value}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ThemeModeContext.Provider>
  );
}

export function useThemeMode() {
  const value = useContext(ThemeModeContext);
  if (!value) {
    throw new Error("useThemeMode must be used within ThemeModeProvider");
  }

  return value;
}

function readStoredModePreference(): ThemeModePreference {
  if (typeof window === "undefined") {
    return "system";
  }

  const value = window.localStorage.getItem(storageKey);
  return value === "light" || value === "dark" || value === "system"
    ? value
    : "system";
}
