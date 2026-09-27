// types.ts — data types following the backend JSON structs.

export interface DriveUI {
  name: string;
  mountPoint: string;
  totalBytes: number;
  freeBytes: number;
  root: boolean;
  removable: boolean;
}

/** Every cleanup category the backend can return; locales must cover all of them. */
export type CategoryId =
  | "nodejs" | "go" | "rust" | "gradle" | "maven" | "cpp" | "python" | "dotnet"
  | "xcode" | "android" | "temp" | "ai" | "electron" | "docker" | "brew" | "php"
  | "pip" | "ruby" | "flutter" | "terraform" | "nuget" | "jetbrains" | "webbuild"
  | "emulators";

export interface CategoryUI {
  id: CategoryId;
  icon: string;
  optIn: boolean;
}

export interface ScanItem {
  path: string;
  size: number;
  category: string;
  /** Set when this path is the target of a symlinked cache location. */
  linkFrom?: string;
  /** Set when another item already contains this one, so its bytes are counted there. */
  nestedIn?: string;
}

export interface ScanResult {
  items: ScanItem[];
  totalBytes: number;
  itemCount: number;
  dirsSkipped: number;
  partial: boolean;
}

export interface ScanProgress {
  phase: string;
  path: string;
  itemsFound: number;
  totalItems: number;
  bytesSoFar: number;
  dirsVisited: number;
}

export interface FolderEntry {
  name: string;
  path: string;
  isDir: boolean;
  isLink: boolean;
  size: number;
}

export interface UpdateInfo {
  available: boolean;
  current: string;
  latest: string;
  notes: string;
  asset: string;
  url: string;
  bytes: number;
  digest: string;
  page: string;
}

export interface UpdateProgress {
  path: string;
  bytes: number;
  total: number;
  speed: number;
  percent: number;
  phase: string;
}

export interface UpdateResult {
  path: string;
  error?: string;
}

export interface MachineInfo {
  os: string;
  arch: string;
  cpuCores: number;
  memoryBytes: number;
  hostname: string;
  user: string;
  home: string;
  goVersion: string;
}

export interface AppInstallInfo {
  version: string;
  path: string;
  bytes: number;
}

export interface CacheReport {
  files: number;
  bytes: number;
}

export interface DeleteProgress {
  done: number;
  total: number;
  bytes: number;
  path: string;
}

export interface FailureItem {
  path: string;
  message: string;
}

export interface DeleteResult {
  ok: number;
  failed: number;
  bytes: number;
  failures: FailureItem[];
  duration: number;
}

export type Theme = "light" | "dark" | "system";
export type Lang = "id" | "en";
