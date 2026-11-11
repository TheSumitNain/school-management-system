import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system";
export type ThemePalette = "navy" | "green" | "purple" | "teal";

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
      palette: "navy",
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
    },
  ),
);

export function resolveTheme(preference: ThemePreference): "light" | "dark" {
  if (preference !== "system") return preference;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
