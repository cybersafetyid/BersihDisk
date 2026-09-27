// ConfirmModal.tsx — confirmation dialog before deletion.
import { useEffect } from "react";
import { Icon } from "../common/Icon";
import { formatSize, formatNumber } from "../lib/format";
import { useI18n } from "../i18n/i18n";

export type DeleteMode = "trash" | "permanent";

interface Props {
  itemCount: number;
  totalBytes: number;
  mode: DeleteMode;
  onMode: (m: DeleteMode) => void;
  onConfirm: () => void;
  onCancel: () => void;
  busy: boolean;
}

export function ConfirmModal({ itemCount, totalBytes, mode, onMode, onConfirm, onCancel, busy }: Props) {
  const { t, p } = useI18n();

  useEffect(() => {
    const esc = (e: KeyboardEvent) => { if (e.key === "Escape" && !busy) onCancel(); };
    window.addEventListener("keydown", esc);
    return () => window.removeEventListener("keydown", esc);
  }, [onCancel, busy]);

  return (
    <div className="modal-backdrop" onClick={busy ? undefined : onCancel}>
      <div className={`modal ${mode === "permanent" ? "modal-danger" : ""}`} onClick={(e) => e.stopPropagation()}>
        <h2>{mode === "permanent" ? t("confirm.titlePermanent") : t("confirm.titleTrash")}</h2>
        <p className="modal-summary">
          <strong>{formatNumber(itemCount)}</strong> {p("plurals.folder", itemCount)} ·{" "}
          <strong>{formatSize(totalBytes)}</strong> {t("confirm.willFree")}
        </p>

        <div className="mode-picker">
          <button className={`mode-card ${mode === "trash" ? "active" : ""}`} onClick={() => onMode("trash")}>
            <span className="mode-icon"><Icon name="trash" size={22} /></span>
            <span className="mode-title">{t("confirm.trash")}</span>
            <span className="mode-desc">{t("confirm.trashDesc")}</span>
          </button>
          <button className={`mode-card danger ${mode === "permanent" ? "active" : ""}`} onClick={() => onMode("permanent")}>
            <span className="mode-icon"><Icon name="cpu" size={22} /></span>
            <span className="mode-title">{t("confirm.permanent")}</span>
            <span className="mode-desc">{t("confirm.permanentDesc")}</span>
          </button>
        </div>

        {mode === "permanent" && (
          <p className="modal-warning">
            <Icon name="triangle-alert" size={16} />
            <span>{t("confirm.warning")}</span>
          </p>
        )}

        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel} disabled={busy}>{t("confirm.cancel")}</button>
          <button className={`btn ${mode === "permanent" ? "btn-danger" : "btn-primary"}`} onClick={onConfirm} disabled={busy}>
            {busy ? t("confirm.busy") : mode === "permanent" ? t("confirm.yesPermanent") : t("confirm.yesTrash")}
          </button>
        </div>
      </div>
    </div>
  );
}
