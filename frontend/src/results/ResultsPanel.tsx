// ResultsPanel.tsx — scan results: accordion per category, folder browsing with
// real sizes, multi-select, and per-item handoff to the OS file manager.
import { useEffect, useMemo, useRef, useState } from "react";
import type { MouseEvent } from "react";
import { Icon } from "../common/Icon";
import { RiskBadge } from "../common/RiskBadge";
import { bulkSelectable } from "../lib/risk";
import { formatSize, formatNumber, shortPath } from "../lib/format";
import { listFolder } from "../backend";
import { coveredByAncestor, hasSelectedBelow } from "../lib/paths";
import { useI18n } from "../i18n/i18n";
import type { TFunc } from "../i18n/i18n";
import type { ScanResult, ScanItem, CategoryUI, FolderEntry, Level } from "../lib/types";

interface Node {
  path: string;
  name: string;
  size: number;
  isDir: boolean;
  note?: string;
  /** Risk of the scan item this row is (children inherit it and show no badge). */
  level?: Level;
  reasons?: string[];
  categoryNote?: string;
}

const baseName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() ?? p;

function nodeFromItem(it: ScanItem, t: TFunc["t"]): Node {
  const categoryNote = it.level !== "safe" ? t(`categories.${it.category}.risk`) : undefined;
  const note = it.linkFrom
    ? t("tree.symlinkFrom", { path: shortPath(it.linkFrom, 46) })
    : it.nestedIn
      ? t("tree.nestedIn", { path: shortPath(it.nestedIn, 46) })
      : undefined;
  return {
    path: it.path, name: baseName(it.path), size: it.size, isDir: true, note,
    level: it.level, reasons: it.reasons,
    categoryNote: categoryNote && categoryNote !== `categories.${it.category}.risk` ? categoryNote : undefined,
  };
}

function nodeFromEntry(e: FolderEntry, t: TFunc["t"]): Node {
  return {
    path: e.path,
    name: e.name,
    size: e.size,
    isDir: e.isDir,
    note: e.isLink ? t("tree.linkNote") : undefined,
  };
}

interface RowProps {
  node: Node;
  depth: number;
  selected: Set<string>;
  onToggle: (path: string) => void;
  onSelectMany: (paths: string[], select: boolean) => void;
  onReveal: (path: string) => void;
  onEntries: (entries: { path: string; size: number }[]) => void;
  /** Set on rows inside an opened folder: releases one child from a selected ancestor. */
  onExclude?: (path: string) => void;
}

function Row({ node, depth, selected, onToggle, onSelectMany, onReveal, onEntries, onExclude }: RowProps) {
  const { t } = useI18n();
  // A row is ticked when it, or a folder above it, is selected: deleting the folder
  // removes the row too, so the UI must not show it as unselected.
  const inherited = useMemo(() => coveredByAncestor(selected, node.path), [selected, node.path]);
  const checked = selected.has(node.path) || inherited;
  const partial = useMemo(
    () => !checked && node.isDir && hasSelectedBelow(selected, node.path),
    [checked, node.isDir, selected, node.path],
  );
  const box = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (box.current) box.current.indeterminate = partial;
  }, [partial]);
  const [open, setOpen] = useState(false);
  const [kids, setKids] = useState<Node[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    if (kids || loading) return;
    setLoading(true);
    setError(null);
    try {
      const loaded = (await listFolder(node.path)).map((e) => nodeFromEntry(e, t));
      setKids(loaded);
      onEntries(loaded.map((n) => ({ path: n.path, size: n.size })));
    } catch (e) {
      setError(t("tree.readFailed", { detail: String(e) }));
    } finally {
      setLoading(false);
    }
  };

  // Buttons live inside the row's label, so their clicks must not toggle the checkbox.
  const stop = (fn: () => void) => (e: MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    fn();
  };

  // Unticking a child of a selected folder: drop the folder from the selection and
  // keep every sibling, so exactly the one child is left out.
  const excludeChild = (childPath: string) => {
    if (selected.has(node.path)) onToggle(node.path);
    if (inherited) onExclude?.(node.path);
    onSelectMany((kids ?? []).map((k) => k.path).filter((p) => p !== childPath), true);
  };

  const onBoxChange = () => {
    if (!inherited) return onToggle(node.path);
    // Covered from above: only rows inside an opened folder can be released.
    if (onExclude) onExclude(node.path);
  };

  const toggleOpen = () => {
    const next = !open;
    setOpen(next);
    if (next) void load();
  };

  const indent = 12 + depth * 18;
  const blocked = node.level === "blocked";
  const level = node.level ?? "safe";

  return (
    <>
      <label className={`item-row ${checked ? "checked" : ""} ${level !== "safe" ? `row-${level}` : ""}`} style={{ paddingLeft: indent }}>
        <input
          ref={box}
          type="checkbox"
          checked={checked && !blocked}
          disabled={blocked || (inherited && !onExclude)}
          onChange={onBoxChange}
        />
        {node.isDir ? (
          <button
            className={`tree-expander ${open ? "open" : ""}`}
            data-tip={t(open ? "tree.collapse" : "tree.expand")}
            aria-label={t(open ? "tree.collapse" : "tree.expand")}
            aria-expanded={open}
            onClick={stop(toggleOpen)}
          >
            <Icon name="chevron-right" size={12} className="tree-chevron" />
          </button>
        ) : (
          <span className="tree-spacer" />
        )}
        <span className="item-main" onDoubleClick={node.isDir ? toggleOpen : undefined}>
          <span className="item-path" title={node.path}>
            {depth === 0 ? shortPath(node.path, 64) : node.name}
          </span>
          {node.note && <span className="item-note">{node.note}</span>}
        </span>
        {depth === 0 && <RiskBadge level={level} reasons={node.reasons} categoryNote={node.categoryNote} />}
        <span className="item-size">{formatSize(node.size)}</span>
        <button
          className="tree-reveal"
          title={t("tree.reveal")}
          aria-label={t("tree.revealAria")}
          onClick={stop(() => onReveal(node.path))}
        >
          <Icon name="folder-open" size={15} />
        </button>
      </label>

      {open && (
        <div className="tree-kids">
          {loading && <div className="tree-status" style={{ paddingLeft: indent + 18 }}>{t("tree.loading")}</div>}
          {error && <div className="tree-status" style={{ paddingLeft: indent + 18 }}>{error}</div>}
          {kids?.map((k) => (
            <Row
              key={k.path}
              node={k}
              depth={depth + 1}
              selected={selected}
              onToggle={onToggle}
              onSelectMany={onSelectMany}
              onReveal={onReveal}
              onEntries={onEntries}
              onExclude={excludeChild}
            />
          ))}
          {kids && kids.length === 0 && !loading && !error && (
            <div className="tree-status" style={{ paddingLeft: indent + 18 }}>{t("tree.empty")}</div>
          )}
        </div>
      )}
    </>
  );
}

interface Props {
  result: ScanResult;
  categories: CategoryUI[];
  selected: Set<string>;
  onToggleItem: (path: string) => void;
  onSelectMany: (paths: string[], select: boolean) => void;
  onReveal: (path: string) => void;
  onEntries: (entries: { path: string; size: number }[]) => void;
  /** Changes when the tree on disk changed, so open folders reload. */
  refreshKey?: number;
}

export function ResultsPanel({ result, categories, selected, onToggleItem, onSelectMany, onReveal, onEntries, refreshKey = 0 }: Props) {
  const { t, p } = useI18n();
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());

  // Group by category, sorted by total size desc.
  const groups = useMemo(() => {
    const m = new Map<string, ScanItem[]>();
    for (const it of result.items) {
      const arr = m.get(it.category) ?? [];
      arr.push(it);
      m.set(it.category, arr);
    }
    return [...m.entries()].sort(
      (a, b) => b[1].reduce((s, i) => s + i.size, 0) - a[1].reduce((s, i) => s + i.size, 0),
    );
  }, [result]);

  const categoryName = (id: string) => t(`categories.${id}.name`);
  const categoryIcon = (id: string) => categories.find((c) => c.id === id)?.icon ?? "eraser";

  // Bulk actions only tick plainly safe items; a risky one must be ticked by hand.
  const pickable = (items: ScanItem[]) => items.filter(bulkSelectable).map((i) => i.path);

  const toggleGroup = (items: ScanItem[]) => {
    const paths = pickable(items);
    const all = paths.length > 0 && paths.every((p) => selected.has(p));
    onSelectMany(paths, !all);
  };

  const pickLarge = () => {
    const large = pickable(result.items.filter((i) => i.size >= 100 * 1024 * 1024));
    onSelectMany(result.items.map((i) => i.path), false);
    onSelectMany(large, true);
  };

  const riskyCount = result.items.filter((i) => i.level === "caution" || i.level === "danger").length;
  const blockedCount = result.items.filter((i) => i.level === "blocked").length;

  return (
    <div className="results-panel">
      <div className="results-toolbar">
        <div className="results-summary">
          <strong>{formatNumber(result.itemCount)}</strong> {p("plurals.folder", result.itemCount)} ·{" "}
          <strong className="accent">{formatSize(result.totalBytes)}</strong> {t("results.canFree")}
          {result.partial && <span className="badge badge-risk"> {t("results.partial")}</span>}
          {result.dirsSkipped > 0 && (
            <span className="muted"> {t("results.skipped", { count: formatNumber(result.dirsSkipped) })}</span>
          )}
        </div>
        <div className="results-actions">
          <button className="btn btn-ghost btn-small" onClick={pickLarge}>{t("results.selectLarge")}</button>
          <button className="btn btn-ghost btn-small" onClick={() => onSelectMany(pickable(result.items), true)}>
            {t("results.selectAll")}
          </button>
          <button className="btn btn-ghost btn-small" onClick={() => onSelectMany(result.items.map((i) => i.path), false)}>
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

      {groups.length === 0 && <p className="results-empty">{t("results.empty")}</p>}

      <div className="results-groups">
        {groups.map(([catId, items]) => {
          const total = items.reduce((s, i) => s + i.size, 0);
          const groupCollapsed = collapsed.has(catId);
          const selectable = pickable(items);
          const allChecked = selectable.length > 0 && selectable.every((p) => selected.has(p));
          return (
            <div key={catId} className="group">
              <div className="group-head">
                <button
                  className="group-toggle"
                  onClick={() => setCollapsed((s) => { const n = new Set(s); n.has(catId) ? n.delete(catId) : n.add(catId); return n; })}
                >
                  <span className={`arrow ${groupCollapsed ? "folded" : ""}`}>▾</span>
                  <span className="group-icon"><Icon name={categoryIcon(catId)} size={17} /></span>
                  <span className="group-name">{categoryName(catId)}</span>
                  <span className="group-info">
                    {t("results.groupInfo", { count: items.length, size: formatSize(total) })}
                  </span>
                </button>
                <button className="btn btn-ghost btn-small" onClick={() => toggleGroup(items)} disabled={selectable.length === 0}>
                  {t(allChecked ? "results.deselectGroup" : "results.selectGroup")}
                </button>
              </div>
              {!groupCollapsed && (
                <div className="group-body">
                  {items.map((it) => (
                    <Row
                      key={`${it.path}:${refreshKey}`}
                      node={nodeFromItem(it, t)}
                      depth={0}
                      selected={selected}
                      onToggle={onToggleItem}
                      onSelectMany={onSelectMany}
                      onReveal={onReveal}
                      onEntries={onEntries}
                    />
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
