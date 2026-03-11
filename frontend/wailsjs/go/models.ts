export namespace main {
	
	export class AppInfo {
	    name: string;
	    repo: string;
	    photo: string;
	    match: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.repo = source["repo"];
	        this.photo = source["photo"];
	        this.match = source["match"];
	        this.summary = source["summary"];
	    }
	}
	export class PlatformDownload {
	    platform: string;
	    available: boolean;
	    asset_name: string;
	    download_url: string;
	    arch: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformDownload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.available = source["available"];
	        this.asset_name = source["asset_name"];
	        this.download_url = source["download_url"];
	        this.arch = source["arch"];
	    }
	}
	export class AppReleaseDetail {
	    repo: string;
	    match: string;
	    release_tag: string;
	    release_name: string;
	    release_body: string;
	    release_published_at: string;
	    readme: string;
	    readme_source_url: string;
	    readme_branch: string;
	    readme_file_path: string;
	    downloads: Record<string, PlatformDownload>;
	
	    static createFrom(source: any = {}) {
	        return new AppReleaseDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.match = source["match"];
	        this.release_tag = source["release_tag"];
	        this.release_name = source["release_name"];
	        this.release_body = source["release_body"];
	        this.release_published_at = source["release_published_at"];
	        this.readme = source["readme"];
	        this.readme_source_url = source["readme_source_url"];
	        this.readme_branch = source["readme_branch"];
	        this.readme_file_path = source["readme_file_path"];
	        this.downloads = this.convertValues(source["downloads"], PlatformDownload, true);
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
	export class DownloadTaskSnapshot {
	    task_id: string;
	    status: string;
	    progress: number;
	    downloaded_bytes: number;
	    total_bytes: number;
	    file_path: string;
	    error: string;
	    download_url: string;
	    file_name: string;
	    platform: string;
	    started_at: string;
	    updated_at: string;
	    temp_file: string;
	    etag: string;
	    accept_ranges: string;
	    resume_offset: number;
	
	    static createFrom(source: any = {}) {
	        return new DownloadTaskSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.downloaded_bytes = source["downloaded_bytes"];
	        this.total_bytes = source["total_bytes"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	        this.download_url = source["download_url"];
	        this.file_name = source["file_name"];
	        this.platform = source["platform"];
	        this.started_at = source["started_at"];
	        this.updated_at = source["updated_at"];
	        this.temp_file = source["temp_file"];
	        this.etag = source["etag"];
	        this.accept_ranges = source["accept_ranges"];
	        this.resume_offset = source["resume_offset"];
	    }
	}
	
	export class StartDownloadRequest {
	    download_url: string;
	    file_name: string;
	    platform: string;
	    download_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new StartDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.download_url = source["download_url"];
	        this.file_name = source["file_name"];
	        this.platform = source["platform"];
	        this.download_dir = source["download_dir"];
	    }
	}

}

