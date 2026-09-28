// PlanModal.tsx — review, confirm and watch the uninstall of one package.
// Phases: loading the plan → review (steps, warnings, mode, acknowledgement) →
// running → result. The backend runs only the steps of the plan it produced.
import { useEffect, useMemo, useState } from "react";
import { Icon } from "../common/Icon";
import { RiskAck } from "../common/RiskAck";
import { ModePicker } from "../results/ModePicker";
import type { DeleteMode } from "../results/ConfirmModal";
import { StepList, CopyCommand } from "./StepList";
import { formatSize } from "../lib/format";
import { needsAck } from "../lib/risk";
import { useI18n } from "../i18n/i18n";
import * as api from "../backend";
import type { InstalledItem, UninstallPlan, UninstallProgress, UninstallResult } from "../lib/types";

interface Props {
  item: InstalledItem;
  /** `changed` is true when a run happened, so the list must be reloaded. */
  onClose: (changed: boolean) => void;
}

type Phase = "loading" | "review" | "running" | "done";

export function PlanModal({ item, onClose }: Props) {
  const { t } = useI18n();
  const [phase, setPhase] = useState<Phase>("loading");
  const [plan, setPlan] = useState<UninstallPlan | null>(null);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [mode, setMode] = useState<DeleteMode>("trash");
  const [ack, setAck] = useState(false);
  const [progress, setProgress] = useState<UninstallProgress | null>(null);
  const [result, setResult] = useState<UninstallResult | null>(null);

  useEffect(() => {
    let live = true;
    api.planUninstall(item.id)
      .then((p) => {
        if (!live) return;
        setPlan(p);
        setSelected(new Set(p.steps.filter((s) => s.selected && s.level !== "blocked").map((s) => s.id)));
        setPhase("review");
      })
      .catch((e) => {
        if (!live) return;
        setError(String(e));
        setPhase("review");
      });
    return () => { live = false; };
  }, [item.id]);

  useEffect(() => {
    const off1 = api.onUninstallProgress(setProgress);
    const off2 = api.onUninstallFinished((r) => { setResult(r); setPhase("done"); });
    return () => { off1(); off2(); };
  }, []);

  const chosen = useMemo(() => plan?.steps.filter((s) => selected.has(s.id)) ?? [], [plan, selected]);
  const mustAck = chosen.some((s) => needsAck(s.level));
  const removable = (s: { kind: string; manual?: string }) => s.kind === "path" && !s.manual;
  const freed = chosen.reduce((sum, s) => (removable(s) ? sum + (s.size ?? 0) : sum), 0);
  const hasPaths = chosen.some(removable);

  const toggle = (id: string) =>
    setSelected((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });

  const run = () => {
    if (!plan) return;
    setPhase("running");
    setProgress({ done: 0, total: chosen.length, step: "", bytes: 0 });
    api.startUninstall({ planId: plan.id, steps: chosen.map((s) => s.id), mode, acknowledged: ack });
  };

  const results = useMemo(() => new Map((result?.steps ?? []).map((r) => [r.id, r])), [result]);
  const busy = phase === "running";
  const percent = progress && progress.total > 0 ? (progress.done / progress.total) * 100 : 0;
  const close = () => onClose(phase === "done");

  return (
    <div className="modal-backdrop" onClick={busy ? undefined : close}>
      <div className="modal modal-wide" onClick={(e) => e.stopPropagation()}>
        <h2>{t("uninstall.planTitle", { name: item.name })}</h2>

        {phase === "loading" && <p className="modal-summary">{t("uninstall.planning")}</p>}
        {error && <p className="modal-warning"><Icon name="triangle-alert" size={16} /><span>{error}</span></p>}

        {plan && phase === "review" && (
          <>
            <p className="modal-summary">
              {t("uninstall.planSummary", { steps: chosen.length, size: formatSize(freed) })}
            </p>
            {(plan.warnings ?? []).map((w) => (
              <div key={w.code} className="risk-banner">
                <Icon name="triangle-alert" size={16} />
                <span>{t(`uninstall.warn.${w.code}`, { detail: w.detail ?? "" })}</span>
              </div>
            ))}
            {plan.steps.length === 0 && <p className="empty">{t("uninstall.nothingFound")}</p>}
          </>
        )}

        {phase === "running" && progress && (
          <>
            <div className="progress-track"><div className="progress-fill" style={{ width: `${percent}%` }} /></div>
            <div className="progress-meta">
              <span>{t("uninstall.progress", { done: progress.done, total: progress.total })}</span>
              <span className="item-path" title={progress.step}>{progress.step}</span>
            </div>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => api.cancelUninstall()}>{t("delete.cancel")}</button>
            </div>
          </>
        )}

        {phase === "done" && result && (
          <div className={`result-summary ${result.failed > 0 || result.error ? "has-failed" : ""}`}>
            {result.error
              ? <span>{result.error}</span>
              : <span>{t("uninstall.result", { ok: result.ok, failed: result.failed, skipped: result.skipped, size: formatSize(result.bytes) })}</span>}
            {result.aborted && <span className="muted"> {t("uninstall.aborted")}</span>}
          </div>
        )}

        {plan && (phase === "review" || phase === "done") && (
          <div className="step-scroll">
            <StepList
              steps={plan.steps}
              selected={selected}
              onToggle={toggle}
              results={phase === "done" ? results : undefined}
            />
          </div>
        )}

        {phase === "done" && (result?.manual?.length ?? 0) > 0 && (
          <div className="manual-block">
            <p className="muted">{t("uninstall.manualIntro")}</p>
            {result!.manual!.map((c) => <CopyCommand key={c} text={c} />)}
          </div>
        )}

        {plan && phase === "review" && (
          <>
            {hasPaths && <ModePicker mode={mode} onMode={setMode} />}
            {mode === "permanent" && hasPaths && (
              <p className="modal-warning"><Icon name="triangle-alert" size={16} /><span>{t("confirm.warning")}</span></p>
            )}
            {mustAck && <RiskAck checked={ack} onChange={setAck} />}
          </>
        )}

        {!busy && (
          <div className="modal-actions">
            <button className="btn btn-ghost" onClick={close}>
              {phase === "done" ? t("common.close") : t("confirm.cancel")}
            </button>
            {phase === "review" && plan && (
              <button
                className="btn btn-danger"
                onClick={run}
                disabled={chosen.length === 0 || (mustAck && !ack)}
              >
                {t("uninstall.confirm", { name: item.name })}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
