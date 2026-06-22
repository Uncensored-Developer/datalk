import AccountCircleOutlinedIcon from "@mui/icons-material/AccountCircleOutlined";
import AddOutlinedIcon from "@mui/icons-material/AddOutlined";
import ChatOutlinedIcon from "@mui/icons-material/ChatOutlined";
import ChevronLeftOutlinedIcon from "@mui/icons-material/ChevronLeftOutlined";
import ChevronRightOutlinedIcon from "@mui/icons-material/ChevronRightOutlined";
import CloseOutlinedIcon from "@mui/icons-material/CloseOutlined";
import DarkModeOutlinedIcon from "@mui/icons-material/DarkModeOutlined";
import DeleteOutlineOutlinedIcon from "@mui/icons-material/DeleteOutlineOutlined";
import DashboardOutlinedIcon from "@mui/icons-material/DashboardOutlined";
import KeyboardArrowDownOutlinedIcon from "@mui/icons-material/KeyboardArrowDownOutlined";
import LightModeOutlinedIcon from "@mui/icons-material/LightModeOutlined";
import LogoutOutlinedIcon from "@mui/icons-material/LogoutOutlined";
import MenuOutlinedIcon from "@mui/icons-material/MenuOutlined";
import PeopleOutlineOutlinedIcon from "@mui/icons-material/PeopleOutlineOutlined";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";
import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import TuneOutlinedIcon from "@mui/icons-material/TuneOutlined";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import Divider from "@mui/material/Divider";
import Drawer from "@mui/material/Drawer";
import FormControl from "@mui/material/FormControl";
import IconButton from "@mui/material/IconButton";
import InputLabel from "@mui/material/InputLabel";
import List from "@mui/material/List";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Toolbar from "@mui/material/Toolbar";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { ConfirmDialog } from "../common/ConfirmDialog";
import { EmptyState } from "../common/EmptyState";
import { LoadingState } from "../common/LoadingState";
import {
  useThemeMode,
  type ThemeModePreference,
} from "../../providers/ThemeModeProvider";
import type { Connection, Conversation, UserRole } from "../../types/api";
import { errorMessage } from "../../utils/errors";

const drawerWidth = 304;

type AppShellProps = {
  title: string;
  children: ReactNode;
};

const pageItems: Array<{
  label: string;
  path: string;
  icon: ReactNode;
  roles?: UserRole[];
}> = [
  { label: "Start", path: "/", icon: <DashboardOutlinedIcon /> },
  { label: "Chat", path: "/chat", icon: <ChatOutlinedIcon /> },
  { label: "Connections", path: "/connections", icon: <StorageOutlinedIcon /> },
  { label: "Profile", path: "/profile", icon: <AccountCircleOutlinedIcon /> },
  {
    label: "Users",
    path: "/users",
    icon: <PeopleOutlineOutlinedIcon />,
    roles: ["owner", "admin"],
  },
  {
    label: "Provider Configs",
    path: "/provider-configs",
    icon: <SettingsOutlinedIcon />,
    roles: ["owner", "admin"],
  },
];

export function AppShell({ title, children }: AppShellProps) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [pagesAnchor, setPagesAnchor] = useState<HTMLElement | null>(null);
  const { logout, user } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const visiblePageItems = pageItems.filter(
    (item) => !item.roles || (user && item.roles.includes(user.role)),
  );

  const drawer = <ConversationNavigation onSelect={() => setMobileOpen(false)} />;

  return (
    <Box sx={{ display: "flex", minHeight: "100vh", bgcolor: "background.default" }}>
      <Box
        component="nav"
        sx={{
          width: { md: sidebarCollapsed ? 0 : drawerWidth },
          flexShrink: { md: 0 },
          transition: (theme) =>
            theme.transitions.create("width", {
              duration: theme.transitions.duration.shorter,
            }),
        }}
      >
        <Drawer
          ModalProps={{ keepMounted: true }}
          onClose={() => setMobileOpen(false)}
          open={mobileOpen}
          sx={{
            display: { xs: "block", md: "none" },
            "& .MuiDrawer-paper": { width: drawerWidth, maxWidth: "88vw" },
          }}
          variant="temporary"
        >
          <Stack direction="row" justifyContent="flex-end" sx={{ p: 1 }}>
            <IconButton aria-label="Close conversations" onClick={() => setMobileOpen(false)}>
              <CloseOutlinedIcon />
            </IconButton>
          </Stack>
          {drawer}
        </Drawer>
        <Drawer
          open
          sx={{
            display: { xs: "none", md: sidebarCollapsed ? "none" : "block" },
            "& .MuiDrawer-paper": {
              boxSizing: "border-box",
              width: drawerWidth,
              borderRightColor: "divider",
            },
          }}
          variant="permanent"
        >
          {drawer}
        </Drawer>
      </Box>

      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box
          component="header"
          sx={{
            position: "sticky",
            top: 0,
            zIndex: (theme) => theme.zIndex.appBar,
            borderBottom: "1px solid",
            borderColor: "divider",
            bgcolor: "background.paper",
          }}
        >
          <Toolbar sx={{ gap: 1.5 }}>
            <IconButton
              aria-label="Open conversations"
              edge="start"
              onClick={() => setMobileOpen(true)}
              sx={{ display: { md: "none" } }}
            >
              <MenuOutlinedIcon />
            </IconButton>
            <Tooltip title={sidebarCollapsed ? "Show conversations" : "Hide conversations"}>
              <IconButton
                aria-label={sidebarCollapsed ? "Show conversations" : "Hide conversations"}
                edge="start"
                onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
                sx={{ display: { xs: "none", md: "inline-flex" } }}
              >
                {sidebarCollapsed ? <ChevronRightOutlinedIcon /> : <ChevronLeftOutlinedIcon />}
              </IconButton>
            </Tooltip>
            <Typography component="h1" fontWeight={800} sx={{ flex: 1 }} variant="h2">
              {title}
            </Typography>
            <ThemeModeControl />
            <Button
              aria-controls={pagesAnchor ? "page-menu" : undefined}
              aria-expanded={pagesAnchor ? "true" : undefined}
              aria-haspopup="menu"
              endIcon={<KeyboardArrowDownOutlinedIcon />}
              onClick={(event) => setPagesAnchor(event.currentTarget)}
              variant="outlined"
              sx={{ display: { xs: "none", sm: "inline-flex" } }}
            >
              Pages
            </Button>
            <Tooltip title="Pages">
              <IconButton
                aria-controls={pagesAnchor ? "page-menu" : undefined}
                aria-expanded={pagesAnchor ? "true" : undefined}
                aria-haspopup="menu"
                aria-label="Open pages"
                onClick={(event) => setPagesAnchor(event.currentTarget)}
                sx={{ display: { xs: "inline-flex", sm: "none" } }}
              >
                <DashboardOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Menu
              anchorEl={pagesAnchor}
              id="page-menu"
              onClose={() => setPagesAnchor(null)}
              open={Boolean(pagesAnchor)}
            >
              {visiblePageItems.map((item) => (
                <MenuItem
                  key={item.path}
                  onClick={() => {
                    setPagesAnchor(null);
                    navigate(item.path);
                  }}
                  selected={
                    item.path === "/"
                      ? location.pathname === "/"
                      : location.pathname.startsWith(item.path)
                  }
                >
                  <ListItemIcon>{item.icon}</ListItemIcon>
                  <ListItemText>{item.label}</ListItemText>
                </MenuItem>
              ))}
            </Menu>
            <Tooltip title="Sign out">
              <IconButton
                aria-label="Sign out"
                onClick={() => {
                  void logout().finally(() => navigate("/login", { replace: true }));
                }}
              >
                <LogoutOutlinedIcon />
              </IconButton>
            </Tooltip>
          </Toolbar>
        </Box>

        <Box component="main" sx={{ p: { xs: 2, sm: 3 }, maxWidth: 1280 }}>
          {children}
        </Box>
      </Box>
    </Box>
  );
}

function ConversationNavigation({ onSelect }: { onSelect: () => void }) {
  const { apiClient, user } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const [selectedConnectionID, setSelectedConnectionID] = useState("");
  const [deletingConversation, setDeletingConversation] = useState<Conversation | null>(null);
  const activeConversationID = conversationIDFromPath(location.pathname);

  const connectionsQuery = useQuery({
    queryKey: ["connections"],
    queryFn: () => apiClient.get<Connection[]>("/connections"),
  });
  const conversationsPath = selectedConnectionID
    ? `/chat/conversations?connection_id=${selectedConnectionID}&limit=50&offset=0`
    : "/chat/conversations?limit=50&offset=0";
  const conversationsQuery = useQuery({
    queryKey: ["chat-conversations", selectedConnectionID],
    queryFn: () => apiClient.get<Conversation[]>(conversationsPath),
    retry: false,
  });
  const deleteMutation = useMutation({
    mutationFn: (conversation: Conversation) =>
      apiClient.delete<void>(`/chat/conversations/${conversation.id}`),
    onSuccess() {
      if (deletingConversation?.id === activeConversationID) {
        navigate("/chat");
      }
      setDeletingConversation(null);
      void queryClient.invalidateQueries({ queryKey: ["chat-conversations"] });
    },
  });

  const connections = Array.isArray(connectionsQuery.data) ? connectionsQuery.data : [];
  const conversations = Array.isArray(conversationsQuery.data) ? conversationsQuery.data : [];

  return (
    <Stack sx={{ height: "100%" }}>
      <Toolbar sx={{ px: 2 }}>
        <Stack direction="row" alignItems="center" spacing={1.25} sx={{ minWidth: 0, flex: 1 }}>
          <StorageOutlinedIcon color="primary" />
          <Typography component="div" fontWeight={800} noWrap>
            Datalk
          </Typography>
        </Stack>
        <Tooltip title="New conversation">
          <span>
            <IconButton
              aria-label="New conversation"
              color="primary"
              disabled={connectionsQuery.isLoading || connections.length === 0}
              onClick={() => {
                navigate(selectedConnectionID ? `/chat?connection_id=${selectedConnectionID}` : "/chat");
                onSelect();
              }}
            >
              <AddOutlinedIcon />
            </IconButton>
          </span>
        </Tooltip>
      </Toolbar>
      <Divider />
      <Stack spacing={1.5} sx={{ p: 2 }}>
        <Typography component="h2" fontWeight={800}>
          Conversations
        </Typography>
        <FormControl fullWidth size="small">
          <InputLabel id="connection-filter-label">Connection</InputLabel>
          <Select
            label="Connection"
            labelId="connection-filter-label"
            value={selectedConnectionID}
            onChange={(event) => setSelectedConnectionID(event.target.value)}
          >
            <MenuItem value="">All connections</MenuItem>
            {connections.map((connection) => (
              <MenuItem key={connection.id} value={String(connection.id)}>
                {connection.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>
      <Divider />
      <Box sx={{ flex: 1, minHeight: 0, overflowY: "auto", px: 1, py: 1.5 }}>
        {connectionsQuery.isLoading || conversationsQuery.isLoading ? (
          <LoadingState label="Loading conversations" />
        ) : null}
        {connectionsQuery.isError ? (
          <Alert severity="error">{errorMessage(connectionsQuery.error)}</Alert>
        ) : null}
        {conversationsQuery.isError ? (
          <Alert
            severity="error"
            action={
              <Button color="inherit" onClick={() => void conversationsQuery.refetch()} size="small">
                Retry
              </Button>
            }
          >
            {errorMessage(conversationsQuery.error)}
          </Alert>
        ) : null}
        {deleteMutation.isError ? (
          <Alert severity="error" sx={{ mb: 1 }}>
            {errorMessage(deleteMutation.error)}
          </Alert>
        ) : null}
        {!connectionsQuery.isLoading &&
        !conversationsQuery.isLoading &&
        !conversationsQuery.isError &&
        conversations.length === 0 ? (
          <EmptyState
            title="No conversations"
            description="Create a conversation to start asking questions."
          />
        ) : null}
        <List disablePadding>
          {conversations.map((conversation) => (
            <ListItemButton
              key={conversation.id}
              onClick={() => {
                navigate(`/chat/${conversation.id}`);
                onSelect();
              }}
              selected={activeConversationID === conversation.id}
              sx={{
                borderRadius: 1,
                mb: 0.5,
                alignItems: "flex-start",
                gap: 1,
              }}
            >
              <ListItemText
                primary={conversationTitle(conversation)}
                primaryTypographyProps={{ fontWeight: 800, noWrap: true }}
                secondary={`Connection ${conversation.connection_id}`}
              />
              <Tooltip title="Delete conversation">
                <IconButton
                  aria-label={`Delete ${conversationTitle(conversation)}`}
                  color="error"
                  edge="end"
                  disabled={deleteMutation.isPending}
                  onClick={(event) => {
                    event.stopPropagation();
                    setDeletingConversation(conversation);
                  }}
                  size="small"
                >
                  {deleteMutation.isPending &&
                  deletingConversation?.id === conversation.id ? (
                    <CircularProgress color="inherit" size={18} />
                  ) : (
                    <DeleteOutlineOutlinedIcon fontSize="small" />
                  )}
                </IconButton>
              </Tooltip>
            </ListItemButton>
          ))}
        </List>
      </Box>
      <Divider />
      <Stack spacing={0.5} sx={{ p: 2 }}>
        <Typography color="text.secondary" variant="caption">
          Signed in
        </Typography>
        <Typography fontWeight={700} noWrap>
          {user?.name ?? "Unknown user"}
        </Typography>
      </Stack>
      <ConfirmDialog
        open={Boolean(deletingConversation)}
        title="Delete conversation"
        description={`Delete ${deletingConversation ? conversationTitle(deletingConversation) : "this chat"}?`}
        confirmLabel="Delete"
        destructive
        onCancel={() => setDeletingConversation(null)}
        onConfirm={() => {
          if (deletingConversation) {
            deleteMutation.mutate(deletingConversation);
          }
        }}
        isLoading={deleteMutation.isPending}
      />
    </Stack>
  );
}

function conversationIDFromPath(pathname: string) {
  const match = pathname.match(/^\/chat\/(\d+)$/);
  return match ? Number(match[1]) : null;
}

function conversationTitle(conversation: Conversation) {
  return conversation.title?.trim() || "New Chat";
}

type ThemeModeControlProps = {
  compact?: boolean;
};

export function ThemeModeControl({ compact = false }: ThemeModeControlProps) {
  const { modePreference, resolvedMode, setModePreference } = useThemeMode();

  return (
    <Stack direction="row" alignItems="center" spacing={1}>
      <Tooltip title={`Theme: ${resolvedMode}`}>
        <TuneOutlinedIcon color="action" fontSize="small" />
      </Tooltip>
      <Select
        aria-label="Theme mode"
        onChange={(event) =>
          setModePreference(event.target.value as ThemeModePreference)
        }
        size="small"
        value={modePreference}
        sx={{ minWidth: compact ? 44 : { xs: 44, sm: 132 } }}
      >
        <MenuItem value="system">
          <Stack direction="row" spacing={1} alignItems="center">
            <TuneOutlinedIcon fontSize="small" />
            <Box
              component="span"
              sx={{ display: compact ? "none" : { xs: "none", sm: "inline" } }}
            >
              System
            </Box>
          </Stack>
        </MenuItem>
        <MenuItem value="light">
          <Stack direction="row" spacing={1} alignItems="center">
            <LightModeOutlinedIcon fontSize="small" />
            <Box
              component="span"
              sx={{ display: compact ? "none" : { xs: "none", sm: "inline" } }}
            >
              Light
            </Box>
          </Stack>
        </MenuItem>
        <MenuItem value="dark">
          <Stack direction="row" spacing={1} alignItems="center">
            <DarkModeOutlinedIcon fontSize="small" />
            <Box
              component="span"
              sx={{ display: compact ? "none" : { xs: "none", sm: "inline" } }}
            >
              Dark
            </Box>
          </Stack>
        </MenuItem>
      </Select>
    </Stack>
  );
}
