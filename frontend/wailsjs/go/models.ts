export namespace lottery {
	
	export class Game {
	    type: string;
	    numbers: number[];
	    cost: number;
	    expectedReturn: number;
	    probability: number;
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.numbers = source["numbers"];
	        this.cost = source["cost"];
	        this.expectedReturn = source["expectedReturn"];
	        this.probability = source["probability"];
	    }
	}
	export class Stats {
	    totalDraws: number;
	    analyzedDraws: number;
	    numberFrequency: Record<number, number>;
	    sumDistribution: Record<number, number>;
	    recentTrends: number[];
	    hotNumbers: number[];
	    coldNumbers: number[];
	    patterns: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalDraws = source["totalDraws"];
	        this.analyzedDraws = source["analyzedDraws"];
	        this.numberFrequency = source["numberFrequency"];
	        this.sumDistribution = source["sumDistribution"];
	        this.recentTrends = source["recentTrends"];
	        this.hotNumbers = source["hotNumbers"];
	        this.coldNumbers = source["coldNumbers"];
	        this.patterns = source["patterns"];
	    }
	}
	export class Strategy {
	    games: Game[];
	    totalCost: number;
	    budget: number;
	    expectedReturn: number;
	    reasoning: string;
	    statistics: Stats;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Strategy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.games = this.convertValues(source["games"], Game);
	        this.totalCost = source["totalCost"];
	        this.budget = source["budget"];
	        this.expectedReturn = source["expectedReturn"];
	        this.reasoning = source["reasoning"];
	        this.statistics = this.convertValues(source["statistics"], Stats);
	        this.createdAt = this.convertValues(source["createdAt"], null);
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

export namespace main {
	
	export class ConfigData {
	    claudeApiKey: string;
	    claudeModel: string;
	    timeoutSec: number;
	    maxTokens: number;
	    verbose: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.claudeApiKey = source["claudeApiKey"];
	        this.claudeModel = source["claudeModel"];
	        this.timeoutSec = source["timeoutSec"];
	        this.maxTokens = source["maxTokens"];
	        this.verbose = source["verbose"];
	    }
	}
	export class ConnectionStatus {
	    caixaAPI: boolean;
	    caixaError?: string;
	    claudeAPI: boolean;
	    claudeError?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.caixaAPI = source["caixaAPI"];
	        this.caixaError = source["caixaError"];
	        this.claudeAPI = source["claudeAPI"];
	        this.claudeError = source["claudeError"];
	    }
	}
	export class StrategyResponse {
	    success: boolean;
	    strategy?: lottery.Strategy;
	    confidence: number;
	    error?: string;
	    availableLotteries?: string[];
	    failedLotteries?: string[];
	
	    static createFrom(source: any = {}) {
	        return new StrategyResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.strategy = this.convertValues(source["strategy"], lottery.Strategy);
	        this.confidence = source["confidence"];
	        this.error = source["error"];
	        this.availableLotteries = source["availableLotteries"];
	        this.failedLotteries = source["failedLotteries"];
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
	export class UserPreferences {
	    lotteryTypes: string[];
	    budget: number;
	    strategy: string;
	    avoidPatterns: boolean;
	    favoriteNumbers: number[];
	    excludeNumbers: number[];
	
	    static createFrom(source: any = {}) {
	        return new UserPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lotteryTypes = source["lotteryTypes"];
	        this.budget = source["budget"];
	        this.strategy = source["strategy"];
	        this.avoidPatterns = source["avoidPatterns"];
	        this.favoriteNumbers = source["favoriteNumbers"];
	        this.excludeNumbers = source["excludeNumbers"];
	    }
	}

}

