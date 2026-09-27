// format.ts — display formatting utilities (Indonesian UI strings).

const UNITS = ["B", "KB", "MB", "GB", "TB"];

export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), UNITS.length - 1);
  const value = bytes / Math.pow(1024, i);
  const decimals = value >= 100 || i === 0 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(decimals).replace(".", ",")} ${UNITS[i]}`;
}

export function formatNumber(n: number): string {
  return new Intl.NumberFormat("id-ID").format(n);
}

export function formatPercent(part: number, total: number): string {
  if (total <= 0) return "0%";
  const p = Math.round((part / total) * 100);
  return `${Math.min(100, Math.max(0, p))}%`;
}

// shortPath truncates long paths in the middle: /a/b/…/c/d
export function shortPath(p: string, max = 58): string {
  if (p.length <= max) return p;
  const head = p.slice(0, Math.floor(max * 0.45));
  const tail = p.slice(-Math.floor(max * 0.4));
  return `${head}…${tail}`;
}

export function lastName(p: string): string {
  const parts = p.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] ?? p;
}
