export namespace main {
	
	export class CreaturePageResult {
	    creatures: models.Creature[];
	    total: number;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreaturePageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.creatures = this.convertValues(source["creatures"], models.Creature);
	        this.total = source["total"];
	        this.hasMore = source["hasMore"];
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
	export class DataItem {
	    key: string;
	    label: string;
	    source: string;
	    kind: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DataItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.source = source["source"];
	        this.kind = source["kind"];
	        this.count = source["count"];
	    }
	}
	export class DataStatus {
	    icons: number;
	    maps: number;
	    npcImages: number;
	    talentBgs: number;
	    datasets: DataItem[];
	
	    static createFrom(source: any = {}) {
	        return new DataStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.icons = source["icons"];
	        this.maps = source["maps"];
	        this.npcImages = source["npcImages"];
	        this.talentBgs = source["talentBgs"];
	        this.datasets = this.convertValues(source["datasets"], DataItem);
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
	export class FixMissingIconsResult {
	    totalMissing: number;
	    fixed: number;
	    failed: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FixMissingIconsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalMissing = source["totalMissing"];
	        this.fixed = source["fixed"];
	        this.failed = source["failed"];
	        this.message = source["message"];
	    }
	}
	export class FlightConnection {
	    pathId: number;
	    from: number;
	    to: number;
	    waypoints: number[][];
	
	    static createFrom(source: any = {}) {
	        return new FlightConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pathId = source["pathId"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.waypoints = source["waypoints"];
	    }
	}
	export class FlightContinent {
	    mapId: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new FlightContinent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mapId = source["mapId"];
	        this.name = source["name"];
	    }
	}
	export class TransportRoute {
	    id: number;
	    type: string;
	    here: string;
	    dest: string;
	    destContinent: string;
	    sameContinent: boolean;
	    waypoints: number[][];
	
	    static createFrom(source: any = {}) {
	        return new TransportRoute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.here = source["here"];
	        this.dest = source["dest"];
	        this.destContinent = source["destContinent"];
	        this.sameContinent = source["sameContinent"];
	        this.waypoints = source["waypoints"];
	    }
	}
	export class FlightNode {
	    id: number;
	    name: string;
	    alliance: boolean;
	    horde: boolean;
	    zone: string;
	    px: number;
	    py: number;
	
	    static createFrom(source: any = {}) {
	        return new FlightNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.alliance = source["alliance"];
	        this.horde = source["horde"];
	        this.zone = source["zone"];
	        this.px = source["px"];
	        this.py = source["py"];
	    }
	}
	export class FlightData {
	    nodes: FlightNode[];
	    connections: FlightConnection[];
	    transports: TransportRoute[];
	
	    static createFrom(source: any = {}) {
	        return new FlightData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], FlightNode);
	        this.connections = this.convertValues(source["connections"], FlightConnection);
	        this.transports = this.convertValues(source["transports"], TransportRoute);
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
	
	export class ImageResult {
	    data: string;
	    mimeType: string;
	    source: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.mimeType = source["mimeType"];
	        this.source = source["source"];
	        this.error = source["error"];
	    }
	}
	export class ImportReport {
	    success: boolean;
	    title: string;
	    lines: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.title = source["title"];
	        this.lines = source["lines"];
	    }
	}
	export class LegacyLootItem {
	    itemId: number;
	    itemName: string;
	    iconName: string;
	    quality: number;
	    dropChance?: string;
	    slotType?: string;
	    spellId?: number;
	
	    static createFrom(source: any = {}) {
	        return new LegacyLootItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.itemName = source["itemName"];
	        this.iconName = source["iconName"];
	        this.quality = source["quality"];
	        this.dropChance = source["dropChance"];
	        this.slotType = source["slotType"];
	        this.spellId = source["spellId"];
	    }
	}
	export class LegacyBossLoot {
	    bossName: string;
	    items: LegacyLootItem[];
	
	    static createFrom(source: any = {}) {
	        return new LegacyBossLoot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bossName = source["bossName"];
	        this.items = this.convertValues(source["items"], LegacyLootItem);
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
	
	export class TalentRank {
	    spellId: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new TalentRank(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.description = source["description"];
	    }
	}
	export class Talent {
	    id: number;
	    row: number;
	    col: number;
	    name: string;
	    icon: string;
	    maxRank: number;
	    ranks: TalentRank[];
	    reqTalent: number;
	    reqRank: number;
	
	    static createFrom(source: any = {}) {
	        return new Talent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.row = source["row"];
	        this.col = source["col"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.maxRank = source["maxRank"];
	        this.ranks = this.convertValues(source["ranks"], TalentRank);
	        this.reqTalent = source["reqTalent"];
	        this.reqRank = source["reqRank"];
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
	export class TalentTree {
	    id: number;
	    name: string;
	    order: number;
	    background: string;
	    talents: Talent[];
	
	    static createFrom(source: any = {}) {
	        return new TalentTree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.order = source["order"];
	        this.background = source["background"];
	        this.talents = this.convertValues(source["talents"], Talent);
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
	export class TalentClassData {
	    class: string;
	    trees: TalentTree[];
	
	    static createFrom(source: any = {}) {
	        return new TalentClassData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.trees = this.convertValues(source["trees"], TalentTree);
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
	export class TalentClassInfo {
	    class: string;
	    classId: number;
	    name?: string;
	    color?: string;
	
	    static createFrom(source: any = {}) {
	        return new TalentClassInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.classId = source["classId"];
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	
	
	
	export class UpdateInfo {
	    current: string;
	    latest: string;
	    updateAvailable: boolean;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.updateAvailable = source["updateAvailable"];
	        this.url = source["url"];
	    }
	}
	export class WhatsNewEntry {
	    type: string;
	    id: number;
	    name: string;
	    change: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatsNewEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.change = source["change"];
	    }
	}
	export class WhatsNewGroup {
	    type: string;
	    label: string;
	    added: number;
	    changed: number;
	    entries: WhatsNewEntry[];
	
	    static createFrom(source: any = {}) {
	        return new WhatsNewGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	        this.added = source["added"];
	        this.changed = source["changed"];
	        this.entries = this.convertValues(source["entries"], WhatsNewEntry);
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
	export class WhatsNewReport {
	    baseline: string;
	    groups: WhatsNewGroup[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatsNewReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseline = source["baseline"];
	        this.groups = this.convertValues(source["groups"], WhatsNewGroup);
	        this.error = source["error"];
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
	export class WorldConn {
	    from: number;
	    to: number;
	
	    static createFrom(source: any = {}) {
	        return new WorldConn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from = source["from"];
	        this.to = source["to"];
	    }
	}
	export class WorldTransport {
	    type: string;
	    aMap: number;
	    aPx: number;
	    aPy: number;
	    aName: string;
	    bMap: number;
	    bPx: number;
	    bPy: number;
	    bName: string;
	
	    static createFrom(source: any = {}) {
	        return new WorldTransport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.aMap = source["aMap"];
	        this.aPx = source["aPx"];
	        this.aPy = source["aPy"];
	        this.aName = source["aName"];
	        this.bMap = source["bMap"];
	        this.bPx = source["bPx"];
	        this.bPy = source["bPy"];
	        this.bName = source["bName"];
	    }
	}
	export class WorldNode {
	    id: number;
	    name: string;
	    alliance: boolean;
	    horde: boolean;
	    mapId: number;
	    zone: string;
	    px: number;
	    py: number;
	
	    static createFrom(source: any = {}) {
	        return new WorldNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.alliance = source["alliance"];
	        this.horde = source["horde"];
	        this.mapId = source["mapId"];
	        this.zone = source["zone"];
	        this.px = source["px"];
	        this.py = source["py"];
	    }
	}
	export class WorldData {
	    continents: FlightContinent[];
	    nodes: WorldNode[];
	    connections: WorldConn[];
	    transports: WorldTransport[];
	
	    static createFrom(source: any = {}) {
	        return new WorldData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.continents = this.convertValues(source["continents"], FlightContinent);
	        this.nodes = this.convertValues(source["nodes"], WorldNode);
	        this.connections = this.convertValues(source["connections"], WorldConn);
	        this.transports = this.convertValues(source["transports"], WorldTransport);
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
	
	
	export class ZoneNode {
	    id: number;
	    name: string;
	    alliance: boolean;
	    horde: boolean;
	    px: number;
	    py: number;
	    dests: string[];
	
	    static createFrom(source: any = {}) {
	        return new ZoneNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.alliance = source["alliance"];
	        this.horde = source["horde"];
	        this.px = source["px"];
	        this.py = source["py"];
	        this.dests = source["dests"];
	    }
	}
	export class ZoneData {
	    mapKey: string;
	    mapId: number;
	    zone: string;
	    nodes: ZoneNode[];
	
	    static createFrom(source: any = {}) {
	        return new ZoneData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mapKey = source["mapKey"];
	        this.mapId = source["mapId"];
	        this.zone = source["zone"];
	        this.nodes = this.convertValues(source["nodes"], ZoneNode);
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

export namespace models {
	
	export class AtlasTable {
	    key: string;
	    displayName: string;
	
	    static createFrom(source: any = {}) {
	        return new AtlasTable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	    }
	}
	export class BeastFamily {
	    family: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new BeastFamily(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class Category {
	    id: number;
	    key: string;
	    name: string;
	    parentId?: number;
	    type: string;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.type = source["type"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class CraftReagent {
	    entry: number;
	    name: string;
	    quality: number;
	    iconPath: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new CraftReagent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.quality = source["quality"];
	        this.iconPath = source["iconPath"];
	        this.count = source["count"];
	    }
	}
	export class Creature {
	    entry: number;
	    name: string;
	    subname?: string;
	    levelMin: number;
	    levelMax: number;
	    healthMin: number;
	    healthMax: number;
	    manaMin: number;
	    manaMax: number;
	    goldMin: number;
	    goldMax: number;
	    type: number;
	    typeName: string;
	    rank: number;
	    rankName: string;
	    faction: number;
	    npcFlags: number;
	    minDmg: number;
	    maxDmg: number;
	    armor: number;
	    holyRes: number;
	    fireRes: number;
	    natureRes: number;
	    frostRes: number;
	    shadowRes: number;
	    arcaneRes: number;
	    displayId1: number;
	
	    static createFrom(source: any = {}) {
	        return new Creature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.healthMin = source["healthMin"];
	        this.healthMax = source["healthMax"];
	        this.manaMin = source["manaMin"];
	        this.manaMax = source["manaMax"];
	        this.goldMin = source["goldMin"];
	        this.goldMax = source["goldMax"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.rank = source["rank"];
	        this.rankName = source["rankName"];
	        this.faction = source["faction"];
	        this.npcFlags = source["npcFlags"];
	        this.minDmg = source["minDmg"];
	        this.maxDmg = source["maxDmg"];
	        this.armor = source["armor"];
	        this.holyRes = source["holyRes"];
	        this.fireRes = source["fireRes"];
	        this.natureRes = source["natureRes"];
	        this.frostRes = source["frostRes"];
	        this.shadowRes = source["shadowRes"];
	        this.arcaneRes = source["arcaneRes"];
	        this.displayId1 = source["displayId1"];
	    }
	}
	export class CreatureAbility {
	    id: number;
	    name: string;
	    icon: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreatureAbility(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	    }
	}
	export class CreatureSpawn {
	    mapId: number;
	    x: number;
	    y: number;
	    z: number;
	
	    static createFrom(source: any = {}) {
	        return new CreatureSpawn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mapId = source["mapId"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.z = source["z"];
	    }
	}
	export class QuestRelation {
	    entry: number;
	    name: string;
	    title?: string;
	    level?: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestRelation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.title = source["title"];
	        this.level = source["level"];
	        this.type = source["type"];
	    }
	}
	export class LootItem {
	    itemId: number;
	    name: string;
	    iconPath: string;
	    quality: number;
	    chance: number;
	    minCount: number;
	    maxCount: number;
	
	    static createFrom(source: any = {}) {
	        return new LootItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.quality = source["quality"];
	        this.chance = source["chance"];
	        this.minCount = source["minCount"];
	        this.maxCount = source["maxCount"];
	    }
	}
	export class CreatureDetail {
	    entry: number;
	    name: string;
	    subname?: string;
	    levelMin: number;
	    levelMax: number;
	    healthMin: number;
	    healthMax: number;
	    manaMin: number;
	    manaMax: number;
	    goldMin: number;
	    goldMax: number;
	    type: number;
	    typeName: string;
	    rank: number;
	    rankName: string;
	    faction: number;
	    npcFlags: number;
	    minDmg: number;
	    maxDmg: number;
	    armor: number;
	    holyRes: number;
	    fireRes: number;
	    natureRes: number;
	    frostRes: number;
	    shadowRes: number;
	    arcaneRes: number;
	    displayId1: number;
	    loot: LootItem[];
	    startsQuests: QuestRelation[];
	    endsQuests: QuestRelation[];
	    abilities: CreatureAbility[];
	    spawns: CreatureSpawn[];
	
	    static createFrom(source: any = {}) {
	        return new CreatureDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.healthMin = source["healthMin"];
	        this.healthMax = source["healthMax"];
	        this.manaMin = source["manaMin"];
	        this.manaMax = source["manaMax"];
	        this.goldMin = source["goldMin"];
	        this.goldMax = source["goldMax"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.rank = source["rank"];
	        this.rankName = source["rankName"];
	        this.faction = source["faction"];
	        this.npcFlags = source["npcFlags"];
	        this.minDmg = source["minDmg"];
	        this.maxDmg = source["maxDmg"];
	        this.armor = source["armor"];
	        this.holyRes = source["holyRes"];
	        this.fireRes = source["fireRes"];
	        this.natureRes = source["natureRes"];
	        this.frostRes = source["frostRes"];
	        this.shadowRes = source["shadowRes"];
	        this.arcaneRes = source["arcaneRes"];
	        this.displayId1 = source["displayId1"];
	        this.loot = this.convertValues(source["loot"], LootItem);
	        this.startsQuests = this.convertValues(source["startsQuests"], QuestRelation);
	        this.endsQuests = this.convertValues(source["endsQuests"], QuestRelation);
	        this.abilities = this.convertValues(source["abilities"], CreatureAbility);
	        this.spawns = this.convertValues(source["spawns"], CreatureSpawn);
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
	export class CreatureDrop {
	    entry: number;
	    name: string;
	    levelMin: number;
	    levelMax: number;
	    chance: number;
	
	    static createFrom(source: any = {}) {
	        return new CreatureDrop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.chance = source["chance"];
	    }
	}
	
	export class CreatureType {
	    type: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new CreatureType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class Faction {
	    id: number;
	    name: string;
	    description: string;
	    side: number;
	    categoryId: number;
	
	    static createFrom(source: any = {}) {
	        return new Faction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.side = source["side"];
	        this.categoryId = source["categoryId"];
	    }
	}
	export class FactionNpc {
	    entry: number;
	    name: string;
	    subname: string;
	    levelMin: number;
	    levelMax: number;
	
	    static createFrom(source: any = {}) {
	        return new FactionNpc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	    }
	}
	export class FactionDetail {
	    id: number;
	    name: string;
	    description: string;
	    side: number;
	    sideName: string;
	    categoryId: number;
	    creatures?: Creature[];
	    quests?: QuestRelation[];
	    questGivers?: FactionNpc[];
	    members?: FactionNpc[];
	
	    static createFrom(source: any = {}) {
	        return new FactionDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.side = source["side"];
	        this.sideName = source["sideName"];
	        this.categoryId = source["categoryId"];
	        this.creatures = this.convertValues(source["creatures"], Creature);
	        this.quests = this.convertValues(source["quests"], QuestRelation);
	        this.questGivers = this.convertValues(source["questGivers"], FactionNpc);
	        this.members = this.convertValues(source["members"], FactionNpc);
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
	
	export class FavoriteCategory {
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class FavoriteItem {
	    id: number;
	    itemEntry: number;
	    category: string;
	    addedAt: string;
	    itemName?: string;
	    itemQuality?: number;
	    iconPath?: string;
	    itemLevel?: number;
	    status: number;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.itemEntry = source["itemEntry"];
	        this.category = source["category"];
	        this.addedAt = source["addedAt"];
	        this.itemName = source["itemName"];
	        this.itemQuality = source["itemQuality"];
	        this.iconPath = source["iconPath"];
	        this.itemLevel = source["itemLevel"];
	        this.status = source["status"];
	    }
	}
	export class FavoriteResult {
	    success: boolean;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class GameObject {
	    entry: number;
	    name: string;
	    type: number;
	    typeName: string;
	    displayId: number;
	    size: number;
	    data?: number[];
	
	    static createFrom(source: any = {}) {
	        return new GameObject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.displayId = source["displayId"];
	        this.size = source["size"];
	        this.data = source["data"];
	    }
	}
	export class ObjectSpawn {
	    mapId: number;
	    zoneName: string;
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new ObjectSpawn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mapId = source["mapId"];
	        this.zoneName = source["zoneName"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class GameObjectDetail {
	    entry: number;
	    name: string;
	    type: number;
	    typeName: string;
	    displayId: number;
	    faction: number;
	    flags: number;
	    size: number;
	    data0: number;
	    data1: number;
	    reqSkill?: string;
	    reqSkillLevel?: number;
	    startsQuests?: QuestRelation[];
	    endsQuests?: QuestRelation[];
	    contains?: LootItem[];
	    spawns?: ObjectSpawn[];
	
	    static createFrom(source: any = {}) {
	        return new GameObjectDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.displayId = source["displayId"];
	        this.faction = source["faction"];
	        this.flags = source["flags"];
	        this.size = source["size"];
	        this.data0 = source["data0"];
	        this.data1 = source["data1"];
	        this.reqSkill = source["reqSkill"];
	        this.reqSkillLevel = source["reqSkillLevel"];
	        this.startsQuests = this.convertValues(source["startsQuests"], QuestRelation);
	        this.endsQuests = this.convertValues(source["endsQuests"], QuestRelation);
	        this.contains = this.convertValues(source["contains"], LootItem);
	        this.spawns = this.convertValues(source["spawns"], ObjectSpawn);
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
	export class InventorySlot {
	    class: number;
	    subClass: number;
	    inventoryType: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new InventorySlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.subClass = source["subClass"];
	        this.inventoryType = source["inventoryType"];
	        this.name = source["name"];
	    }
	}
	export class Item {
	    entry: number;
	    name: string;
	    description?: string;
	    quality: number;
	    itemLevel: number;
	    requiredLevel: number;
	    class: number;
	    subClass: number;
	    inventoryType: number;
	    iconPath: string;
	    sellPrice?: number;
	    buyPrice?: number;
	    allowableClass?: number;
	    allowableRace?: number;
	    bonding?: number;
	    maxDurability?: number;
	    maxCount?: number;
	    armor?: number;
	    statType1?: number;
	    statValue1?: number;
	    statType2?: number;
	    statValue2?: number;
	    statType3?: number;
	    statValue3?: number;
	    statType4?: number;
	    statValue4?: number;
	    statType5?: number;
	    statValue5?: number;
	    statType6?: number;
	    statValue6?: number;
	    statType7?: number;
	    statValue7?: number;
	    statType8?: number;
	    statValue8?: number;
	    statType9?: number;
	    statValue9?: number;
	    statType10?: number;
	    statValue10?: number;
	    delay?: number;
	    dmgMin1?: number;
	    dmgMax1?: number;
	    dmgType1?: number;
	    dmgMin2?: number;
	    dmgMax2?: number;
	    dmgType2?: number;
	    holyRes?: number;
	    fireRes?: number;
	    natureRes?: number;
	    frostRes?: number;
	    shadowRes?: number;
	    arcaneRes?: number;
	    spellId1?: number;
	    spellTrigger1?: number;
	    spellId2?: number;
	    spellTrigger2?: number;
	    spellId3?: number;
	    spellTrigger3?: number;
	    setId?: number;
	    dropRate?: string;
	    containerSlots?: number;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.quality = source["quality"];
	        this.itemLevel = source["itemLevel"];
	        this.requiredLevel = source["requiredLevel"];
	        this.class = source["class"];
	        this.subClass = source["subClass"];
	        this.inventoryType = source["inventoryType"];
	        this.iconPath = source["iconPath"];
	        this.sellPrice = source["sellPrice"];
	        this.buyPrice = source["buyPrice"];
	        this.allowableClass = source["allowableClass"];
	        this.allowableRace = source["allowableRace"];
	        this.bonding = source["bonding"];
	        this.maxDurability = source["maxDurability"];
	        this.maxCount = source["maxCount"];
	        this.armor = source["armor"];
	        this.statType1 = source["statType1"];
	        this.statValue1 = source["statValue1"];
	        this.statType2 = source["statType2"];
	        this.statValue2 = source["statValue2"];
	        this.statType3 = source["statType3"];
	        this.statValue3 = source["statValue3"];
	        this.statType4 = source["statType4"];
	        this.statValue4 = source["statValue4"];
	        this.statType5 = source["statType5"];
	        this.statValue5 = source["statValue5"];
	        this.statType6 = source["statType6"];
	        this.statValue6 = source["statValue6"];
	        this.statType7 = source["statType7"];
	        this.statValue7 = source["statValue7"];
	        this.statType8 = source["statType8"];
	        this.statValue8 = source["statValue8"];
	        this.statType9 = source["statType9"];
	        this.statValue9 = source["statValue9"];
	        this.statType10 = source["statType10"];
	        this.statValue10 = source["statValue10"];
	        this.delay = source["delay"];
	        this.dmgMin1 = source["dmgMin1"];
	        this.dmgMax1 = source["dmgMax1"];
	        this.dmgType1 = source["dmgType1"];
	        this.dmgMin2 = source["dmgMin2"];
	        this.dmgMax2 = source["dmgMax2"];
	        this.dmgType2 = source["dmgType2"];
	        this.holyRes = source["holyRes"];
	        this.fireRes = source["fireRes"];
	        this.natureRes = source["natureRes"];
	        this.frostRes = source["frostRes"];
	        this.shadowRes = source["shadowRes"];
	        this.arcaneRes = source["arcaneRes"];
	        this.spellId1 = source["spellId1"];
	        this.spellTrigger1 = source["spellTrigger1"];
	        this.spellId2 = source["spellId2"];
	        this.spellTrigger2 = source["spellTrigger2"];
	        this.spellId3 = source["spellId3"];
	        this.spellTrigger3 = source["spellTrigger3"];
	        this.setId = source["setId"];
	        this.dropRate = source["dropRate"];
	        this.containerSlots = source["containerSlots"];
	    }
	}
	export class ItemSubClass {
	    class: number;
	    subClass: number;
	    name: string;
	    inventorySlots?: InventorySlot[];
	
	    static createFrom(source: any = {}) {
	        return new ItemSubClass(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.subClass = source["subClass"];
	        this.name = source["name"];
	        this.inventorySlots = this.convertValues(source["inventorySlots"], InventorySlot);
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
	export class ItemClass {
	    class: number;
	    name: string;
	    subClasses?: ItemSubClass[];
	
	    static createFrom(source: any = {}) {
	        return new ItemClass(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.class = source["class"];
	        this.name = source["name"];
	        this.subClasses = this.convertValues(source["subClasses"], ItemSubClass);
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
	export class ItemContainer {
	    entry: number;
	    name: string;
	    kind: string;
	    quality: number;
	    iconPath: string;
	    chance: number;
	    skill?: string;
	    skillReq?: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemContainer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.quality = source["quality"];
	        this.iconPath = source["iconPath"];
	        this.chance = source["chance"];
	        this.skill = source["skill"];
	        this.skillReq = source["skillReq"];
	    }
	}
	export class ItemCraftSource {
	    spellId: number;
	    spellName: string;
	    spellIcon: string;
	    skillName?: string;
	    reqSkill?: number;
	    producedCount?: number;
	    reagents?: CraftReagent[];
	
	    static createFrom(source: any = {}) {
	        return new ItemCraftSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.spellName = source["spellName"];
	        this.spellIcon = source["spellIcon"];
	        this.skillName = source["skillName"];
	        this.reqSkill = source["reqSkill"];
	        this.producedCount = source["producedCount"];
	        this.reagents = this.convertValues(source["reagents"], CraftReagent);
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
	export class ItemReagentUse {
	    spellId: number;
	    spellName: string;
	    spellIcon: string;
	    skillName?: string;
	    reqSkill?: number;
	    reagentCount?: number;
	    producedItem?: number;
	    producedName?: string;
	    producedIcon?: string;
	    producedQuality?: number;
	    producedCount?: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemReagentUse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.spellName = source["spellName"];
	        this.spellIcon = source["spellIcon"];
	        this.skillName = source["skillName"];
	        this.reqSkill = source["reqSkill"];
	        this.reagentCount = source["reagentCount"];
	        this.producedItem = source["producedItem"];
	        this.producedName = source["producedName"];
	        this.producedIcon = source["producedIcon"];
	        this.producedQuality = source["producedQuality"];
	        this.producedCount = source["producedCount"];
	    }
	}
	export class ItemVendor {
	    entry: number;
	    name: string;
	    levelMin: number;
	    levelMax: number;
	    cost: number;
	    stock: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemVendor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.cost = source["cost"];
	        this.stock = source["stock"];
	    }
	}
	export class ItemDrop {
	    entry: number;
	    name: string;
	    quality: number;
	    chance: number;
	    minCount: number;
	    maxCount: number;
	    iconPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ItemDrop(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.quality = source["quality"];
	        this.chance = source["chance"];
	        this.minCount = source["minCount"];
	        this.maxCount = source["maxCount"];
	        this.iconPath = source["iconPath"];
	    }
	}
	export class QuestReward {
	    entry: number;
	    title: string;
	    level: number;
	    isChoice: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QuestReward(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.level = source["level"];
	        this.isChoice = source["isChoice"];
	    }
	}
	export class ItemDetail {
	    entry: number;
	    name: string;
	    description?: string;
	    quality: number;
	    itemLevel: number;
	    requiredLevel: number;
	    class: number;
	    subClass: number;
	    inventoryType: number;
	    iconPath: string;
	    sellPrice?: number;
	    buyPrice?: number;
	    allowableClass?: number;
	    allowableRace?: number;
	    bonding?: number;
	    maxDurability?: number;
	    maxCount?: number;
	    armor?: number;
	    statType1?: number;
	    statValue1?: number;
	    statType2?: number;
	    statValue2?: number;
	    statType3?: number;
	    statValue3?: number;
	    statType4?: number;
	    statValue4?: number;
	    statType5?: number;
	    statValue5?: number;
	    statType6?: number;
	    statValue6?: number;
	    statType7?: number;
	    statValue7?: number;
	    statType8?: number;
	    statValue8?: number;
	    statType9?: number;
	    statValue9?: number;
	    statType10?: number;
	    statValue10?: number;
	    delay?: number;
	    dmgMin1?: number;
	    dmgMax1?: number;
	    dmgType1?: number;
	    dmgMin2?: number;
	    dmgMax2?: number;
	    dmgType2?: number;
	    holyRes?: number;
	    fireRes?: number;
	    natureRes?: number;
	    frostRes?: number;
	    shadowRes?: number;
	    arcaneRes?: number;
	    spellId1?: number;
	    spellTrigger1?: number;
	    spellId2?: number;
	    spellTrigger2?: number;
	    spellId3?: number;
	    spellTrigger3?: number;
	    setId?: number;
	    dropRate?: string;
	    containerSlots?: number;
	    displayId: number;
	    flags: number;
	    buyCount: number;
	    stackable: number;
	    material: number;
	    droppedBy: CreatureDrop[];
	    rewardFrom: QuestReward[];
	    contains: ItemDrop[];
	    soldBy: ItemVendor[];
	    createdBy: ItemCraftSource[];
	    reagentFor: ItemReagentUse[];
	    containedIn: ItemContainer[];
	    containedInItem: ItemContainer[];
	    gatheredFrom: ItemContainer[];
	    objectiveOf: QuestReward[];
	    startsQuest?: QuestReward;
	
	    static createFrom(source: any = {}) {
	        return new ItemDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.quality = source["quality"];
	        this.itemLevel = source["itemLevel"];
	        this.requiredLevel = source["requiredLevel"];
	        this.class = source["class"];
	        this.subClass = source["subClass"];
	        this.inventoryType = source["inventoryType"];
	        this.iconPath = source["iconPath"];
	        this.sellPrice = source["sellPrice"];
	        this.buyPrice = source["buyPrice"];
	        this.allowableClass = source["allowableClass"];
	        this.allowableRace = source["allowableRace"];
	        this.bonding = source["bonding"];
	        this.maxDurability = source["maxDurability"];
	        this.maxCount = source["maxCount"];
	        this.armor = source["armor"];
	        this.statType1 = source["statType1"];
	        this.statValue1 = source["statValue1"];
	        this.statType2 = source["statType2"];
	        this.statValue2 = source["statValue2"];
	        this.statType3 = source["statType3"];
	        this.statValue3 = source["statValue3"];
	        this.statType4 = source["statType4"];
	        this.statValue4 = source["statValue4"];
	        this.statType5 = source["statType5"];
	        this.statValue5 = source["statValue5"];
	        this.statType6 = source["statType6"];
	        this.statValue6 = source["statValue6"];
	        this.statType7 = source["statType7"];
	        this.statValue7 = source["statValue7"];
	        this.statType8 = source["statType8"];
	        this.statValue8 = source["statValue8"];
	        this.statType9 = source["statType9"];
	        this.statValue9 = source["statValue9"];
	        this.statType10 = source["statType10"];
	        this.statValue10 = source["statValue10"];
	        this.delay = source["delay"];
	        this.dmgMin1 = source["dmgMin1"];
	        this.dmgMax1 = source["dmgMax1"];
	        this.dmgType1 = source["dmgType1"];
	        this.dmgMin2 = source["dmgMin2"];
	        this.dmgMax2 = source["dmgMax2"];
	        this.dmgType2 = source["dmgType2"];
	        this.holyRes = source["holyRes"];
	        this.fireRes = source["fireRes"];
	        this.natureRes = source["natureRes"];
	        this.frostRes = source["frostRes"];
	        this.shadowRes = source["shadowRes"];
	        this.arcaneRes = source["arcaneRes"];
	        this.spellId1 = source["spellId1"];
	        this.spellTrigger1 = source["spellTrigger1"];
	        this.spellId2 = source["spellId2"];
	        this.spellTrigger2 = source["spellTrigger2"];
	        this.spellId3 = source["spellId3"];
	        this.spellTrigger3 = source["spellTrigger3"];
	        this.setId = source["setId"];
	        this.dropRate = source["dropRate"];
	        this.containerSlots = source["containerSlots"];
	        this.displayId = source["displayId"];
	        this.flags = source["flags"];
	        this.buyCount = source["buyCount"];
	        this.stackable = source["stackable"];
	        this.material = source["material"];
	        this.droppedBy = this.convertValues(source["droppedBy"], CreatureDrop);
	        this.rewardFrom = this.convertValues(source["rewardFrom"], QuestReward);
	        this.contains = this.convertValues(source["contains"], ItemDrop);
	        this.soldBy = this.convertValues(source["soldBy"], ItemVendor);
	        this.createdBy = this.convertValues(source["createdBy"], ItemCraftSource);
	        this.reagentFor = this.convertValues(source["reagentFor"], ItemReagentUse);
	        this.containedIn = this.convertValues(source["containedIn"], ItemContainer);
	        this.containedInItem = this.convertValues(source["containedInItem"], ItemContainer);
	        this.gatheredFrom = this.convertValues(source["gatheredFrom"], ItemContainer);
	        this.objectiveOf = this.convertValues(source["objectiveOf"], QuestReward);
	        this.startsQuest = this.convertValues(source["startsQuest"], QuestReward);
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
	
	
	export class ItemSetBrowse {
	    itemsetId: number;
	    name: string;
	    itemIds: number[];
	    itemCount: number;
	    skillId: number;
	    skillLevel: number;
	    classMask: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemSetBrowse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemsetId = source["itemsetId"];
	        this.name = source["name"];
	        this.itemIds = source["itemIds"];
	        this.itemCount = source["itemCount"];
	        this.skillId = source["skillId"];
	        this.skillLevel = source["skillLevel"];
	        this.classMask = source["classMask"];
	    }
	}
	export class SetBonus {
	    threshold: number;
	    spellId: number;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new SetBonus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threshold = source["threshold"];
	        this.spellId = source["spellId"];
	        this.description = source["description"];
	    }
	}
	export class ItemSetDetail {
	    itemsetId: number;
	    name: string;
	    items: Item[];
	    bonuses: SetBonus[];
	
	    static createFrom(source: any = {}) {
	        return new ItemSetDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemsetId = source["itemsetId"];
	        this.name = source["name"];
	        this.items = this.convertValues(source["items"], Item);
	        this.bonuses = this.convertValues(source["bonuses"], SetBonus);
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
	export class ItemSetInfo {
	    name: string;
	    items: string[];
	    bonuses: string[];
	
	    static createFrom(source: any = {}) {
	        return new ItemSetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.items = source["items"];
	        this.bonuses = source["bonuses"];
	    }
	}
	
	
	
	
	export class ObjectType {
	    id: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ObjectType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class Quest {
	    entry: number;
	    title: string;
	    questLevel: number;
	    minLevel: number;
	    type: number;
	    zoneOrSort: number;
	    categoryName: string;
	    requiredRaces: number;
	    requiredClasses: number;
	    srcItemId: number;
	    rewardXp: number;
	    rewardMoney: number;
	    prevQuestId: number;
	    nextQuestId: number;
	    exclusiveGroup: number;
	    nextQuestInChain: number;
	
	    static createFrom(source: any = {}) {
	        return new Quest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.questLevel = source["questLevel"];
	        this.minLevel = source["minLevel"];
	        this.type = source["type"];
	        this.zoneOrSort = source["zoneOrSort"];
	        this.categoryName = source["categoryName"];
	        this.requiredRaces = source["requiredRaces"];
	        this.requiredClasses = source["requiredClasses"];
	        this.srcItemId = source["srcItemId"];
	        this.rewardXp = source["rewardXp"];
	        this.rewardMoney = source["rewardMoney"];
	        this.prevQuestId = source["prevQuestId"];
	        this.nextQuestId = source["nextQuestId"];
	        this.exclusiveGroup = source["exclusiveGroup"];
	        this.nextQuestInChain = source["nextQuestInChain"];
	    }
	}
	export class QuestCategory {
	    id: number;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new QuestCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class QuestCategoryEnhanced {
	    id: number;
	    groupId: number;
	    name: string;
	    questCount: number;
	
	    static createFrom(source: any = {}) {
	        return new QuestCategoryEnhanced(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.groupId = source["groupId"];
	        this.name = source["name"];
	        this.questCount = source["questCount"];
	    }
	}
	export class QuestCategoryGroup {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestCategoryGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class QuestClass {
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestClass(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class QuestSeriesItem {
	    entry: number;
	    title: string;
	    depth: number;
	
	    static createFrom(source: any = {}) {
	        return new QuestSeriesItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.depth = source["depth"];
	    }
	}
	export class QuestReputation {
	    factionId: number;
	    name: string;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new QuestReputation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.factionId = source["factionId"];
	        this.name = source["name"];
	        this.value = source["value"];
	    }
	}
	export class QuestObjective {
	    entry: number;
	    name: string;
	    count: number;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestObjective(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.count = source["count"];
	        this.kind = source["kind"];
	    }
	}
	export class QuestItem {
	    entry: number;
	    name: string;
	    iconPath: string;
	    count: number;
	    quality: number;
	
	    static createFrom(source: any = {}) {
	        return new QuestItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.iconPath = source["iconPath"];
	        this.count = source["count"];
	        this.quality = source["quality"];
	    }
	}
	export class QuestRewardSpell {
	    spellId: number;
	    name: string;
	    iconName: string;
	
	    static createFrom(source: any = {}) {
	        return new QuestRewardSpell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.name = source["name"];
	        this.iconName = source["iconName"];
	    }
	}
	export class QuestDetail {
	    entry: number;
	    title: string;
	    details: string;
	    objectives: string;
	    offerRewardText?: string;
	    endText?: string;
	    questLevel: number;
	    minLevel: number;
	    type: number;
	    zoneOrSort: number;
	    categoryName: string;
	    requiredRaces?: number;
	    side: string;
	    raceNames: string;
	    requiredClasses?: number;
	    classes: QuestClass[];
	    rewardXp: number;
	    rewardMoney: number;
	    rewardSpell?: number;
	    rewardSpellInfo?: QuestRewardSpell;
	    rewardItems: QuestItem[];
	    choiceItems: QuestItem[];
	    requiredItems: QuestItem[];
	    requiredObjectives: QuestObjective[];
	    reputation: QuestReputation[];
	    starters: QuestRelation[];
	    enders: QuestRelation[];
	    series: QuestSeriesItem[];
	    prevQuests: QuestSeriesItem[];
	    exclusiveQuests: QuestSeriesItem[];
	
	    static createFrom(source: any = {}) {
	        return new QuestDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.details = source["details"];
	        this.objectives = source["objectives"];
	        this.offerRewardText = source["offerRewardText"];
	        this.endText = source["endText"];
	        this.questLevel = source["questLevel"];
	        this.minLevel = source["minLevel"];
	        this.type = source["type"];
	        this.zoneOrSort = source["zoneOrSort"];
	        this.categoryName = source["categoryName"];
	        this.requiredRaces = source["requiredRaces"];
	        this.side = source["side"];
	        this.raceNames = source["raceNames"];
	        this.requiredClasses = source["requiredClasses"];
	        this.classes = this.convertValues(source["classes"], QuestClass);
	        this.rewardXp = source["rewardXp"];
	        this.rewardMoney = source["rewardMoney"];
	        this.rewardSpell = source["rewardSpell"];
	        this.rewardSpellInfo = this.convertValues(source["rewardSpellInfo"], QuestRewardSpell);
	        this.rewardItems = this.convertValues(source["rewardItems"], QuestItem);
	        this.choiceItems = this.convertValues(source["choiceItems"], QuestItem);
	        this.requiredItems = this.convertValues(source["requiredItems"], QuestItem);
	        this.requiredObjectives = this.convertValues(source["requiredObjectives"], QuestObjective);
	        this.reputation = this.convertValues(source["reputation"], QuestReputation);
	        this.starters = this.convertValues(source["starters"], QuestRelation);
	        this.enders = this.convertValues(source["enders"], QuestRelation);
	        this.series = this.convertValues(source["series"], QuestSeriesItem);
	        this.prevQuests = this.convertValues(source["prevQuests"], QuestSeriesItem);
	        this.exclusiveQuests = this.convertValues(source["exclusiveQuests"], QuestSeriesItem);
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
	
	
	
	
	
	
	
	export class RacialSpell {
	    id: number;
	    name: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new RacialSpell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	    }
	}
	export class RaceClass {
	    id: number;
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new RaceClass(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class Race {
	    id: number;
	    name: string;
	    fileString: string;
	    prefix: string;
	    faction: string;
	    info: string;
	    abilities: string[];
	    classes: RaceClass[];
	    racials: RacialSpell[];
	
	    static createFrom(source: any = {}) {
	        return new Race(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.fileString = source["fileString"];
	        this.prefix = source["prefix"];
	        this.faction = source["faction"];
	        this.info = source["info"];
	        this.abilities = source["abilities"];
	        this.classes = this.convertValues(source["classes"], RaceClass);
	        this.racials = this.convertValues(source["racials"], RacialSpell);
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
	
	
	export class SearchFilter {
	    query: string;
	    quality?: number[];
	    class?: number[];
	    subClass?: number[];
	    inventoryType?: number[];
	    minLevel?: number;
	    maxLevel?: number;
	    minReqLevel?: number;
	    maxReqLevel?: number;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.quality = source["quality"];
	        this.class = source["class"];
	        this.subClass = source["subClass"];
	        this.inventoryType = source["inventoryType"];
	        this.minLevel = source["minLevel"];
	        this.maxLevel = source["maxLevel"];
	        this.minReqLevel = source["minReqLevel"];
	        this.maxReqLevel = source["maxReqLevel"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class Spell {
	    entry: number;
	    name: string;
	    subname: string;
	    description: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new Spell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	    }
	}
	export class SearchResult {
	    items: Item[];
	    creatures?: Creature[];
	    quests?: Quest[];
	    spells?: Spell[];
	    objects?: GameObject[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Item);
	        this.creatures = this.convertValues(source["creatures"], Creature);
	        this.quests = this.convertValues(source["quests"], Quest);
	        this.spells = this.convertValues(source["spells"], Spell);
	        this.objects = this.convertValues(source["objects"], GameObject);
	        this.totalCount = source["totalCount"];
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
	
	
	export class SpellClass {
	    id: number;
	    name: string;
	    skillCount: number;
	    color?: string;
	
	    static createFrom(source: any = {}) {
	        return new SpellClass(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.skillCount = source["skillCount"];
	        this.color = source["color"];
	    }
	}
	export class SpellRewardQuest {
	    entry: number;
	    title: string;
	    level: number;
	    side: string;
	
	    static createFrom(source: any = {}) {
	        return new SpellRewardQuest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.level = source["level"];
	        this.side = source["side"];
	    }
	}
	export class SpellTrainerNpc {
	    entry: number;
	    name: string;
	    levelMin: number;
	    levelMax: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellTrainerNpc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	    }
	}
	export class SpellUsedByItem {
	    entry: number;
	    name: string;
	    quality: number;
	    iconPath: string;
	    triggerType: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellUsedByItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.quality = source["quality"];
	        this.iconPath = source["iconPath"];
	        this.triggerType = source["triggerType"];
	    }
	}
	export class SpellEffectInfo {
	    index: number;
	    effect: string;
	    auraName?: string;
	    value?: string;
	    radius?: string;
	    mechanic?: string;
	    triggerSpell?: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellEffectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.effect = source["effect"];
	        this.auraName = source["auraName"];
	        this.value = source["value"];
	        this.radius = source["radius"];
	        this.mechanic = source["mechanic"];
	        this.triggerSpell = source["triggerSpell"];
	    }
	}
	export class SpellDetail {
	    entry: number;
	    school: number;
	    category: number;
	    castUI: number;
	    dispel: number;
	    mechanic: number;
	    attributes: number;
	    attributesEx: number;
	    attributesEx2: number;
	    attributesEx3: number;
	    attributesEx4: number;
	    stances: number;
	    stancesNot: number;
	    targets: number;
	    targetCreatureType: number;
	    requiresSpellFocus: number;
	    casterAuraState: number;
	    targetAuraState: number;
	    castingTimeIndex: number;
	    recoveryTime: number;
	    categoryRecoveryTime: number;
	    interruptFlags: number;
	    auraInterruptFlags: number;
	    channelInterruptFlags: number;
	    procFlags: number;
	    procChance: number;
	    procCharges: number;
	    maxLevel: number;
	    baseLevel: number;
	    spellLevel: number;
	    durationIndex: number;
	    powerType: number;
	    manaCost: number;
	    manCostPerLevel: number;
	    manaPerSecond: number;
	    manaPerSecondPerLevel: number;
	    rangeIndex: number;
	    speed: number;
	    modelNextSpell: number;
	    stackAmount: number;
	    totem1: number;
	    totem2: number;
	    reagent1: number;
	    reagent2: number;
	    reagent3: number;
	    reagent4: number;
	    reagent5: number;
	    reagent6: number;
	    reagent7: number;
	    reagent8: number;
	    reagentCount1: number;
	    reagentCount2: number;
	    reagentCount3: number;
	    reagentCount4: number;
	    reagentCount5: number;
	    reagentCount6: number;
	    reagentCount7: number;
	    reagentCount8: number;
	    equippedItemClass: number;
	    equippedItemSubClassMask: number;
	    equippedItemInventoryTypeMask: number;
	    effect1: number;
	    effect2: number;
	    effect3: number;
	    effectDieSides1: number;
	    effectDieSides2: number;
	    effectDieSides3: number;
	    effectBaseDice1: number;
	    effectBaseDice2: number;
	    effectBaseDice3: number;
	    effectDicePerLevel1: number;
	    effectDicePerLevel2: number;
	    effectDicePerLevel3: number;
	    effectRealPointsPerLevel1: number;
	    effectRealPointsPerLevel2: number;
	    effectRealPointsPerLevel3: number;
	    effectBasePoints1: number;
	    effectBasePoints2: number;
	    effectBasePoints3: number;
	    effectBonusCoefficient1: number;
	    effectBonusCoefficient2: number;
	    effectBonusCoefficient3: number;
	    effectMechanic1: number;
	    effectMechanic2: number;
	    effectMechanic3: number;
	    effectImplicitTargetA1: number;
	    effectImplicitTargetA2: number;
	    effectImplicitTargetA3: number;
	    effectImplicitTargetB1: number;
	    effectImplicitTargetB2: number;
	    effectImplicitTargetB3: number;
	    effectRadiusIndex1: number;
	    effectRadiusIndex2: number;
	    effectRadiusIndex3: number;
	    effectApplyAuraName1: number;
	    effectApplyAuraName2: number;
	    effectApplyAuraName3: number;
	    effectAmplitude1: number;
	    effectAmplitude2: number;
	    effectAmplitude3: number;
	    effectMultipleValue1: number;
	    effectMultipleValue2: number;
	    effectMultipleValue3: number;
	    effectChainTarget1: number;
	    effectChainTarget2: number;
	    effectChainTarget3: number;
	    effectItemType1: number;
	    effectItemType2: number;
	    effectItemType3: number;
	    effectMiscValue1: number;
	    effectMiscValue2: number;
	    effectMiscValue3: number;
	    effectTriggerSpell1: number;
	    effectTriggerSpell2: number;
	    effectTriggerSpell3: number;
	    effectPointsPerComboPoint1: number;
	    effectPointsPerComboPoint2: number;
	    effectPointsPerComboPoint3: number;
	    spellVisual1: number;
	    spellVisual2: number;
	    spellIconId: number;
	    activeIconId: number;
	    spellPriority: number;
	    name: string;
	    nameFlags: number;
	    nameSubtext: string;
	    nameSubtextFlags: number;
	    description: string;
	    descriptionFlags: number;
	    auraDescription: string;
	    auraDescriptionFlags: number;
	    manaCostPercentage: number;
	    startRecoveryCategory: number;
	    startRecoveryTime: number;
	    minTargetLevel: number;
	    maxTargetLevel: number;
	    spellFamilyName: number;
	    spellFamilyFlags: number;
	    maxAffectedTargets: number;
	    dmgClass: number;
	    preventionType: number;
	    stanceBarOrder: number;
	    dmgMultiplier1: number;
	    dmgMultiplier2: number;
	    dmgMultiplier3: number;
	    minFactionId: number;
	    minReputation: number;
	    requiredAuraVision: number;
	    customFlags: number;
	    icon: string;
	    schoolName: string;
	    toolTip: string;
	    castTime: string;
	    range: string;
	    duration: string;
	    power: string;
	    cooldown: string;
	    gcd: string;
	    proc: string;
	    mechanicName: string;
	    dispelType: string;
	    effects?: SpellEffectInfo[];
	    flags?: string[];
	    usedByItems?: SpellUsedByItem[];
	    taughtByNpcs?: SpellTrainerNpc[];
	    taughtByItems?: SpellUsedByItem[];
	    taughtByQuests?: SpellRewardQuest[];
	
	    static createFrom(source: any = {}) {
	        return new SpellDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.school = source["school"];
	        this.category = source["category"];
	        this.castUI = source["castUI"];
	        this.dispel = source["dispel"];
	        this.mechanic = source["mechanic"];
	        this.attributes = source["attributes"];
	        this.attributesEx = source["attributesEx"];
	        this.attributesEx2 = source["attributesEx2"];
	        this.attributesEx3 = source["attributesEx3"];
	        this.attributesEx4 = source["attributesEx4"];
	        this.stances = source["stances"];
	        this.stancesNot = source["stancesNot"];
	        this.targets = source["targets"];
	        this.targetCreatureType = source["targetCreatureType"];
	        this.requiresSpellFocus = source["requiresSpellFocus"];
	        this.casterAuraState = source["casterAuraState"];
	        this.targetAuraState = source["targetAuraState"];
	        this.castingTimeIndex = source["castingTimeIndex"];
	        this.recoveryTime = source["recoveryTime"];
	        this.categoryRecoveryTime = source["categoryRecoveryTime"];
	        this.interruptFlags = source["interruptFlags"];
	        this.auraInterruptFlags = source["auraInterruptFlags"];
	        this.channelInterruptFlags = source["channelInterruptFlags"];
	        this.procFlags = source["procFlags"];
	        this.procChance = source["procChance"];
	        this.procCharges = source["procCharges"];
	        this.maxLevel = source["maxLevel"];
	        this.baseLevel = source["baseLevel"];
	        this.spellLevel = source["spellLevel"];
	        this.durationIndex = source["durationIndex"];
	        this.powerType = source["powerType"];
	        this.manaCost = source["manaCost"];
	        this.manCostPerLevel = source["manCostPerLevel"];
	        this.manaPerSecond = source["manaPerSecond"];
	        this.manaPerSecondPerLevel = source["manaPerSecondPerLevel"];
	        this.rangeIndex = source["rangeIndex"];
	        this.speed = source["speed"];
	        this.modelNextSpell = source["modelNextSpell"];
	        this.stackAmount = source["stackAmount"];
	        this.totem1 = source["totem1"];
	        this.totem2 = source["totem2"];
	        this.reagent1 = source["reagent1"];
	        this.reagent2 = source["reagent2"];
	        this.reagent3 = source["reagent3"];
	        this.reagent4 = source["reagent4"];
	        this.reagent5 = source["reagent5"];
	        this.reagent6 = source["reagent6"];
	        this.reagent7 = source["reagent7"];
	        this.reagent8 = source["reagent8"];
	        this.reagentCount1 = source["reagentCount1"];
	        this.reagentCount2 = source["reagentCount2"];
	        this.reagentCount3 = source["reagentCount3"];
	        this.reagentCount4 = source["reagentCount4"];
	        this.reagentCount5 = source["reagentCount5"];
	        this.reagentCount6 = source["reagentCount6"];
	        this.reagentCount7 = source["reagentCount7"];
	        this.reagentCount8 = source["reagentCount8"];
	        this.equippedItemClass = source["equippedItemClass"];
	        this.equippedItemSubClassMask = source["equippedItemSubClassMask"];
	        this.equippedItemInventoryTypeMask = source["equippedItemInventoryTypeMask"];
	        this.effect1 = source["effect1"];
	        this.effect2 = source["effect2"];
	        this.effect3 = source["effect3"];
	        this.effectDieSides1 = source["effectDieSides1"];
	        this.effectDieSides2 = source["effectDieSides2"];
	        this.effectDieSides3 = source["effectDieSides3"];
	        this.effectBaseDice1 = source["effectBaseDice1"];
	        this.effectBaseDice2 = source["effectBaseDice2"];
	        this.effectBaseDice3 = source["effectBaseDice3"];
	        this.effectDicePerLevel1 = source["effectDicePerLevel1"];
	        this.effectDicePerLevel2 = source["effectDicePerLevel2"];
	        this.effectDicePerLevel3 = source["effectDicePerLevel3"];
	        this.effectRealPointsPerLevel1 = source["effectRealPointsPerLevel1"];
	        this.effectRealPointsPerLevel2 = source["effectRealPointsPerLevel2"];
	        this.effectRealPointsPerLevel3 = source["effectRealPointsPerLevel3"];
	        this.effectBasePoints1 = source["effectBasePoints1"];
	        this.effectBasePoints2 = source["effectBasePoints2"];
	        this.effectBasePoints3 = source["effectBasePoints3"];
	        this.effectBonusCoefficient1 = source["effectBonusCoefficient1"];
	        this.effectBonusCoefficient2 = source["effectBonusCoefficient2"];
	        this.effectBonusCoefficient3 = source["effectBonusCoefficient3"];
	        this.effectMechanic1 = source["effectMechanic1"];
	        this.effectMechanic2 = source["effectMechanic2"];
	        this.effectMechanic3 = source["effectMechanic3"];
	        this.effectImplicitTargetA1 = source["effectImplicitTargetA1"];
	        this.effectImplicitTargetA2 = source["effectImplicitTargetA2"];
	        this.effectImplicitTargetA3 = source["effectImplicitTargetA3"];
	        this.effectImplicitTargetB1 = source["effectImplicitTargetB1"];
	        this.effectImplicitTargetB2 = source["effectImplicitTargetB2"];
	        this.effectImplicitTargetB3 = source["effectImplicitTargetB3"];
	        this.effectRadiusIndex1 = source["effectRadiusIndex1"];
	        this.effectRadiusIndex2 = source["effectRadiusIndex2"];
	        this.effectRadiusIndex3 = source["effectRadiusIndex3"];
	        this.effectApplyAuraName1 = source["effectApplyAuraName1"];
	        this.effectApplyAuraName2 = source["effectApplyAuraName2"];
	        this.effectApplyAuraName3 = source["effectApplyAuraName3"];
	        this.effectAmplitude1 = source["effectAmplitude1"];
	        this.effectAmplitude2 = source["effectAmplitude2"];
	        this.effectAmplitude3 = source["effectAmplitude3"];
	        this.effectMultipleValue1 = source["effectMultipleValue1"];
	        this.effectMultipleValue2 = source["effectMultipleValue2"];
	        this.effectMultipleValue3 = source["effectMultipleValue3"];
	        this.effectChainTarget1 = source["effectChainTarget1"];
	        this.effectChainTarget2 = source["effectChainTarget2"];
	        this.effectChainTarget3 = source["effectChainTarget3"];
	        this.effectItemType1 = source["effectItemType1"];
	        this.effectItemType2 = source["effectItemType2"];
	        this.effectItemType3 = source["effectItemType3"];
	        this.effectMiscValue1 = source["effectMiscValue1"];
	        this.effectMiscValue2 = source["effectMiscValue2"];
	        this.effectMiscValue3 = source["effectMiscValue3"];
	        this.effectTriggerSpell1 = source["effectTriggerSpell1"];
	        this.effectTriggerSpell2 = source["effectTriggerSpell2"];
	        this.effectTriggerSpell3 = source["effectTriggerSpell3"];
	        this.effectPointsPerComboPoint1 = source["effectPointsPerComboPoint1"];
	        this.effectPointsPerComboPoint2 = source["effectPointsPerComboPoint2"];
	        this.effectPointsPerComboPoint3 = source["effectPointsPerComboPoint3"];
	        this.spellVisual1 = source["spellVisual1"];
	        this.spellVisual2 = source["spellVisual2"];
	        this.spellIconId = source["spellIconId"];
	        this.activeIconId = source["activeIconId"];
	        this.spellPriority = source["spellPriority"];
	        this.name = source["name"];
	        this.nameFlags = source["nameFlags"];
	        this.nameSubtext = source["nameSubtext"];
	        this.nameSubtextFlags = source["nameSubtextFlags"];
	        this.description = source["description"];
	        this.descriptionFlags = source["descriptionFlags"];
	        this.auraDescription = source["auraDescription"];
	        this.auraDescriptionFlags = source["auraDescriptionFlags"];
	        this.manaCostPercentage = source["manaCostPercentage"];
	        this.startRecoveryCategory = source["startRecoveryCategory"];
	        this.startRecoveryTime = source["startRecoveryTime"];
	        this.minTargetLevel = source["minTargetLevel"];
	        this.maxTargetLevel = source["maxTargetLevel"];
	        this.spellFamilyName = source["spellFamilyName"];
	        this.spellFamilyFlags = source["spellFamilyFlags"];
	        this.maxAffectedTargets = source["maxAffectedTargets"];
	        this.dmgClass = source["dmgClass"];
	        this.preventionType = source["preventionType"];
	        this.stanceBarOrder = source["stanceBarOrder"];
	        this.dmgMultiplier1 = source["dmgMultiplier1"];
	        this.dmgMultiplier2 = source["dmgMultiplier2"];
	        this.dmgMultiplier3 = source["dmgMultiplier3"];
	        this.minFactionId = source["minFactionId"];
	        this.minReputation = source["minReputation"];
	        this.requiredAuraVision = source["requiredAuraVision"];
	        this.customFlags = source["customFlags"];
	        this.icon = source["icon"];
	        this.schoolName = source["schoolName"];
	        this.toolTip = source["toolTip"];
	        this.castTime = source["castTime"];
	        this.range = source["range"];
	        this.duration = source["duration"];
	        this.power = source["power"];
	        this.cooldown = source["cooldown"];
	        this.gcd = source["gcd"];
	        this.proc = source["proc"];
	        this.mechanicName = source["mechanicName"];
	        this.dispelType = source["dispelType"];
	        this.effects = this.convertValues(source["effects"], SpellEffectInfo);
	        this.flags = source["flags"];
	        this.usedByItems = this.convertValues(source["usedByItems"], SpellUsedByItem);
	        this.taughtByNpcs = this.convertValues(source["taughtByNpcs"], SpellTrainerNpc);
	        this.taughtByItems = this.convertValues(source["taughtByItems"], SpellUsedByItem);
	        this.taughtByQuests = this.convertValues(source["taughtByQuests"], SpellRewardQuest);
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
	
	
	export class SpellSkill {
	    id: number;
	    categoryId: number;
	    name: string;
	    spellCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SpellSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.categoryId = source["categoryId"];
	        this.name = source["name"];
	        this.spellCount = source["spellCount"];
	    }
	}
	export class SpellSkillCategory {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SpellSkillCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	
	
	export class StatType {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new StatType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class TooltipEffect {
	    text: string;
	    spellId: number;
	
	    static createFrom(source: any = {}) {
	        return new TooltipEffect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.spellId = source["spellId"];
	    }
	}
	export class TooltipData {
	    entry: number;
	    name: string;
	    quality: number;
	    itemLevel?: number;
	    binding?: string;
	    unique?: boolean;
	    typeName?: string;
	    slotName?: string;
	    armor?: number;
	    damageText?: string;
	    speedText?: string;
	    dps?: string;
	    stats?: string[];
	    resistances?: string[];
	    effects?: TooltipEffect[];
	    requiredLevel?: number;
	    sellPrice?: number;
	    durability?: string;
	    classes?: string;
	    races?: string;
	    setInfo?: ItemSetInfo;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new TooltipData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.quality = source["quality"];
	        this.itemLevel = source["itemLevel"];
	        this.binding = source["binding"];
	        this.unique = source["unique"];
	        this.typeName = source["typeName"];
	        this.slotName = source["slotName"];
	        this.armor = source["armor"];
	        this.damageText = source["damageText"];
	        this.speedText = source["speedText"];
	        this.dps = source["dps"];
	        this.stats = source["stats"];
	        this.resistances = source["resistances"];
	        this.effects = this.convertValues(source["effects"], TooltipEffect);
	        this.requiredLevel = source["requiredLevel"];
	        this.sellPrice = source["sellPrice"];
	        this.durability = source["durability"];
	        this.classes = source["classes"];
	        this.races = source["races"];
	        this.setInfo = this.convertValues(source["setInfo"], ItemSetInfo);
	        this.description = source["description"];
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
	
	export class ZoneSpawn {
	    entry: number;
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new ZoneSpawn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class ZoneObject {
	    entry: number;
	    name: string;
	    type: number;
	    typeName: string;
	
	    static createFrom(source: any = {}) {
	        return new ZoneObject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	    }
	}
	export class ZoneQuest {
	    entry: number;
	    title: string;
	    questLevel: number;
	    minLevel: number;
	
	    static createFrom(source: any = {}) {
	        return new ZoneQuest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.questLevel = source["questLevel"];
	        this.minLevel = source["minLevel"];
	    }
	}
	export class ZoneNpc {
	    entry: number;
	    name: string;
	    subname: string;
	    levelMin: number;
	    levelMax: number;
	    rank: number;
	    rankName: string;
	    type: number;
	    typeName: string;
	    npcFlags: number;
	
	    static createFrom(source: any = {}) {
	        return new ZoneNpc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.rank = source["rank"];
	        this.rankName = source["rankName"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.npcFlags = source["npcFlags"];
	    }
	}
	export class ZoneDetail {
	    id: number;
	    name: string;
	    groupName: string;
	    mapName: string;
	    minLevel: number;
	    maxLevel: number;
	    npcs: ZoneNpc[];
	    quests: ZoneQuest[];
	    objects: ZoneObject[];
	    spawns: ZoneSpawn[];
	    objectSpawns: ZoneSpawn[];
	
	    static createFrom(source: any = {}) {
	        return new ZoneDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.groupName = source["groupName"];
	        this.mapName = source["mapName"];
	        this.minLevel = source["minLevel"];
	        this.maxLevel = source["maxLevel"];
	        this.npcs = this.convertValues(source["npcs"], ZoneNpc);
	        this.quests = this.convertValues(source["quests"], ZoneQuest);
	        this.objects = this.convertValues(source["objects"], ZoneObject);
	        this.spawns = this.convertValues(source["spawns"], ZoneSpawn);
	        this.objectSpawns = this.convertValues(source["objectSpawns"], ZoneSpawn);
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
	export class ZoneListEntry {
	    id: number;
	    name: string;
	    groupId: number;
	    groupName: string;
	    npcCount: number;
	    questCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ZoneListEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.groupId = source["groupId"];
	        this.groupName = source["groupName"];
	        this.npcCount = source["npcCount"];
	        this.questCount = source["questCount"];
	    }
	}
	
	
	

}

export namespace services {
	
	export class ImportResult {
	    checked: number;
	    imported: number;
	    failed: number;
	    items: string[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.checked = source["checked"];
	        this.imported = source["imported"];
	        this.failed = source["failed"];
	        this.items = source["items"];
	        this.errors = source["errors"];
	    }
	}
	export class MissingItem {
	    itemId: number;
	    tableKey: string;
	    tableName: string;
	
	    static createFrom(source: any = {}) {
	        return new MissingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.tableKey = source["tableKey"];
	        this.tableName = source["tableName"];
	    }
	}
	export class NpcAbility {
	    spellId: number;
	    name: string;
	    description: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new NpcAbility(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.icon = source["icon"];
	    }
	}
	export class NpcTrainSpell {
	    spellId: number;
	    name: string;
	    subtext: string;
	    level: number;
	    iconName: string;
	
	    static createFrom(source: any = {}) {
	        return new NpcTrainSpell(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.spellId = source["spellId"];
	        this.name = source["name"];
	        this.subtext = source["subtext"];
	        this.level = source["level"];
	        this.iconName = source["iconName"];
	    }
	}
	export class NpcSellItem {
	    itemId: number;
	    name: string;
	    quality: number;
	    iconPath: string;
	    cost: number;
	    stock: number;
	
	    static createFrom(source: any = {}) {
	        return new NpcSellItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.name = source["name"];
	        this.quality = source["quality"];
	        this.iconPath = source["iconPath"];
	        this.cost = source["cost"];
	        this.stock = source["stock"];
	    }
	}
	export class NpcSpawn {
	    mapId: number;
	    zoneName: string;
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new NpcSpawn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mapId = source["mapId"];
	        this.zoneName = source["zoneName"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class NpcQuest {
	    questId: number;
	    title: string;
	    type: string;
	    level: number;
	
	    static createFrom(source: any = {}) {
	        return new NpcQuest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.questId = source["questId"];
	        this.title = source["title"];
	        this.type = source["type"];
	        this.level = source["level"];
	    }
	}
	export class NpcLoot {
	    itemId: number;
	    name: string;
	    chance: number;
	    minCount: number;
	    maxCount: number;
	    quality: number;
	    iconPath: string;
	
	    static createFrom(source: any = {}) {
	        return new NpcLoot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.itemId = source["itemId"];
	        this.name = source["name"];
	        this.chance = source["chance"];
	        this.minCount = source["minCount"];
	        this.maxCount = source["maxCount"];
	        this.quality = source["quality"];
	        this.iconPath = source["iconPath"];
	    }
	}
	export class NpcFullDetails {
	    entry: number;
	    name: string;
	    subname?: string;
	    levelMin: number;
	    levelMax: number;
	    healthMin: number;
	    healthMax: number;
	    manaMin: number;
	    manaMax: number;
	    goldMin: number;
	    goldMax: number;
	    type: number;
	    typeName: string;
	    rank: number;
	    rankName: string;
	    faction: number;
	    npcFlags: number;
	    minDmg: number;
	    maxDmg: number;
	    armor: number;
	    holyRes: number;
	    fireRes: number;
	    natureRes: number;
	    frostRes: number;
	    shadowRes: number;
	    arcaneRes: number;
	    displayId1: number;
	    infobox: Record<string, string>;
	    mapUrl: string;
	    modelImageUrl: string;
	    factionName: string;
	    factionId: number;
	    zoneName: string;
	    x: number;
	    y: number;
	    loot: NpcLoot[];
	    quests: NpcQuest[];
	    abilities: NpcAbility[];
	    spawns: NpcSpawn[];
	    sells: NpcSellItem[];
	    trains: NpcTrainSpell[];
	
	    static createFrom(source: any = {}) {
	        return new NpcFullDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.subname = source["subname"];
	        this.levelMin = source["levelMin"];
	        this.levelMax = source["levelMax"];
	        this.healthMin = source["healthMin"];
	        this.healthMax = source["healthMax"];
	        this.manaMin = source["manaMin"];
	        this.manaMax = source["manaMax"];
	        this.goldMin = source["goldMin"];
	        this.goldMax = source["goldMax"];
	        this.type = source["type"];
	        this.typeName = source["typeName"];
	        this.rank = source["rank"];
	        this.rankName = source["rankName"];
	        this.faction = source["faction"];
	        this.npcFlags = source["npcFlags"];
	        this.minDmg = source["minDmg"];
	        this.maxDmg = source["maxDmg"];
	        this.armor = source["armor"];
	        this.holyRes = source["holyRes"];
	        this.fireRes = source["fireRes"];
	        this.natureRes = source["natureRes"];
	        this.frostRes = source["frostRes"];
	        this.shadowRes = source["shadowRes"];
	        this.arcaneRes = source["arcaneRes"];
	        this.displayId1 = source["displayId1"];
	        this.infobox = source["infobox"];
	        this.mapUrl = source["mapUrl"];
	        this.modelImageUrl = source["modelImageUrl"];
	        this.factionName = source["factionName"];
	        this.factionId = source["factionId"];
	        this.zoneName = source["zoneName"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.loot = this.convertValues(source["loot"], NpcLoot);
	        this.quests = this.convertValues(source["quests"], NpcQuest);
	        this.abilities = this.convertValues(source["abilities"], NpcAbility);
	        this.spawns = this.convertValues(source["spawns"], NpcSpawn);
	        this.sells = this.convertValues(source["sells"], NpcSellItem);
	        this.trains = this.convertValues(source["trains"], NpcTrainSpell);
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
	
	
	
	
	
	export class RemoteItem {
	    entry: number;
	    name: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.name = source["name"];
	        this.url = source["url"];
	    }
	}
	export class RemoteQuest {
	    entry: number;
	    title: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteQuest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = source["entry"];
	        this.title = source["title"];
	        this.url = source["url"];
	    }
	}
	export class SyncItemResult {
	    success: boolean;
	    itemId: number;
	    name?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncItemResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.itemId = source["itemId"];
	        this.name = source["name"];
	        this.error = source["error"];
	    }
	}
	export class SyncSpellResult {
	    success: boolean;
	    spellId: number;
	    name?: string;
	    description?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncSpellResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.spellId = source["spellId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.error = source["error"];
	    }
	}

}

