import { useMemo, useState } from "react";
import { Dropdown } from "../common/Dropdown";
import { Icon } from "../common/Icon";
import { useI18n } from "../i18n/i18n";
import type { CategoryUI } from "../lib/types";

type View = "grid" | "list";
const VIEW_KEY = "bersihdisk.categoryView";

function loadView(): View {
  try {
    return localStorage.getItem(VIEW_KEY) === "list" ? "list" : "grid";
  } catch {
    return "grid"; // storage can be blocked; the default still works
  }
}

interface Props {
  categories: CategoryUI[];
  selected: Set<string>;
  onToggle: (id: string) => void;
  /** Replaces the whole selection (used by the bulk buttons). */
  onReplace: (ids: Set<string>) => void;
}

export function CategoryPanel({ categories, selected, onToggle, onReplace }: Props) {
  const { t } = useI18n();
  const [view, setView] = useState<View>(loadView);
  const [query, setQuery] = useState("");

  const chooseView = (next: View) => {
    setView(next);
    try {
      localStorage.setItem(VIEW_KEY, next);
    } catch {
      /* not persisted; still applies for this session */
    }
  };

  // A category "has a note" when the backend rates any of its locations above safe.
  const riskOf = (c: CategoryUI) => {
    if (c.risk === "safe") return "";
    const risk = t(`categories.${c.id}.risk`);
    return risk !== `categories.${c.id}.risk` ? risk : t("safety.reason.categoryNote");
  };
  const allIds = new Set(categories.map((c) => c.id));
  const noNoteIds = new Set(categories.filter((c) => c.risk === "safe").map((c) => c.id));
  const sameAs = (ids: Set<string>) => ids.size === selected.size && [...ids].every((id) => selected.has(id));

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return categories;
    return categories.filter((c) => {
      const name = t(`categories.${c.id}.name`).toLowerCase();
      const desc = t(`categories.${c.id}.desc`).toLowerCase();
      return name.includes(q) || desc.includes(q) || c.id.toLowerCase().includes(q);
    });
  }, [categories, query, t]);

  return (
    <>
      <div className="category-toolbar">
        <div className="category-toolbar-left">
          <Dropdown
            label={t("category.select")}
            icon="list-checks"
            items={[
              { key: "all", icon: "list-checks", label: t("category.selectAll"), checked: sameAs(allIds), onSelect: () => onReplace(allIds) },
              { key: "none", icon: "x", label: t("category.selectNone"), checked: selected.size === 0, onSelect: () => onReplace(new Set()) },
              { key: "safe", icon: "shield-check", label: t("category.selectNoNote"), checked: sameAs(noNoteIds), onSelect: () => onReplace(noNoteIds) },
            ]}
          />
          <span className="category-count-badge">
            {t("category.selectedCount", { count: selected.size, total: categories.length })}
          </span>
        </div>

        <div className="category-toolbar-right">
          <div className="category-search">
            <Icon name="search" size={13} className="search-icon" />
            <input
              type="text"
              placeholder={t("category.searchPlaceholder")}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="search-input"
            />
            {query && (
              <button className="search-clear" onClick={() => setQuery("")} title="Clear">
                <Icon name="x" size={12} />
              </button>
            )}
          </div>

          <div className="view-toggle" role="group" aria-label={t("category.view")}>
            {(["grid", "list"] as const).map((v) => (
              <button
                key={v}
                className={view === v ? "active" : ""}
                onClick={() => chooseView(v)}
                aria-pressed={view === v}
                aria-label={t(v === "grid" ? "category.viewGrid" : "category.viewList")}
                data-tip={t(v === "grid" ? "category.viewGrid" : "category.viewList")}
              >
                <Icon name={v === "grid" ? "layout-grid" : "list"} size={16} />
              </button>
            ))}
          </div>
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="results-empty-filter">
          <Icon name="search" size={24} />
          <p>{t("uninstall.empty")}</p>
        </div>
      ) : (
        <div className={`category-grid ${view === "list" ? "as-list" : ""}`}>
          {filtered.map((c, i) => {
            const active = selected.has(c.id);
            const desc = t(`categories.${c.id}.desc`);
            const risk = riskOf(c);
            return (
              <button
                key={c.id}
                className={`category-card ${active ? "active" : ""} ${c.optIn ? "optin" : ""}`}
                style={{ animationDelay: `${i * 30}ms` }}
                onClick={() => onToggle(c.id)}
                aria-pressed={active}
                title={risk || desc}
              >
                <div className="category-top">
                  <div className="category-icon-badge">
                    <Icon name={c.icon} size={20} />
                  </div>
                  <div className={`check-indicator ${active ? "on" : ""}`} aria-hidden="true">
                    {active && <Icon name="check" size={12} />}
                  </div>
                </div>
                <div className="category-name">{t(`categories.${c.id}.name`)}</div>
                <div className="category-desc">{desc}</div>
                <div className="category-badges">
                  {c.optIn && <span className="badge badge-optional">{t("category.optional")}</span>}
                  {risk && (
                    <span className={`badge badge-${c.risk === "danger" ? "risk" : "caution"}`} title={risk}>
                      <Icon name="triangle-alert" size={11} /> {t("category.hasNote")}
                    </span>
                  )}
                </div>
              </button>
            );
          })}
        </div>
      )}
    </>
  );
}

