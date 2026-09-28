// StepList.tsx — the reviewable steps of an uninstall plan, grouped by kind, each
// with its own checkbox, risk badge, size and (for admin-only steps) the command
// to run by hand.
import { useState } from "react";
import { Icon } from "../common/Icon";
import { RiskBadge } from "../common/RiskBadge";
import { formatSize, shortPath } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { PlanStep, StepKind, StepResult } from "../lib/types";

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
              return (
                <label key={s.id} className={`step-row ${selected.has(s.id) ? "checked" : ""} ${blocked ? "row-blocked" : ""}`}>
                  {res ? (
                    <span className={`step-status status-${res.status}`}>{t(`uninstall.status.${res.status}`)}</span>
                  ) : results ? (
                    <span className="step-status">—</span> // not part of the run
                  ) : (
                    <input
                      type="checkbox"
                      checked={selected.has(s.id) && !blocked}
                      disabled={blocked}
                      onChange={() => onToggle(s.id)}
                    />
                  )}
                  <span className="step-main">
                    <span className="step-label" title={s.label}>{kind === "path" ? shortPath(s.label, 62) : s.label}</span>
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
