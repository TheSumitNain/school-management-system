import { Moon, Sun } from "lucide-react";
import { useThemeStore } from "../../features/theme/theme-store";

export function ThemeToggle() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const isDark = document.documentElement.dataset.theme === "dark";

  return (
    <button
      type="button"
      className="theme-toggle"
      onClick={() => setPreference(isDark ? "light" : "dark")}
      aria-label={`Switch to ${isDark ? "light" : "dark"} theme`}
      title={`Switch to ${isDark ? "light" : "dark"} theme (current preference: ${preference})`}
    >
      {isDark ? <Sun size={16} /> : <Moon size={16} />}
      <span className="theme-toggle-label">{isDark ? "Light" : "Dark"}</span>
    </button>
  );
}
