import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system";
export type ThemePalette = "cobalt" | "alpine" | "evergreen" | "lagoon" | "iris" | "amethyst" | "blush" | "terracotta" | "saffron" | "dune" | "glacier" | "obsidian" | "graphite" | "ruby" | "copper";

type ThemeState = {
  preference: ThemePreference;
  palette: ThemePalette;
  setPreference: (preference: ThemePreference) => void;
  setPalette: (palette: ThemePalette) => void;
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      preference: "system",
      palette: "cobalt",
      setPreference: (preference) => set({ preference }),
      setPalette: (palette) => set({ palette }),
    }),
    {
      name: "schooldesk.theme",
      storage: createJSONStorage(() => localStorage),
      // Keep compatibility with existing users who only have a saved mode.
      partialize: (state) => ({
        preference: state.preference,
        palette: state.palette,
      }),
      version: 4,
      migrate: (persistedState) => {
        const state = persistedState as { preference?: ThemePreference; palette?: string };
        const validPalettes: ThemePalette[] = ["cobalt", "alpine", "evergreen", "lagoon", "iris", "amethyst", "blush", "terracotta", "saffron", "dune", "glacier", "obsidian", "graphite", "ruby", "copper"];
        return { ...state, palette: validPalettes.includes(state.palette as ThemePalette) ? state.palette as ThemePalette : "cobalt" };
      },
    },
  ),
);

export function resolveTheme(preference: ThemePreference): "light" | "dark" {
  if (preference !== "system") return preference;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
