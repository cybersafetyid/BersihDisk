// ResultsPanel.tsx — scan results: accordion per category, folder browsing with
// real sizes, multi-select, and per-item handoff to the OS file manager.
import { useMemo, useState } from "react";
import type { MouseEvent } from "react";
import { Icon } from "../common/Icon";
import { formatSize, formatNumber, shortPath } from "../lib/format";
import { listFolder } from "../backend";
import { useI18n } from "../i18n/i18n";
import type { TFunc } from "../i18n/i18n";
import type { ScanResult, ScanItem, CategoryUI, FolderEntry } from "../lib/types";

interface Node {
  path: string;
  name: string;
  size: number;
  isDir: boolean;
  note?: string;
}

const baseName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() ?? p;

function nodeFromItem(it: ScanItem, t: TFunc["t"]): Node {
  const note = it.linkFrom
    ? t("tree.symlinkFrom", { path: shortPath(it.linkFrom, 46) })
    : it.nestedIn
      ? t("tree.nestedIn", { path: shortPath(it.nestedIn, 46) })
      : undefined;
  return { path: it.path, name: baseName(it.path), size: it.size, isDir: true, note };
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
  onReveal: (path: string) => void;
  onEntries: (entries: { path: string; size: number }[]) => void;
}

function Row({ node, depth, selected, onToggle, onReveal, onEntries }: RowProps) {
  const { t } = useI18n();
  const checked = selected.has(node.path);
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

  const indent = 12 + depth * 18;

  return (
    <>
      <label className={`item-row ${checked ? "checked" : ""}`} style={{ paddingLeft: indent }}>
        <input type="checkbox" checked={checked} onChange={() => onToggle(node.path)} />
        {node.isDir ? (
          <button
            className="tree-expander"
            title={t(open ? "tree.collapse" : "tree.expand")}
            onClick={stop(() => {
              const next = !open;
              setOpen(next);
              if (next) void load();
            })}
          >
            <span className={`arrow ${open ? "" : "folded"}`}>▾</span>
          </button>
        ) : (
          <span className="tree-spacer" />
        )}
        <span className="item-main">
          <span className="item-path" title={node.path}>
            {depth === 0 ? shortPath(node.path, 64) : node.name}
          </span>
          {node.note && <span className="item-note">{node.note}</span>}
        </span>
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
            <Row key={k.path} node={k} depth={depth + 1} selected={selected} onToggle={onToggle} onReveal={onReveal} onEntries={onEntries} />
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
}

export function ResultsPanel({ result, categories, selected, onToggleItem, onSelectMany, onReveal, onEntries }: Props) {
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

  const toggleGroup = (items: ScanItem[]) => {
    const paths = items.map((i) => i.path);
    const all = paths.every((p) => selected.has(p));
    onSelectMany(paths, !all);
  };

  const pickLarge = () => {
    const large = result.items.filter((i) => i.size >= 100 * 1024 * 1024).map((i) => i.path);
    onSelectMany(result.items.map((i) => i.path), false);
    onSelectMany(large, true);
  };

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
          <button className="btn btn-ghost btn-small" onClick={() => onSelectMany(result.items.map((i) => i.path), true)}>
            {t("results.selectAll")}
          </button>
          <button className="btn btn-ghost btn-small" onClick={() => onSelectMany(result.items.map((i) => i.path), false)}>
            {t("results.deselectAll")}
          </button>
        </div>
      </div>

      <div className="results-groups">
        {groups.map(([catId, items]) => {
          const total = items.reduce((s, i) => s + i.size, 0);
          const groupCollapsed = collapsed.has(catId);
          const allChecked = items.every((i) => selected.has(i.path));
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
                <button className="btn btn-ghost btn-small" onClick={() => toggleGroup(items)}>
                  {t(allChecked ? "results.deselectGroup" : "results.selectGroup")}
                </button>
              </div>
              {!groupCollapsed && (
                <div className="group-body">
                  {items.map((it) => (
                    <Row
                      key={it.path}
                      node={nodeFromItem(it, t)}
                      depth={0}
                      selected={selected}
                      onToggle={onToggleItem}
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
