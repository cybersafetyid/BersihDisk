// SettingsMenu.tsx — gear button in the header that opens the settings menu.
import { useEffect, useRef, useState } from "react";
import { Icon } from "../common/Icon";
import { useI18n } from "../i18n/i18n";
import { visibleSections } from "./sections";
import type { Section } from "./sections";

interface Props {
  iconSupported: boolean;
  onOpen: (section: Section) => void;
}

/** Keep in step with the .settings-dropdown.closing animation in settings.css. */
const EXIT_MS = 140;

export function SettingsMenu({ iconSupported, onOpen }: Props) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  // The dropdown stays mounted while its exit animation plays, then unmounts.
  const [mounted, setMounted] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  const items = visibleSections(iconSupported);

  useEffect(() => {
    if (open) {
      setMounted(true);
      return;
    }
    const id = window.setTimeout(() => setMounted(false), EXIT_MS);
    return () => window.clearTimeout(id);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const away = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) setOpen(false);
    };
    const esc = (e: KeyboardEvent) => { if (e.key === "Escape") setOpen(false); };
    document.addEventListener("mousedown", away);
    document.addEventListener("keydown", esc);
    return () => {
      document.removeEventListener("mousedown", away);
      document.removeEventListener("keydown", esc);
    };
  }, [open]);

  return (
    <div className="settings-menu" ref={wrap}>
      <button
        className={`btn btn-ghost btn-small settings-trigger-btn ${open ? "active" : ""}`}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        title={t("settings.entry")}
      >
        <span className="settings-gear-wrap">
          <Icon name="settings" size={15} />
        </span>
        <span className="theme-label">{t("settings.entry")}</span>
        <Icon name="chevron-down" size={12} className="settings-chevron" />
      </button>

      {mounted && (
        <div className={`settings-dropdown ${open ? "" : "closing"}`} role="menu">
          <div className="dropdown-brand-header">
            <span className="dropdown-brand-title">{t("settings.title")}</span>
          </div>

          <div className="dropdown-items-scroll">
            {items.map((s, i) => {
              const isFirstOfGroup = i === 0 || items[i - 1].group !== s.group;
              return (
                <div key={s.key} className="menu-row" style={{ "--i": i } as React.CSSProperties}>
                  {isFirstOfGroup && (
                    <div className="menu-group-header">
                      <span>{t(`settings.groups.${s.group}`)}</span>
                    </div>
                  )}
                  <button
                    role="menuitem"
                    className="menu-item"
                    onClick={() => { setOpen(false); onOpen(s.key); }}
                  >
                    <span className="menu-icon-wrap">
                      <Icon name={s.icon} size={14} />
                    </span>
                    <span className="menu-item-text">{t(`settings.sections.${s.key}`)}</span>
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
