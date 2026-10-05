// App.tsx — main flow: drives → categories → scan → results → confirm → delete,
// plus the settings screen. Every string comes from the locale catalog.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { DriveSelector } from "./drives/DriveSelector";
import { CategoryPanel } from "./categories/CategoryPanel";
import { ScanProgressView } from "./scan/ScanProgress";
import { ResultsPanel } from "./results/ResultsPanel";
import { ConfirmModal, type DeleteMode, type RiskEntry } from "./results/ConfirmModal";
import { UninstallPage } from "./uninstall/UninstallPage";
import { AnalyzePage } from "./analyze/AnalyzePage";
import { ModeNav } from "./common/ModeNav";
import { Icon } from "./common/Icon";
import { DeleteProgressView } from "./results/DeleteProgress";
import { ThemeSwitch } from "./common/ThemeSwitch";
import { UpdateControl } from "./update/UpdateControl";
import { SettingsMenu } from "./settings/SettingsMenu";
import { SettingsPage } from "./settings/SettingsPage";
import type { Section } from "./settings/sections";
import { Toast, type ToastData } from "./common/Toast";
import brandLogo from "./assets/brand/logo.svg";

import { useAppliedSettings } from "./hooks/useAppliedSettings";
import { useI18n } from "./i18n/i18n";
import { formatSize, formatNumber } from "./lib/format";
import { isInside } from "./lib/paths";
import { itemFor } from "./lib/risk";
import * as api from "./backend";
import type { DriveUI, CategoryUI, ScanItem, ScanResult, ScanProgress, DeleteProgress } from "./lib/types";

/** Totals as the scanner computes them: an item nested in another is counted once. */
function tally(items: ScanItem[]): Pick<ScanResult, "items" | "totalBytes" | "itemCount"> {
  const paths = new Set(items.map((i) => i.path));
  const totalBytes = items.reduce((s, i) => (i.nestedIn && paths.has(i.nestedIn) ? s : s + i.size), 0);
  return { items, totalBytes, itemCount: items.length };
}

/** The result without the items `isGone` says were deleted. */
function withoutItems(r: ScanResult, isGone: (path: string) => boolean): ScanResult {
  return { ...r, ...tally(r.items.filter((i) => !isGone(i.path))) };
}

/** The result with fresh sizes; an item that measures 0 no longer exists. */
function resized(r: ScanResult, sizes: Map<string, number>): ScanResult {
  const items = r.items
    .map((i) => ({ ...i, size: sizes.get(i.path) ?? i.size }))
    .filter((i) => i.size > 0)
    .sort((a, b) => b.size - a.size);
  return { ...r, ...tally(items) };
}

type Stage = "select" | "results" | "delete" | "settings" | "uninstall" | "analyze";

/** A delete waiting for confirmation, from either the cleaner or the analyzer. */
interface PendingDelete {
  paths: string[];
  sizes: number[];
  risks: RiskEntry[];
  totalBytes: number;
  count: number;
  origin: "results" | "analyze";
}

export default function App() {
  const settings = useAppliedSettings();
  const { t, p } = useI18n();

  // Data from the backend.
  const [drives, setDrives] = useState<DriveUI[]>([]);
  const [categories, setCategories] = useState<CategoryUI[]>([]);
  const [loading, setLoading] = useState(true);
  const [iconSupported, setIconSupported] = useState(false);

  // User selections.
  const [selectedDrives, setSelectedDrives] = useState<Set<string>>(new Set());
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set());

  // Flow.
  const [stage, setStage] = useState<Stage>("select");
  const [section, setSection] = useState<Section>("language");
  const [updateSignal, setUpdateSignal] = useState(0);
  const [scanProgress, setScanProgress] = useState<ScanProgress | null>(null);
  const [result, setResult] = useState<ScanResult | null>(null);
  const resultRef = useRef<ScanResult | null>(null);
  resultRef.current = result;
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [pending, setPending] = useState<PendingDelete | null>(null);
  const [deleteMode, setDeleteMode] = useState<DeleteMode>("trash");
  const lastMode = useRef<DeleteMode>("trash");
  const lastPaths = useRef<string[]>([]);
  // Which screen started the running delete, so the flow returns there.
  const deleteOrigin = useRef<"results" | "analyze">("results");
  // Bumped after a delete so open folders in the results reload their contents.
  const [refreshKey, setRefreshKey] = useState(0);
  const [deleteProgress, setDeleteProgress] = useState<DeleteProgress | null>(null);
  // The scroll container is <main>; a long page (results, analyzer, uninstall)
  // shows a "back to top" button once it has scrolled far enough.
  const contentRef = useRef<HTMLElement>(null);
  const [showToTop, setShowToTop] = useState(false);
  const [toasts, setToasts] = useState<ToastData[]>([]);
  const toastId = useRef(0);

  const pushToast = useCallback((next: Omit<ToastData, "id">) => {
    setToasts((prev) => [...prev, { ...next, id: ++toastId.current }]);
  }, []);
  const closeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((x) => x.id !== id));
  }, []);
  const notify = useCallback(
    (message: string, detail?: string, variant: "success" | "error" | "info" = "info") =>
      pushToast({ message, detail, variant }),
    [pushToast],
  );

  // Opens a path in Finder / Explorer and reports a failure as a toast.
  const revealPath = useCallback(
    (path: string) => {
      api.reveal(path).catch((e) => notify(t("toast.revealFailed"), String(e), "error"));
    },
    [notify, t],
  );

  // Load drives + categories on mount; default: root drive + non-optIn categories.
  useEffect(() => {
    (async () => {
      try {
        const [d, c] = await Promise.all([api.detectDrives(), api.listCategories()]);
        setDrives(d);
        setCategories(c);
        setSelectedDrives(new Set(d.filter((x) => x.root).map((x) => x.mountPoint)));
        setSelectedCategories(new Set(c.filter((x) => !x.optIn).map((x) => x.id)));
      } catch (e) {
        notify(t("toast.loadFailed"), String(e), "error");
      } finally {
        setLoading(false);
      }
    })();
    api.appIconSupported().then(setIconSupported).catch(() => setIconSupported(false));
  }, [notify, t]);

  // Backend event listeners.
  useEffect(() => {
    const off1 = api.onScanProgress((prog) => setScanProgress(prog));
    const off2 = api.onScanFinished((r) => {
      setResult(r);
      // A nested item is removed together with the item containing it, so it
      // starts unchecked rather than promising the same bytes twice.
      // Only plainly safe items start ticked; risky and protected ones are the user's call.
      const paths = new Set(r.items.map((i) => i.path));
      setSelectedItems(
        new Set(r.items.filter((i) => i.level === "safe" && (!i.nestedIn || !paths.has(i.nestedIn))).map((i) => i.path)),
      );
      setKnownSizes(new Map(r.items.map((i) => [i.path, i.size])));
      setScanProgress(null);
      setStage("results");
      if (r.partial) notify(t("toast.scanCancelled"), t("toast.scanCancelledDetail"), "info");
    });
    const off3 = api.onDeleteProgress((prog) => setDeleteProgress(prog));
    const off4 = api.onDeleteFinished((r) => {
      setDeleteProgress(null);
      setPending(null);
      const fromAnalyze = deleteOrigin.current === "analyze";
      // Stay on the screen the delete started from so the user can pick more;
      // only what is gone leaves the list.
      setStage(fromAnalyze ? "analyze" : "results");
      const failed = new Set((r.failures ?? []).map((f) => f.path));
      const gone = lastPaths.current.filter((p) => !failed.has(p));
      const isGone = (p: string) => gone.some((g) => p === g || isInside(p, g));
      if (!fromAnalyze) {
        setSelectedItems((prev) => new Set([...prev].filter((p) => !isGone(p))));
        setResult((prev) => (prev ? withoutItems(prev, isGone) : prev));
      }
      // The analyzer re-measures the folder in view when this key changes.
      setRefreshKey((n) => n + 1);
      // What is left may have shrunk (a child was deleted): re-measure it.
      let left: string[] = [];
      if (!fromAnalyze) left = (resultRef.current?.items ?? []).filter((i) => !isGone(i.path)).map((i) => i.path);
      if (left.length > 0) {
        api.measurePaths(left).then((sizes) => {
          const fresh = new Map(left.map((p, i) => [p, sizes[i]]));
          setResult((prev) => (prev ? resized(prev, fresh) : prev));
          setKnownSizes((prev) => {
            const next = new Map([...prev].filter(([p]) => !isGone(p)));
            fresh.forEach((size, p) => size > 0 && next.set(p, size));
            return next;
          });
        }).catch(() => {});
      }
      // Free space changed; refresh the drive cards.
      api.detectDrives().then(setDrives).catch(() => {});
      // Name the first failures so the user can see which paths stayed and why.
      const failures = (r.failures ?? []).slice(0, 3).map(
        (f) => `${f.path}: ${f.message}${f.hint ? ` — ${t(`safety.hint.${f.hint}`)}` : ""}`,
      );
      const more = r.failed > failures.length ? ` (+${r.failed - failures.length})` : "";
      notify(
        t(lastMode.current === "trash" ? "toast.trashResult" : "toast.deleteResult", { ok: formatNumber(r.ok), size: formatSize(r.bytes) }),
        r.failed > 0
          ? [t("toast.deleteFailedDetail", { failed: r.failed }), ...failures].join(" · ") + more
          : undefined,
        r.failed > 0 ? "info" : "success",
      );
    });
    return () => { off1(); off2(); off3(); off4(); };
  }, [notify, t]);

  const toggleDrive = (mp: string) =>
    setSelectedDrives((s) => { const n = new Set(s); n.has(mp) ? n.delete(mp) : n.add(mp); return n; });
  const toggleCategory = (id: string) =>
    setSelectedCategories((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const toggleItem = (path: string) =>
    setSelectedItems((s) => { const n = new Set(s); n.has(path) ? n.delete(path) : n.add(path); return n; });
  const selectMany = (paths: string[], select: boolean) =>
    setSelectedItems((s) => { const n = new Set(s); paths.forEach((x) => (select ? n.add(x) : n.delete(x))); return n; });

  const startScan = () => {
    if (selectedDrives.size === 0 || selectedCategories.size === 0) return;
    setResult(null);
    setScanProgress({ phase: "search", path: "", itemsFound: 0, totalItems: 0, bytesSoFar: 0, dirsVisited: 0 });
    api.startScan({ roots: [...selectedDrives], categories: [...selectedCategories] });
  };

  const cancelScan = () => api.cancelScan();

  // Sizes of everything the UI has measured — scan results plus folders the user
  // opened — so an arbitrary selection can be totalled and a delete can report
  // what it actually freed.
  const [knownSizes, setKnownSizes] = useState<Map<string, number>>(new Map());
  const registerSizes = useCallback((entries: { path: string; size: number }[]) => {
    setKnownSizes((prev) => {
      const next = new Map(prev);
      for (const e of entries) next.set(e.path, e.size);
      return next;
    });
  }, []);

  const selectedBytes = useMemo(() => {
    const paths = [...selectedItems];
    let total = 0;
    for (const x of paths) {
      // Anything inside an already selected folder is counted once, by that folder.
      if (paths.some((q) => q !== x && isInside(x, q))) continue;
      total += knownSizes.get(x) ?? 0;
    }
    return total;
  }, [selectedItems, knownSizes]);

  // What the confirmation dialog must warn about: every selected location the
  // backend rated above safe (a folder picked inside a result inherits its item's level).
  const risks = useMemo<RiskEntry[]>(() => {
    if (!result) return [];
    const out: RiskEntry[] = [];
    for (const path of selectedItems) {
      const it = itemFor(result.items, path);
      if (!it || (it.level !== "caution" && it.level !== "danger")) continue;
      const note = t(`categories.${it.category}.risk`);
      out.push({ path, level: it.level, reasons: it.reasons, note: note !== `categories.${it.category}.risk` ? note : undefined });
    }
    return out;
  }, [result, selectedItems, t]);

  // Open the shared confirmation for the cleaner's selection.
  const openCleanConfirm = () => {
    const paths = [...selectedItems];
    setPending({
      paths,
      sizes: paths.map((x) => knownSizes.get(x) ?? 0),
      risks,
      totalBytes: selectedBytes,
      count: paths.length,
      origin: "results",
    });
  };

  // Open the same confirmation for a selection made in the drive analyzer.
  const openAnalyzeConfirm = (paths: string[], sizes: number[], analyzeRisks: RiskEntry[], totalBytes: number) => {
    setPending({ paths, sizes, risks: analyzeRisks, totalBytes, count: paths.length, origin: "analyze" });
  };

  const confirmDelete = (mode: DeleteMode, acknowledged: boolean) => {
    if (!pending) return;
    const { paths, sizes, origin } = pending;
    lastMode.current = mode;
    lastPaths.current = paths;
    deleteOrigin.current = origin;
    setPending(null);
    // The cleaner hides its results behind the progress screen; the analyzer keeps
    // its chart visible while the shared progress overlay runs on top.
    if (origin === "results") setStage("delete");
    setDeleteProgress({ done: 0, total: paths.length, bytes: 0, path: "" });
    api.startDelete({ paths, sizes, mode, acknowledged });
  };

  const canScan = selectedDrives.size > 0 && selectedCategories.size > 0 && stage === "select";
  const selectedCount = selectedItems.size;

  const goClean = () => setStage(result ? "results" : "select");

  const openSettings = (next: Section) => {
    setSection(next);
    setStage("settings");
  };

  return (
    <div className="app">
      <header className="header">
        <div className="brand">
          <span className="brand-icon">
            <img src={brandLogo} alt="" width={52} height={52} />
          </span>
          <div>
            <h1>{t("app.title")}</h1>
            <p>{t("app.tagline")}</p>
          </div>
        </div>
        <ModeNav
          mode={stage === "uninstall" ? "uninstall" : stage === "analyze" ? "analyze" : "clean"}
          onMode={(m) => {
            if (m === "uninstall") setStage("uninstall");
            else if (m === "analyze") setStage("analyze");
            else goClean();
          }}
        />
        <div className="header-actions">
          <ThemeSwitch />
          <UpdateControl onNotify={notify} auto={settings.autoUpdate} openSignal={updateSignal} />
          <SettingsMenu iconSupported={iconSupported} onOpen={openSettings} />
        </div>
      </header>

      <main
        className="content"
        ref={contentRef}
        onScroll={(e) => setShowToTop(e.currentTarget.scrollTop > 320)}
      >
        {loading ? (
          <div className="skeleton-wrap">
            <div className="skeleton skeleton-card" />
            <div className="skeleton skeleton-card" />
            <div className="skeleton skeleton-card" />
          </div>
        ) : stage === "settings" ? (
          <SettingsPage
            section={section}
            onSection={setSection}
            onClose={() => setStage(result ? "results" : "select")}
            onNotify={notify}
            iconSupported={iconSupported}
            onCheckUpdate={() => setUpdateSignal((n) => n + 1)}
          />
        ) : stage === "uninstall" ? (
          <UninstallPage onNotify={notify} onFreed={() => api.detectDrives().then(setDrives).catch(() => {})} />
        ) : stage === "select" && (
          <section className="fade-in">
            <h2 className="section-title">{t("drive.step")}</h2>
            <DriveSelector drives={drives} selected={selectedDrives} onToggle={toggleDrive} />

            <h2 className="section-title">{t("category.step")}</h2>
            <CategoryPanel
              categories={categories}
              selected={selectedCategories}
              onToggle={toggleCategory}
              onReplace={setSelectedCategories}
            />

            <div className="action-bar">
              <span className="action-info">
                {t("scan.selection", {
                  drives: selectedDrives.size,
                  driveWord: p("plurals.drive", selectedDrives.size),
                  categories: selectedCategories.size,
                  categoryWord: p("plurals.category", selectedCategories.size),
                })}
              </span>
              <button className="btn btn-primary btn-large" disabled={!canScan} onClick={startScan}>
                {t("scan.start")}
              </button>
            </div>
          </section>
        )}

        {stage === "analyze" && (
          <AnalyzePage
            drives={drives}
            refreshKey={refreshKey}
            onNotify={notify}
            onReveal={revealPath}
            onDelete={openAnalyzeConfirm}
          />
        )}

        {stage === "results" && result && (
          <section className="fade-in">
            <h2 className="section-title">{t("results.title")}</h2>
            <ResultsPanel
              result={result}
              categories={categories}
              selected={selectedItems}
              onToggleItem={toggleItem}
              onSelectMany={selectMany}
              onReveal={revealPath}
              onEntries={registerSizes}
              refreshKey={refreshKey}
            />
            <div className="action-bar">
              <button className="btn btn-ghost" onClick={() => { setStage("select"); setResult(null); }}>
                {t("results.rescan")}
              </button>
              <div className="action-right">
                <span className="action-info">
                  {t("results.selectedCount", { count: selectedCount, size: formatSize(selectedBytes) })}
                </span>
                <button
                  className="btn btn-primary btn-large"
                  disabled={selectedCount === 0}
                  onClick={openCleanConfirm}
                >
                  {t("results.deleteSelected")}
                </button>
              </div>
            </div>
          </section>
        )}
      </main>

      {scanProgress && <ScanProgressView progress={scanProgress} onCancel={cancelScan} />}

      {pending && (
        <ConfirmModal
          itemCount={pending.count}
          totalBytes={pending.totalBytes}
          mode={deleteMode}
          onMode={setDeleteMode}
          onConfirm={confirmDelete}
          onCancel={() => setPending(null)}
          busy={false}
          risks={pending.risks}
        />
      )}
      {deleteProgress && <DeleteProgressView progress={deleteProgress} onCancel={() => api.cancelDelete()} />}

      <div className="toast-container">
        {toasts.map((x) => <Toast key={x.id} toast={x} onClose={closeToast} />)}
      </div>

      {showToTop && (
        <button
          className="to-top"
          title={t("common.backToTop")}
          aria-label={t("common.backToTop")}
          onClick={() => contentRef.current?.scrollTo({ top: 0, behavior: "smooth" })}
        >
          <Icon name="chevron-up" size={18} />
        </button>
      )}
    </div>
  );
}
