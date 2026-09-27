// settings/sections.ts — the settings sections, shared by the menu and the page.
export type Section =
  | "language" | "appearance" | "icon" | "updates" | "cache"
  | "about" | "system" | "app" | "privacy" | "terms" | "feedback" | "donate";

export interface SectionMeta {
  key: Section;
  icon: string;
  group: "preferences" | "information" | "legal";
}

export const sections: SectionMeta[] = [
  { key: "language", icon: "globe", group: "preferences" },
  { key: "appearance", icon: "palette", group: "preferences" },
  { key: "icon", icon: "eraser", group: "preferences" },
  { key: "updates", icon: "refresh", group: "preferences" },
  { key: "cache", icon: "trash", group: "preferences" },
  { key: "about", icon: "info", group: "information" },
  { key: "system", icon: "cpu", group: "information" },
  { key: "app", icon: "harddrive", group: "information" },
  { key: "privacy", icon: "shield", group: "legal" },
  { key: "terms", icon: "scale", group: "legal" },
  { key: "feedback", icon: "mail", group: "legal" },
  { key: "donate", icon: "heart", group: "legal" },
];

/** Sections shown on this platform — the icon picker needs runtime support. */
export function visibleSections(iconSupported: boolean): SectionMeta[] {
  return sections.filter((s) => s.key !== "icon" || iconSupported);
}
