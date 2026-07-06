import AnnouncementBar from './components/AnnouncementBar'
import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Products from './components/Products'
import CodeSection from './components/CodeSection'
import India from './components/India'
import Comparison from './components/Comparison'
import OpenSource from './components/OpenSource'
import Pricing from './components/Pricing'
import CTA from './components/CTA'
import Footer from './components/Footer'

function App() {
    return (
        <div className="min-h-screen bg-surface">
            <AnnouncementBar />
            <Navbar />
            <main>
                <Hero />
                <Products />
                <CodeSection />
                <India />
                <Comparison />
                <OpenSource />
                <Pricing />
                <CTA />
            </main>
            <Footer />
        </div>
    )
}

export default App
