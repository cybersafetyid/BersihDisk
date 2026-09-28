// UninstallPage.tsx — everything installed that BersihDisk knows how to remove:
// applications, runtimes/SDKs and packages of the package managers. Picking one
// opens its plan for review; nothing is removed from this page directly.
import { useCallback, useEffect, useMemo, useState } from "react";
import { Icon } from "../common/Icon";
import { PlanModal } from "./PlanModal";
import { AppLogo } from "./AppLogo";
import { shortPath } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import * as api from "../backend";
import type { InstalledItem, InstalledKind } from "../lib/types";

const kinds: { key: InstalledKind; icon: string }[] = [
  { key: "app", icon: "monitor" },
  { key: "runtime", icon: "cpu" },
  { key: "package", icon: "package" },
];

const PAGE = 150; // rows rendered at once; a machine can have hundreds of packages

interface Props {
  onNotify: (message: string, detail?: string, variant?: "success" | "error" | "info") => void;
  /** Called after a plan ran, so the drive cards can refresh their free space. */
  onFreed: () => void;
}

/** Placeholder rows with the same anatomy as a real row: icon, two text lines, button. */
function InstalledSkeleton() {
  const widths = [38, 52, 30, 46, 34, 44]; // varied so it reads as content, not a pattern
  return (
    <div className="installed-list" aria-busy="true" aria-live="polite">
      {widths.map((w, i) => (
        <div key={i} className="installed-row skeleton-item" style={{ animationDelay: `${i * 60}ms` }}>
          <span className="installed-icon skeleton" />
          <span className="installed-main">
            <span className="skeleton skeleton-line" style={{ width: `${w}%`, height: 14 }} />
            <span className="skeleton skeleton-line" style={{ width: `${w + 18}%`, height: 10 }} />
          </span>
          <span className="skeleton skeleton-btn" />
        </div>
      ))}
    </div>
  );
}

export function UninstallPage({ onNotify, onFreed }: Props) {
  const { t } = useI18n();
  const [items, setItems] = useState<InstalledItem[]>([]);
  const [unavailable, setUnavailable] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [kind, setKind] = useState<InstalledKind>("app");
  const [provider, setProvider] = useState("all");
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(PAGE);
  const [open, setOpen] = useState<InstalledItem | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api.listPackages()
      .then((r) => { setItems(r.packages); setUnavailable(r.unavailable); })
      .catch((e) => onNotify(t("uninstall.loadFailed"), String(e), "error"))
      .finally(() => setLoading(false));
  }, [onNotify, t]);

  useEffect(load, [load]);
  useEffect(() => setLimit(PAGE), [kind, provider, query]);
  useEffect(() => setProvider("all"), [kind]);

  const ofKind = useMemo(() => items.filter((i) => i.kind === kind), [items, kind]);
  const providers = useMemo(() => [...new Set(ofKind.map((i) => i.provider))].sort(), [ofKind]);
  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    return ofKind.filter(
      (i) => (provider === "all" || i.provider === provider) && (!q || i.name.toLowerCase().includes(q)),
    );
  }, [ofKind, provider, query]);

  const closePlan = (changed: boolean) => {
    setOpen(null);
    if (changed) {
      onFreed();
      load(); // what was removed must leave the list
    }
  };

  return (
    <section className="fade-in">
      <h2 className="section-title">{t("uninstall.title")}</h2>
      <p className="page-lead">{t("uninstall.lead")}</p>

      <div className="uninstall-toolbar">
        <div className="view-toggle" role="tablist" aria-label={t("uninstall.kinds")}>
          {kinds.map((k) => (
            <button
              key={k.key}
              role="tab"
              aria-selected={kind === k.key}
              className={`kind-tab ${kind === k.key ? "active" : ""}`}
              onClick={() => setKind(k.key)}
            >
              <Icon name={k.icon} size={15} /> {t(`uninstall.kind.${k.key}`)}
              <span className="kind-count">{items.filter((i) => i.kind === k.key).length}</span>
            </button>
          ))}
        </div>
        <label className="search-box">
          <Icon name="search" size={15} />
          <input
            type="search"
            value={query}
            placeholder={t("uninstall.search")}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <button className="btn btn-ghost btn-small btn-refresh" onClick={load} disabled={loading}>
          <Icon name="refresh" size={14} /> {t("uninstall.refresh")}
        </button>
      </div>

      {providers.length > 1 && (
        <div className="provider-chips">
          {["all", ...providers].map((p) => (
            <button key={p} className={`chip ${provider === p ? "active" : ""}`} onClick={() => setProvider(p)}>
              {p === "all" ? t("uninstall.all") : t(`uninstall.provider.${p}`)}
            </button>
          ))}
        </div>
      )}

      {unavailable.length > 0 && (
        <div className="risk-banner">
          <Icon name="triangle-alert" size={16} />
          <span>{t("uninstall.unavailable", { list: unavailable.map((p) => t(`uninstall.provider.${p}`)).join(", ") })}</span>
        </div>
      )}

      {loading ? (
        <InstalledSkeleton />
      ) : shown.length === 0 ? (
        <p className="empty">{t("uninstall.empty")}</p>
      ) : (
        <div className="installed-list">
          {shown.slice(0, limit).map((i) => (
            <div key={i.id} className="installed-row">
              <span className="installed-icon">
                {i.provider === "apps" || i.provider === "appx" ? (
                  <AppLogo id={i.id} fallback={i.icon} />
                ) : (
                  <Icon name={i.icon} size={20} />
                )}
              </span>
              <span className="installed-main">
                <span className="installed-name">
                  {i.name}
                  {i.version && <span className="installed-version">{i.version}</span>}
                </span>
                <span className="item-note">
                  {t(`uninstall.provider.${i.provider}`)}
                  {i.path ? ` · ${shortPath(i.path, 60)}` : ""}
                  {(i.notes ?? []).map((n) => ` · ${t(`uninstall.note.${n}`)}`).join("")}
                </span>
              </span>
              <button className="btn btn-ghost btn-small btn-uninstall" onClick={() => setOpen(i)}>
                {t("uninstall.action")}
              </button>
            </div>
          ))}
          {shown.length > limit && (
            <button className="btn btn-ghost" onClick={() => setLimit((n) => n + PAGE)}>
              {t("uninstall.showMore", { count: shown.length - limit })}
            </button>
          )}
        </div>
      )}

      {open && <PlanModal item={open} onClose={closePlan} />}
    </section>
  );
}
