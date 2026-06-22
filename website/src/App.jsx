import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Features from './components/Features'
import CodeExample from './components/CodeExample'
import UseCases from './components/UseCases'
import Comparison from './components/Comparison'
import Playground from './components/Playground'
import Docs from './components/Docs'
import Pricing from './components/Pricing'
import CTA from './components/CTA'
import Footer from './components/Footer'

function App() {
    return (
        <div className="grain min-h-screen bg-[#050505]">
            <Navbar />
            <main>
                <Hero />
                <Features />
                <Playground />
                <CodeExample />
                <UseCases />
                <Comparison />
                <Docs />
                <Pricing />
                <CTA />
            </main>
            <Footer />
        </div>
    )
}

export default App
