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
