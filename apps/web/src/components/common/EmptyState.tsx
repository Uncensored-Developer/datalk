import InboxOutlinedIcon from "@mui/icons-material/InboxOutlined";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import type { ReactNode } from "react";

type EmptyStateProps = {
  title: string;
  description?: string;
  action?: ReactNode;
};

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <Paper variant="outlined" sx={{ p: 3 }}>
      <Stack alignItems="center" spacing={1.5} textAlign="center">
        <InboxOutlinedIcon color="disabled" fontSize="large" />
        <Stack spacing={0.5}>
          <Typography fontWeight={700}>{title}</Typography>
          {description ? (
            <Typography color="text.secondary" variant="body2">
              {description}
            </Typography>
          ) : null}
        </Stack>
        {typeof action === "string" ? (
          <Button variant="contained">{action}</Button>
        ) : (
          action
        )}
      </Stack>
    </Paper>
  );
}
