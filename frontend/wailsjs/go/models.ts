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
	
	    static createFrom(source: any = {}) {
	        return new CategoryUI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.icon = source["icon"];
	        this.optIn = source["optIn"];
	    }
	}
	export class DeleteRequest {
	    paths: string[];
	    sizes: number[];
	    mode: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.sizes = source["sizes"];
	        this.mode = source["mode"];
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

