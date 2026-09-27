// ScanProgress.tsx — modal two-phase scan progress; backdrop is non-dismissable
// on purpose so a stray click can't abandon a running scan (cancel = explicit).
import { formatSize, formatNumber } from "../lib/format";
import { shortPath } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { ScanProgress } from "../lib/types";

interface Props {
  progress: ScanProgress;
  onCancel: () => void;
}

export function ScanProgressView({ progress, onCancel }: Props) {
  const { t } = useI18n();
  const searching = progress.phase === "search";
  const percent = !searching && progress.totalItems > 0 ? (progress.itemsFound / progress.totalItems) * 100 : 0;

  return (
    <div className="modal-backdrop">
      <div className="progress-panel">
        <div className="progress-head">
          <div className="progress-phase">
            <span className={`phase-dot ${searching ? "on" : "done"}`}>1</span>
            <span className={searching ? "on" : "done"}>{t("scan.searching")}</span>
            <span className={`phase-dot ${searching ? "" : "on"}`}>2</span>
            <span className={searching ? "" : "on"}>{t("scan.measuring")}</span>
          </div>
          <button className="btn btn-ghost btn-small" onClick={onCancel}>{t("scan.cancel")}</button>
        </div>
        <div className="progress-track">
          <div
            className={`progress-fill ${searching ? "indeterminate" : ""}`}
            style={searching ? undefined : { width: `${percent}%` }}
          />
        </div>
        <div className="progress-meta">
          <span>
            {searching
              ? t("scan.dirsChecked", { count: formatNumber(progress.dirsVisited) })
              : t("scan.measured", { done: progress.itemsFound, total: progress.totalItems })}
          </span>
          <span>
            {searching
              ? t("scan.candidates", { count: formatNumber(progress.itemsFound) })
              : progress.bytesSoFar > 0 && t("scan.recorded", { size: formatSize(progress.bytesSoFar) })}
          </span>
        </div>
        {progress.path && <div className="progress-path" title={progress.path}>{shortPath(progress.path, 70)}</div>}
      </div>
    </div>
  );
}
