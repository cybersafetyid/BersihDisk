// ModePicker.tsx — the Trash / Permanent choice, shared by every destructive dialog.
import { Icon } from "../common/Icon";
import { useI18n } from "../i18n/i18n";
import type { DeleteMode } from "./ConfirmModal";

interface Props {
  mode: DeleteMode;
  onMode: (m: DeleteMode) => void;
  /** Permanent is unavailable (risky selections go to the Trash only). */
  lockTrash?: boolean;
}

export function ModePicker({ mode, onMode, lockTrash }: Props) {
  const { t } = useI18n();
  return (
    <div className="mode-picker">
      <button className={`mode-card ${mode === "trash" ? "active" : ""}`} onClick={() => onMode("trash")}>
        <span className="mode-icon"><Icon name="trash" size={22} /></span>
        <span className="mode-title">{t("confirm.trash")}</span>
        <span className="mode-desc">{t("confirm.trashDesc")}</span>
      </button>
      <button
        className={`mode-card danger ${mode === "permanent" ? "active" : ""}`}
        onClick={() => onMode("permanent")}
        disabled={lockTrash}
        title={lockTrash ? t("safety.trashOnly") : undefined}
      >
        <span className="mode-icon"><Icon name="cpu" size={22} /></span>
        <span className="mode-title">{t("confirm.permanent")}</span>
        <span className="mode-desc">{lockTrash ? t("safety.trashOnly") : t("confirm.permanentDesc")}</span>
      </button>
    </div>
  );
}
