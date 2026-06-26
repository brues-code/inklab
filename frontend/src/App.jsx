import { useState, useCallback } from 'react'
import AtlasLootPage from './pages/AtlasLootPage/AtlasLootPage'
import DatabasePage from './pages/DatabasePage/DatabasePage'
import SearchPage from './pages/SearchPage/SearchPage'
import TalentsPage from './pages/TalentsPage'
import FlightsPage from './pages/FlightsPage'
import SyncPage from './pages/SyncPage'
import ToolsPage from './pages/ToolsPage'
import FavoritesPage from './pages/FavoritesPage/FavoritesPage'
import { TabButton, UpdateBanner, DataStatusBanner } from './components/ui'

function App() {
    const [activeTab, setActiveTab] = useState('database')
    
    // Pending navigation target (from SearchPage to Database)
    const [pendingNavigation, setPendingNavigation] = useState(null)

    // Handle navigation from SearchPage - switch to database tab and open item
    const handleSearchNavigate = useCallback((type, entry) => {
        console.log(`[App] Search navigation: ${type} #${entry}`)
        setPendingNavigation({ type, entry })
        setActiveTab('database')
    }, [])

    // Clear pending navigation (called by DatabasePage after receiving it)
    const clearPendingNavigation = useCallback(() => {
        setPendingNavigation(null)
    }, [])

    return (
        <div className="h-screen flex flex-col bg-bg-dark text-white">
            <UpdateBanner />
            <DataStatusBanner onGoToTools={() => setActiveTab('tools')} />
            {/* Header */}
            <header className="bg-gradient-to-b from-[#2a2a3a] to-bg-main border-b-[3px] border-bg-dark px-5 py-3 flex items-center justify-between">
                <div className="flex items-center gap-5">
                    <h1 className="text-2xl text-wow-gold font-normal drop-shadow-md flex items-center gap-2.5">
                        <img src="/logo.png" alt="InkLab" className="w-12 h-12" />
                        InkLab
                    </h1>
                    <nav className="flex gap-1">
                        <TabButton 
                            active={activeTab === 'database'} 
                            onClick={() => setActiveTab('database')}
                        >
                            Database
                        </TabButton>
                        <TabButton 
                            active={activeTab === 'atlas'} 
                            onClick={() => setActiveTab('atlas')}
                        >
                            AtlasLoot
                        </TabButton>
                        <TabButton 
                            active={activeTab === 'favorites'} 
                            onClick={() => setActiveTab('favorites')}
                        >
                            Favorites
                        </TabButton>
                        <TabButton
                            active={activeTab === 'search'}
                            onClick={() => setActiveTab('search')}
                        >
                            Search
                        </TabButton>
                        <TabButton
                            active={activeTab === 'talents'}
                            onClick={() => setActiveTab('talents')}
                        >
                            Talents
                        </TabButton>
                        <TabButton
                            active={activeTab === 'flights'}
                            onClick={() => setActiveTab('flights')}
                        >
                            Flights
                        </TabButton>
                        <TabButton
                            active={activeTab === 'tools'}
                            onClick={() => setActiveTab('tools')}
                        >
                            Import
                        </TabButton>
                        <TabButton
                            active={activeTab === 'sync'}
                            onClick={() => setActiveTab('sync')}
                        >
                            Sync
                        </TabButton>
                    </nav>
                </div>
            </header>

            {/* Main Content */}
            <main className="flex-1 overflow-hidden">
                {activeTab === 'atlas' && <AtlasLootPage />}
                {activeTab === 'database' && (
                    <DatabasePage 
                        pendingNavigation={pendingNavigation}
                        onNavigationHandled={clearPendingNavigation}
                    />
                )}
                {activeTab === 'search' && (
                    <SearchPage 
                        onNavigate={handleSearchNavigate}
                    />
                )}
                {activeTab === 'favorites' && (
                    <FavoritesPage
                        onNavigate={handleSearchNavigate}
                    />
                )}
                {activeTab === 'talents' && <TalentsPage />}
                {activeTab === 'flights' && <FlightsPage />}
                {activeTab === 'tools' && <ToolsPage onNavigate={handleSearchNavigate} />}
                {activeTab === 'sync' && <SyncPage />}
            </main>
        </div>
    )
}

export default App
