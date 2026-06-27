import {
    createRootRoute,
    createRoute,
    createRouter,
    createHashHistory,
    redirect,
} from '@tanstack/react-router'

import RootLayout from './routes/RootLayout'
import DatabaseTabs from './routes/DatabaseTabs'
import DatabaseDetail from './routes/DatabaseDetail'
import AtlasDetail from './routes/AtlasDetail'

import AtlasLootPage from './pages/AtlasLootPage/AtlasLootPage'
import FavoritesPage from './pages/FavoritesPage/FavoritesPage'
import TalentsPage from './pages/TalentsPage'
import MapsPage from './pages/MapsPage'
import ToolsPage from './pages/ToolsPage'
import SyncPage from './pages/SyncPage'

const rootRoute = createRootRoute({ component: RootLayout })

// "/" → default landing
const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    beforeLoad: () => {
        throw redirect({ to: '/database/$tab', params: { tab: 'items' } })
    },
})

// "/database" → first tab
const databaseIndexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/database',
    beforeLoad: () => {
        throw redirect({ to: '/database/$tab', params: { tab: 'items' } })
    },
})

// "/database/$tab" — the tabs view; detail renders in its <Outlet>
const databaseTabRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/database/$tab',
    component: DatabaseTabs,
})

// "/database/$tab/$type/$id" — entity detail overlay
const databaseDetailRoute = createRoute({
    getParentRoute: () => databaseTabRoute,
    path: '$type/$id',
    component: DatabaseDetail,
})

// "/atlas" — loot browser; detail renders in its <Outlet>
const atlasRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/atlas',
    component: AtlasLootPage,
})

const atlasDetailRoute = createRoute({
    getParentRoute: () => atlasRoute,
    path: '$type/$id',
    component: AtlasDetail,
})

const favoritesRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/favorites',
    component: FavoritesPage,
})

// Talents: class in the path (refresh-stable, Back walks between classes).
// "/talents" lands on the last-viewed class. The working build lives in session
// memory, not the URL — sharing is via the copyable build code.
const talentsIndexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/talents',
    component: TalentsPage,
})
const talentsClassRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/talents/$class',
    component: TalentsPage,
})

const mapsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/maps',
    component: MapsPage,
})
const toolsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/tools',
    component: ToolsPage,
})
const syncRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/sync',
    component: SyncPage,
})

const routeTree = rootRoute.addChildren([
    indexRoute,
    databaseIndexRoute,
    databaseTabRoute.addChildren([databaseDetailRoute]),
    atlasRoute.addChildren([atlasDetailRoute]),
    favoritesRoute,
    talentsIndexRoute,
    talentsClassRoute,
    mapsRoute,
    toolsRoute,
    syncRoute,
])

// Hash history: the Wails embedded asset server only serves "/", so keeping the
// route in the URL fragment avoids any need for SPA fallback rewriting.
export const router = createRouter({
    routeTree,
    history: createHashHistory(),
    defaultPreload: false,
})

declare module '@tanstack/react-router' {
    interface Register {
        router: typeof router
    }
}
