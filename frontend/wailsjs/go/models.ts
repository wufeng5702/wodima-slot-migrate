export namespace android {
	
	export class SlotRow {
	    id: number;
	    slotIndex: number;
	    userAccount: string;
	    jsonString: string;
	    jsonSize: number;
	    jsonPreview: string;
	
	    static createFrom(source: any = {}) {
	        return new SlotRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.slotIndex = source["slotIndex"];
	        this.userAccount = source["userAccount"];
	        this.jsonString = source["jsonString"];
	        this.jsonSize = source["jsonSize"];
	        this.jsonPreview = source["jsonPreview"];
	    }
	}
	export class WifiResult {
	    url: string;
	    localUrl: string;
	    allUrls: string[];
	    token: string;
	    debugInfo: string;
	
	    static createFrom(source: any = {}) {
	        return new WifiResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.localUrl = source["localUrl"];
	        this.allUrls = source["allUrls"];
	        this.token = source["token"];
	        this.debugInfo = source["debugInfo"];
	    }
	}

}

export namespace migrate {
	
	export class MigrateResult {
	    id: number;
	    slotIndex: number;
	    targetFile: string;
	    backupFile: string;
	    success: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new MigrateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.slotIndex = source["slotIndex"];
	        this.targetFile = source["targetFile"];
	        this.backupFile = source["backupFile"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class SlotSelection {
	    id: number;
	    slotIndex: number;
	    jsonString: string;
	
	    static createFrom(source: any = {}) {
	        return new SlotSelection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.slotIndex = source["slotIndex"];
	        this.jsonString = source["jsonString"];
	    }
	}

}

export namespace steam {
	
	export class SteamUser {
	    steamId: string;
	    remotePath: string;
	
	    static createFrom(source: any = {}) {
	        return new SteamUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steamId = source["steamId"];
	        this.remotePath = source["remotePath"];
	    }
	}
	export class Info {
	    steamPath: string;
	    users: SteamUser[];
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steamPath = source["steamPath"];
	        this.users = this.convertValues(source["users"], SteamUser);
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

