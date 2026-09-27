// backend.ts — wrapper around the generated Wails bindings (frontend/wailsjs)
// with our own types so the frontend stays decoupled from codegen details.
import * as BackendRaw from "../wailsjs/go/main/App";
import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
import type {
  DriveUI, CategoryUI, ScanResult, ScanProgress, DeleteProgress, DeleteResult,
  FolderEntry, UpdateInfo, UpdateProgress, UpdateResult, MachineInfo, AppInstallInfo, CacheReport,
} from "./lib/types";

const B = BackendRaw as any;

export const detectDrives = (): Promise<DriveUI[]> => B.DetectDrives();
export const listCategories = (): Promise<CategoryUI[]> => B.ListCategories();
export const startScan = (req: { roots: string[]; categories: string[] }): Promise<string> =>
  B.StartScan(req);
export const cancelScan = (): Promise<void> => B.CancelScan();
export const startDelete = (req: { paths: string[]; sizes: number[]; mode: string }): Promise<string> =>
  B.StartDelete(req);
export const cancelDelete = (): Promise<void> => B.CancelDelete();
export const appInfo = (): Promise<Record<string, unknown>> => B.AppInfo();

// Folder browsing and OS handoff.
export const listFolder = (path: string): Promise<FolderEntry[]> => B.ListFolder(path);
export const reveal = (path: string): Promise<void> => B.Reveal(path);

// In-app updates from GitHub Releases.
export const checkUpdate = (): Promise<UpdateInfo> => B.CheckUpdate();
export const startUpdateDownload = (url: string, digest: string): Promise<string> =>
  B.StartUpdateDownload(url, digest);
export const cancelUpdate = (): Promise<void> => B.CancelUpdate();
export const installUpdate = (path: string): Promise<void> => B.InstallUpdate(path);

// Settings page data and actions.
export const systemInfo = (): Promise<MachineInfo> => B.SystemInfo();
export const appDetails = (): Promise<AppInstallInfo> => B.AppDetails();
export const clearCache = (): Promise<CacheReport> => B.ClearCache();
export const appIconSupported = (): Promise<boolean> => B.AppIconSupported();
export const setAppIcon = (pngBase64: string): Promise<void> => B.SetAppIcon(pngBase64);
export const openLink = (url: string): Promise<void> => B.OpenLink(url);

export function onScanProgress(cb: (p: ScanProgress) => void) {
  EventsOn("scan:progress", cb);
  return () => EventsOff("scan:progress");
}
export function onScanFinished(cb: (r: ScanResult) => void) {
  EventsOn("scan:finished", cb);
  return () => EventsOff("scan:finished");
}
export function onDeleteProgress(cb: (p: DeleteProgress) => void) {
  EventsOn("delete:progress", cb);
  return () => EventsOff("delete:progress");
}
export function onDeleteFinished(cb: (r: DeleteResult) => void) {
  EventsOn("delete:finished", cb);
  return () => EventsOff("delete:finished");
}
export function onUpdateProgress(cb: (p: UpdateProgress) => void) {
  EventsOn("update:progress", cb);
  return () => EventsOff("update:progress");
}
export function onUpdateFinished(cb: (r: UpdateResult) => void) {
  EventsOn("update:finished", cb);
  return () => EventsOff("update:finished");
}
