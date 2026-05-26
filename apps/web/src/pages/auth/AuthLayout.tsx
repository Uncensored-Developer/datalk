import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import type { ReactNode } from "react";
import { ThemeModeControl } from "../../components/layout/AppShell";

type AuthLayoutProps = {
  title: string;
  subtitle: string;
  children: ReactNode;
};

export function AuthLayout({ title, subtitle, children }: AuthLayoutProps) {
  return (
    <Box
      sx={{
        minHeight: "100vh",
        bgcolor: "background.default",
        display: "grid",
        alignItems: "center",
        py: 4,
      }}
    >
      <Container maxWidth="xs">
        <Stack spacing={2.5}>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Stack direction="row" spacing={1.25} alignItems="center">
              <StorageOutlinedIcon color="primary" />
              <Typography component="div" fontWeight={800}>
                Datalk
              </Typography>
            </Stack>
            <ThemeModeControl compact />
          </Stack>

          <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
            <Stack spacing={2.5}>
              <Stack spacing={0.5}>
                <Typography component="h1" variant="h1">
                  {title}
                </Typography>
                <Typography color="text.secondary">{subtitle}</Typography>
              </Stack>
              {children}
            </Stack>
          </Paper>
        </Stack>
      </Container>
    </Box>
  );
}
