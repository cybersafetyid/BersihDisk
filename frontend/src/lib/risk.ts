// risk.ts — helpers over the backend's risk levels, shared by the cleaner and
// the uninstaller so both treat "needs a warning" the same way.
import { isInside } from "./paths";
import type { Level, ScanItem } from "./types";

const rank: Record<Level, number> = { safe: 0, caution: 1, danger: 2, blocked: 3 };

/** 0 (safe) … 3 (blocked), for sorting. */
export function severity(level: Level): number {
  return rank[level];
}

export function worse(a: Level, b: Level): Level {
  return rank[b] > rank[a] ? b : a;
}

/** A level the user has to acknowledge before deleting. */
export function needsAck(level: Level): boolean {
  return level === "caution" || level === "danger";
}

/** The scan item a path is, or sits inside (the innermost one). */
export function itemFor(items: ScanItem[], path: string): ScanItem | undefined {
  let best: ScanItem | undefined;
  for (const it of items) {
    if ((it.path === path || isInside(path, it.path)) && (!best || it.path.length > best.path.length)) best = it;
  }
  return best;
}

/** Items a bulk action ("select all", "select ≥ 100 MB") may tick: never risky or protected ones. */
export function bulkSelectable(it: ScanItem): boolean {
  return it.level === "safe";
}
