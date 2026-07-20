import ErrorBoundary from './components/ErrorBoundary'
import { Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import Login from './pages/Login'
import DashboardLayout from "./components/layout/DashboardLayout"
import { useAuth } from './auth/AuthProvider'

// Customer Portal Pages

// Quotes

// Checkout (public)

// Smart Dunning

// Finance

// Settings

// Multi-tenant + tax + GenAI

const Dashboard = lazy(() => import('./pages/Dashboard'))
// Route-level code splitting: every page beyond the login/home critical path
// loads on demand, so the entry chunk stays small and Tremor/recharts land in
// route chunks. All pages are default exports, which lazy() consumes directly.
const Customers = lazy(() => import('./pages/Customers'))
const CreateCustomer = lazy(() => import('./pages/CreateCustomer'))
const Plans = lazy(() => import('./pages/Plans'))
const CreatePlan = lazy(() => import('./pages/CreatePlan'))
const Register = lazy(() => import('./pages/Register'))
const ForgotPassword = lazy(() => import('./pages/ForgotPassword'))
const ResetPassword = lazy(() => import('./pages/ResetPassword'))
const AcceptInvite = lazy(() => import('./pages/AcceptInvite'))
const Security = lazy(() => import('./pages/Security'))
const Subscriptions = lazy(() => import('./pages/Subscriptions'))
const CreateSubscription = lazy(() => import('./pages/CreateSubscription'))
const Invoices = lazy(() => import('./pages/Invoices'))
const Coupons = lazy(() => import('./pages/Coupons'))
const Metering = lazy(() => import('./pages/Metering'))
const Wallets = lazy(() => import('./pages/Wallets'))
const AuditLog = lazy(() => import('./pages/AuditLog'))
const CreateCoupon = lazy(() => import('./pages/CreateCoupon'))
const Usage = lazy(() => import('./pages/Usage'))
const Developers = lazy(() => import('./pages/Developers'))
const Integrations = lazy(() => import('./pages/Integrations'))
const Ledger = lazy(() => import('./pages/Ledger'))
const CreditNotes = lazy(() => import('./pages/CreditNotes'))
const CreateCreditNote = lazy(() => import('./pages/CreateCreditNote'))
const Settings = lazy(() => import('./pages/Settings'))
const Team = lazy(() => import('./pages/Team'))
const Notifications = lazy(() => import('./pages/Notifications'))
const Profile = lazy(() => import('./pages/Profile'))
const Referrals = lazy(() => import('./pages/Referrals'))
const Gifts = lazy(() => import('./pages/Gifts'))
const PortalLogin = lazy(() => import('./pages/portal/PortalLogin'))
const PortalDashboard = lazy(() => import('./pages/portal/PortalDashboard'))
const PortalVerify = lazy(() => import('./pages/portal/PortalVerify'))
const PortalRedeem = lazy(() => import('./pages/portal/PortalRedeem'))
const Quotes = lazy(() => import('./pages/Quotes'))
const CreateQuote = lazy(() => import('./pages/CreateQuote'))
const Checkout = lazy(() => import('./pages/Checkout'))
const DunningDashboard = lazy(() => import('./pages/DunningDashboard'))
const DunningCampaigns = lazy(() => import('./pages/DunningCampaigns'))
const CancelFlows = lazy(() => import('./pages/CancelFlows'))
const Churn = lazy(() => import('./pages/Churn'))
const Mandates = lazy(() => import('./pages/Mandates'))
const Disputes = lazy(() => import('./pages/Disputes'))
const OfflinePayments = lazy(() => import('./pages/OfflinePayments'))
const FinanceReconciliation = lazy(() => import('./pages/FinanceReconciliation'))
const RevenueRecognition = lazy(() => import('./pages/RevenueRecognition'))
const RevenueWaterfall = lazy(() => import('./pages/RevenueWaterfall'))
const TrialBalance = lazy(() => import('./pages/TrialBalance'))
const MRRWaterfall = lazy(() => import('./pages/MRRWaterfall'))
const InvoiceAging = lazy(() => import('./pages/InvoiceAging'))
const UnitEconomics = lazy(() => import('./pages/UnitEconomics'))
const ExecutiveSummary = lazy(() => import('./pages/ExecutiveSummary'))
const RevenueByPlan = lazy(() => import('./pages/RevenueByPlan'))
const RevenueByGeography = lazy(() => import('./pages/RevenueByGeography'))
const IRPSettings = lazy(() => import('./pages/settings/IRPSettings'))
const GSTSettings = lazy(() => import('./pages/settings/GSTSettings'))
const TaxNexusSettings = lazy(() => import('./pages/settings/TaxNexusSettings'))
const Organizations = lazy(() => import('./pages/Organizations'))
const GSTReturns = lazy(() => import('./pages/GSTReturns'))
const AskAnalytics = lazy(() => import('./pages/AskAnalytics'))

const PageFallback = () => (
    <div className="flex h-full min-h-[40vh] w-full items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    </div>
)

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
                <Suspense fallback={<PageFallback />}>
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
                            <Route path="/products" element={<Navigate to="/plans" replace />} />
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
                </Suspense>
            </ErrorBoundary>
        </div>
    );
}

export default App
