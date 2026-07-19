import ErrorBoundary from './components/ErrorBoundary'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import Dashboard from './pages/Dashboard'
import Customers from './pages/Customers'
import CreateCustomer from './pages/CreateCustomer'
import Plans from './pages/Plans'
import CreatePlan from './pages/CreatePlan'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import AcceptInvite from './pages/AcceptInvite'
import Security from './pages/Security'
import Subscriptions from './pages/Subscriptions'
import CreateSubscription from './pages/CreateSubscription'
import Invoices from './pages/Invoices'
import Products from './pages/Products'
import Coupons from './pages/Coupons'
import Metering from './pages/Metering'
import Wallets from './pages/Wallets'
import AuditLog from './pages/AuditLog'
import CreateCoupon from './pages/CreateCoupon'
import Usage from './pages/Usage'
import Developers from './pages/Developers'
import Integrations from './pages/Integrations'
import Ledger from './pages/Ledger'
import CreditNotes from './pages/CreditNotes'
import CreateCreditNote from './pages/CreateCreditNote'
import Settings from './pages/Settings'
import Team from './pages/Team'
import Notifications from './pages/Notifications'
import Profile from './pages/Profile'
import Referrals from './pages/Referrals'
import Gifts from './pages/Gifts'
import DashboardLayout from "./components/layout/DashboardLayout"
import { useAuth } from './auth/AuthProvider'

// Customer Portal Pages
import PortalLogin from './pages/portal/PortalLogin'
import PortalDashboard from './pages/portal/PortalDashboard'
import PortalVerify from './pages/portal/PortalVerify'
import PortalRedeem from './pages/portal/PortalRedeem'

// Quotes
import Quotes from './pages/Quotes'
import CreateQuote from './pages/CreateQuote'

// Checkout (public)
import Checkout from './pages/Checkout'

// Smart Dunning
import DunningDashboard from './pages/DunningDashboard'
import DunningCampaigns from './pages/DunningCampaigns'
import CancelFlows from './pages/CancelFlows'
import Churn from './pages/Churn'
import Mandates from './pages/Mandates'
import Disputes from './pages/Disputes'
import OfflinePayments from './pages/OfflinePayments'

// Finance
import FinanceReconciliation from './pages/FinanceReconciliation'
import RevenueRecognition from './pages/RevenueRecognition'
import RevenueWaterfall from './pages/RevenueWaterfall'
import TrialBalance from './pages/TrialBalance'
import MRRWaterfall from './pages/MRRWaterfall'
import InvoiceAging from './pages/InvoiceAging'
import UnitEconomics from './pages/UnitEconomics'
import ExecutiveSummary from './pages/ExecutiveSummary'
import RevenueByPlan from './pages/RevenueByPlan'
import RevenueByGeography from './pages/RevenueByGeography'

// Settings
import IRPSettings from './pages/settings/IRPSettings'
import GSTSettings from './pages/settings/GSTSettings'
import TaxNexusSettings from './pages/settings/TaxNexusSettings'

// Multi-tenant + tax + GenAI
import Organizations from './pages/Organizations'
import GSTReturns from './pages/GSTReturns'
import AskAnalytics from './pages/AskAnalytics'

const PrivateRoute = () => {
    const { isAuthenticated, loading } = useAuth();

    if (loading) {
        return (
            <div className="flex h-screen w-full items-center justify-center bg-gray-50 dark:bg-stone-950">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
            </div>
        )
    }

    return isAuthenticated ? <Outlet /> : <Navigate to="/login" />;
};

function App() {
    const { isAuthenticated } = useAuth();

    return (
        <div className="bg-background-dark min-h-screen">
            <ErrorBoundary>
                <Routes>
                    <Route path="/login" element={!isAuthenticated ? <Login /> : <Navigate to="/" />} />
                    <Route path="/register" element={!isAuthenticated ? <Register /> : <Navigate to="/" />} /> {/* Added Register Route */}
                    <Route path="/forgot-password" element={!isAuthenticated ? <ForgotPassword /> : <Navigate to="/" />} />
                    <Route path="/reset-password" element={<ResetPassword />} />
                    <Route path="/accept-invite" element={<AcceptInvite />} />

                    {/* Hosted Checkout (public) */}
                    <Route path="/checkout/:id" element={<Checkout />} />

                    {/* Customer Portal Routes (public) */}
                    <Route path="/portal/login" element={<PortalLogin />} />
                    <Route path="/portal/verify" element={<PortalVerify />} />
                    <Route path="/portal/dashboard" element={<PortalDashboard />} />
                    <Route path="/portal/redeem" element={<PortalRedeem />} />

                    {/* Protected Routes */}
                    <Route element={<PrivateRoute />}>
                        <Route element={<DashboardLayout />}>
                            <Route path="/" element={<Dashboard />} />
                            <Route path="/customers" element={<Customers />} />
                            <Route path="/customers/new" element={<CreateCustomer />} />
                            <Route path="/plans" element={<Plans />} />
                            <Route path="/plans/new" element={<CreatePlan />} />
                            <Route path="/subscriptions" element={<Subscriptions />} />
                            <Route path="/subscriptions/new" element={<CreateSubscription />} />
                            <Route path="/invoices" element={<Invoices />} />
                            <Route path="/products" element={<Products />} />
                            <Route path="/coupons" element={<Coupons />} />
                            <Route path="/coupons/new" element={<CreateCoupon />} />
                            <Route path="/usage" element={<Usage />} />
                            <Route path="/developers" element={<Developers />} />
                            <Route path="/integrations" element={<Integrations />} />
                            <Route path="/ledger" element={<Ledger />} />
                            <Route path="/metering" element={<Metering />} />
                            <Route path="/wallets" element={<Wallets />} />
                            <Route path="/audit-log" element={<AuditLog />} />
                            <Route path="/finance/reconciliation" element={<FinanceReconciliation />} />
                            <Route path="/finance/trial-balance" element={<TrialBalance />} />
                            <Route path="/finance/revenue-recognition" element={<RevenueRecognition />} />
                            <Route path="/finance/revenue-waterfall" element={<RevenueWaterfall />} />
                            <Route path="/finance/mrr-waterfall" element={<MRRWaterfall />} />
                            <Route path="/finance/invoice-aging" element={<InvoiceAging />} />
                            <Route path="/finance/unit-economics" element={<UnitEconomics />} />
                            <Route path="/overview" element={<ExecutiveSummary />} />
                            <Route path="/finance/revenue-by-plan" element={<RevenueByPlan />} />
                            <Route path="/finance/revenue-by-geography" element={<RevenueByGeography />} />
                            <Route path="/credit-notes" element={<CreditNotes />} />
                            <Route path="/credit-notes/new" element={<CreateCreditNote />} />
                            <Route path="/quotes" element={<Quotes />} />
                            <Route path="/quotes/new" element={<CreateQuote />} />
                            <Route path="/settings" element={<Settings />} />
                            <Route path="/security" element={<Security />} />
                            <Route path="/team" element={<Team />} />
                            <Route path="/notifications" element={<Notifications />} />
                            <Route path="/profile" element={<Profile />} />
                            <Route path="/referrals" element={<Referrals />} />
                            <Route path="/gifts" element={<Gifts />} />
                            <Route path="/dunning" element={<DunningDashboard />} />
                            <Route path="/dunning/campaigns" element={<DunningCampaigns />} />
                            <Route path="/cancel-flows" element={<CancelFlows />} />
                            <Route path="/churn" element={<Churn />} />
                            <Route path="/mandates" element={<Mandates />} />
                            <Route path="/disputes" element={<Disputes />} />
                            <Route path="/payments/offline" element={<OfflinePayments />} />
                            <Route path="/settings/irp" element={<IRPSettings />} />
                            <Route path="/settings/gst" element={<GSTSettings />} />
                            <Route path="/settings/tax-nexus" element={<TaxNexusSettings />} />
                            <Route path="/organizations" element={<Organizations />} />
                            <Route path="/finance/gst-returns" element={<GSTReturns />} />
                            <Route path="/ask" element={<AskAnalytics />} />
                        </Route>
                    </Route>

                    {/* Fallback */}
                    <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
            </ErrorBoundary>
        </div>
    );
}

export default App
