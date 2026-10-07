// Dropdown.tsx — a button that opens a small action menu (custom, so it looks the
// same on every OS, unlike a native <select>). Closes on outside click, Escape or
// choosing an item; arrow keys move between items; the menu animates out.
import { useEffect, useRef, useState } from "react";
import { Icon } from "./Icon";

export interface DropdownItem {
  key: string;
  icon: string;
  label: string;
  /** Shown with a check mark when it describes the current state. */
  checked?: boolean;
  onSelect: () => void;
}

interface Props {
  label: string;
  icon?: string;
  items: DropdownItem[];
}

/** Keep in step with the .settings-dropdown.closing animation in settings.css. */
const EXIT_MS = 140;

export function Dropdown({ label, icon, items }: Props) {
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      setMounted(true);
      return;
    }
    const id = window.setTimeout(() => setMounted(false), EXIT_MS);
    return () => window.clearTimeout(id);
  }, [open]);

  // Focus the first item when the menu opens.
  useEffect(() => {
    if (open && mounted) menu.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
  }, [open, mounted]);

  useEffect(() => {
    if (!open) return;
    const away = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", away);
    return () => document.removeEventListener("mousedown", away);
  }, [open]);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false);
      trigger.current?.focus();
      return;
    }
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    e.preventDefault();
    const els = [...(menu.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])];
    if (els.length === 0) return;
    const at = els.indexOf(document.activeElement as HTMLElement);
    const next = e.key === "ArrowDown" ? (at + 1) % els.length : (at - 1 + els.length) % els.length;
    els[next].focus();
  };

  return (
    <div className="dropdown" ref={wrap} onKeyDown={onKeyDown}>
      <button
        ref={trigger}
        className={`btn btn-ghost btn-small dropdown-trigger ${open ? "active" : ""}`}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
      >
        {icon && <Icon name={icon} size={15} />}
        <span>{label}</span>
        <Icon name="chevron-down" size={14} className="dropdown-chevron" />
      </button>

      {mounted && (
        <div ref={menu} className={`settings-dropdown dropdown-menu ${open ? "" : "closing"}`} role="menu">
          {items.map((it, i) => (
            <div key={it.key} className="menu-row" style={{ "--i": i } as React.CSSProperties}>
              <button
                role="menuitem"
                className="menu-item"
                onClick={() => {
                  setOpen(false);
                  it.onSelect();
                  trigger.current?.focus();
                }}
              >
                <Icon name={it.icon} size={15} />
                <span className="menu-label">{it.label}</span>
                {it.checked && (
                  <span className="menu-check" aria-hidden="true">
                    <Icon name="check" size={12} />
                  </span>
                )}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
