// ConfirmModal.tsx — confirmation dialog before deletion. Anything above "safe"
// is listed with the reasons and needs an explicit acknowledgement; a "danger"
// selection can only go to the Trash.
import { useEffect, useState } from "react";
import { RiskBadge, useReasonText } from "../common/RiskBadge";
import { RiskAck } from "../common/RiskAck";
import { ModePicker } from "./ModePicker";
import { Icon } from "../common/Icon";
import { formatSize, formatNumber, shortPath } from "../lib/format";
import { needsAck, severity, worse } from "../lib/risk";
import { useI18n } from "../i18n/i18n";
import type { Level } from "../lib/types";

export type DeleteMode = "trash" | "permanent";

/** One selected location that is not plainly safe. */
export interface RiskEntry {
  path: string;
  level: Level;
  reasons?: string[];
  /** Category note, shown for the "categoryNote" reason. */
  note?: string;
}

interface Props {
  itemCount: number;
  totalBytes: number;
  mode: DeleteMode;
  onMode: (m: DeleteMode) => void;
  onConfirm: (mode: DeleteMode, acknowledged: boolean) => void;
  onCancel: () => void;
  busy: boolean;
  risks: RiskEntry[];
}

const SHOWN = 5;

export function ConfirmModal({ itemCount, totalBytes, mode, onMode, onConfirm, onCancel, busy, risks }: Props) {
  const { t, p } = useI18n();
  const reasonText = useReasonText();
  const [ack, setAck] = useState(false);

  const worst = risks.reduce<Level>((w, r) => worse(w, r.level), "safe");
  const mustAck = needsAck(worst);
  const trashOnly = worst === "danger";
  const effective: DeleteMode = trashOnly ? "trash" : mode;

  useEffect(() => {
    const esc = (e: KeyboardEvent) => { if (e.key === "Escape" && !busy) onCancel(); };
    window.addEventListener("keydown", esc);
    return () => window.removeEventListener("keydown", esc);
  }, [onCancel, busy]);

  const ordered = [...risks].sort((a, b) => severity(b.level) - severity(a.level));

  return (
    <div className="modal-backdrop" onClick={busy ? undefined : onCancel}>
      <div
        className={`modal ${effective === "permanent" || worst === "danger" ? "modal-danger" : ""}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header-row">
          <span className={`modal-header-icon ${effective === "permanent" || worst === "danger" ? "danger" : "accent"}`}>
            <Icon name={effective === "permanent" ? "eraser" : "trash"} size={22} />
          </span>
          <div>
            <h2>{effective === "permanent" ? t("confirm.titlePermanent") : t("confirm.titleTrash")}</h2>
            <p className="modal-summary">
              <strong>{formatNumber(itemCount)}</strong> {p("plurals.folder", itemCount)} ·{" "}
              <strong>{formatSize(totalBytes)}</strong> {t("confirm.willFree")}
            </p>
          </div>
        </div>

        {risks.length > 0 && (
          <div className={`risk-panel risk-${worst}`}>
            <div className="risk-panel-title">
              <Icon name={worst === "danger" ? "shield-alert" : "triangle-alert"} size={16} />
              {t(worst === "danger" ? "safety.confirmDanger" : "safety.confirmCaution", { count: risks.length })}
            </div>
            <ul className="risk-list">
              {ordered.slice(0, SHOWN).map((r) => (
                <li key={r.path}>
                  <div className="risk-list-head">
                    <RiskBadge level={r.level} />
                    <span className="risk-list-path" title={r.path}>{shortPath(r.path, 46)}</span>
                  </div>
                  <div className="risk-list-why">
                    {(r.reasons ?? []).map((c) => (c === "categoryNote" && r.note ? r.note : reasonText([c])[0])).join(" · ")}
                  </div>
                </li>
              ))}
            </ul>
            {risks.length > SHOWN && (
              <div className="muted">{t("safety.andMore", { count: risks.length - SHOWN })}</div>
            )}
          </div>
        )}

        <ModePicker mode={effective} onMode={onMode} lockTrash={trashOnly} />

        {effective === "permanent" && (
          <p className="modal-warning">
            <Icon name="triangle-alert" size={16} />
            <span>{t("confirm.warning")}</span>
          </p>
        )}

        {mustAck && <RiskAck checked={ack} onChange={setAck} danger={worst === "danger"} />}

        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel} disabled={busy}>{t("confirm.cancel")}</button>
          <button
            className={`btn ${effective === "permanent" ? "btn-danger" : "btn-primary"}`}
            onClick={() => onConfirm(effective, ack)}
            disabled={busy || (mustAck && !ack)}
          >
            {busy ? t("confirm.busy") : effective === "permanent" ? t("confirm.yesPermanent") : t("confirm.yesTrash")}
          </button>
        </div>
      </div>
    </div>
  );
}
