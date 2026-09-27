// i18n.ts — translation lookup over the locale catalogs. Keys are dotted paths;
// a missing key returns the key itself so the mistake is visible in the UI.
import { useMemo } from "react";
import type { Lang } from "../lib/types";
import { useSettings } from "../lib/settings";
import id from "./locales/id";
import en from "./locales/en";

export type Dictionary = typeof id;

export const catalogs: Record<Lang, Dictionary> = { id, en };

export const availableLangs: { value: Lang; labelKey: string }[] = [
  { value: "id", labelKey: "lang.id" },
  { value: "en", labelKey: "lang.en" },
];

type Vars = Record<string, string | number>;

function lookup(dict: Dictionary, path: string): string | string[] | undefined {
  let node: unknown = dict;
  for (const part of path.split(".")) {
    if (typeof node !== "object" || node === null) return undefined;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" || Array.isArray(node) ? (node as string | string[]) : undefined;
}

function interpolate(template: string, vars?: Vars): string {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (match, key: string) =>
    key in vars ? String(vars[key]) : match,
  );
}

/** One string, falling back to Indonesian and finally to the key itself. */
export function translate(lang: Lang, path: string, vars?: Vars): string {
  const found = lookup(catalogs[lang], path) ?? lookup(catalogs.id, path);
  if (found === undefined) return path;
  return interpolate(Array.isArray(found) ? found.join(" ") : found, vars);
}

/** Several paragraphs, for the long-form pages. */
export function translateList(lang: Lang, path: string): string[] {
  const found = lookup(catalogs[lang], path) ?? lookup(catalogs.id, path);
  if (Array.isArray(found)) return found.map((p) => interpolate(p));
  if (typeof found === "string") return [found];
  return [path];
}

export interface TFunc {
  lang: Lang;
  t: (path: string, vars?: Vars) => string;
  tl: (path: string) => string[];
  /** Picks the plural form of a "plurals.<base>One/Many" pair. */
  p: (base: string, n: number) => string;
}

export function useT(lang: Lang): TFunc {
  return useMemo(
    () => ({
      lang,
      t: (path: string, vars?: Vars) => translate(lang, path, vars),
      tl: (path: string) => translateList(lang, path),
      p: (base: string, n: number) =>
        translate(lang, `${base}${lang === "en" && n === 1 ? "One" : "Many"}`),
    }),
    [lang],
  );
}

/** Convenience hook: translations for the language currently in settings. */
export function useI18n(): TFunc {
  return useT(useSettings().lang);
}
