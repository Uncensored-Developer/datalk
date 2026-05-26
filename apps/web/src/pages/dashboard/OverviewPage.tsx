import ApiOutlinedIcon from "@mui/icons-material/ApiOutlined";
import DevicesOutlinedIcon from "@mui/icons-material/DevicesOutlined";
import PaletteOutlinedIcon from "@mui/icons-material/PaletteOutlined";
import SecurityOutlinedIcon from "@mui/icons-material/SecurityOutlined";
import Chip from "@mui/material/Chip";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { EmptyState } from "../../components/common/EmptyState";
import { apiBaseUrl } from "../../config";

const foundationItems = [
  {
    title: "Theme modes",
    description: "Light, dark, and system preferences are available globally.",
    icon: <PaletteOutlinedIcon color="primary" />,
  },
  {
    title: "Responsive shell",
    description: "Desktop sidebar and mobile drawer navigation are wired.",
    icon: <DevicesOutlinedIcon color="primary" />,
  },
  {
    title: "Session context",
    description: "Local session storage and auth context are ready for flows.",
    icon: <SecurityOutlinedIcon color="primary" />,
  },
  {
    title: "API client",
    description: "JSON requests, bearer auth, errors, and refresh retry exist.",
    icon: <ApiOutlinedIcon color="primary" />,
  },
];

export function OverviewPage() {
  return (
    <Stack spacing={3}>
      <Stack spacing={1}>
        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={1}
          alignItems={{ xs: "flex-start", sm: "center" }}
        >
          <Typography variant="h1">React app foundation</Typography>
          <Chip label="Step 3" size="small" color="primary" />
        </Stack>
        <Typography color="text.secondary" sx={{ maxWidth: 720 }}>
          Auth routes, session persistence, protected navigation, logout, and
          current-user password changes are now wired into the foundation.
        </Typography>
      </Stack>

      <Grid container spacing={2}>
        {foundationItems.map((item) => (
          <Grid key={item.title} size={{ xs: 12, sm: 6, lg: 3 }}>
            <Paper variant="outlined" sx={{ height: "100%", p: 2, minHeight: 156 }}>
              <Stack spacing={1.5}>
                {item.icon}
                <Stack spacing={0.5}>
                  <Typography fontWeight={800}>{item.title}</Typography>
                  <Typography color="text.secondary" variant="body2">
                    {item.description}
                  </Typography>
                </Stack>
              </Stack>
            </Paper>
          </Grid>
        ))}
      </Grid>

      <Paper variant="outlined" sx={{ p: 2 }}>
        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={1}
          alignItems={{ xs: "flex-start", sm: "center" }}
          justifyContent="space-between"
        >
          <Stack spacing={0.5}>
            <Typography fontWeight={800}>API base</Typography>
            <Typography color="text.secondary" variant="body2">
              {apiBaseUrl}
            </Typography>
          </Stack>
          <Chip label="same-origin ready" size="small" />
        </Stack>
      </Paper>

      <EmptyState
        title="Domain screens come next"
        description="Users, connections, provider configs, and chat will be added in later steps."
      />
    </Stack>
  );
}
