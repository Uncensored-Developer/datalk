import ConstructionOutlinedIcon from "@mui/icons-material/ConstructionOutlined";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";

type PlaceholderPageProps = {
  title: string;
};

export function PlaceholderPage({ title }: PlaceholderPageProps) {
  return (
    <Paper variant="outlined" sx={{ p: { xs: 2, sm: 3 } }}>
      <Stack spacing={1.5}>
        <ConstructionOutlinedIcon color="primary" />
        <Stack spacing={0.5}>
          <Typography variant="h1">{title}</Typography>
          <Typography color="text.secondary">
            This protected route is reserved for a later implementation step.
          </Typography>
        </Stack>
      </Stack>
    </Paper>
  );
}
