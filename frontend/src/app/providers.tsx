import { useEffect, type PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useThemeStore, resolveTheme } from "../features/theme/theme-store";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: (failureCount, error) => {
        const status = typeof error === "object" && error !== null && "status" in error
          ? Number((error as { status?: unknown }).status)
          : 0;
        return failureCount < 1 && (status === 0 || status >= 500);
      },
    },
    mutations: { retry: 0 },
  },
});

function ThemeController({ children }: PropsWithChildren) {
  const preference = useThemeStore((state) => state.preference);
  const palette = useThemeStore((state) => state.palette);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const theme = resolveTheme(preference);
      document.documentElement.dataset.theme = theme;
      document.documentElement.dataset.palette = palette;
      document.documentElement.style.colorScheme = theme;
    };
    apply();
    if (preference !== "system") return;
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [preference, palette]);

  return children;
}

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeController>{children}</ThemeController>
    </QueryClientProvider>
  );
}

export { queryClient };
