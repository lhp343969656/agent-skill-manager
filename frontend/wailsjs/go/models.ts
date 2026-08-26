export namespace github {
	
	export class Version {
	    Kind: string;
	    Display: string;
	    Ref: string;
	    SHA: string;
	
	    static createFrom(source: any = {}) {
	        return new Version(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Kind = source["Kind"];
	        this.Display = source["Display"];
	        this.Ref = source["Ref"];
	        this.SHA = source["SHA"];
	    }
	}

}

export namespace main {
	
	export class CheckUpdateResult {
	    skillId: string;
	    isLocal: boolean;
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    currentCommit: string;
	    latestCommit: string;
	    updateNotes: string;
	    checkError: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckUpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillId = source["skillId"];
	        this.isLocal = source["isLocal"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.currentCommit = source["currentCommit"];
	        this.latestCommit = source["latestCommit"];
	        this.updateNotes = source["updateNotes"];
	        this.checkError = source["checkError"];
	    }
	}
	export class InstallRequest {
	    url: string;
	    gitRef: string;
	    note: string;
	    skillPath: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.gitRef = source["gitRef"];
	        this.note = source["note"];
	        this.skillPath = source["skillPath"];
	    }
	}
	export class InstallResult {
	    skillId: string;
	    installPath: string;
	    syncedAgents: string[];
	    conflicts: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillId = source["skillId"];
	        this.installPath = source["installPath"];
	        this.syncedAgents = source["syncedAgents"];
	        this.conflicts = source["conflicts"];
	    }
	}
	export class LocalInstallResult {
	    skillId: string;
	    skillName: string;
	    installPath: string;
	    syncedAgents: string[];
	    conflicts: string[];
	
	    static createFrom(source: any = {}) {
	        return new LocalInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillId = source["skillId"];
	        this.skillName = source["skillName"];
	        this.installPath = source["installPath"];
	        this.syncedAgents = source["syncedAgents"];
	        this.conflicts = source["conflicts"];
	    }
	}
	export class RepoSkill {
	    name: string;
	    relPath: string;
	
	    static createFrom(source: any = {}) {
	        return new RepoSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.relPath = source["relPath"];
	    }
	}
	export class ScanResult {
	    owner: string;
	    repo: string;
	    subPath: string;
	    defaultRef: string;
	    description: string;
	    stars: number;
	    versions: github.Version[];
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.owner = source["owner"];
	        this.repo = source["repo"];
	        this.subPath = source["subPath"];
	        this.defaultRef = source["defaultRef"];
	        this.description = source["description"];
	        this.stars = source["stars"];
	        this.versions = this.convertValues(source["versions"], github.Version);
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
	export class UninstallResult {
	    skillId: string;
	    removed: string[];
	    failed: string[];
	
	    static createFrom(source: any = {}) {
	        return new UninstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skillId = source["skillId"];
	        this.removed = source["removed"];
	        this.failed = source["failed"];
	    }
	}

}

export namespace models {
	
	export class Agent {
	    id: string;
	    adapterId: string;
	    displayName: string;
	    skillsPath: string;
	    enabled: boolean;
	    detected: boolean;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Agent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.adapterId = source["adapterId"];
	        this.displayName = source["displayName"];
	        this.skillsPath = source["skillsPath"];
	        this.enabled = source["enabled"];
	        this.detected = source["detected"];
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Skill {
	    id: string;
	    displayName: string;
	    sourceUrl: string;
	    repositoryOwner: string;
	    repositoryName: string;
	    repositoryPath: string;
	    currentVersionId: string;
	    note: string;
	    displayVersion: string;
	    // Go type: time
	    installedAt: any;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.sourceUrl = source["sourceUrl"];
	        this.repositoryOwner = source["repositoryOwner"];
	        this.repositoryName = source["repositoryName"];
	        this.repositoryPath = source["repositoryPath"];
	        this.currentVersionId = source["currentVersionId"];
	        this.note = source["note"];
	        this.displayVersion = source["displayVersion"];
	        this.installedAt = this.convertValues(source["installedAt"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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

