// CategoryPanel.tsx — category card grid with official SVG icons.
// Names and descriptions come from the locale catalog, keyed by category ID.
import { Icon } from "../common/Icon";
import { useI18n } from "../i18n/i18n";
import type { CategoryUI } from "../lib/types";

interface Props {
  categories: CategoryUI[];
  selected: Set<string>;
  onToggle: (id: string) => void;
}

export function CategoryPanel({ categories, selected, onToggle }: Props) {
  const { t } = useI18n();

  return (
    <div className="category-grid">
      {categories.map((c, i) => {
        const active = selected.has(c.id);
        const desc = t(`categories.${c.id}.desc`);
        const risk = t(`categories.${c.id}.risk`);
        const hasRisk = risk !== `categories.${c.id}.risk`;
        return (
          <button
            key={c.id}
            className={`category-card ${active ? "active" : ""} ${c.optIn ? "optin" : ""}`}
            style={{ animationDelay: `${i * 40}ms` }}
            onClick={() => onToggle(c.id)}
            aria-pressed={active}
            title={hasRisk ? risk : desc}
          >
            <div className="category-top">
              <span className="category-icon">
                <Icon name={c.icon} size={22} />
              </span>
              <span className={`check ${active ? "on" : ""}`}>✓</span>
            </div>
            <div className="category-name">{t(`categories.${c.id}.name`)}</div>
            <div className="category-desc">{desc}</div>
            <div className="category-badges">
              {c.optIn && <span className="badge badge-optional">{t("category.optional")}</span>}
              {hasRisk && (
                <span className="badge badge-risk" title={risk}>
                  <Icon name="triangle-alert" size={11} /> {t("category.hasNote")}
                </span>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
}
