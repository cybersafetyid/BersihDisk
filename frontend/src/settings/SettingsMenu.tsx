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

export function SettingsMenu({ iconSupported, onOpen }: Props) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  const items = visibleSections(iconSupported);

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
        className={`btn btn-ghost btn-small ${open ? "active" : ""}`}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        title={t("settings.entry")}
      >
        <Icon name="settings" size={15} />
        <span className="theme-label">{t("settings.entry")}</span>
      </button>

      {open && (
        <div className="settings-dropdown" role="menu">
          {items.map((s, i) => (
            <div key={s.key}>
              {i > 0 && items[i - 1].group !== s.group && <div className="menu-sep" />}
              <button
                role="menuitem"
                className="menu-item"
                onClick={() => { setOpen(false); onOpen(s.key); }}
              >
                <Icon name={s.icon} size={15} />
                <span>{t(`settings.sections.${s.key}`)}</span>
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
