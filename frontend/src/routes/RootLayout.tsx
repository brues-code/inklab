import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import { UpdateBanner, DataStatusBanner, GlobalSearch } from '../components/ui'
import { TooltipProvider } from '../hooks/useTooltipContext'

// Shared classes mirroring the old TabButton 'tab' variant so the header nav
// keeps its look now that buttons are router <Link>s.
const NAV_BASE =
    'px-4 py-2 font-bold text-sm cursor-pointer transition-all duration-200 border ' +
    'bg-transparent border-transparent text-wow-gold uppercase text-[13px] rounded-none hover:bg-bg-hover'
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
        <Link to={to} params={params} activeOptions={{ exact: false }} className={NAV_BASE} activeProps={{ className: NAV_ACTIVE }}>
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
        <div className="h-screen flex flex-col bg-bg-dark text-white">
            <UpdateBanner />
            <DataStatusBanner onGoToTools={() => navigate({ to: '/tools' })} />

            {/* Header */}
            <header className="bg-gradient-to-b from-[#2a2a3a] to-bg-main border-b-[3px] border-bg-dark px-5 py-3 flex items-center justify-between">
                <div className="flex items-center gap-5">
                    <h1 className="text-2xl text-wow-gold font-normal drop-shadow-md flex items-center gap-2.5">
                        <img src="/logo.png" alt="InkLab" className="w-12 h-12" />
                        InkLab
                    </h1>
                    <nav className="flex gap-1">
                        <NavTab to="/database">Database</NavTab>
                        <NavTab to="/atlas">AtlasLoot</NavTab>
                        <NavTab to="/favorites">Favorites</NavTab>
                        <NavTab to="/talents">Talents</NavTab>
                        <NavTab to="/maps">Maps</NavTab>
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
