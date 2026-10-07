// DriveSelector.tsx — multi-select drive cards with capacity bars and modern metrics.
import { Icon } from "../common/Icon";
import { formatSize } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { DriveUI } from "../lib/types";

interface Props {
  drives: DriveUI[];
  selected: Set<string>;
  onToggle: (mountPoint: string) => void;
}

export function DriveSelector({ drives, selected, onToggle }: Props) {
  const { t } = useI18n();

  if (drives.length === 0) {
    return <div className="empty">{t("drive.none")}</div>;
  }
  return (
    <div className="drive-grid">
      {drives.map((d, i) => {
        const used = d.usedBytes;
        const percent = d.totalBytes > 0 ? Math.min(100, (used / d.totalBytes) * 100) : 0;
        const active = selected.has(d.mountPoint);
        const iconName = d.removable ? "usb" : d.root ? "monitor" : "harddrive";
        const tagLabel = d.root
          ? t("drive.systemRoot")
          : d.removable
          ? t("drive.removable")
          : t("drive.internal");

        return (
          <button
            key={d.mountPoint}
            className={`drive-card ${active ? "active" : ""}`}
            style={{ animationDelay: `${i * 60}ms` }}
            onClick={() => onToggle(d.mountPoint)}
            aria-pressed={active}
            title={d.mountPoint}
          >
            <div className="drive-head">
              <div className="drive-icon-badge">
                <Icon name={iconName} size={18} />
              </div>
              <div className="drive-head-info">
                <div className="drive-name-row">
                  <span className="drive-name">{d.name}</span>
                  <span className={`drive-tag ${d.root ? "tag-root" : d.removable ? "tag-usb" : ""}`}>
                    {tagLabel}
                  </span>
                </div>
                <div className="drive-path-micro" title={d.mountPoint}>
                  {d.mountPoint}
                </div>
              </div>
              <div className={`check-indicator ${active ? "on" : ""}`} aria-hidden="true">
                {active && <Icon name="check" size={12} />}
              </div>
            </div>

            <div className="drive-bar-wrap">
              <div className="drive-bar">
                <div
                  className={`drive-bar-fill ${percent > 90 ? "critical" : percent > 75 ? "high" : ""}`}
                  style={{ width: `${percent}%` }}
                />
              </div>
              <div className="drive-percent-badge">
                <span className={percent > 90 ? "critical-text" : percent > 75 ? "warning-text" : ""}>
                  {Math.round(percent)}%
                </span>
              </div>
            </div>

            <div className="drive-stats-row">
              <div className="drive-stat-col">
                <span className="drive-stat-val">{formatSize(used)}</span>
                <span className="drive-stat-lbl">{t("drive.used", { size: "" }).trim() || "Terpakai"}</span>
              </div>
              <div className="drive-stat-divider" />
              <div className="drive-stat-col">
                <span className="drive-stat-val free-val">{formatSize(d.freeBytes)}</span>
                <span className="drive-stat-lbl">{t("drive.free", { size: "" }).trim() || "Bebas"}</span>
              </div>
              <div className="drive-stat-divider" />
              <div className="drive-stat-col">
                <span className="drive-stat-val total-val">{formatSize(d.totalBytes)}</span>
                <span className="drive-stat-lbl">{t("drive.total", { size: "" }).trim() || "Total"}</span>
              </div>
            </div>
          </button>
        );
      })}
    </div>
  );
}

