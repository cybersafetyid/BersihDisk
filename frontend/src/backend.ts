// backend.ts — wrapper around the generated Wails bindings (frontend/wailsjs)
// with our own types so the frontend stays decoupled from codegen details.
import * as BackendRaw from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import type {
  DriveUI, CategoryUI, ScanResult, ScanProgress, DeleteProgress, DeleteResult,
  FolderEntry, UpdateInfo, UpdateProgress, UpdateResult, MachineInfo, AppInstallInfo, CacheReport,
  PackageList, UninstallPlan, UninstallProgress, UninstallResult,
  AnalyzeResult, AnalyzeProgress,
} from "./lib/types";

const B = BackendRaw as any;

export const detectDrives = (): Promise<DriveUI[]> => B.DetectDrives();
export const listCategories = (): Promise<CategoryUI[]> => B.ListCategories();
export const startScan = (req: { roots: string[]; categories: string[] }): Promise<string> =>
  B.StartScan(req);
export const cancelScan = (): Promise<void> => B.CancelScan();
export const startDelete = (req: { paths: string[]; sizes: number[]; mode: string; acknowledged: boolean }): Promise<string> =>
  B.StartDelete(req);
export const cancelDelete = (): Promise<void> => B.CancelDelete();
export const appInfo = (): Promise<Record<string, unknown>> => B.AppInfo();

// Drive analyzer: size a folder's children and drill in, like a disk map.
export const startAnalyze = (path: string): Promise<string> => B.StartAnalyze(path);
export const cancelAnalyze = (): Promise<void> => B.CancelAnalyze();
export function onAnalyzeProgress(cb: (p: AnalyzeProgress) => void) {
  return EventsOn("analyze:progress", cb);
}
export function onAnalyzeFinished(cb: (r: AnalyzeResult) => void) {
  return EventsOn("analyze:finished", cb);
}

// Folder browsing and OS handoff.
export const listFolder = (path: string): Promise<FolderEntry[]> => B.ListFolder(path);
export const measurePaths = (paths: string[]): Promise<number[]> => B.MeasurePaths(paths);
export const reveal = (path: string): Promise<void> => B.Reveal(path);

// Uninstaller: list what is installed, review a plan, run the chosen steps.
export const listPackages = (): Promise<PackageList> => B.ListPackages();
export const planUninstall = (packageId: string): Promise<UninstallPlan> => B.PlanUninstall(packageId);
export const startUninstall = (req: { planId: string; steps: string[]; mode: string; acknowledged: boolean }): Promise<string> =>
  B.StartUninstall(req);
/** The application's own icon as a data URL; "" when it has none to show. */
export const appIcon = (packageId: string): Promise<string> => B.AppIcon(packageId);
export const cancelUninstall = (): Promise<void> => B.CancelUninstall();

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

// EventsOn returns a function that removes only this listener, unlike EventsOff(name),
// which would drop every listener of the event.
export function onScanProgress(cb: (p: ScanProgress) => void) {
  return EventsOn("scan:progress", cb);
}
export function onScanFinished(cb: (r: ScanResult) => void) {
  return EventsOn("scan:finished", cb);
}
export function onDeleteProgress(cb: (p: DeleteProgress) => void) {
  return EventsOn("delete:progress", cb);
}
export function onDeleteFinished(cb: (r: DeleteResult) => void) {
  return EventsOn("delete:finished", cb);
}
export function onUpdateProgress(cb: (p: UpdateProgress) => void) {
  return EventsOn("update:progress", cb);
}
export function onUpdateFinished(cb: (r: UpdateResult) => void) {
  return EventsOn("update:finished", cb);
}
export function onUninstallProgress(cb: (p: UninstallProgress) => void) {
  return EventsOn("uninstall:progress", cb);
}
export function onUninstallFinished(cb: (r: UninstallResult) => void) {
  return EventsOn("uninstall:finished", cb);
}
