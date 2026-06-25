package main

import (
	"context"
	_ "embed" // Use blank import to ensure it sticks, though explicit usage should be enough
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"inklab/backend/database"
	"inklab/backend/datatools"
	"inklab/backend/services"

	"github.com/joho/godotenv"
)

// App struct
type App struct {
	ctx     context.Context
	db      *database.SQLiteDB
	DataDir string // Path to data directory

	// Repositories
	itemRepo      *database.ItemRepository
	creatureRepo  *database.CreatureRepository
	questRepo     *database.QuestRepository
	spellRepo     *database.SpellRepository
	lootRepo      *database.LootRepository
	factionRepo   *database.FactionRepository
	objectRepo    *database.GameObjectRepository
	zoneRepo      *database.ZoneRepository
	categoryRepo  *database.CategoryRepository
	atlasLootRepo *database.AtlasLootRepository
	favoriteRepo  *database.FavoriteRepository

	// Cache for category lookups
	categoryCache      map[int]*database.Category
	rootCategoryByName map[string]int

	// Services
	npcService  *services.NpcService
	syncService *services.SyncService
	scraper     *services.ScraperService
	mysqlDB     *database.MySQLConnection

	// Cached client MPQ source for on-demand model rendering. The mpq set is not
	// concurrency-safe, so clientSrcMu guards both the (re)open and every use.
	clientSrc    datatools.ClientFiles
	clientSrcDir string
	clientSrcMu  sync.Mutex

	// Mode
	isDevMode bool
}

// NewApp creates a new App application struct
func NewApp(dataDir string, isDevMode bool) *App {
	return &App{
		DataDir:            dataDir,
		isDevMode:          isDevMode,
		categoryCache:      make(map[int]*database.Category),
		rootCategoryByName: make(map[string]int),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	fmt.Println("Initializing InkLab (SQLite Version)...")

	// Load .env. godotenv.Load() only reads the current working directory, which
	// is the project root under `wails dev` but is unpredictable for a packaged
	// build — so also look next to the executable and in the data dir. Without
	// this, MySQL-backed sync (e.g. creature spawns) silently does nothing in a
	// release build even when MySQL is reachable.
	envCandidates := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		envCandidates = append(envCandidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	envCandidates = append(envCandidates, filepath.Join(a.DataDir, ".env"))
	envLoaded := false
	for _, p := range envCandidates {
		if err := godotenv.Load(p); err == nil {
			fmt.Printf("✓ Loaded environment from %s\n", p)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		fmt.Printf("Warning: no .env found (looked in %v), MySQL features disabled\n", envCandidates)
	}

	// Initialize SQLite database
	dbPath := filepath.Join(a.DataDir, "inklab.db")

	db, err := database.NewSQLiteDB(dbPath)
	if err != nil {
		fmt.Printf("ERROR: Failed to open database: %v\n", err)
		return
	}

	// Ensure schema exists
	if err := db.InitSchema(); err != nil {
		fmt.Printf("ERROR: Failed to initialize schema: %v\n", err)
		return
	}

	a.db = db

	// Initialize all repositories
	a.itemRepo = database.NewItemRepository(db)
	a.creatureRepo = database.NewCreatureRepository(db)
	a.questRepo = database.NewQuestRepository(db)
	a.spellRepo = database.NewSpellRepository(db)
	a.lootRepo = database.NewLootRepository(db)
	a.factionRepo = database.NewFactionRepository(db)
	a.objectRepo = database.NewGameObjectRepository(db)
	a.zoneRepo = database.NewZoneRepository(db)
	a.categoryRepo = database.NewCategoryRepository(db)
	a.atlasLootRepo = database.NewAtlasLootRepository(db)
	a.favoriteRepo = database.NewFavoriteRepository(db)

	// Initialize favorites schema
	if err := a.favoriteRepo.InitSchema(); err != nil {
		fmt.Printf("ERROR: Failed to initialize favorites schema: %v\n", err)
	}

	// Initialize MySQL (Optional)
	mysqlUser := os.Getenv("MYSQL_USER")
	if mysqlUser != "" {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			os.Getenv("MYSQL_USER"),
			os.Getenv("MYSQL_PASSWORD"),
			os.Getenv("MYSQL_HOST"),
			os.Getenv("MYSQL_PORT"),
			os.Getenv("MYSQL_DATABASE"),
		)
		mysqlConn, err := database.NewMySQLConnection(dsn)
		if err != nil {
			fmt.Printf("MySQL Connection Failed: %v\n", err)
		} else {
			a.mysqlDB = mysqlConn
			fmt.Println("✓ MySQL Connected")
			// Inject into CreatureRepository
			a.creatureRepo.SetMySQL(mysqlConn.DB())
		}
	}

	// Print stats
	itemCount, _ := a.itemRepo.GetItemCount()
	catCount, _ := a.categoryRepo.GetCategoryCount()
	fmt.Printf("✓ Database Connected: %s\n", dbPath)
	fmt.Printf("  - Items: %d\n", itemCount)
	fmt.Printf("  - Categories: %d\n", catCount)

	// Build category cache
	a.buildCategoryCache()

	// Data import using importers
	// dataDir is already set in a.DataDir

	// If database is already populated (itemCount > 0), skip costly imports
	if itemCount > 0 {
		fmt.Println("Database already populated. Skipping initialization imports.")
	} else {

		// Import Item Sets
		fmt.Println("Checking item sets...")
		itemSetImporter := database.NewItemSetImporter(db)
		if err := itemSetImporter.CheckAndImport(a.DataDir); err != nil {
			fmt.Printf("ERROR: Failed to import item sets: %v\n", err)
		}

		// Import Factions
		fmt.Println("Checking faction data...")
		factionImporter := database.NewFactionImporter(db)
		factionImporter.CheckAndImport(a.DataDir)

	}

	// Import Metadata (Zones, Skills) - Always run this as it checks internally and initializes static data
	fmt.Println("Checking metadata...")
	metadataImporter := database.NewMetadataImporter(db)
	metadataImporter.ImportAll(a.DataDir)

	// Faction templates (NPC -> faction membership). Always check: the JSON is
	// generated by the client DBC export, which may happen after first run.
	database.NewFactionImporter(db).CheckAndImportTemplates(a.DataDir)

	// MySQL is initialized once above from the MYSQL_* env vars (loaded from
	// .env), in both dev and production. The previous dev-only block here passed
	// the ".env" path itself as the DSN, which always failed with "invalid DSN:
	// missing the slash separating the database name".

	// 4. Import Data (Developer Mode Only - users use pre-built DB)
	if a.isDevMode {
		// Import AtlasLoot
		fmt.Println("Checking AtlasLoot data...")
		alImporter := database.NewAtlasLootImporter(a.db)
		if err := alImporter.CheckAndImport(a.DataDir); err != nil {
			fmt.Printf("ERROR: Failed to import AtlasLoot: %v\n", err)
		}

		// Import MySQL Tables
		a.importFullTables(a.DataDir)
	}

	// Icon downloading is now on-demand via fix button
	// No need to auto-download on startup

	// Initialize NPC Service
	a.scraper = services.NewScraperService()
	a.npcService = services.NewNpcService(a.db.DB(), a.mysqlDB, a.scraper, a.itemRepo, a.creatureRepo, a.DataDir)
	a.syncService = services.NewSyncService(a.db.DB())

	// Async sync creature spawns for dev convenience
	if a.isDevMode && a.mysqlDB != nil {
		var spawnCount int
		a.db.DB().QueryRow("SELECT COUNT(*) FROM creature_spawn").Scan(&spawnCount)

		if spawnCount == 0 {
			fmt.Println("⚡ Starting async creature spawn sync (First Run)...")
			go func() {
				// No progress callback necessary for background startup task
				err := a.npcService.SyncAllCreatureSpawns(nil)
				if err != nil {
					fmt.Printf("Startup spawn sync warning: %v\n", err)
				} else {
					fmt.Println("✓ Creature spawn sync complete")
				}
			}()
		} else {
			fmt.Printf("⏭️  creature_spawn already has %d rows, skipping startup sync\n", spawnCount)
		}

		// Same one-time sync for game-object spawns.
		var goSpawnCount int
		a.db.DB().QueryRow("SELECT COUNT(*) FROM gameobject_spawn").Scan(&goSpawnCount)
		if goSpawnCount == 0 {
			fmt.Println("⚡ Starting async gameobject spawn sync (First Run)...")
			go func() {
				if err := a.npcService.SyncAllGameObjectSpawns(nil); err != nil {
					fmt.Printf("Startup gameobject spawn sync warning: %v\n", err)
				} else {
					fmt.Println("✓ GameObject spawn sync complete")
				}
			}()
		} else {
			fmt.Printf("⏭️  gameobject_spawn already has %d rows, skipping startup sync\n", goSpawnCount)
		}
	}

	fmt.Println("✓ InkLab ready!")
}

// importFullTables imports data from MySQL if available
// The MySQL importer checks each table individually - only empty tables are imported
func (a *App) importFullTables(dataDir string) {
	if a.mysqlDB == nil {
		fmt.Println("⚠️ No MySQL connection available. Database import skipped.")
		return
	}

	fmt.Println("⚡ Checking database tables and importing from MySQL if needed...")
	importer := database.NewMySQLImporter(a.db.DB(), a.mysqlDB.DB())
	if err := importer.ImportAllFromMySQL(); err != nil {
		fmt.Printf("❌ MySQL Import Failed: %v\n", err)
	} else {
		fmt.Println("✓ MySQL Import Check Complete")
	}

	// Apply the DBC-derived icon mappings onto the freshly imported templates.
	// These are generated by cmd/dbc2json (item_icons.json / spells_enhanced.json).
	gen := database.NewGeneratedImporter(a.db.DB())
	if err := gen.ImportSpellsFromDBC(filepath.Join(dataDir, "spells_enhanced.json")); err != nil {
		fmt.Printf("⚠️ Spell DBC import failed: %v\n", err)
	}
	if err := gen.ImportItemIcons(filepath.Join(dataDir, "item_icons.json")); err != nil {
		fmt.Printf("⚠️ Item icon import failed: %v\n", err)
	}
	if err := gen.ImportSpellIcons(filepath.Join(dataDir, "spells_enhanced.json")); err != nil {
		fmt.Printf("⚠️ Spell icon import failed: %v\n", err)
	}
	if err := gen.ImportTalents(filepath.Join(dataDir, "talents.json")); err != nil {
		fmt.Printf("⚠️ Talent import failed: %v\n", err)
	}
	// Resolve $-placeholders against the just-imported (DBC-authoritative) values.
	services.NewSyncService(a.db.DB()).FullSyncSpells(0, false, "", 0, nil)
}

// buildCategoryCache builds a cache of categories for faster lookups
func (a *App) buildCategoryCache() {
	roots, err := a.categoryRepo.GetRootCategories()
	if err != nil {
		return
	}

	for _, cat := range roots {
		a.categoryCache[cat.ID] = cat
		a.rootCategoryByName[cat.Name] = cat.ID
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.clientSrcMu.Lock()
	if a.clientSrc != nil {
		a.clientSrc.Close()
		a.clientSrc = nil
	}
	a.clientSrcMu.Unlock()
	if a.db != nil {
		a.db.Close()
	}
}

// WaitForReady waits for the app to be ready (max 5 seconds)
func (a *App) WaitForReady() bool {
	return a.db != nil
}
