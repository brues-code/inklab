# InkLab - World of Warcraft Database Browser

A comprehensive desktop application for browsing and exploring World of Warcraft (Turtle WoW) game data, built with Wails, Go, and React.

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
   - Syncs missing or updated data directly from `turtlecraft.gg` via the built-in Sync Service
   - No external database dependencies required

2. **Developer Mode** (Optional):
   - Can connect to a local MySQL instance for custom data export
   - Python export scripts available in `scripts/` (Optional usage)
   - Useful for initial database population or large schema updates

**Sync Service (`backend/services/`)**:

- Scrapes and parses data from `database.turtlecraft.gg`
- Supports Items, Spells, Quests, and Icons
- Multi-threaded worker pools for fast synchronization
- "AtlasLoot Missing" mode to find gaps in local data

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+
- Wails v2.11+

### Installation

```bash
# Clone the repository
git clone https://github.com/oplancelot/InkLab.git
cd InkLab

# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev
```

### First Run

On first startup, the application will:

1. Initialize the SQLite database connection
2. Validate the integrity of `data/inklab.db`
3. Ready to use immediately

**Note**: You can use the **Settings** page in the app to update your local database with the latest changes from Turtle WoW (Sync Items, Spells, Quests, or missing AtlasLoot items).

## Development

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

   - Use the in-app Settings to sync data from `turtlecraft.gg`.
   - This approach is incremental and does not require external tools.

2. **Developer Export (Legacy/Full Rebuild)**:
   - If you have a local Turtle WoW MySQL database, you can use `scripts/export_all_data.py` to dump JSONs.
   - Using `wails dev` with no existing DB will trigger an import from `data/*.json`.

### Icon Management

Icons are automatically downloaded on-demand or via the "Auto-fix" option in Settings:

1. **Wowhead CDN** (`wow.zamimg.com`) - Primary source
2. **Turtle WoW Database** (`database.turtlecraft.gg`) - Fallback
3. **Trinity AoWoW** (`aowow.trinitycore.info`) - Fallback

Icons are cached in `data/icons/` for offline use.

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
