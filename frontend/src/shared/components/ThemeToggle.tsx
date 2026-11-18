import { Moon, Sun } from "lucide-react";
import { resolveTheme, useThemeStore } from "../../features/theme/theme-store";

export function ThemeToggle() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const darkStart = useThemeStore((state) => state.darkStart);
  const lightStart = useThemeStore((state) => state.lightStart);
  const activeTheme = resolveTheme(preference, darkStart, lightStart);
  const nextTheme = activeTheme === "dark" ? "light" : "dark";

  return (
    <button
      type="button"
      className="theme-mode-toggle"
      onClick={() => setPreference(nextTheme)}
      aria-label={`Switch to ${nextTheme} mode`}
      title={`Switch to ${nextTheme} mode`}
    >
      {activeTheme === "dark" ? <Sun size={17} /> : <Moon size={17} />}
      <span>{activeTheme === "dark" ? "Dark" : "Light"}</span>
    </button>
  );
}
