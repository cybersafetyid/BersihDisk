// paths.ts — path helpers shared by the selection maths.

/** Normalises Windows separators so comparisons work on every platform. */
function toSlash(p: string): string {
  return p.split("\\").join("/");
}

/** True when `child` sits strictly below `parent`. */
export function isInside(child: string, parent: string): boolean {
  const c = toSlash(child);
  const p = toSlash(parent).replace(/\/+$/, "");
  if (!p) return false;
  return c !== p && c.startsWith(`${p}/`);
}

/** True when an ancestor of `path` is selected, so the selection already covers it. */
export function coveredByAncestor(selected: Set<string>, path: string): boolean {
  for (const s of selected) if (isInside(path, s)) return true;
  return false;
}

/** True when something below `path` is selected (drives the "partial" checkbox state). */
export function hasSelectedBelow(selected: Set<string>, path: string): boolean {
  for (const s of selected) if (isInside(s, path)) return true;
  return false;
}
