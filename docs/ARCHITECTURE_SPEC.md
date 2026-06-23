# InkLab Project Specification & Data Architecture

## 1. Project Overview

InkLab is a desktop database companion application for Octo WoW, built using Wails (Go + React). It provides a rich, responsive interface for browsing items, NPCs, quests, and other game data.

## 2. Core Philosophy

**The Release Build is Standalone.**

- The final distributed application (`inklab.exe`) **must not** depend on an external MySQL server or require the user to have specific developer environments set up.
- All data required for the application to function must be contained within:
  1. **SQLite Database** (`inklab.db`): The single source of truth for the runtime application.
  2. **Data Assets** (`data/`): Images, icons, and maps.
  3. **Application Binary**: The compiled Wails frontend/backend.

## 3. Data Sources & hierarchy

The data within `inklab.db` is aggregated from specific sources with a strict hierarchy of authority.

### A. Primary Truth (Web Sources)

These are the authoritative sources for game data values.

1. **octowow.st/db** (OctoWow Database): Primary source for server-specific custom content (Items, Quests, Spells, NPCs, Icons). It runs the same AoWoW database engine as the former `database.turtlecraft.gg` site, but with a different page structure (title suffix, mapper/coords format, infobox markup, breadcrumb-based item class) — the scrapers/parsers in `backend/parsers` were adapted to it. The base URL lives in a single constant (`services.DatabaseBaseURL` / `DATABASE_BASE_URL`).
2. **Wowhead (Classic)**: Fallback source for Vanilla/Classic data (NPC details, Maps, Lore, Standard Drop rates).

### B. Development Data Source (Local MySQL)

_Role: Ingestion & Schema Reference_

- **Usage**: Used **only** during the development/data curation phase.
- **Source**: Local `tw_world` MySQL database (standard Mangos/Turtle-WoW core).
- **Purpose**:
  - To bulk populate the SQLite database initially.
  - To validate table structures.
  - To extract relationships (NPC loot tables, Quest starters/enders).
- **Restriction**: MySQL code must be isolated. The release build should function perfectly without a MySQL connection (graceful degradation or build tags).

### C. AtlasLoot Data

_Role: UI Structure & Categorization_

- **Source**: `AtlasLoot` Lua tables converted to JSON.
- **Purpose**: Provides the "Dungeon Journal" style navigation structure (Server -> Dungeon -> Boss -> Loot).
- **Processing**: We process these JSONs to build the navigation tree in SQLite.

## 4. Workflows

### Development Workflow

1. **Ingest**: Developer runs importers that pull data from Local MySQL or Scrape Web Sources.
2. **Store**: Data is normalized and stored in `inklab.db` (SQLite).
3. **Verify**: UI reads solely from SQLite to display data.

### Release Workflow

1. **Build**: The application is compiled.
2. **Package**: The populated `inklab.db` is bundled (or downloaded on first run) along with the executable.
3. **Run**: The User runs `inklab.exe`. It reads from `inklab.db`. It may perform live web-scraping (e.g., Wowhead) for ephemeral data but should cache it to SQLite.

## 5. Deprecations

- **Raw MySQL Exports (JSON)**: `item_template.json` and similar raw dumps are deprecated in favor of direct MySQL->SQLite ingestion during development or direct Web->SQLite syncing.
- **Direct MySQL Runtime Dependency**: The app must never crash if MySQL is absent.

## 6. Technical Stack

- **Frontend**: React, TailwindCSS (Vanilla CSS preference), HeroIcons.
- **Backend**: Go (Wails).
- **Database**: SQLite (via `modernc.org/sqlite` or CGO-free driver).
- **External Comms**: HTTP Client for scraping/syncing.

## 7. Software Design Principles (KISS, DRY, SOLID)

To ensure long-term maintainability and robustness, the codebase adheres to the following principles:

### SOLID

- **Single Responsibility (SRP)**: Each service has a distinct purpose.
  - `ScraperService`: Purely for HTML parsing and web extraction.
  - `NpcService`: Orchestrates data flow (Cache -> Sync -> DB), acting as the business logic layer.
  - `MySQLConnection`: Handles raw database connectivity only.
- **Dependency Inversion (DIP)**: Services depend on interfaces (e.g., `HttpClient`) rather than concrete implementations, facilitating testing and configuration.

### DRY (Don't Repeat Yourself)

- **Unified Data Access**: All data retrieval logic (e.g., "Check local cache, if missing then sync") is centralized in Services (e.g., `NpcService`). The Frontend never calls raw SQL or external APIs directly; it asks the Service for data, and the Service handles the sourcing strategy.
- **Shared Schemas**: Database schemas (`generated_schema.go`) are shared for 1:1 mapping where possible to avoid redefining structures.

### KISS (Keep It Simple, Stupid)

- **Architecture**: The "SQLite as Source of Truth" model simplifies the release build significantly. There is no complex runtime config for the end-user.
- **Read-Through Caching**: The application logic is straightforward: "Read Local. If missing, Fetch Remote & Save Local." This avoids complex synchronization states or distributed transaction requirements.

---

_Created: 2026-01-09_
