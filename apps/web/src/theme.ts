import { createTheme, type PaletteMode } from "@mui/material/styles";

export function createAppTheme(mode: PaletteMode) {
  const isDark = mode === "dark";

  return createTheme({
    palette: {
      mode,
      primary: {
        main: isDark ? "#60a5fa" : "#2563eb",
      },
      secondary: {
        main: isDark ? "#2dd4bf" : "#0f766e",
      },
      background: {
        default: isDark ? "#111827" : "#f8fafc",
        paper: isDark ? "#1f2937" : "#ffffff",
      },
      text: {
        primary: isDark ? "#f9fafb" : "#111827",
        secondary: isDark ? "#d1d5db" : "#4b5563",
      },
      divider: isDark ? "rgba(255, 255, 255, 0.12)" : "rgba(17, 24, 39, 0.12)",
    },
    shape: {
      borderRadius: 8,
    },
    typography: {
      fontFamily:
        "Inter, system-ui, -apple-system, BlinkMacSystemFont, sans-serif",
      h1: {
        fontSize: "2rem",
        fontWeight: 700,
        letterSpacing: 0,
      },
      h2: {
        fontSize: "1.25rem",
        fontWeight: 700,
        letterSpacing: 0,
      },
      button: {
        fontWeight: 600,
        letterSpacing: 0,
        textTransform: "none",
      },
    },
    components: {
      MuiButton: {
        defaultProps: {
          disableElevation: true,
        },
        styleOverrides: {
          root: {
            minHeight: 36,
          },
        },
      },
      MuiCard: {
        styleOverrides: {
          root: {
            borderRadius: 8,
          },
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: {
            backgroundImage: "none",
          },
        },
      },
    },
  });
}
