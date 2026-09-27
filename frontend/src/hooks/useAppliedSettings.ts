// useAppliedSettings.ts — pushes persisted settings onto the document: theme,
// accent colour, interface language and window title.
import { useEffect } from "react";
import { useSettings } from "../lib/settings";
import type { Settings } from "../lib/settings";
import { translate } from "../i18n/i18n";

export function useAppliedSettings(): Settings {
  const settings = useSettings();

  useEffect(() => {
    const root = document.documentElement;
    const dark = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      root.dataset.theme = settings.theme === "system" ? (dark.matches ? "dark" : "light") : settings.theme;
    };
    apply();
    if (settings.theme !== "system") return;
    dark.addEventListener("change", apply);
    return () => dark.removeEventListener("change", apply);
  }, [settings.theme]);

  useEffect(() => {
    document.documentElement.dataset.accent = settings.accent;
  }, [settings.accent]);

  useEffect(() => {
    document.documentElement.lang = settings.lang;
    document.title = translate(settings.lang, "app.title");
  }, [settings.lang]);

  return settings;
}
