# InkLab - World of Warcraft Database Browser

A comprehensive desktop application for browsing and exploring World of Warcraft (Octo WoW) game data, built with Wails, Go, and React.

## Features

### Database Browser

- **Items**: Complete item database with detailed statistics
  - Search by name, class, subclass, and inventory slot
  - WoW-style tooltips with complete item information
  - Icon display with local cache and CDN fallback
- **AtlasLoot Integration**: Complete loot table browser

  - 7 categories: Instances, Sets, Factions, PvP, World Bosses, World Events, Crafting
  - Hierarchical navigation (Category → Module → Table → Items)
  - Drop chance information where available

- **Creatures**: Browse creature database

  - Search by name and type
  - Paginated results for performance
  - View creature loot tables

- **Quests**: Explore quest database

  - Browse by zone or quest category
  - View quest details and objectives

- **Spells**: Search spell database

  - Browse by class and skill category
  - View spell effects and icons

- **Game Objects**: Browse object database

  - Search by name and type
  - View object loot tables

- **Factions**: View faction database
  - Reputation and faction rewards

## Architecture

### Technology Stack

- **Backend**: Go 1.24 + Wails v2.11
- **Frontend**: React 18 + TypeScript + Vite
- **Database**: SQLite 3
- **Styling**: Tailwind CSS with custom WoW theme

### Data Pipeline

The application supports two modes of data operation:

1. **End User Mode** (Default):

   - Uses the embedded SQLite database (`data/inklab.db`)
   - Syncs missing or updated data directly from `octowow.st` via the built-in Sync Service
   - No external database dependencies required

2. **Developer Mode** (Optional):
   - Can connect to a local MySQL instance for custom data export
   - Python export scripts available in `scripts/` (Optional usage)
   - Useful for initial database population or large schema updates

**Sync Service (`backend/services/`)**:

- Scrapes and parses data from `octowow.st/db`
- Supports Items, NPCs, Quests, and Objects (spell text is resolved locally from client DBC data)
- Multi-threaded worker pools for fast synchronization
- "AtlasLoot Missing" mode to find gaps in local data

## Installation

1. **Download** the latest release for your platform from the [Releases page](https://github.com/brues-code/inklab/releases/latest).

2. **Extract** the archive anywhere and run the executable (`InkLab.exe` on Windows). The database ships embedded in the binary and is unpacked into a `data/` folder next to it on first launch — no external database or other setup required.

3. **Run the Client Data import.** Icons, zone maps, and other client-derived reference data are built locally from your WoW client. In the app, open **Tools → Import → Client Data (icons, maps, DBC)**, set the base path to your client folder (the one containing `Data\*.MPQ`), and run it once.

Upgrading is the same: newer releases refresh the embedded database automatically on launch, and your locally-built icons and maps are kept.

## Development

### Building from Source

Prerequisites: Go 1.24+, Node.js 18+, Wails v2.11+

```bash
# Clone the repository
git clone https://github.com/brues-code/inklab.git
cd inklab

# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev

# Or produce a production binary in build/bin
wails build
```

### Database Schema

The application uses a SQLite database with 30+ tables:

**Core Tables**:

- `item_template`: Items (1:1 MySQL mapping)
- `creature_template`: Creatures
- `quest_template`: Quests
- `spell_template`: Spells
- `gameobject_template`: Objects

**AtlasLoot Tables**:

- `atlasloot_categories`: Categories
- `atlasloot_modules`: Modules
- `atlasloot_tables`: Loot tables
- `atlasloot_items`: Loot entries

**Loot Tables**:

- `creature_loot_template`
- `item_loot_template`
- `gameobject_loot_template`
- `reference_loot_template`
- `disenchant_loot_template`

### Data Update Workflow

1. **Sync Service (Recommended)**:

   - Use the in-app **Sync** page to sync data from `octowow.st`.
   - This approach is incremental and does not require external tools.

2. **Developer Export (Legacy/Full Rebuild)**:
   - If you have a local Turtle WoW MySQL database, you can use `scripts/export_all_data.py` to dump JSONs.
   - Using `wails dev` with no existing DB will trigger an import from `data/*.json`.

### Icon Management

Icon images are never downloaded from the network. The Client Data import decodes them straight from your WoW client's MPQ archives into `data/icons/`, and the UI serves them from there. The icon-fix tooling only discovers missing icon *names* (e.g. `inv_sword_01`) from `octowow.st/db` so the locally-extracted images can resolve; anything still unresolved falls back to a question-mark placeholder.

## Data Sources

- **Turtle-WoW Emulation Server Source Code**: https://github.com/brian8544/turtle-wow

## Key Technologies

- **Wails**: Go-powered desktop apps with web UI
- **SQLite**: Embedded database (no server needed)
- **Code Generation**: Python scripts auto-generate Go code
- **React Hooks**: Modern state management
- **Tailwind CSS**: Utility-first styling

## Future Enhancements

- Talent tree browser and calculator
- Equipment set manager
- Stat calculator and comparison
- DPS simulator
- Enchant and gem browser
- Character planner
- Export/import functionality

## Contributing

This project is for educational purposes and community use. Contributions welcome!

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

**Built with ❤️ for the Turtle WoW Community**
