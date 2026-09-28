// App.tsx — main flow: drives → categories → scan → results → confirm → delete,
// plus the settings screen. Every string comes from the locale catalog.
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { DriveSelector } from "./drives/DriveSelector";
import { CategoryPanel } from "./categories/CategoryPanel";
import { ScanProgressView } from "./scan/ScanProgress";
import { ResultsPanel } from "./results/ResultsPanel";
import { ConfirmModal, type DeleteMode } from "./results/ConfirmModal";
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
import * as api from "./backend";
import type { DriveUI, CategoryUI, ScanResult, ScanProgress, DeleteProgress } from "./lib/types";

type Stage = "select" | "results" | "delete" | "settings";

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
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteMode, setDeleteMode] = useState<DeleteMode>("trash");
  const lastMode = useRef<DeleteMode>("trash");
  const [deleteProgress, setDeleteProgress] = useState<DeleteProgress | null>(null);
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
      const paths = new Set(r.items.map((i) => i.path));
      setSelectedItems(
        new Set(r.items.filter((i) => !i.nestedIn || !paths.has(i.nestedIn)).map((i) => i.path)),
      );
      setKnownSizes(new Map(r.items.map((i) => [i.path, i.size])));
      setScanProgress(null);
      setStage("results");
      if (r.partial) notify(t("toast.scanCancelled"), t("toast.scanCancelledDetail"), "info");
    });
    const off3 = api.onDeleteProgress((prog) => setDeleteProgress(prog));
    const off4 = api.onDeleteFinished((r) => {
      setDeleteProgress(null);
      setConfirmOpen(false);
      setStage("select");
      setResult(null);
      setScanProgress(null);
      setSelectedItems(new Set());
      setKnownSizes(new Map());
      // Free space changed; refresh the drive cards.
      api.detectDrives().then(setDrives).catch(() => {});
      // Name the first failures so the user can see which paths stayed and why.
      const failures = (r.failures ?? []).slice(0, 3).map((f) => `${f.path}: ${f.message}`);
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

  const confirmDelete = () => {
    const paths = [...selectedItems];
    lastMode.current = deleteMode;
    setConfirmOpen(false);
    setStage("delete");
    setDeleteProgress({ done: 0, total: paths.length, bytes: 0, path: "" });
    api.startDelete({ paths, sizes: paths.map((x) => knownSizes.get(x) ?? 0), mode: deleteMode });
  };

  const canScan = selectedDrives.size > 0 && selectedCategories.size > 0 && stage === "select";
  const selectedCount = selectedItems.size;

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
        <div className="header-actions">
          <ThemeSwitch />
          <UpdateControl onNotify={notify} auto={settings.autoUpdate} openSignal={updateSignal} />
          <SettingsMenu iconSupported={iconSupported} onOpen={openSettings} />
        </div>
      </header>

      <main className="content">
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
        ) : stage === "select" && (
          <section className="fade-in">
            <h2 className="section-title">{t("drive.step")}</h2>
            <DriveSelector drives={drives} selected={selectedDrives} onToggle={toggleDrive} />

            <h2 className="section-title">{t("category.step")}</h2>
            <CategoryPanel categories={categories} selected={selectedCategories} onToggle={toggleCategory} />

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
                  onClick={() => setConfirmOpen(true)}
                >
                  {t("results.deleteSelected")}
                </button>
              </div>
            </div>
          </section>
        )}
      </main>

      {scanProgress && <ScanProgressView progress={scanProgress} onCancel={cancelScan} />}

      {confirmOpen && result && (
        <ConfirmModal
          itemCount={selectedItems.size}
          totalBytes={selectedBytes}
          mode={deleteMode}
          onMode={setDeleteMode}
          onConfirm={confirmDelete}
          onCancel={() => setConfirmOpen(false)}
          busy={false}
        />
      )}
      {stage === "delete" && deleteProgress && <DeleteProgressView progress={deleteProgress} onCancel={() => api.cancelDelete()} />}

      <div className="toast-container">
        {toasts.map((x) => <Toast key={x.id} toast={x} onClose={closeToast} />)}
      </div>
    </div>
  );
}
