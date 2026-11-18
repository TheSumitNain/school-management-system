import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system" | "schedule";
export type ThemePalette = "cobalt" | "alpine" | "evergreen" | "lagoon" | "iris" | "amethyst" | "blush" | "terracotta" | "saffron" | "dune" | "glacier" | "obsidian" | "graphite" | "ruby" | "copper";

type ThemeState = {
  preference: ThemePreference;
  palette: ThemePalette;
  darkStart: string;
  lightStart: string;
  setPreference: (preference: ThemePreference) => void;
  setPalette: (palette: ThemePalette) => void;
  setSchedule: (darkStart: string, lightStart: string) => void;
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      preference: "system",
      palette: "cobalt",
      darkStart: "19:00",
      lightStart: "06:00",
      setPreference: (preference) => set({ preference }),
      setPalette: (palette) => set({ palette }),
      setSchedule: (darkStart, lightStart) => set({ darkStart, lightStart }),
    }),
    {
      name: "schooldesk.theme",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        preference: state.preference,
        palette: state.palette,
        darkStart: state.darkStart,
        lightStart: state.lightStart,
      }),
      version: 5,
      migrate: (persistedState) => {
        const state = persistedState as { preference?: string; palette?: string; darkStart?: string; lightStart?: string };
        const validPalettes: ThemePalette[] = ["cobalt", "alpine", "evergreen", "lagoon", "iris", "amethyst", "blush", "terracotta", "saffron", "dune", "glacier", "obsidian", "graphite", "ruby", "copper"];
        const validPreferences: ThemePreference[] = ["light", "dark", "system", "schedule"];
        return {
          ...state,
          preference: validPreferences.includes(state.preference as ThemePreference) ? state.preference as ThemePreference : "system",
          palette: validPalettes.includes(state.palette as ThemePalette) ? state.palette as ThemePalette : "cobalt",
          darkStart: /^([01]\d|2[0-3]):[0-5]\d$/.test(state.darkStart ?? "") ? state.darkStart! : "19:00",
          lightStart: /^([01]\d|2[0-3]):[0-5]\d$/.test(state.lightStart ?? "") ? state.lightStart! : "06:00",
        };
      },
    },
  ),
);

export function resolveTheme(
  preference: ThemePreference,
  darkStart = "19:00",
  lightStart = "06:00",
  now = new Date(),
): "light" | "dark" {
  if (preference === "light" || preference === "dark") return preference;
  if (preference === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  const toMinutes = (time: string) => {
    const [hours, minutes] = time.split(":").map(Number);
    return hours * 60 + minutes;
  };
  const darkMinute = toMinutes(darkStart);
  const lightMinute = toMinutes(lightStart);
  if (darkMinute === lightMinute) {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  const currentMinute = now.getHours() * 60 + now.getMinutes();
  const isDark = darkMinute < lightMinute
    ? currentMinute >= darkMinute && currentMinute < lightMinute
    : currentMinute >= darkMinute || currentMinute < lightMinute;
  return isDark ? "dark" : "light";
}
