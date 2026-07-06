import { Link, useLocation } from "react-router-dom";
import Icon from "../ui/Icon";

const SidebarItem = ({ to, icon, label, isActive }) => {
    return (
        <Link
            to={to}
            className={`group flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-all duration-200 ${isActive
                ? "bg-gray-100 text-gray-900 font-semibold dark:bg-zinc-800 dark:text-white" // Active state
                : "text-gray-500 hover:bg-gray-50 hover:text-gray-900 dark:text-zinc-500 dark:hover:bg-zinc-800/50 dark:hover:text-zinc-300" // Inactive state
                }`}
        >
            <Icon name={icon} className="text-[20px]" filled={isActive} />
            <p>
                {label}
            </p>
        </Link>
    );
};

const Sidebar = () => {
    const location = useLocation();
    const path = location.pathname;

    return (
        <aside className="fixed inset-y-0 left-0 z-20 flex w-64 flex-col justify-between border-r border-gray-100 bg-white p-6 transition-transform lg:static lg:translate-x-0 dark:border-zinc-800 dark:bg-zinc-900">
            <div className="flex flex-col gap-6">
                {/* Brand */}
                <div className="flex items-center gap-3 px-2">
                    <div className="flex size-8 items-center justify-center rounded-lg bg-black text-white dark:bg-white dark:text-black">
                        <Icon name="layers" className="text-lg" />
                    </div>
                    <div className="flex flex-col">
                        <h1 className="text-base font-bold text-gray-900 dark:text-white tracking-tight">
                            Recurso
                        </h1>
                    </div>
                </div>

                {/* Navigation */}
                <nav className="flex flex-col gap-1.5">
                    <SidebarItem
                        to="/"
                        icon="home"
                        label="Home"
                        isActive={path === "/"}
                    />
                    <div className="pt-4 pb-2">
                        <p className="px-3 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-zinc-500">
                            Core
                        </p>
                    </div>
                    <SidebarItem
                        to="/customers"
                        icon="group"
                        label="Customers"
                        isActive={path.startsWith("/customers")}
                    />
                    <SidebarItem
                        to="/products"
                        icon="inventory_2"
                        label="Products"
                        isActive={path.startsWith("/products")}
                    />
                    <SidebarItem
                        to="/plans"
                        icon="layers"
                        label="Plans"
                        isActive={path.startsWith("/plans")}
                    />
                    <SidebarItem
                        to="/coupons"
                        icon="local_offer"
                        label="Coupons"
                        isActive={path.startsWith("/coupons")}
                    />
                    <SidebarItem
                        to="/subscriptions"
                        icon="autorenew"
                        label="Subscriptions"
                        isActive={path.startsWith("/subscriptions")}
                    />
                    <SidebarItem
                        to="/invoices"
                        icon="receipt_long"
                        label="Invoices"
                        isActive={path.startsWith("/invoices")}
                    />
                    <SidebarItem
                        to="/credit-notes"
                        icon="description"
                        label="Credit Notes"
                        isActive={path.startsWith("/credit-notes")}
                    />
                    <SidebarItem
                        to="/quotes"
                        icon="request_quote"
                        label="Quotes"
                        isActive={path.startsWith("/quotes")}
                    />

                    <div className="pt-4 pb-2">
                        <p className="px-3 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-zinc-500">
                            Growth
                        </p>
                    </div>
                    <SidebarItem
                        to="/referrals"
                        icon="campaign"
                        label="Referrals"
                        isActive={path.startsWith("/referrals")}
                    />
                    <SidebarItem
                        to="/gifts"
                        icon="redeem"
                        label="Gifts"
                        isActive={path.startsWith("/gifts")}
                    />

                    <div className="pt-4 pb-2">
                        <p className="px-3 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-zinc-500">
                            Finance
                        </p>
                    </div>
                    <SidebarItem
                        to="/ledger"
                        icon="account_balance"
                        label="Financials"
                        isActive={path.startsWith("/ledger")}
                    />
                    <SidebarItem
                        to="/finance/reconciliation"
                        icon="fact_check"
                        label="Reconciliation"
                        isActive={path.startsWith("/finance/reconciliation")}
                    />
                    <SidebarItem
                        to="/usage"
                        icon="bar_chart"
                        label="Usage"
                        isActive={path.startsWith("/usage")}
                    />
                    <SidebarItem
                        to="/dunning"
                        icon="psychology"
                        label="Smart Dunning"
                        isActive={path.startsWith("/dunning")}
                    />

                    <div className="pt-4 pb-2">
                        <p className="px-3 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-zinc-500">
                            System
                        </p>
                    </div>
                    <SidebarItem
                        to="/developers"
                        icon="code"
                        label="Developers"
                        isActive={path.startsWith("/developers")}
                    />
                </nav>
            </div>

            <div className="flex flex-col gap-2 border-t border-gray-100 pt-6 dark:border-zinc-800">
                <SidebarItem
                    to="/settings"
                    icon="settings"
                    label="Settings"
                    isActive={path.startsWith("/settings")}
                />
            </div>
        </aside>
    );
};

export default Sidebar;
