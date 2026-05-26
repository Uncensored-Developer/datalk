import ChatOutlinedIcon from "@mui/icons-material/ChatOutlined";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useNavigate } from "react-router-dom";

export function OverviewPage() {
  const navigate = useNavigate();

  return (
    <Stack
      alignItems="center"
      justifyContent="center"
      spacing={3}
      sx={{ minHeight: "calc(100vh - 160px)", textAlign: "center" }}
    >
      <Stack spacing={1} sx={{ maxWidth: 560 }}>
        <Typography variant="h1">Start a conversation</Typography>
        <Typography color="text.secondary">
          Ask a question against an available database connection.
        </Typography>
      </Stack>
      <Button
        size="large"
        startIcon={<ChatOutlinedIcon />}
        variant="contained"
        onClick={() => navigate("/chat")}
      >
        Start conversation
      </Button>
    </Stack>
  );
}
