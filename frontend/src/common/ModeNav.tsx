// ModeNav.tsx — the two jobs of the app: clean caches, uninstall software.
import { Icon } from "./Icon";
import { useI18n } from "../i18n/i18n";

export type Mode = "clean" | "analyze" | "uninstall";

const modes: { key: Mode; icon: string }[] = [
  { key: "clean", icon: "eraser" },
  { key: "analyze", icon: "pie-chart" },
  { key: "uninstall", icon: "trash" },
];

export function ModeNav({ mode, onMode }: { mode: Mode; onMode: (m: Mode) => void }) {
  const { t } = useI18n();
  return (
    <nav className="mode-nav" role="tablist" aria-label={t("nav.label")}>
      {modes.map((m) => (
        <button
          key={m.key}
          role="tab"
          aria-selected={mode === m.key}
          className={mode === m.key ? "active" : ""}
          onClick={() => onMode(m.key)}
        >
          <Icon name={m.icon} size={15} />
          {t(`nav.${m.key}`)}
        </button>
      ))}
    </nav>
  );
}
