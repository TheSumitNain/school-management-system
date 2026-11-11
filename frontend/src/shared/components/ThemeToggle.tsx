import { useThemeStore, type ThemePalette, type ThemePreference } from "../../features/theme/theme-store";

const paletteOptions: { value: ThemePalette; label: string }[] = [
  { value: "navy", label: "Classic Navy" },
  { value: "green", label: "Forest Green" },
  { value: "purple", label: "Royal Purple" },
  { value: "teal", label: "Ocean Teal" },
];

const appearanceOptions: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

export function ThemeToggle() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const palette = useThemeStore((state) => state.palette);
  const setPalette = useThemeStore((state) => state.setPalette);

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
      <label className="theme-mode-control">
        <span className="sr-only">Appearance mode</span>
        <select
          aria-label="Appearance mode"
          value={preference}
          onChange={(event) => setPreference(event.target.value as ThemePreference)}
        >
          {appearanceOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </label>
    </div>
  );
}
