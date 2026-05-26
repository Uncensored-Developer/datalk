import AccountCircleOutlinedIcon from "@mui/icons-material/AccountCircleOutlined";
import ChatOutlinedIcon from "@mui/icons-material/ChatOutlined";
import CloseOutlinedIcon from "@mui/icons-material/CloseOutlined";
import DarkModeOutlinedIcon from "@mui/icons-material/DarkModeOutlined";
import DashboardOutlinedIcon from "@mui/icons-material/DashboardOutlined";
import LightModeOutlinedIcon from "@mui/icons-material/LightModeOutlined";
import LogoutOutlinedIcon from "@mui/icons-material/LogoutOutlined";
import MenuOutlinedIcon from "@mui/icons-material/MenuOutlined";
import PeopleOutlineOutlinedIcon from "@mui/icons-material/PeopleOutlineOutlined";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";
import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import TuneOutlinedIcon from "@mui/icons-material/TuneOutlined";
import Box from "@mui/material/Box";
import Divider from "@mui/material/Divider";
import Drawer from "@mui/material/Drawer";
import IconButton from "@mui/material/IconButton";
import List from "@mui/material/List";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import Toolbar from "@mui/material/Toolbar";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useState, type ReactNode } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import {
  useThemeMode,
  type ThemeModePreference,
} from "../../providers/ThemeModeProvider";
import type { UserRole } from "../../types/api";

const drawerWidth = 256;

type AppShellProps = {
  title: string;
  children: ReactNode;
};

const navItems: Array<{
  label: string;
  path: string;
  icon: ReactNode;
  roles?: UserRole[];
}> = [
  { label: "Overview", path: "/", icon: <DashboardOutlinedIcon /> },
  { label: "Profile", path: "/profile", icon: <AccountCircleOutlinedIcon /> },
  { label: "Chat", path: "/chat", icon: <ChatOutlinedIcon /> },
  { label: "Connections", path: "/connections", icon: <StorageOutlinedIcon /> },
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
  const { logout, user } = useAuth();
  const navigate = useNavigate();
  const visibleNavItems = navItems.filter(
    (item) => !item.roles || (user && item.roles.includes(user.role)),
  );

  const drawer = (
    <Stack sx={{ height: "100%" }}>
      <Toolbar sx={{ px: 2 }}>
        <Stack direction="row" alignItems="center" spacing={1.25}>
          <StorageOutlinedIcon color="primary" />
          <Typography component="div" fontWeight={800}>
            Datalk
          </Typography>
        </Stack>
      </Toolbar>
      <Divider />
      <List sx={{ px: 1, py: 1.5 }}>
        {visibleNavItems.map((item) => (
          <ListItemButton
            component={NavLink}
            key={item.path}
            onClick={() => setMobileOpen(false)}
            sx={{
              borderRadius: 1,
              mb: 0.5,
              "&.active": {
                bgcolor: "action.selected",
                color: "primary.main",
                "& .MuiListItemIcon-root": { color: "primary.main" },
              },
            }}
            to={item.path}
          >
            <ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        ))}
      </List>
      <Box sx={{ flex: 1 }} />
      <Divider />
      <Stack spacing={1} sx={{ p: 2 }}>
        <Typography color="text.secondary" variant="caption">
          Signed in
        </Typography>
        <Typography fontWeight={700} noWrap>
          {user?.name ?? "Unknown user"}
        </Typography>
      </Stack>
    </Stack>
  );

  return (
    <Box sx={{ display: "flex", minHeight: "100vh", bgcolor: "background.default" }}>
      <Box
        component="nav"
        sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}
      >
        <Drawer
          ModalProps={{ keepMounted: true }}
          onClose={() => setMobileOpen(false)}
          open={mobileOpen}
          sx={{
            display: { xs: "block", md: "none" },
            "& .MuiDrawer-paper": { width: drawerWidth },
          }}
          variant="temporary"
        >
          <Stack direction="row" justifyContent="flex-end" sx={{ p: 1 }}>
            <IconButton aria-label="Close navigation" onClick={() => setMobileOpen(false)}>
              <CloseOutlinedIcon />
            </IconButton>
          </Stack>
          {drawer}
        </Drawer>
        <Drawer
          open
          sx={{
            display: { xs: "none", md: "block" },
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
              aria-label="Open navigation"
              edge="start"
              onClick={() => setMobileOpen(true)}
              sx={{ display: { md: "none" } }}
            >
              <MenuOutlinedIcon />
            </IconButton>
            <Typography component="h1" fontWeight={800} sx={{ flex: 1 }} variant="h2">
              {title}
            </Typography>
            <ThemeModeControl />
            <Tooltip title={user ? user.email : "Not signed in"}>
              <IconButton aria-label="Current user" component={NavLink} to="/profile">
                <AccountCircleOutlinedIcon />
              </IconButton>
            </Tooltip>
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
