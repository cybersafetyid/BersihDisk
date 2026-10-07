// Toast.tsx — auto-dismissing toast notifications.
import { useEffect } from "react";
import { useI18n } from "../i18n/i18n";
import { Icon } from "./Icon";

export interface ToastData {
  id: number;
  message: string;
  detail?: string;
  variant: "success" | "error" | "info";
  action?: { label: string; run: () => void };
}

export function Toast({ toast, onClose }: { toast: ToastData; onClose: (id: number) => void }) {
  const { t } = useI18n();

  useEffect(() => {
    const t = setTimeout(() => onClose(toast.id), toast.action ? 8000 : 4500);
    return () => clearTimeout(t);
  }, [toast, onClose]);

  const iconName = toast.variant === "success" ? "check" : toast.variant === "error" ? "x" : "info";

  return (
    <div className={`toast toast-${toast.variant}`}>
      <span className={`toast-icon toast-icon-${toast.variant}`}>
        <Icon name={iconName} size={14} />
      </span>
      <div className="toast-body">
        <div className="toast-message">{toast.message}</div>
        {toast.detail && <div className="toast-detail">{toast.detail}</div>}
      </div>
      {toast.action && (
        <button
          className="toast-action"
          onClick={() => { toast.action!.run(); onClose(toast.id); }}
        >
          {toast.action.label}
        </button>
      )}
      <button className="toast-close" onClick={() => onClose(toast.id)} aria-label={t("common.close")}>
        <Icon name="x" size={14} />
      </button>
    </div>
  );
}
