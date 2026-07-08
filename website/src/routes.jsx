import App from './App'
import VsPage from './pages/VsPage'
import { competitorList } from './data/competitors'

export const routes = [
    {
        path: '/',
        element: <App />,
    },
    ...competitorList.map((data) => ({
        path: `/vs/${data.slug}`,
        element: <VsPage data={data} />,
    })),
]
