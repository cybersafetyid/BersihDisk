// DriveSelector.tsx — multi-select drive cards with capacity bars.
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
              <Icon name={iconName} size={20} />
              <span className="drive-name">{d.name}</span>
              <span className={`check ${active ? "on" : ""}`}>✓</span>
            </div>
            <div className="drive-bar">
              <div
                className={`drive-bar-fill ${percent > 90 ? "critical" : percent > 75 ? "high" : ""}`}
                style={{ width: `${percent}%` }}
              />
            </div>
            <div className="drive-detail">
              <span>{t("drive.used", { size: formatSize(used) })}</span>
              <span>{t("drive.free", { size: formatSize(d.freeBytes) })}</span>
            </div>
            <div className="drive-path">{d.mountPoint}</div>
          </button>
        );
      })}
    </div>
  );
}
