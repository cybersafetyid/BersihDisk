// AnalyzePage.tsx — the "scan drive" view: pick a drive, see what takes up the
// most space as a DaisyDisk-style chart, drill into folders, and hand the biggest
// items to the same guarded delete flow the cleaner uses. Sensitive locations
// keep their risk badge and a blocked one can never be selected.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Icon } from "../common/Icon";
import { RiskBadge } from "../common/RiskBadge";
import { DriveSelector } from "../drives/DriveSelector";
import { Sunburst, paletteColor, type Ring } from "./Sunburst";
import type { RiskEntry } from "../results/ConfirmModal";
import { formatSize, formatNumber, formatPercent, shortPath } from "../lib/format";
import { isInside } from "../lib/paths";
import * as api from "../backend";
import { useI18n } from "../i18n/i18n";
import type { AnalyzeEntry, AnalyzeProgress, DriveUI, Level } from "../lib/types";

/** What the UI remembers about every entry it has seen, for selection maths. */
interface Known {
  size: number;
  level: Level;
  reasons?: string[];
}

interface Props {
  drives: DriveUI[];
  /** Bumped after a delete so the current folder is re-measured. */
  refreshKey: number;
  onNotify: (message: string, detail?: string, variant?: "success" | "error" | "info") => void;
  onReveal: (path: string) => void;
  onDelete: (paths: string[], sizes: number[], risks: RiskEntry[], totalBytes: number) => void;
}

const LIST_CAP = 150;
const baseName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() ?? p;

export function AnalyzePage({ drives, refreshKey, onNotify, onReveal, onDelete }: Props) {
  const { t, p } = useI18n();

  const [drive, setDrive] = useState<string | null>(null);
  const [rings, setRings] = useState<Ring[]>([]);
  const [analyzing, setAnalyzing] = useState(false);
  const [progress, setProgress] = useState<AnalyzeProgress | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [known, setKnown] = useState<Map<string, Known>>(new Map());
  const [hovered, setHovered] = useState<AnalyzeEntry | null>(null);
  const [showAll, setShowAll] = useState(false);

  // The path a finished listing is expected to belong to, and how many rings to
  // keep. A stale result (from a listing we replaced) is ignored.
  const pending = useRef<{ path: string; truncateTo: number } | null>(null);
  const ringsRef = useRef<Ring[]>([]);
  ringsRef.current = rings;
  const refreshSeen = useRef(refreshKey);

  // navigate starts a listing of `path`, keeping the first `truncateTo` rings.
  const navigate = useCallback((path: string, truncateTo: number) => {
    pending.current = { path, truncateTo };
    setRings((rs) => rs.slice(0, truncateTo));
    setError(null);
    setAnalyzing(true);
    setProgress({ path, dirs: 0, total: 0 });
    api.startAnalyze(path).catch((e) => {
      setAnalyzing(false);
      setError(String(e));
    });
  }, []);

  // Backend events.
  useEffect(() => {
    const offProg = api.onAnalyzeProgress((prog) => setProgress(prog));
    const offDone = api.onAnalyzeFinished((res) => {
      const want = pending.current;
      if (!want || res.path !== want.path) return; // replaced by a newer listing
      pending.current = null;
      setAnalyzing(false);
      setProgress(null);
      if (res.error) {
        setError(res.error);
        onNotify(t("analyze.failed"), res.error, "error");
        return;
      }
      const ring: Ring = { path: res.path, totalBytes: res.totalBytes, entries: res.entries, partial: res.partial };
      setRings((rs) => [...rs.slice(0, want.truncateTo), ring]);
      // Remember sizes and risk levels for the selection maths and confirmation.
      setKnown((prev) => {
        const next = new Map(prev);
        for (const e of res.entries) next.set(e.path, { size: e.size, level: e.level, reasons: e.reasons });
        return next;
      });
      setShowAll(false);
    });
    return () => { offProg(); offDone(); };
  }, [onNotify, t]);

  // After a delete, re-measure the folder in view and forget what is gone.
  useEffect(() => {
    if (refreshSeen.current === refreshKey) return;
    refreshSeen.current = refreshKey;
    const rs = ringsRef.current;
    if (rs.length === 0) return;
    setSelected(new Set());
    navigate(rs[rs.length - 1].path, rs.length - 1);
  }, [refreshKey, navigate]);

  const chooseDrive = (mount: string) => {
    setDrive(mount);
    setSelected(new Set());
    setKnown(new Map());
    setRings([]);
    navigate(mount, 0);
  };

  const current = rings.length > 0 ? rings[rings.length - 1] : null;

  const drill = (entry: AnalyzeEntry, ringIndex: number) => navigate(entry.path, ringIndex + 1);
  const up = () => setRings((rs) => rs.slice(0, -1));
  const toCrumb = (index: number) => setRings((rs) => rs.slice(0, index + 1));

  const toggle = (entry: AnalyzeEntry) => {
    // A blocked location can never be deleted, and a skipped one was never
    // measured (its size is unknown), so neither is selectable.
    if (entry.level === "blocked" || entry.skipped) return;
    setSelected((s) => {
      const n = new Set(s);
      n.has(entry.path) ? n.delete(entry.path) : n.add(entry.path);
      return n;
    });
  };
  const selectMany = (paths: string[], on: boolean) =>
    setSelected((s) => {
      const n = new Set(s);
      for (const x of paths) (on ? n.add(x) : n.delete(x));
      return n;
    });

  const safePaths = useMemo(
    () => (current ? current.entries.filter((e) => e.level === "safe").map((e) => e.path) : []),
    [current],
  );
  const largePaths = useMemo(
    () => (current ? current.entries.filter((e) => e.level === "safe" && e.size >= 100 * 1024 * 1024).map((e) => e.path) : []),
    [current],
  );

  // The locations actually deleted: drop any selected path that sits inside
  // another selected one, because its ancestor removes it too (the deleter
  // dedups the same way). Keeps the promised space, the count, and the delete
  // payload consistent with what will really be removed.
  const deletePaths = useMemo(() => {
    const paths = [...selected];
    return paths.filter((p) => !paths.some((q) => q !== p && isInside(p, q)));
  }, [selected]);

  const selectedBytes = useMemo(() => {
    let total = 0;
    for (const path of deletePaths) total += known.get(path)?.size ?? 0;
    return total;
  }, [deletePaths, known]);

  const risks = useMemo<RiskEntry[]>(() => {
    const out: RiskEntry[] = [];
    for (const path of deletePaths) {
      const k = known.get(path);
      if (!k || (k.level !== "caution" && k.level !== "danger")) continue;
      out.push({ path, level: k.level, reasons: k.reasons });
    }
    return out;
  }, [deletePaths, known]);

  const riskyCount = current ? current.entries.filter((e) => e.level === "caution" || e.level === "danger").length : 0;
  const blockedCount = current ? current.entries.filter((e) => e.level === "blocked").length : 0;

  const requestDelete = () => {
    if (deletePaths.length === 0) return;
    onDelete(deletePaths, deletePaths.map((x) => known.get(x)?.size ?? 0), risks, selectedBytes);
  };

  const cancel = () => {
    api.cancelAnalyze().catch(() => {
      // Only if the binding itself fails do we drop the in-flight listing. On the
      // normal path the backend stops, finishes the listing as partial, and emits
      // it — the finished handler then renders the partial result with its badge.
      pending.current = null;
      setAnalyzing(false);
      setProgress(null);
    });
  };

  return (
    <section className="analyze fade-in">
      <div className="analyze-head">
        <div>
          <h2 className="section-title">{t("analyze.title")}</h2>
          <p className="analyze-lead">{t("analyze.lead")}</p>
        </div>
        {drive && (
          <button className="btn btn-ghost btn-small" onClick={() => { setDrive(null); setRings([]); cancel(); }}>
            {t("analyze.changeDrive")}
          </button>
        )}
      </div>

      {drives.length === 0 && <div className="empty">{t("drive.none")}</div>}

      {drives.length > 0 && !drive && (
        <DriveSelector drives={drives} selected={new Set()} onToggle={chooseDrive} />
      )}

      {drive && (
        <>
          <nav className="crumbs" aria-label={t("analyze.crumbs")}>
            <span className="crumb-root" aria-hidden="true"><Icon name="harddrive" size={15} /></span>
            <div className="crumb-track">
              {rings.map((r, i) => {
                const last = i === rings.length - 1;
                const label = i === 0 ? (drives.find((d) => d.mountPoint === r.path)?.name ?? r.path) : baseName(r.path);
                return (
                  <span key={r.path} className="crumb-item">
                    {i > 0 && <Icon name="chevron-right" size={13} className="crumb-sep" />}
                    <button
                      className={`crumb ${last ? "current" : ""}`}
                      onClick={() => !last && toCrumb(i)}
                      aria-current={last ? "page" : undefined}
                      title={r.path}
                    >
                      {label}
                    </button>
                  </span>
                );
              })}
            </div>
            <span className="crumb-status">
              {analyzing
                ? t("analyze.working")
                : `${formatNumber(current?.entries.length ?? 0)} ${p("plurals.item", current?.entries.length ?? 0)}`}
            </span>
          </nav>

          {error && (
            <div className="risk-banner analyze-error">
              <Icon name="triangle-alert" size={16} />
              <span>{t("analyze.failed")}: {error}</span>
            </div>
          )}

          {current && (
            <div className="analyze-body">
              <div className="analyze-chart">
                <Sunburst
                  rings={rings}
                  selected={selected}
                  onDrill={drill}
                  onSelect={toggle}
                  onHover={setHovered}
                  onUp={up}
                />
                <div className="analyze-hover">
                  {hovered ? (
                    <>
                      <span className="hover-name">{hovered.name}</span>
                      <span className="hover-size">{formatSize(hovered.size)}</span>
                      {hovered.level !== "safe" && <RiskBadge level={hovered.level} reasons={hovered.reasons} />}
                      <span className="hover-path" title={hovered.path}>{shortPath(hovered.path, 54)}</span>
                    </>
                  ) : (
                    <span className="muted">{t("analyze.hoverHint")}</span>
                  )}
                </div>
              </div>

              <div className="analyze-panel">
                <div className="results-toolbar">
                  <div className="results-summary">
                    <strong className="accent">{formatSize(current.totalBytes)}</strong> {t("analyze.inFolder")} ·{" "}
                    {formatNumber(current.entries.length)} {p("plurals.item", current.entries.length)}
                    {current.partial && <span className="badge badge-risk"> {t("analyze.partial")}</span>}
                  </div>
                  <div className="results-actions">
                    <button className="btn btn-ghost btn-small" onClick={() => selectMany(largePaths, true)} disabled={largePaths.length === 0}>
                      {t("results.selectLarge")}
                    </button>
                    <button className="btn btn-ghost btn-small" onClick={() => selectMany(safePaths, true)} disabled={safePaths.length === 0}>
                      {t("results.selectAll")}
                    </button>
                    <button className="btn btn-ghost btn-small" onClick={() => setSelected(new Set())} disabled={selected.size === 0}>
                      {t("results.deselectAll")}
                    </button>
                  </div>
                </div>

                {(riskyCount > 0 || blockedCount > 0) && (
                  <div className="risk-banner">
                    <Icon name="triangle-alert" size={16} />
                    <span>
                      {riskyCount > 0 && t("safety.bannerRisky", { count: riskyCount })}
                      {riskyCount > 0 && blockedCount > 0 && " "}
                      {blockedCount > 0 && t("safety.bannerBlocked", { count: blockedCount })}
                    </span>
                  </div>
                )}

                <div className="analyze-list">
                  {current.entries.length === 0 && <p className="results-empty">{t("analyze.empty")}</p>}
                  {(showAll ? current.entries : current.entries.slice(0, LIST_CAP)).map((e, i) => {
                    const checked = selected.has(e.path);
                    const blocked = e.level === "blocked";
                    const isDir = e.isDir && !e.skipped;
                    return (
                      <div
                        key={e.path}
                        className={`analyze-row ${checked ? "checked" : ""} ${e.level !== "safe" ? `row-${e.level}` : ""}`}
                      >
                        <input
                          type="checkbox"
                          checked={checked}
                          disabled={blocked || e.skipped}
                          onChange={() => toggle(e)}
                          aria-label={t("analyze.select", { name: e.name })}
                        />
                        <span className="analyze-swatch" style={{ background: e.level === "blocked" ? "var(--border)" : paletteColor(i) }} />
                        {isDir ? (
                          <button className="analyze-name analyze-drill" onClick={() => drill(e, rings.length - 1)} title={t("analyze.open", { name: e.name })}>
                            <Icon name="folder" size={15} />
                            <span className="analyze-name-text">{e.name}</span>
                          </button>
                        ) : (
                          <span className="analyze-name" title={e.path}>
                            <Icon name={e.isLink ? "copy" : "file"} size={15} />
                            <span className="analyze-name-text">{e.name}</span>
                          </span>
                        )}
                        <RiskBadge level={e.level} reasons={e.reasons} />
                        <span className="analyze-size">{formatSize(e.size)}</span>
                        <span className="analyze-percent">{formatPercent(e.size, current.totalBytes)}</span>
                        <button className="tree-reveal" title={t("tree.reveal")} onClick={() => onReveal(e.path)}>
                          <Icon name="folder-open" size={15} />
                        </button>
                      </div>
                    );
                  })}
                  {!showAll && current.entries.length > LIST_CAP && (
                    <button className="btn btn-ghost btn-small analyze-more" onClick={() => setShowAll(true)}>
                      {t("analyze.showMore", { count: formatNumber(current.entries.length - LIST_CAP) })}
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          <div className="action-bar">
            {deletePaths.length === 0 ? (
              <span className="action-info muted">{t("analyze.selectHint")}</span>
            ) : (
              <span className="action-info">{t("results.selectedCount", { count: deletePaths.length, size: formatSize(selectedBytes) })}</span>
            )}
            <div className="action-right">
              <button className="btn btn-primary btn-large" disabled={deletePaths.length === 0} onClick={requestDelete}>
                {t("results.deleteSelected")}
              </button>
            </div>
          </div>

          {analyzing && (
            <div className="modal-backdrop">
              <div className="progress-panel">
                <div className="progress-head">
                  <div className="progress-phase">
                    <span className="phase-dot on"><Icon name="pie-chart" size={13} /></span>
                    <span className="on">{t("analyze.scanning")}</span>
                  </div>
                  <button className="btn btn-ghost btn-small" onClick={cancel}>{t("scan.cancel")}</button>
                </div>
                <div className="progress-track"><div className="progress-fill indeterminate" /></div>
                <div className="progress-meta">
                  <span>{t("analyze.reading", { count: formatNumber(progress?.dirs ?? 0) })}</span>
                  <span className="muted">{t("analyze.measureNote")}</span>
                </div>
                {progress?.path && <div className="progress-path" title={progress.path}>{shortPath(progress.path, 70)}</div>}
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}
