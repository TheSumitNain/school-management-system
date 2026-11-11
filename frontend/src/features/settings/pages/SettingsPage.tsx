import { Monitor, Moon, Sun } from "lucide-react";
import { useThemeStore, type ThemePalette, type ThemePreference } from "../../theme/theme-store";

const modes: { value: ThemePreference; label: string; description: string; icon: typeof Sun }[] = [
  { value: "light", label: "Light", description: "A bright, clear workspace.", icon: Sun },
  { value: "dark", label: "Dark", description: "A darker interface for low-light use.", icon: Moon },
  { value: "system", label: "System", description: "Follow your device appearance setting.", icon: Monitor },
];
const palettes: { value: ThemePalette; label: string }[] = [
  { value: "midnight", label: "Midnight Blue" },
  { value: "green", label: "Forest Green" },
  { value: "graphite", label: "Graphite Black" },
  { value: "teal", label: "Ocean Teal" },
  { value: "rose", label: "Rose Blush" },
  { value: "sky", label: "Sky Blue" },
  { value: "amber", label: "Warm Amber" },
  { value: "slate", label: "Soft Slate" },
];

export function SettingsPage() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const palette = useThemeStore((state) => state.palette);
  const setPalette = useThemeStore((state) => state.setPalette);

  return (
    <div className="content appearance-page">
      <div className="heading">
        <div>
          <span className="eyebrow">PREFERENCES</span>
          <h1>Settings</h1>
          <p>Manage how SchoolDesk looks on this device.</p>
        </div>
      </div>
      <section className="appearance-section" aria-labelledby="appearance-mode-title">
        <h2 id="appearance-mode-title">Appearance mode</h2>
        <p>Choose Light or Dark, or let SchoolDesk follow your device setting. Your choice is saved automatically.</p>
        <div className="appearance-options">
          {modes.map(({ value, label, description, icon: Icon }) => (
            <button key={value} type="button" className="appearance-option" aria-pressed={preference === value} onClick={() => setPreference(value)}>
              <Icon size={18} />
              <strong>{label}</strong>
              <small>{description}</small>
            </button>
          ))}
        </div>
      </section>
      <section className="appearance-section" aria-labelledby="brand-palette-title">
        <h2 id="brand-palette-title">Brand palette</h2>
        <p>Select the accent color used across navigation, buttons and highlights.</p>
        <div className="palette-options">
          {palettes.map(({ value, label }) => (
            <button key={value} type="button" className="palette-option" aria-pressed={palette === value} onClick={() => setPalette(value)}>
              <span className={`palette-swatch palette-swatch-${value}`} aria-hidden="true" />
              {label}
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
