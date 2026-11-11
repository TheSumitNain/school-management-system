import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system";
export type ThemePalette = "midnight" | "green" | "graphite" | "teal" | "rose" | "sky" | "amber" | "slate";

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
      palette: "midnight",
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
      version: 1,
      migrate: (persistedState) => {
        const state = persistedState as { preference?: ThemePreference; palette?: string };
        const validPalettes: ThemePalette[] = ["midnight", "green", "graphite", "teal", "rose", "sky", "amber", "slate"];
        const palette: ThemePalette = state.palette === "navy" || state.palette === "ruby"
          ? "midnight"
          : state.palette === "purple" || state.palette === "copper"
            ? "graphite"
            : validPalettes.includes(state.palette as ThemePalette)
              ? state.palette as ThemePalette
              : "midnight";
        return { ...state, palette };
      },
    },
  ),
);

export function resolveTheme(preference: ThemePreference): "light" | "dark" {
  if (preference !== "system") return preference;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
