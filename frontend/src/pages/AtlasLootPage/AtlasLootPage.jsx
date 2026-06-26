import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
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
  const { setHoveredItem, handleMouseMove, handleItemEnter } = useTooltipCtx();

  // Cascading data via Query, keyed by selection. Categories are static for a
  // session; modules/tables key by their parent; loot keys by the full path.
  const categoriesQuery = useQuery({ queryKey: ["atlasCategories"], queryFn: GetCategories, staleTime: Infinity });
  const modulesQuery = useQuery({
    queryKey: ["atlasModules", selectedCategory],
    queryFn: () => GetInstances(selectedCategory),
    enabled: !!selectedCategory,
    staleTime: Infinity,
  });
  const tablesQuery = useQuery({
    queryKey: ["atlasTables", selectedCategory, selectedModule],
    queryFn: () => GetTables(selectedCategory, selectedModule),
    enabled: !!(selectedCategory && selectedModule),
    staleTime: Infinity,
  });

  const categories = categoriesQuery.data || [];
  const modules = modulesQuery.data || [];
  const tables = tablesQuery.data || [];

  const tableKeyOf = (t) => (typeof t === "string" ? t : t?.key || t);

  // The loot table to load: a clicked boss in 3-level categories, else the first
  // table of the selected module in 2-level categories (loaded automatically).
  const effectiveTable = isThreeLevelCategory
    ? selectedTable
    : tables.length
    ? tableKeyOf(tables[0])
    : "";

  const lootQuery = useQuery({
    queryKey: ["atlasLoot", selectedCategory, selectedModule, effectiveTable],
    queryFn: () => GetLoot(selectedCategory, selectedModule, effectiveTable),
    enabled: !!(selectedCategory && selectedModule && effectiveTable),
  });
  const loot = lootQuery.data || null;

  // Selection handlers reset everything below the chosen level (no effects).
  const pickCategory = (cat) => {
    setSelectedCategory(cat);
    setSelectedModule("");
    setSelectedTable("");
    setCategoryFilter("");
    setModuleFilter("");
    setTableFilter("");
    setItemFilter("");
  };
  const pickModule = (mod) => {
    setSelectedModule(mod);
    setSelectedTable("");
    setModuleFilter("");
    setTableFilter("");
    setItemFilter("");
  };
  const pickTable = (tableKey) => {
    setSelectedTable(tableKey);
    setTableFilter("");
    setItemFilter("");
  };

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

  // Render loot content (shared between 2-level and 3-level views)
  const renderLootContent = () => {
    const showPrompt = isThreeLevelCategory ? !selectedTable : !selectedModule;
    const showLoading = !showPrompt && (lootQuery.isLoading || (!isThreeLevelCategory && tablesQuery.isLoading));
    return (
    <>
      {showLoading && (
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

      {!showPrompt && !showLoading && filteredItems.length === 0 && (
        <div className="flex-1 flex items-center justify-center text-gray-600 italic">
          No loot data found
        </div>
      )}

      {showPrompt && (
        <div className="flex-1 flex items-center justify-center text-gray-600 italic">
          {isThreeLevelCategory ? "Select a boss to view loot" : "Select a module to view items"}
        </div>
      )}
    </>
    );
  };

  // Dynamic grid layout based on category type
  const gridLayout = isThreeLevelCategory 
    ? "200px 200px 200px 1fr" 
    : "200px 200px 1fr";

  return (
    <PageLayout>
      {/* Main Loot Browser - Hidden when detail active */}
      <div className={`flex flex-col h-full flex-1 overflow-hidden ${detailActive ? 'hidden' : ''}`}>
        {categoriesQuery.isError && (
          <div className="mx-3 mt-3 p-3 bg-red-900/30 border border-red-500/30 rounded flex items-center gap-3 text-red-400">
            <span>❌</span>
            <span>Error loading categories</span>
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
              {categoriesQuery.isLoading && (
                <div className="p-4 text-center text-wow-gold italic animate-pulse">
                  Loading...
                </div>
              )}
              {filteredCategories.map((cat) => (
                <ListItem
                  key={cat}
                  active={selectedCategory === cat}
                  onClick={() => pickCategory(cat)}
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
              {modulesQuery.isLoading && (
                <div className="p-4 text-center text-wow-gold italic animate-pulse">
                  Loading...
                </div>
              )}
              {filteredModules.map((mod) => (
                <ListItem
                  key={mod}
                  active={selectedModule === mod}
                  onClick={() => pickModule(mod)}
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
                {tablesQuery.isLoading && (
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
                      onClick={() => pickTable(tableKey)}
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
