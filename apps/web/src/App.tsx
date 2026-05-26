import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Container from "@mui/material/Container";
import Divider from "@mui/material/Divider";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "/api";

export function App() {
  return (
    <Box sx={{ minHeight: "100vh", bgcolor: "background.default" }}>
      <Box
        component="header"
        sx={{
          borderBottom: "1px solid",
          borderColor: "divider",
          bgcolor: "background.paper",
        }}
      >
        <Container maxWidth="lg">
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            spacing={2}
            sx={{ minHeight: 64 }}
          >
            <Stack direction="row" alignItems="center" spacing={1.5}>
              <StorageOutlinedIcon color="primary" />
              <Typography component="h1" variant="h2">
                Datalk
              </Typography>
            </Stack>
            <Chip label="Frontend foundation" size="small" color="primary" />
          </Stack>
        </Container>
      </Box>

      <Container component="main" maxWidth="lg" sx={{ py: 4 }}>
        <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
          <Stack spacing={2.5}>
            <Stack spacing={0.5}>
              <Typography variant="h1">React app shell</Typography>
              <Typography color="text.secondary">
                Material UI, React Router, TanStack Query, TypeScript, and Vite
                are wired and ready for the product screens.
              </Typography>
            </Stack>

            <Divider />

            <Stack
              direction={{ xs: "column", sm: "row" }}
              spacing={1.5}
              alignItems={{ xs: "stretch", sm: "center" }}
            >
              <Button variant="contained" startIcon={<StorageOutlinedIcon />}>
                App shell ready
              </Button>
              <Typography variant="body2" color="text.secondary">
                API base: {apiBaseUrl}
              </Typography>
            </Stack>
          </Stack>
        </Paper>
      </Container>
    </Box>
  );
}
