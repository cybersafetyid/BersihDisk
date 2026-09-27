// settings.ts — persisted app settings, shared as one external store so theme,
// language and accent stay in sync everywhere without prop drilling.
import { useSyncExternalStore } from "react";
import type { Lang, Theme } from "./types";

export type AccentKey = "blue" | "violet" | "teal" | "amber" | "rose";
export type IconKey = "default" | "midnight" | "citrus";

export interface Settings {
  lang: Lang;
  theme: Theme;
  accent: AccentKey;
  autoUpdate: boolean;
  icon: IconKey;
}

const STORAGE_KEY = "bersihdisk.settings.v1";

const fallback: Settings = {
  lang: "id",
  theme: "system",
  accent: "blue",
  autoUpdate: false,
  icon: "default",
};

function load(): Settings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;
    const saved = JSON.parse(raw) as Partial<Settings>;
    return {
      lang: saved.lang === "en" ? "en" : "id",
      theme: saved.theme === "light" || saved.theme === "dark" ? saved.theme : "system",
      accent: saved.accent ?? fallback.accent,
      autoUpdate: Boolean(saved.autoUpdate),
      icon: saved.icon ?? fallback.icon,
    };
  } catch {
    return fallback;
  }
}

let current: Settings = load();
const listeners = new Set<() => void>();

function emit() {
  for (const l of listeners) l();
}

export function getSettings(): Settings {
  return current;
}

export function updateSettings(patch: Partial<Settings>) {
  current = { ...current, ...patch };
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(current));
  } catch {
    // Storage can be full or blocked; the in-memory value still applies.
  }
  emit();
}

/** Removes every stored setting and returns the previous values. */
export function resetSettings(): Settings {
  const previous = current;
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // ignore
  }
  current = { ...fallback };
  emit();
  return previous;
}

export function useSettings(): Settings {
  return useSyncExternalStore(
    (cb) => {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
    getSettings,
    getSettings,
  );
}
