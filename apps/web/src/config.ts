export const apiBaseUrl = normalizeBaseUrl(
  import.meta.env.VITE_API_BASE_URL ?? "/api",
);

function normalizeBaseUrl(value: string) {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}
