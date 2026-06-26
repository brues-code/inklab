import { useState, useEffect, useMemo } from "react";
import { Outlet, useNavigate, useChildMatches } from "@tanstack/react-router";
import {
  GetCategories,
  GetInstances,
  GetTables,
} from "../../../wailsjs/go/main/App";
import {
  PageLayout,
  ContentGrid,
  SidebarPanel,
  ContentPanel,
  ScrollList,
  SectionHeader,
  ListItem,
  LootItem,
} from "../../components/ui";
import { useTooltipCtx } from "../../hooks/useTooltipContext";
import { filterItems } from "../../utils/databaseApi";

// Direct call to GetLoot - using window binding
const GetLoot = (category, instance, boss) => {
  if (window?.go?.main?.App?.GetLoot) {
    return window.go.main.App.GetLoot(category, instance, boss);
  }
  return Promise.resolve({ bossName: boss, items: [] });
};

// Categories that use 3-level hierarchy (Category → Instance → Boss)
const THREE_LEVEL_CATEGORIES = ["Dungeons", "Raids", "Collections", "Sets", "Crafting", "PvP", "PvP Rewards"];

function AtlasLootPage() {
  const [categories, setCategories] = useState([]);
  const [modules, setModules] = useState([]);
  const [tables, setTables] = useState([]);
  const [loot, setLoot] = useState(null);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const [selectedCategory, setSelectedCategory] = useState("");
  const [selectedModule, setSelectedModule] = useState("");
  const [selectedTable, setSelectedTable] = useState("");

  // Filter states for each column
  const [categoryFilter, setCategoryFilter] = useState("");
  const [moduleFilter, setModuleFilter] = useState("");
  const [tableFilter, setTableFilter] = useState("");
  const [itemFilter, setItemFilter] = useState("");

  // Detail view navigation — routed; Back uses browser history. The loot
  // selection state is preserved across detail visits because this route stays
  // mounted while the detail renders in <Outlet>.
  const navigate = useNavigate();
  const detailActive = useChildMatches().length > 0;

  const navigateTo = (type, entry) =>
    navigate({ to: "/atlas/$type/$id", params: { type, id: String(entry) } });

  // Check if current category uses 3-level hierarchy
  const isThreeLevelCategory = THREE_LEVEL_CATEGORIES.includes(selectedCategory);

  // Shared app-wide tooltip (single instance lives at the router root).
  const {
    setHoveredItem,
    loadTooltipData,
    handleMouseMove,
    handleItemEnter,
  } = useTooltipCtx();

  // Filtered lists
  const filteredCategories = useMemo(
    () => filterItems(categories, categoryFilter),
    [categories, categoryFilter]
  );
  const filteredModules = useMemo(
    () => filterItems(modules, moduleFilter),
    [modules, moduleFilter]
  );
  const filteredTables = useMemo(() => {
    const tablesWithNames = tables.map((t) => {
      if (typeof t === "string") {
        return { original: t, name: t };
      } else {
        return { original: t, name: t.displayName || t.key || t };
      }
    });
    return filterItems(tablesWithNames, tableFilter);
  }, [tables, tableFilter]);
  const filteredItems = useMemo(() => {
    if (!loot?.items) return [];
    return filterItems(loot.items, itemFilter);
  }, [loot, itemFilter]);

  // Load categories on mount
  useEffect(() => {
    setLoading(true);
    GetCategories()
      .then((cats) => {
        setCategories(cats || []);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to load categories:", err);
        setError("Error loading categories");
        setLoading(false);
      });
  }, []);

  // Load modules when category changes
  useEffect(() => {
    if (selectedCategory) {
      setLoading(true);
      setModules([]);
      setTables([]);
      setLoot(null);
      setSelectedModule("");
      setSelectedTable("");
      setModuleFilter("");
      setTableFilter("");
      setItemFilter("");

      GetInstances(selectedCategory)
        .then((mods) => {
          setModules(mods || []);
          setLoading(false);
        })
        .catch((err) => {
          console.error("Failed to load modules:", err);
          setLoading(false);
        });
    }
  }, [selectedCategory]);

  // Load tables when module changes (only for 3-level categories)
  useEffect(() => {
    if (selectedModule && selectedCategory && isThreeLevelCategory) {
      setLoading(true);
      setTables([]);
      setLoot(null);
      setSelectedTable("");
      setTableFilter("");
      setItemFilter("");

      GetTables(selectedCategory, selectedModule)
        .then((tbls) => {
          setTables(tbls || []);
          setLoading(false);
        })
        .catch((err) => {
          console.error("Failed to load tables:", err);
          setLoading(false);
        });
    }
  }, [selectedModule, isThreeLevelCategory]);

  // Preload tooltips when loot changes (idempotent — cached/deduped)
  useEffect(() => {
    if (loot?.items) {
      loot.items.slice(0, 20).forEach((item) => {
        if (item.itemId && item.itemId > 0) loadTooltipData(item.itemId);
      });
    }
  }, [loot, loadTooltipData]);

  // Load loot when table is clicked (3-level) or when module is clicked directly (2-level)
  const loadLoot = (table, moduleOverride = null) => {
    const mod = moduleOverride || selectedModule;
    setSelectedTable(table);
    setLoot(null);
    setLoading(true);

    GetLoot(selectedCategory, mod, table)
      .then((result) => {
        setLoot(result);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to load loot:", err);
        setLoading(false);
      });
  };

  // Handle module click - different behavior for 2-level vs 3-level
  const handleModuleClick = (mod) => {
    setSelectedModule(mod);
    setModuleFilter("");
    
    if (!isThreeLevelCategory) {
      // For 2-level categories, load the first table directly
      setLoading(true);
      GetTables(selectedCategory, mod)
        .then((tbls) => {
          if (tbls && tbls.length > 0) {
            // Get the first table key
            const firstTable = typeof tbls[0] === 'string' ? tbls[0] : (tbls[0].key || tbls[0]);
            loadLoot(firstTable, mod);
          } else {
            setLoading(false);
          }
        })
        .catch((err) => {
          console.error("Failed to load tables:", err);
          setLoading(false);
        });
    }
  };

  // Render loot content (shared between 2-level and 3-level views)
  const renderLootContent = () => (
    <>
      {loading && !loot && (selectedTable || (!isThreeLevelCategory && selectedModule)) && (
        <div className="flex-1 flex items-center justify-center text-wow-gold italic animate-pulse">
          Loading loot...
        </div>
      )}

      {filteredItems.length > 0 && (
        <ScrollList className="grid grid-cols-1 xl:grid-cols-2 gap-1 p-2 auto-rows-min">
          {filteredItems.map((item, idx) => {
            const itemId = item.itemId || item.entry || item.id;
            const spellId = item.spellId;
            const uniqueKey = itemId || spellId || idx;
            
            return (
              <LootItem
                key={uniqueKey}
                item={{
                  entry: itemId,
                  spellId: spellId,
                  name: item.itemName || item.name,
                  quality: item.quality,
                  iconPath: item.iconName || item.iconPath,
                  dropChance: item.dropChance,
                }}
                showDropChance
                onClick={() => {
                  if (itemId) {
                    navigateTo('item', itemId);
                  } else if (spellId) {
                    // For now, spells might need a different view or external link
                    // But user requested to see "spell", so we might need a SpellDetailView potentially
                    // or just log it for now as the current system might not fully support spell details page yet.
                    console.log("Clicked spell:", spellId);
                    // navigateTo('spell', spellId); // Only if we implement SpellView
                  }
                }}
                onMouseEnter={() => itemId && handleItemEnter(itemId)}
                onMouseMove={(e) => itemId && handleMouseMove(e, itemId)}
                onMouseLeave={() => setHoveredItem(null)}
              />
            );
          })}
        </ScrollList>
      )}

      {!loading && filteredItems.length === 0 && (selectedTable || (!isThreeLevelCategory && selectedModule)) && (
        <div className="flex-1 flex items-center justify-center text-gray-600 italic">
          No loot data found
        </div>
      )}

      {!selectedTable && !(!isThreeLevelCategory && selectedModule) && (
        <div className="flex-1 flex items-center justify-center text-gray-600 italic">
          {isThreeLevelCategory ? "Select a boss to view loot" : "Select a module to view items"}
        </div>
      )}
    </>
  );

  // Dynamic grid layout based on category type
  const gridLayout = isThreeLevelCategory 
    ? "200px 200px 200px 1fr" 
    : "200px 200px 1fr";

  return (
    <PageLayout>
      {/* Main Loot Browser - Hidden when detail active */}
      <div className={`flex flex-col h-full flex-1 overflow-hidden ${detailActive ? 'hidden' : ''}`}>
        {error && (
          <div className="mx-3 mt-3 p-3 bg-red-900/30 border border-red-500/30 rounded flex items-center gap-3 text-red-400">
            <span>❌</span>
            <span>{error}</span>
          </div>
        )}

        <ContentGrid columns={gridLayout}>
          {/* Column 1: Categories */}
          <SidebarPanel>
            <SectionHeader
              title={`Categories (${filteredCategories.length})`}
              placeholder="Filter categories..."
              onFilterChange={setCategoryFilter}
            />
            <ScrollList>
              {loading && categories.length === 0 && (
                <div className="p-4 text-center text-wow-gold italic animate-pulse">
                  Loading...
                </div>
              )}
              {filteredCategories.map((cat) => (
                <ListItem
                  key={cat}
                  active={selectedCategory === cat}
                  onClick={() => {
                    setSelectedCategory(cat);
                    setCategoryFilter("");
                  }}
                >
                  {cat}
                </ListItem>
              ))}
            </ScrollList>
          </SidebarPanel>

          {/* Column 2: Modules/Instances */}
          <SidebarPanel>
            <SectionHeader
              title={
                selectedCategory
                  ? `${selectedCategory} (${filteredModules.length})`
                  : "Select Category"
              }
              placeholder={isThreeLevelCategory ? "Filter instances..." : "Filter modules..."}
              onFilterChange={setModuleFilter}
            />
            <ScrollList>
              {loading && modules.length === 0 && selectedCategory && (
                <div className="p-4 text-center text-wow-gold italic animate-pulse">
                  Loading...
                </div>
              )}
              {filteredModules.map((mod) => (
                <ListItem
                  key={mod}
                  active={selectedModule === mod}
                  onClick={() => handleModuleClick(mod)}
                >
                  {mod}
                </ListItem>
              ))}
            </ScrollList>
          </SidebarPanel>

          {/* Column 3: Tables/Bosses (only for 3-level categories) */}
          {isThreeLevelCategory && (
            <SidebarPanel>
              <SectionHeader
                title={
                  selectedModule
                    ? `${selectedModule} (${filteredTables.length})`
                    : "Select Instance"
                }
                placeholder="Filter bosses..."
                onFilterChange={setTableFilter}
              />
              <ScrollList>
                {loading && tables.length === 0 && selectedModule && (
                  <div className="p-4 text-center text-wow-gold italic animate-pulse">
                    Loading...
                  </div>
                )}
                {filteredTables.map((tbl, idx) => {
                  const originalTable = tbl.original;
                  const tableKey =
                    typeof originalTable === "string"
                      ? originalTable
                      : originalTable.key || originalTable;
                  return (
                    <ListItem
                      key={tableKey || idx}
                      active={selectedTable === tableKey}
                      onClick={() => {
                        loadLoot(tableKey);
                        setTableFilter("");
                      }}
                    >
                      {tbl.name}
                    </ListItem>
                  );
                })}
              </ScrollList>
            </SidebarPanel>
          )}

          {/* Final Column: Loot Display */}
          <ContentPanel>
            <SectionHeader
              title={
                loot ? `${loot.bossName} (${filteredItems.length})` : "Loot Table"
              }
              placeholder="Filter items..."
              onFilterChange={setItemFilter}
            />
            {renderLootContent()}
          </ContentPanel>
        </ContentGrid>
      </div>

      {/* Detail View Overlay (routed) */}
      <Outlet />
    </PageLayout>
  );
}

export default AtlasLootPage;
