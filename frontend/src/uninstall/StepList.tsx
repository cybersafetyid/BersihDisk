// StepList.tsx — the reviewable steps of an uninstall plan, grouped by kind, each
// with its own checkbox, risk badge, size and (for admin-only steps) the command
// to run by hand. Supports deep inspection into folder paths and their children.
import { useState } from "react";
import { Icon } from "../common/Icon";
import { RiskBadge } from "../common/RiskBadge";
import { formatSize, shortPath } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { PlanStep, StepKind, StepResult, FolderEntry } from "../lib/types";
import { listFolder, reveal } from "../backend";

const order: StepKind[] = ["command", "path", "profile", "registry", "envpath"];
const icons: Record<StepKind, string> = {
  command: "terminal", path: "folder-open", profile: "list", registry: "settings", envpath: "list-checks",
};

interface Props {
  steps: PlanStep[];
  selected: Set<string>;
  onToggle: (id: string) => void;
  /** After a run: the outcome per step id, shown instead of the checkbox. */
  results?: Map<string, StepResult>;
}

/** Shows a manual command with a copy button that flips to "Copied" for a moment. */
export function CopyCommand({ text }: { text: string }) {
  const { t } = useI18n();
  const [done, setDone] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(text).then(() => {
      setDone(true);
      setTimeout(() => setDone(false), 1500);
    }).catch(() => {});
  };
  return (
    <div className="manual-cmd">
      <code>{text}</code>
      <button className="btn btn-ghost btn-small" onClick={copy}>
        <Icon name="copy" size={13} /> {t(done ? "common.copied" : "common.copy")}
      </button>
    </div>
  );
}

function PathStepItem({
  step,
  checked,
  blocked,
  result,
  onToggle,
}: {
  step: PlanStep;
  checked: boolean;
  blocked: boolean;
  result?: StepResult;
  onToggle: (id: string) => void;
}) {
  const { t, p } = useI18n();
  const [open, setOpen] = useState(false);
  const [kids, setKids] = useState<FolderEntry[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toggleOpen = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const next = !open;
    setOpen(next);
    if (next && !kids && !loading) {
      setLoading(true);
      setError(null);
      try {
        const loaded = await listFolder(step.label);
        setKids(loaded);
      } catch (err) {
        setError(String(err));
      } finally {
        setLoading(false);
      }
    }
  };

  const handleReveal = (e: React.MouseEvent, path: string) => {
    e.preventDefault();
    e.stopPropagation();
    reveal(path).catch(() => {});
  };

  const ext = step.label.includes(".") && !step.label.startsWith(".")
    ? step.label.split(".").pop()?.toUpperCase()
    : null;

  return (
    <div className="step-path-wrapper">
      <label className={`step-row ${checked ? "checked" : ""} ${blocked ? "row-blocked" : ""}`}>
        {result ? (
          <span className={`step-status status-${result.status}`}>{t(`uninstall.status.${result.status}`)}</span>
        ) : (
          <input
            type="checkbox"
            checked={checked && !blocked}
            disabled={blocked}
            onChange={() => onToggle(step.id)}
          />
        )}

        <button
          className={`tree-expander ${open ? "open" : ""}`}
          onClick={toggleOpen}
          title={t(open ? "tree.collapse" : "tree.expand")}
          aria-label={t(open ? "tree.collapse" : "tree.expand")}
        >
          <Icon name="chevron-right" size={12} className="tree-chevron" />
        </button>

        <div className="node-type-badge badge-folder">
          <Icon name={open ? "folder-open" : "folder"} size={14} />
        </div>

        <span className="step-main">
          <span className="step-name-row">
            <span className="step-label" title={step.label}>{shortPath(step.label, 58)}</span>
            {ext && <span className="file-ext-chip">{ext}</span>}
            {kids && <span className="dir-count-chip">{kids.length} {p("plurals.item", kids.length)}</span>}
          </span>
          {step.detail && <pre className="step-detail">{step.detail}</pre>}
          {step.manual && <CopyCommand text={step.manual} />}
          {result?.message && <span className="step-message">{result.message}</span>}
          {result?.hint && <span className="step-hint">{t(`safety.hint.${result.hint}`)}</span>}
        </span>

        <RiskBadge level={step.level} reasons={step.reasons} />
        {!!step.size && <span className="item-size">{formatSize(step.size)}</span>}

        <button
          className="tree-reveal"
          title={t("tree.reveal")}
          aria-label={t("tree.revealAria")}
          onClick={(e) => handleReveal(e, step.label)}
        >
          <Icon name="folder-open" size={14} />
        </button>
      </label>

      {open && (
        <div className="tree-kids-wrapper step-kids-indent">
          <div className="tree-kids-list">
            {loading && (
              <div className="tree-status-row">
                <Icon name="refresh" size={13} className="spin" />
                <span>{t("tree.loading")}</span>
              </div>
            )}
            {error && <div className="tree-status-row error">{error}</div>}
            {kids?.map((child) => {
              const childExt = !child.isDir && child.name.includes(".")
                ? child.name.split(".").pop()?.toUpperCase()
                : null;
              return (
                <div key={child.path} className="step-child-row">
                  <div className={`node-type-badge ${child.isDir ? "badge-folder" : "badge-file"}`}>
                    <Icon name={child.isDir ? "folder" : child.isLink ? "copy" : "file"} size={13} />
                  </div>
                  <span className="step-child-name" title={child.path}>
                    {child.name}
                  </span>
                  {childExt && <span className="file-ext-chip">{childExt}</span>}
                  <span className="step-child-size">{formatSize(child.size)}</span>
                  <button
                    className="tree-reveal"
                    title={t("tree.reveal")}
                    aria-label={t("tree.revealAria")}
                    onClick={(e) => handleReveal(e, child.path)}
                  >
                    <Icon name="folder-open" size={13} />
                  </button>
                </div>
              );
            })}
            {kids && kids.length === 0 && !loading && !error && (
              <div className="tree-status-row muted">{t("tree.empty")}</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export function StepList({ steps, selected, onToggle, results }: Props) {
  const { t } = useI18n();
  return (
    <div className="step-list">
      {order.map((kind) => {
        const group = steps.filter((s) => s.kind === kind);
        if (group.length === 0) return null;
        return (
          <section key={kind} className="step-group">
            <h3><Icon name={icons[kind]} size={14} /> {t(`uninstall.step.${kind}`)}</h3>
            {group.map((s) => {
              const res = results?.get(s.id);
              const blocked = s.level === "blocked";
              const isChecked = selected.has(s.id);

              if (kind === "path") {
                return (
                  <PathStepItem
                    key={s.id}
                    step={s}
                    checked={isChecked}
                    blocked={blocked}
                    result={res}
                    onToggle={onToggle}
                  />
                );
              }

              return (
                <label key={s.id} className={`step-row ${isChecked ? "checked" : ""} ${blocked ? "row-blocked" : ""}`}>
                  {res ? (
                    <span className={`step-status status-${res.status}`}>{t(`uninstall.status.${res.status}`)}</span>
                  ) : results ? (
                    <span className="step-status">—</span>
                  ) : (
                    <input
                      type="checkbox"
                      checked={isChecked && !blocked}
                      disabled={blocked}
                      onChange={() => onToggle(s.id)}
                    />
                  )}
                  <div className="node-type-badge badge-file">
                    <Icon name={icons[kind]} size={14} />
                  </div>
                  <span className="step-main">
                    <span className="step-label" title={s.label}>{s.label}</span>
                    {s.detail && <pre className="step-detail">{s.detail}</pre>}
                    {s.manual && <CopyCommand text={s.manual} />}
                    {res?.message && <span className="step-message">{res.message}</span>}
                    {res?.hint && <span className="step-hint">{t(`safety.hint.${res.hint}`)}</span>}
                  </span>
                  <RiskBadge level={s.level} reasons={s.reasons} />
                  {!!s.size && <span className="item-size">{formatSize(s.size)}</span>}
                </label>
              );
            })}
          </section>
        );
      })}
    </div>
  );
}
