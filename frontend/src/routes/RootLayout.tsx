import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import { UpdateBanner, DataStatusBanner, GlobalSearch } from '../components/ui'
import { TooltipProvider } from '../hooks/useTooltipContext'

// Shared classes mirroring the old TabButton 'tab' variant so the header nav
// keeps its look now that buttons are router <Link>s. Paddings and font size
// tighten below xl so the full nav keeps fitting as the window narrows.
const NAV_BASE =
    'px-2 xl:px-4 py-2 font-bold cursor-pointer transition-all duration-200 border ' +
    'bg-transparent border-transparent text-wow-gold uppercase text-[12px] xl:text-[13px] rounded-none hover:bg-bg-hover'
const NAV_ACTIVE = '!bg-bg-active !text-white !border-border-light'

type NavTabProps = {
    to: string
    params?: Record<string, string>
    children: React.ReactNode
}

// A header nav link styled like the old tab buttons. activeOptions exact:false
// so "Database" stays lit across all /database/* routes (tabs + detail).
function NavTab({ to, params, children }: NavTabProps) {
    return (
        <Link
            to={to}
            params={params}
            activeOptions={{ exact: false }}
            className={NAV_BASE}
            activeProps={{ className: NAV_ACTIVE }}
        >
            {children}
        </Link>
    )
}

/**
 * Root layout: the app chrome (banners, header nav, global search) plus the
 * router <Outlet>. The single TooltipProvider here gives every page a shared
 * tooltip cache and the one floating tooltip layer.
 */
export function RootLayout() {
    const navigate = useNavigate()

    return (
        <div className="flex h-screen flex-col bg-bg-dark text-white">
            <UpdateBanner />
            <DataStatusBanner onGoToTools={() => navigate({ to: '/tools' })} />

            {/* Header. Degrades in stages as the window narrows: nav paddings
                tighten (NAV_BASE), the title text drops (logo stays), the
                search shrinks — and as a last resort the nav wraps to a second
                row, so nothing ever overlaps. */}
            <header className="flex items-center justify-between gap-3 border-b-[3px] border-bg-dark bg-gradient-to-b from-[#2a2a3a] to-bg-main px-5 py-3">
                <div className="flex min-w-0 items-center gap-3 xl:gap-5">
                    <h1 className="flex shrink-0 select-none items-center gap-2.5 text-2xl font-normal text-wow-gold drop-shadow-md">
                        <img draggable={false} src="/logo.png" alt="InkLab" className="h-12 w-12" />
                        <span className="hidden lg:inline">InkLab</span>
                    </h1>
                    <nav className="flex flex-wrap gap-1">
                        <NavTab to="/database">Database</NavTab>
                        <NavTab to="/atlas">AtlasLoot</NavTab>
                        <NavTab to="/favorites">Favorites</NavTab>
                        <NavTab to="/talents">Talents</NavTab>
                        <NavTab to="/maps">Maps</NavTab>
                        <NavTab to="/timers">Timers</NavTab>
                        <NavTab to="/tools">Import</NavTab>
                        <NavTab to="/sync">Sync</NavTab>
                    </nav>
                </div>
                <GlobalSearch />
            </header>

            {/* Main content */}
            <main className="flex-1 overflow-hidden">
                <TooltipProvider>
                    <Outlet />
                </TooltipProvider>
            </main>
        </div>
    )
}

export default RootLayout
