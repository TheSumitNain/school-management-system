import { Moon, Sun } from "lucide-react";
import { useThemeStore, type ThemePalette } from "../../features/theme/theme-store";

const paletteOptions: { value: ThemePalette; label: string }[] = [
  { value: "navy", label: "Classic Navy" },
  { value: "green", label: "Forest Green" },
  { value: "purple", label: "Royal Purple" },
  { value: "teal", label: "Ocean Teal" },
];

export function ThemeToggle() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const palette = useThemeStore((state) => state.palette);
  const setPalette = useThemeStore((state) => state.setPalette);
  const isDark = preference === "dark" || (
    preference === "system" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );

  return (
    <div className="theme-controls" aria-label="Appearance settings">
      <label className="theme-palette-control">
        <span className="sr-only">Color theme</span>
        <select
          aria-label="Color theme"
          value={palette}
          onChange={(event) => setPalette(event.target.value as ThemePalette)}
        >
          {paletteOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </label>
      <button
        type="button"
        className="theme-toggle"
        onClick={() => setPreference(isDark ? "light" : "dark")}
        aria-label={`Switch to ${isDark ? "light" : "dark"} theme`}
        title={`Switch to ${isDark ? "light" : "dark"} appearance (current preference: ${preference})`}
      >
        {isDark ? <Sun size={16} /> : <Moon size={16} />}
        <span className="theme-toggle-label">{isDark ? "Light" : "Dark"}</span>
      </button>
    </div>
  );
}
