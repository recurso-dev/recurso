import App from './App'
import VsPage from './pages/VsPage'
import PricingPage from './pages/PricingPage'
import { competitorList } from './data/competitors'

export const routes = [
    {
        path: '/',
        element: <App />,
    },
    {
        path: '/pricing',
        element: <PricingPage />,
    },
    ...competitorList.map((data) => ({
        path: `/vs/${data.slug}`,
        element: <VsPage data={data} />,
    })),
]
