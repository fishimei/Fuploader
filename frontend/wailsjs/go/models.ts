export namespace database {
	
	export class Account {
	    id: number;
	    platform: string;
	    name: string;
	    username: string;
	    avatar: string;
	    cookiePath: string;
	    status: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.avatar = source["avatar"];
	        this.cookiePath = source["cookiePath"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ScheduleConfig {
	    id: number;
	    videosPerDay: number;
	    dailyTimes: string[];
	    startDays: number;
	    timeZone: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.videosPerDay = source["videosPerDay"];
	        this.dailyTimes = source["dailyTimes"];
	        this.startDays = source["startDays"];
	        this.timeZone = source["timeZone"];
	    }
	}
	export class Video {
	    id: number;
	    filename: string;
	    filePath: string;
	    fileSize: number;
	    duration: number;
	    width: number;
	    height: number;
	    title: string;
	    description: string;
	    tags: string[];
	    thumbnail: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Video(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.filePath = source["filePath"];
	        this.fileSize = source["fileSize"];
	        this.duration = source["duration"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.thumbnail = source["thumbnail"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class UploadTask {
	    id: number;
	    videoId: number;
	    video: Video;
	    accountId: number;
	    account: Account;
	    platform: string;
	    status: string;
	    progress: number;
	    scheduleTime?: string;
	    publishUrl: string;
	    errorMsg: string;
	    retryCount: number;
	    createdAt: string;
	    updatedAt: string;
	    title: string;
	    collection: string;
	    shortTitle: string;
	    isOriginal: boolean;
	    originalType: string;
	    location: string;
	    thumbnail: string;
	    syncToutiao: boolean;
	    syncXigua: boolean;
	    isDraft: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UploadTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.videoId = source["videoId"];
	        this.video = this.convertValues(source["video"], Video);
	        this.accountId = source["accountId"];
	        this.account = this.convertValues(source["account"], Account);
	        this.platform = source["platform"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.scheduleTime = source["scheduleTime"];
	        this.publishUrl = source["publishUrl"];
	        this.errorMsg = source["errorMsg"];
	        this.retryCount = source["retryCount"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.title = source["title"];
	        this.collection = source["collection"];
	        this.shortTitle = source["shortTitle"];
	        this.isOriginal = source["isOriginal"];
	        this.originalType = source["originalType"];
	        this.location = source["location"];
	        this.thumbnail = source["thumbnail"];
	        this.syncToutiao = source["syncToutiao"];
	        this.syncXigua = source["syncXigua"];
	        this.isDraft = source["isDraft"];
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

export namespace types {
	
	export class AppStatus {
	    initialized: boolean;
	    error: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.error = source["error"];
	        this.version = source["version"];
	    }
	}
	export class AppVersion {
	    version: string;
	    buildTime: string;
	    goVersion: string;
	    wailsVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new AppVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.buildTime = source["buildTime"];
	        this.goVersion = source["goVersion"];
	        this.wailsVersion = source["wailsVersion"];
	    }
	}
	export class Collection {
	    label: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	    }
	}
	export class CoverInfo {
	    thumbnailPath: string;
	
	    static createFrom(source: any = {}) {
	        return new CoverInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.thumbnailPath = source["thumbnailPath"];
	    }
	}
	export class LogQuery {
	    keyword: string;
	    limit: number;
	    platform: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new LogQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.limit = source["limit"];
	        this.platform = source["platform"];
	        this.level = source["level"];
	    }
	}
	export class PlatformScreenshotConfig {
	    platform: string;
	    name: string;
	    dir: string;
	    screenshotCount: number;
	
	    static createFrom(source: any = {}) {
	        return new PlatformScreenshotConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.screenshotCount = source["screenshotCount"];
	    }
	}
	export class ProductLinkValidationResult {
	    valid: boolean;
	    title?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProductLinkValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.title = source["title"];
	        this.error = source["error"];
	    }
	}
	export class ScreenshotConfig {
	    enabled: boolean;
	    globalDir: string;
	    platformDirs: Record<string, string>;
	    autoClean: boolean;
	    maxAgeDays: number;
	    maxSizeMB: number;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.globalDir = source["globalDir"];
	        this.platformDirs = source["platformDirs"];
	        this.autoClean = source["autoClean"];
	        this.maxAgeDays = source["maxAgeDays"];
	        this.maxSizeMB = source["maxSizeMB"];
	    }
	}
	export class ScreenshotInfo {
	    id: string;
	    filename: string;
	    platform: string;
	    type: string;
	    size: number;
	    // Go type: time
	    createdAt: any;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.platform = source["platform"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.path = source["path"];
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
	export class ScreenshotListResult {
	    list: ScreenshotInfo[];
	    total: number;
	    page: number;
	    pageSize: number;
	    totalSize: number;
	    platformStats: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.list = this.convertValues(source["list"], ScreenshotInfo);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.totalSize = source["totalSize"];
	        this.platformStats = source["platformStats"];
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
	export class ScreenshotQuery {
	    platform?: string;
	    type?: string;
	    startDate?: string;
	    endDate?: string;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new ScreenshotQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.type = source["type"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class SimpleLog {
	    date: string;
	    time: string;
	    message: string;
	    platform: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new SimpleLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.time = source["time"];
	        this.message = source["message"];
	        this.platform = source["platform"];
	        this.level = source["level"];
	    }
	}

}

