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

/** How risky deleting something is; the backend decides, the UI only shows it. */
export type Level = "safe" | "caution" | "danger" | "blocked";

export interface CategoryUI {
  id: CategoryId;
  icon: string;
  optIn: boolean;
  /** Worst level any location of the category can reach. */
  risk: Level;
}

export interface ScanItem {
  path: string;
  size: number;
  category: string;
  /** Set when this path is the target of a symlinked cache location. */
  linkFrom?: string;
  /** Set when another item already contains this one, so its bytes are counted there. */
  nestedIn?: string;
  /** Risk of deleting the item and the reason codes behind it (translated under safety.reason). */
  level: Level;
  reasons?: string[];
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

// ---- drive analyzer (disk usage view) ----

/** One child of an analyzed folder, with its recursive size and deletion risk. */
export interface AnalyzeEntry {
  name: string;
  path: string;
  isDir: boolean;
  isLink: boolean;
  size: number;
  /** Risk of deleting the entry, decided by the backend safety rules. */
  level: Level;
  reasons?: string[];
  /** A virtual/pseudo folder or the OS trash: listed but not measured. */
  skipped?: boolean;
}

export interface AnalyzeResult {
  path: string;
  totalBytes: number;
  entries: AnalyzeEntry[];
  files: number;
  dirs: number;
  skipped: number;
  partial: boolean;
  /** Set when the folder could not be listed. */
  error?: string;
}

export interface AnalyzeProgress {
  path: string;
  dirs: number;
  total: number;
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
  /** Known fixable cause, translated under safety.hint (e.g. fullDiskAccess). */
  hint?: string;
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

// ---- uninstaller ----

export type InstalledKind = "app" | "runtime" | "package";

export interface InstalledItem {
  id: string;
  provider: string;
  kind: InstalledKind;
  name: string;
  version?: string;
  path?: string;
  icon: string;
  /** Hint codes translated under uninstall.note. */
  notes?: string[];
}

export interface PackageList {
  packages: InstalledItem[];
  /** Providers that failed while listing (not merely "not installed"). */
  unavailable: string[];
}

export type StepKind = "command" | "path" | "profile" | "registry" | "envpath";

export interface PlanStep {
  id: string;
  kind: StepKind;
  label: string;
  detail?: string;
  size?: number;
  level: Level;
  reasons?: string[];
  selected: boolean;
  /** Command to run by hand when the step needs administrator rights. */
  manual?: string;
}

export interface PlanWarning {
  code: string;
  detail?: string;
}

export interface UninstallPlan {
  id: string;
  packageId: string;
  name: string;
  steps: PlanStep[];
  warnings?: PlanWarning[];
  totalBytes: number;
}

export interface UninstallProgress {
  done: number;
  total: number;
  step: string;
  bytes: number;
}

export interface StepResult {
  id: string;
  status: "done" | "failed" | "skipped" | "manual";
  message?: string;
  hint?: string;
}

export interface UninstallResult {
  ok: number;
  failed: number;
  skipped: number;
  bytes: number;
  steps: StepResult[];
  manual?: string[];
  aborted?: boolean;
  duration: number;
  /** Set when the request was refused before anything ran. */
  error?: string;
}
