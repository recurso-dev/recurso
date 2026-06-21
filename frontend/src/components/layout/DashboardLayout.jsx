import { Outlet, NavLink, Link, useNavigate } from 'react-router-dom'

import Sidebar from './Sidebar'
import { useAuth } from '../../auth/AuthProvider'

const DashboardLayout = () => {
    const { logout } = useAuth()
    const navigate = useNavigate()

    const handleLogout = () => {
        logout()
        navigate('/login')
    }

    return (
        <div className="relative flex h-auto min-h-screen w-full flex-col bg-background-light dark:bg-background-dark font-sans text-slate-800 dark:text-white">
            <div className="flex h-full min-h-screen w-full">
                {/* SideNavBar */}
                <Sidebar />

                {/* Main Content */}
                <main className="flex flex-1 flex-col min-w-0 bg-gray-50 dark:bg-zinc-950">
                    {/* TopNavBar */}
                    <header className="sticky top-0 z-10 flex h-16 flex-shrink-0 items-center justify-between whitespace-nowrap border-b border-gray-100 bg-white px-8 dark:border-zinc-800 dark:bg-zinc-900">
                        <div className="flex items-center gap-4">
                            {/* Breadcrumb or Page Title placeholder - can be dynamic later */}
                            {/* <h2 className="text-lg font-bold text-gray-900 dark:text-white">Dashboard</h2> */}
                            <div className="hidden md:flex text-sm text-gray-500 dark:text-zinc-400">
                                {/* We could use a hook to update this title based on route */}
                            </div>
                        </div>
                        <div className="flex flex-1 items-center justify-end gap-4">
                            <label className="relative hidden w-full max-w-xs md:block">
                                <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-zinc-500">search</span>
                                <input
                                    className="h-9 w-full rounded-md border border-gray-200 bg-gray-50 pl-10 text-gray-900 placeholder:text-gray-400 focus:border-black focus:ring-1 focus:ring-black dark:border-zinc-800 dark:bg-zinc-800/50 dark:text-white dark:placeholder:text-zinc-500 dark:focus:border-white dark:focus:ring-white transition-all outline-none"
                                    placeholder="Search..."
                                    type="search"
                                />
                            </label>
                            <Link to="/notifications" className="flex h-9 w-9 cursor-pointer items-center justify-center overflow-hidden rounded-md bg-white border border-gray-200 text-gray-500 hover:bg-gray-50 hover:text-black dark:bg-zinc-900 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-white transition-all">
                                <span className="material-symbols-outlined text-[20px]">notifications</span>
                            </Link>
                            <button onClick={handleLogout} className="flex h-9 w-9 cursor-pointer items-center justify-center overflow-hidden rounded-md bg-white border border-gray-200 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:bg-zinc-900 dark:border-zinc-800 dark:text-zinc-400 dark:hover:bg-red-900/20 dark:hover:text-red-500 transition-all" title="Logout">
                                <span className="material-symbols-outlined text-[20px]">logout</span>
                            </button>
                            <Link to="/profile">
                                <div
                                    className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10 border border-gray-200 dark:border-zinc-800"
                                    style={{ backgroundImage: "url('https://api.dicebear.com/7.x/avataaars/svg?seed=Felix')" }}
                                ></div>
                            </Link>
                        </div>
                    </header>

                    {/* Page Content */}
                    <div className="flex-1 overflow-y-auto p-4 sm:p-8">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    )
}

const NavItem = ({ to, icon, label, exact }) => (
    <NavLink
        to={to}
        end={exact}
        className={({ isActive }) => `
            flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors
            ${isActive
                ? 'bg-primary/10 text-primary dark:bg-primary/20'
                : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800'}
        `}
    >
        <span className={`material-symbols-outlined ${icon === 'home' ? 'fill' : ''}`}>{icon}</span>
        <p>{label}</p>
    </NavLink>
)

export default DashboardLayout
