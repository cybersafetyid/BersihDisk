// RiskBadge.tsx — the level of a location as a small badge with a tooltip that
// lists why. Safe items show nothing: the warning is the exception.
import { Icon } from "./Icon";
import { useI18n } from "../i18n/i18n";
import type { Level } from "../lib/types";

interface Props {
  level: Level;
  /** Reason codes; each is translated under safety.reason. */
  reasons?: string[];
  /** Text of the category note, used for the "categoryNote" reason. */
  categoryNote?: string;
}

const icons: Record<Level, string> = {
  safe: "shield-check",
  caution: "triangle-alert",
  danger: "shield-alert",
  blocked: "lock",
};

/** The translated sentences behind a list of reason codes. */
export function useReasonText(categoryNote?: string) {
  const { t } = useI18n();
  return (codes: string[] | undefined) =>
    (codes ?? []).map((c) => (c === "categoryNote" && categoryNote ? categoryNote : t(`safety.reason.${c}`)));
}

export function RiskBadge({ level, reasons, categoryNote }: Props) {
  const { t } = useI18n();
  const text = useReasonText(categoryNote);
  if (level === "safe") return null;
  const tip = text(reasons).join("\n");
  return (
    <span className={`risk-badge risk-${level}`} title={tip}>
      <Icon name={icons[level]} size={11} />
      {t(`safety.level.${level}`)}
    </span>
  );
}
