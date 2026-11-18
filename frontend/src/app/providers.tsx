import { useEffect, type PropsWithChildren } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useThemeStore, resolveTheme } from "../features/theme/theme-store";
import { ToastViewport } from "../shared/components/ToastViewport";

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
  const darkStart = useThemeStore((state) => state.darkStart);
  const lightStart = useThemeStore((state) => state.lightStart);

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const theme = resolveTheme(preference, darkStart, lightStart);
      document.documentElement.dataset.theme = theme;
      document.documentElement.dataset.palette = palette;
      document.documentElement.style.colorScheme = theme;
    };
    apply();

    if (preference === "system") media.addEventListener("change", apply);
    const scheduleTimer = preference === "schedule" ? window.setInterval(apply, 30_000) : undefined;
    return () => {
      if (preference === "system") media.removeEventListener("change", apply);
      if (scheduleTimer !== undefined) window.clearInterval(scheduleTimer);
    };
  }, [preference, palette, darkStart, lightStart]);

  return <>{children}<ToastViewport /></>;
}

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeController>{children}</ThemeController>
    </QueryClientProvider>
  );
}

export { queryClient };
