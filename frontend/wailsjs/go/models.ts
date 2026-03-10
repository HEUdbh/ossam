export namespace main {
	
	export class AppInfo {
	    name: string;
	    repo: string;
	    photo: string;
	    match: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.repo = source["repo"];
	        this.photo = source["photo"];
	        this.match = source["match"];
	    }
	}
	export class AppsConfig {
	    market_name: string;
	    last_updated: string;
	    apps: Record<string, Array<AppInfo>>;
	
	    static createFrom(source: any = {}) {
	        return new AppsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.market_name = source["market_name"];
	        this.last_updated = source["last_updated"];
	        this.apps = this.convertValues(source["apps"], Array<AppInfo>, true);
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

