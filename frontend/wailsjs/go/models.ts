export namespace main {
	
	export class AppStatus {
	    appVersion: string;
	    addonsPath: string;
	    addonsAutoFound: boolean;
	    installedVersion: string;
	    addonInstalled: boolean;
	    autoCheck: boolean;
	    addonName: string;
	    addonRepo: string;
	    elvuiInstalled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appVersion = source["appVersion"];
	        this.addonsPath = source["addonsPath"];
	        this.addonsAutoFound = source["addonsAutoFound"];
	        this.installedVersion = source["installedVersion"];
	        this.addonInstalled = source["addonInstalled"];
	        this.autoCheck = source["autoCheck"];
	        this.addonName = source["addonName"];
	        this.addonRepo = source["addonRepo"];
	        this.elvuiInstalled = source["elvuiInstalled"];
	    }
	}
	export class ElvUIInfo {
	    installed: boolean;
	    installedVersion: string;
	    latestVersion: string;
	    downloadUrl: string;
	    changelog: string;
	    webUrl: string;
	    updateAvailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ElvUIInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.installedVersion = source["installedVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.downloadUrl = source["downloadUrl"];
	        this.changelog = source["changelog"];
	        this.webUrl = source["webUrl"];
	        this.updateAvailable = source["updateAvailable"];
	    }
	}
	export class SelfUpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    updateAvailable: boolean;
	    hasAsset: boolean;
	    assetUrl: string;
	    assetName: string;
	    htmlUrl: string;
	    changelog: string;
	
	    static createFrom(source: any = {}) {
	        return new SelfUpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.hasAsset = source["hasAsset"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.htmlUrl = source["htmlUrl"];
	        this.changelog = source["changelog"];
	    }
	}
	export class UpdateInfo {
	    installedVersion: string;
	    latestVersion: string;
	    releaseName: string;
	    changelog: string;
	    htmlUrl: string;
	    publishedAt: string;
	    assetName: string;
	    assetUrl: string;
	    assetSize: number;
	    updateAvailable: boolean;
	    hasAsset: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installedVersion = source["installedVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseName = source["releaseName"];
	        this.changelog = source["changelog"];
	        this.htmlUrl = source["htmlUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.assetName = source["assetName"];
	        this.assetUrl = source["assetUrl"];
	        this.assetSize = source["assetSize"];
	        this.updateAvailable = source["updateAvailable"];
	        this.hasAsset = source["hasAsset"];
	    }
	}

}

