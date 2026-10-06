// format.ts — display formatting utilities, localised to the active language.
import type { Lang } from "./types";

const UNITS = ["B", "KB", "MB", "GB", "TB"];

// The active language drives number and decimal formatting. It defaults to
// Indonesian (the app's primary locale); the i18n hook syncs it on every change.
let currentLang: Lang = "id";

/** Sets the locale used by the formatters; call once per language change. */
export function setLocale(lang: Lang): void {
  currentLang = lang;
}

// toLocale uses "," for a decimal point, id-ID "." for thousands; en-US the reverse.
const numberLocale = () => (currentLang === "en" ? "en-US" : "id-ID");
const decimalSeparator = () => (currentLang === "en" ? "." : ",");

export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), UNITS.length - 1);
  const value = bytes / Math.pow(1024, i);
  const decimals = value >= 100 || i === 0 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(decimals).replace(".", decimalSeparator())} ${UNITS[i]}`;
}

export function formatNumber(n: number): string {
  return new Intl.NumberFormat(numberLocale()).format(n);
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
