// DeleteProgress.tsx — overlay shown while deletion is running.
import { formatSize } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { DeleteProgress } from "../lib/types";

export function DeleteProgressView({ progress }: { progress: DeleteProgress }) {
  const { t } = useI18n();
  const percent = progress.total > 0 ? (progress.done / progress.total) * 100 : 0;

  return (
    <div className="modal-backdrop">
      <div className="modal modal-small">
        <h2>{t("delete.title")}</h2>
        <div className="progress-track">
          <div className="progress-fill" style={{ width: `${percent}%` }} />
        </div>
        <div className="progress-meta">
          <span>{t("delete.progress", { done: progress.done, total: progress.total })}</span>
          <span>{t("delete.freed", { size: formatSize(progress.bytes) })}</span>
        </div>
      </div>
    </div>
  );
}
