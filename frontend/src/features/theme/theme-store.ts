import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

export type ThemePreference = "light" | "dark" | "system";
type ThemeState = {
  preference: ThemePreference;
  setPreference: (preference: ThemePreference) => void;
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      preference: "system",
      setPreference: (preference) => set({ preference }),
    }),
    {
      name: "schooldesk.theme",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ preference: state.preference }),
    },
  ),
);

export function resolveTheme(preference: ThemePreference): "light" | "dark" {
  if (preference !== "system") return preference;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
