import { Clock3, Monitor, Moon, Sun } from "lucide-react";
import { notifyToast } from "../../../shared/notifications";
import { useThemeStore, type ThemePalette, type ThemePreference } from "../../theme/theme-store";

const modes: { value: ThemePreference; label: string; description: string; icon: typeof Sun }[] = [
  { value: "light", label: "Light", description: "A bright, clear workspace.", icon: Sun },
  { value: "dark", label: "Dark", description: "A darker interface for low-light use.", icon: Moon },
  { value: "system", label: "System", description: "Follow your device appearance setting.", icon: Monitor },
  { value: "schedule", label: "Automatic schedule", description: "Switch between Light and Dark at set times.", icon: Clock3 },
];
const palettes: { value: ThemePalette; label: string }[] = [
  { value: "cobalt", label: "Cobalt" },
  { value: "alpine", label: "Alpine" },
  { value: "evergreen", label: "Evergreen" },
  { value: "lagoon", label: "Lagoon" },
  { value: "iris", label: "Iris" },
  { value: "amethyst", label: "Amethyst" },
  { value: "blush", label: "Blush" },
  { value: "terracotta", label: "Terracotta" },
  { value: "saffron", label: "Saffron" },
  { value: "dune", label: "Dune" },
  { value: "glacier", label: "Glacier" },
  { value: "obsidian", label: "Obsidian" },
  { value: "graphite", label: "Graphite" },
  { value: "ruby", label: "Ruby" },
  { value: "copper", label: "Copper" },
];

export function SettingsPage() {
  const preference = useThemeStore((state) => state.preference);
  const setPreference = useThemeStore((state) => state.setPreference);
  const palette = useThemeStore((state) => state.palette);
  const setPalette = useThemeStore((state) => state.setPalette);
  const darkStart = useThemeStore((state) => state.darkStart);
  const lightStart = useThemeStore((state) => state.lightStart);
  const setSchedule = useThemeStore((state) => state.setSchedule);
  const scheduleValid = darkStart !== lightStart;

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
        <p>Choose Light or Dark, follow your device setting, or let SchoolDesk switch automatically on a daily schedule. Your choice is saved automatically.</p>
        <div className="appearance-options">
          {modes.map(({ value, label, description, icon: Icon }) => (
            <button key={value} type="button" className="appearance-option" aria-pressed={preference === value} onClick={() => { setPreference(value); notifyToast("success", "Appearance mode updated."); }}>
              <Icon size={18} />
              <strong>{label}</strong>
              <small>{description}</small>
            </button>
          ))}
        </div>
        {preference === "schedule" && (
          <div className="theme-schedule" aria-label="Automatic theme schedule">
            <div className="theme-schedule-field">
              <label htmlFor="theme-dark-start">Turn Dark Mode on</label>
              <input id="theme-dark-start" type="time" value={darkStart} onChange={(event) => { setSchedule(event.target.value, lightStart); notifyToast("success", "Theme schedule updated."); }} />
            </div>
            <div className="theme-schedule-field">
              <label htmlFor="theme-light-start">Turn Light Mode on</label>
              <input id="theme-light-start" type="time" value={lightStart} onChange={(event) => { setSchedule(darkStart, event.target.value); notifyToast("success", "Theme schedule updated."); }} />
            </div>
            <p className={scheduleValid ? "theme-schedule-help" : "theme-schedule-help theme-schedule-error"}>
              {scheduleValid
                ? "Uses this device's local time and updates automatically, even if the Settings page is closed."
                : "Choose two different times so the automatic schedule can switch correctly."}
            </p>
          </div>
        )}
      </section>
      <section className="appearance-section" aria-labelledby="brand-palette-title">
        <h2 id="brand-palette-title">Brand palette</h2>
        <p>Choose a coordinated color system for the workspace, header, navigation and interface accents.</p>
        <div className="palette-options">
          {palettes.map(({ value, label }) => (
            <button key={value} type="button" className="palette-option" aria-pressed={palette === value} onClick={() => { setPalette(value); notifyToast("success", "Brand palette updated."); }}>
              <span className={`palette-swatch palette-swatch-${value}`} aria-hidden="true" />
              {label}
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
