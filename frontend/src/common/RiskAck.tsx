// RiskAck.tsx — the "I understand" checkbox that gates any deletion above safe.
import { useI18n } from "../i18n/i18n";

interface Props {
  checked: boolean;
  onChange: (next: boolean) => void;
  /** danger uses the red style. */
  danger?: boolean;
}

export function RiskAck({ checked, onChange, danger }: Props) {
  const { t } = useI18n();
  return (
    <label className={`risk-ack ${danger ? "danger" : ""}`}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span>{t(danger ? "safety.ackDanger" : "safety.ack")}</span>
    </label>
  );
}
