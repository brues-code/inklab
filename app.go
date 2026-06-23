package main

import (
	"context"
	_ "embed" // Use blank import to ensure it sticks, though explicit usage should be enough
	"fmt"
	"os"
	"path/filepath"

	"inklab/backend/database"
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

	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, MySQL features disabled")
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

	// Initialize MySQL Connection (Development mode only)
	if a.isDevMode {
		mysqlDB, err := database.NewMySQLConnection(filepath.Join(".", ".env"))
		if err != nil {
			fmt.Printf("⚠️ MySQL connection failed: %v. NPC sync from MySQL will be unavailable.\n", err)
		} else {
			a.mysqlDB = mysqlDB
			fmt.Println("✓ MySQL Connected (dev mode)")
		}
	}

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
	if a.db != nil {
		a.db.Close()
	}
}

// WaitForReady waits for the app to be ready (max 5 seconds)
func (a *App) WaitForReady() bool {
	return a.db != nil
}
