// appIcons.ts — icon variants offered in settings, plus the encoder the backend
// needs to apply one to the running app.
import type { IconKey } from "./settings";
import citrusIcon from "../assets/icons/citrus.png";
import defaultIcon from "../assets/icons/default.png";
import midnightIcon from "../assets/icons/midnight.png";

export const iconAssets: Record<IconKey, string> = {
  default: defaultIcon,
  midnight: midnightIcon,
  citrus: citrusIcon,
};

export const iconKeys: IconKey[] = ["default", "midnight", "citrus"];

/** Reads an icon asset and returns it as base64 for the binding. */
export async function iconBase64(key: IconKey): Promise<string> {
  const res = await fetch(iconAssets[key]);
  if (!res.ok) throw new Error(`icon asset ${key} could not be loaded`);
  const bytes = new Uint8Array(await res.arrayBuffer());
  let binary = "";
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}
