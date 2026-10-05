export namespace analyzer {
	
	export class Entry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    isLink: boolean;
	    size: number;
	    level: string;
	    reasons?: string[];
	    skipped?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.isLink = source["isLink"];
	        this.size = source["size"];
	        this.level = source["level"];
	        this.reasons = source["reasons"];
	        this.skipped = source["skipped"];
	    }
	}
	export class Result {
	    path: string;
	    totalBytes: number;
	    entries: Entry[];
	    files: number;
	    dirs: number;
	    skipped: number;
	    partial: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.totalBytes = source["totalBytes"];
	        this.entries = this.convertValues(source["entries"], Entry);
	        this.files = source["files"];
	        this.dirs = source["dirs"];
	        this.skipped = source["skipped"];
	        this.partial = source["partial"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace browser {
	
	export class Entry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    isLink: boolean;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.isLink = source["isLink"];
	        this.size = source["size"];
	    }
	}

}

export namespace main {
	
	export class AppInstall {
	    version: string;
	    path: string;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new AppInstall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.path = source["path"];
	        this.bytes = source["bytes"];
	    }
	}
	export class CacheReport {
	    files: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new CacheReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.bytes = source["bytes"];
	    }
	}
	export class CategoryUI {
	    id: string;
	    icon: string;
	    optIn: boolean;
	    risk: string;
	
	    static createFrom(source: any = {}) {
	        return new CategoryUI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.icon = source["icon"];
	        this.optIn = source["optIn"];
	        this.risk = source["risk"];
	    }
	}
	export class DeleteRequest {
	    paths: string[];
	    sizes: number[];
	    mode: string;
	    acknowledged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.sizes = source["sizes"];
	        this.mode = source["mode"];
	        this.acknowledged = source["acknowledged"];
	    }
	}
	export class DriveUI {
	    name: string;
	    mountPoint: string;
	    totalBytes: number;
	    freeBytes: number;
	    root: boolean;
	    removable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DriveUI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.mountPoint = source["mountPoint"];
	        this.totalBytes = source["totalBytes"];
	        this.freeBytes = source["freeBytes"];
	        this.root = source["root"];
	        this.removable = source["removable"];
	    }
	}
	export class ScanRequest {
	    roots: string[];
	    categories: string[];
	
	    static createFrom(source: any = {}) {
	        return new ScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = source["roots"];
	        this.categories = source["categories"];
	    }
	}

}

export namespace sysinfo {
	
	export class Machine {
	    os: string;
	    arch: string;
	    cpuCores: number;
	    memoryBytes: number;
	    hostname: string;
	    user: string;
	    home: string;
	    goVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new Machine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.cpuCores = source["cpuCores"];
	        this.memoryBytes = source["memoryBytes"];
	        this.hostname = source["hostname"];
	        this.user = source["user"];
	        this.home = source["home"];
	        this.goVersion = source["goVersion"];
	    }
	}

}

export namespace uninstall {
	
	export class Installed {
	    id: string;
	    provider: string;
	    kind: string;
	    name: string;
	    version?: string;
	    path?: string;
	    icon: string;
	    notes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Installed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.path = source["path"];
	        this.icon = source["icon"];
	        this.notes = source["notes"];
	    }
	}
	export class ListResult {
	    packages: Installed[];
	    unavailable: string[];
	
	    static createFrom(source: any = {}) {
	        return new ListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.packages = this.convertValues(source["packages"], Installed);
	        this.unavailable = source["unavailable"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Warning {
	    code: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new Warning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.detail = source["detail"];
	    }
	}
	export class Step {
	    id: string;
	    kind: string;
	    label: string;
	    detail?: string;
	    size?: number;
	    level: string;
	    reasons?: string[];
	    selected: boolean;
	    manual?: string;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.detail = source["detail"];
	        this.size = source["size"];
	        this.level = source["level"];
	        this.reasons = source["reasons"];
	        this.selected = source["selected"];
	        this.manual = source["manual"];
	    }
	}
	export class Plan {
	    id: string;
	    packageId: string;
	    name: string;
	    steps: Step[];
	    warnings?: Warning[];
	    totalBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.packageId = source["packageId"];
	        this.name = source["name"];
	        this.steps = this.convertValues(source["steps"], Step);
	        this.warnings = this.convertValues(source["warnings"], Warning);
	        this.totalBytes = source["totalBytes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Request {
	    planId: string;
	    steps: string[];
	    mode: string;
	    acknowledged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.planId = source["planId"];
	        this.steps = source["steps"];
	        this.mode = source["mode"];
	        this.acknowledged = source["acknowledged"];
	    }
	}
	

}

export namespace updater {
	
	export class Info {
	    available: boolean;
	    current: string;
	    latest: string;
	    notes: string;
	    asset: string;
	    url: string;
	    bytes: number;
	    digest: string;
	    page: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.notes = source["notes"];
	        this.asset = source["asset"];
	        this.url = source["url"];
	        this.bytes = source["bytes"];
	        this.digest = source["digest"];
	        this.page = source["page"];
	    }
	}

}

