import React, { useState, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { RemoveFavorite, UpdateFavoriteStatus } from '../../services/api';
import { getQualityColor } from '../../utils/wow';
import { useIcon } from '../../services/useImage';
import { ItemTooltip } from '../../components/ui';
import { useEntityNavigate } from '../../utils/entityNav';
import { tooltipQuery } from '../../hooks/queries/tooltip';
import { useFavorites, useFavoriteCategories } from '../../hooks/queries/favorites';
import { queryKeys } from '../../hooks/queries/keys';
import { queryClient } from '../../queryClient';

// Simple item card for favorites
const FavoriteItemCard = ({ item, onClick, onRemove, onStatusChange }) => {
    const { src } = useIcon(item.iconPath);
    const qualityColor = getQualityColor(item.itemQuality);
    const cardRef = useRef(null);
    const [alignLeft, setAlignLeft] = useState(false);
    
    // Tooltip trigger — data comes from the shared tooltip cache, fetched on hover.
    const [showTooltip, setShowTooltip] = useState(false);
    const { data: tooltipData } = useQuery({ ...tooltipQuery(item.itemEntry), enabled: showTooltip });

    const status = item.status || 0; // 0: None, 1: Obtained, 2: Abandoned

    const handleMouseEnter = () => {
        if (cardRef.current) {
            const rect = cardRef.current.getBoundingClientRect();
            // If space on right is less than tooltip width (320px) + margin (20px), align left
            const spaceRight = window.innerWidth - rect.right;
            setAlignLeft(spaceRight < 340);
        }
        setShowTooltip(true);
    };

    // Status visual helpers
    const getStatusIcon = () => {
        switch (status) {
            case 1: return <span className="text-green-500 font-bold">✓</span>;
            case 2: return <span className="text-red-500 font-bold">✗</span>;
            default: return <span className="opacity-0 group-hover:opacity-30">☐</span>;
        }
    };

    const getCardStyle = () => {
        if (status === 1) return "opacity-60 bg-green-900/10 border-green-900/30";
        if (status === 2) return "opacity-40 grayscale bg-red-900/5";
        return "bg-white/5 hover:bg-white/10";
    };

    const handleStatusClick = (e) => {
        e.stopPropagation();
        // Cycle: 0 -> 1 -> 2 -> 0
        const nextStatus = (status + 1) % 3;
        onStatusChange(item.itemEntry, nextStatus);
    };

    return (
        <div 
            ref={cardRef}
            className="group relative"
            onMouseEnter={handleMouseEnter}
            onMouseLeave={() => setShowTooltip(false)}
        >
            <div 
                className={`flex items-center gap-3 p-2 rounded cursor-pointer border border-transparent transition-all ${getCardStyle()}`}
                onClick={() => onClick('item', item.itemEntry)}
            >
                {/* Status Toggle Box */}
                <div 
                    className={`
                        w-6 h-6 flex items-center justify-center rounded border border-white/20 
                        hover:border-white/60 bg-black/20 flex-shrink-0 transition-colors
                        ${status === 0 ? 'border-white/10' : 'border-white/40'}
                    `}
                    onClick={handleStatusClick}
                    title="Click to cycle: None -> Obtained -> Abandoned"
                >
                    {getStatusIcon()}
                </div>

                {/* Icon */}
                <div className="w-10 h-10 rounded border border-white/20 overflow-hidden flex-shrink-0 relative">
                    <img src={src} alt="" className="w-full h-full object-cover" />
                    {status === 1 && <div className="absolute inset-0 bg-green-500/20" />}
                    {status === 2 && <div className="absolute inset-0 bg-red-500/10" />}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                    <div style={{ color: qualityColor }} className={`font-bold truncate ${status === 2 ? 'line-through decoration-white/30' : ''}`}>
                        {item.itemName || `Item #${item.itemEntry}`}
                    </div>
                    <div className="text-xs text-gray-400 flex gap-2">
                        <span>Lvl {item.itemLevel}</span>
                        <span className="text-gray-500">•</span>
                        <span className="text-gray-400">{item.category || 'Uncategorized'}</span>
                    </div>
                </div>

                {/* Actions */}
                <button 
                    className="opacity-0 group-hover:opacity-100 p-1 text-gray-500 hover:text-red-500 transition-all"
                    onClick={(e) => {
                        e.stopPropagation();
                        onRemove(item.itemEntry);
                    }}
                    title="Remove from favorites"
                >
                    ✕
                </button>
            </div>

            {/* Tooltip */}
            {showTooltip && (
                <div className={`absolute top-0 z-50 w-80 pointer-events-none ${alignLeft ? 'right-full mr-2' : 'left-full ml-2'}`}>
                     <ItemTooltip 
                        item={{ 
                            entry: item.itemEntry, 
                            name: item.itemName, 
                            quality: item.itemQuality 
                        }} 
                        tooltip={tooltipData}
                    />
                </div>
            )}
        </div>
    );
};

const FavoritesPage = () => {
    const entityNavigate = useEntityNavigate();
    const [activeCategory, setActiveCategory] = useState('All');

    const favoritesQuery = useFavorites();
    const categoriesQuery = useFavoriteCategories();

    const favorites = favoritesQuery.data || [];
    const categories = categoriesQuery.data || [];
    const loading = favoritesQuery.isLoading;

    const refresh = () => {
        favoritesQuery.refetch();
        categoriesQuery.refetch();
    };

    const handleRemove = async (entry) => {
        if (window.confirm("Remove this item from favorites?")) {
            await RemoveFavorite(entry);
            queryClient.invalidateQueries({ queryKey: queryKeys.favorites });
            queryClient.invalidateQueries({ queryKey: queryKeys.favoriteCategories });
        }
    };

    const handleStatusChange = async (entry, newStatus) => {
        // Optimistic update written straight into the cache.
        queryClient.setQueryData(queryKeys.favorites, (prev) =>
            (prev || []).map(item => (item.itemEntry === entry ? { ...item, status: newStatus } : item))
        );
        await UpdateFavoriteStatus(entry, newStatus);
    };

    // Group items if showing 'All'? Or just filter
    // User requested "Show by group", so maybe grouping headers is better for 'All' view
    // But filters are also nice.
    
    // Let's implement a filtered view.
    const safeFavorites = favorites || [];
    const filteredItems = activeCategory === 'All' 
        ? safeFavorites 
        : safeFavorites.filter(f => f.category === activeCategory);

    // Group by category for the 'All' view
    const groupedItems = activeCategory === 'All'
        ? safeFavorites.reduce((acc, item) => {
            const cat = item.category || 'Uncategorized';
            if (!acc[cat]) acc[cat] = [];
            acc[cat].push(item);
            return acc;
        }, {})
        : { [activeCategory]: filteredItems };

    return (
        <div className="h-full flex flex-col bg-bg-main overflow-hidden">
            {/* Header */}
            <div className="p-4 border-b border-white/10 flex items-center justify-between bg-bg-dark/50">
                <h2 className="text-xl font-bold text-wow-gold">My Favorites</h2>
                <button 
                    onClick={refresh}
                    className="px-3 py-1 bg-white/5 hover:bg-white/10 rounded text-sm text-gray-300"
                >
                    Refresh
                </button>
            </div>

            <div className="flex-1 flex overflow-hidden">
                {/* Sidebar - Categories */}
                <div className="w-64 bg-bg-dark/30 border-r border-white/10 overflow-y-auto p-2 space-y-1">
                    <button
                        onClick={() => setActiveCategory('All')}
                        className={`w-full text-left px-3 py-2 rounded text-sm font-medium transition-colors flex justify-between ${
                            activeCategory === 'All' 
                                ? 'bg-wow-gold text-black' 
                                : 'text-gray-400 hover:bg-white/5 hover:text-white'
                        }`}
                    >
                        <span>All Items</span>
                        <span className="opacity-60">{favorites.length}</span>
                    </button>
                    
                    <div className="h-px bg-white/10 my-2 mx-1" />

                    {categories.map(cat => (
                        <button
                            key={cat.name}
                            onClick={() => setActiveCategory(cat.name)}
                            className={`w-full text-left px-3 py-2 rounded text-sm transition-colors flex justify-between ${
                                activeCategory === cat.name 
                                    ? 'bg-wow-gold text-black' 
                                    : 'text-gray-400 hover:bg-white/5 hover:text-white'
                            }`}
                        >
                            <span className="truncate">{cat.name || 'Uncategorized'}</span>
                            <span className="opacity-60">{cat.count}</span>
                        </button>
                    ))}
                </div>

                {/* Main Content - Grid */}
                <div className="flex-1 overflow-y-auto p-4">
                    {loading ? (
                        <div className="text-center text-gray-500 mt-20">Loading favorites...</div>
                    ) : favorites.length === 0 ? (
                        <div className="text-center text-gray-500 mt-20">
                            <div className="text-4xl mb-4">❤️</div>
                            No favorites yet. <br/>
                            Go to Database and search for items to add them!
                        </div>
                    ) : (
                        <div className="space-y-8">
                            {Object.entries(groupedItems).map(([category, items]) => (
                                <div key={category}>
                                    <h3 className="text-lg font-bold text-gray-400 mb-3 px-1 flex items-center gap-2">
                                        {category}
                                        <span className="text-xs bg-white/10 px-2 py-0.5 rounded text-gray-500 font-normal">
                                            {items.length}
                                        </span>
                                    </h3>
                                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2">
                                        {items.map(item => (
                                            <FavoriteItemCard
                                                key={item.id}
                                                item={item}
                                                onClick={entityNavigate}
                                                onRemove={handleRemove}
                                                onStatusChange={handleStatusChange}
                                            />
                                        ))}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default FavoritesPage;
