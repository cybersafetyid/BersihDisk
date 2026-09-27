// ThemeSwitch.tsx — three-state theme toggle driven by the settings store.
import { Icon } from "./Icon";
import { useSettings, updateSettings } from "../lib/settings";
import { useI18n } from "../i18n/i18n";
import type { Theme } from "../lib/types";

const OPTIONS: { value: Theme; icon: string; labelKey: string }[] = [
  { value: "light", icon: "sun", labelKey: "theme.light" },
  { value: "dark", icon: "moon", labelKey: "theme.dark" },
  { value: "system", icon: "laptop", labelKey: "theme.system" },
];

export function ThemeSwitch() {
  const { theme } = useSettings();
  const { t } = useI18n();

  return (
    <div className="theme-switch" role="radiogroup" aria-label={t("theme.label")}>
      {OPTIONS.map((o) => (
        <button
          key={o.value}
          role="radio"
          aria-checked={theme === o.value}
          title={t("theme.preset", { name: t(o.labelKey) })}
          className={`theme-btn ${theme === o.value ? "active" : ""}`}
          onClick={() => updateSettings({ theme: o.value })}
        >
          <Icon name={o.icon} size={14} />
          <span className="theme-label">{t(o.labelKey)}</span>
        </button>
      ))}
    </div>
  );
}
