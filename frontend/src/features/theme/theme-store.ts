import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system";
export type ThemePalette = "ruby" | "green" | "copper" | "teal" | "rose" | "sky" | "amber" | "slate";

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
      palette: "ruby",
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
        const state = persistedState as Partial<ThemeState>;
        const palette = state.palette === "navy"
          ? "ruby"
          : state.palette === "purple"
            ? "copper"
            : state.palette ?? "ruby";
        return { ...state, palette };
      },
    },
  ),
);

export function resolveTheme(preference: ThemePreference): "light" | "dark" {
  if (preference !== "system") return preference;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
