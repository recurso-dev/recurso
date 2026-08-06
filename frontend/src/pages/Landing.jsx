import { Link } from "react-router";
import {
  Layers,
  Scale,
  Search,
  RotateCcw,
  Landmark,
  ArrowRight,
  ArrowUpRight,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

// The signed-out front door for the app. Same design language as Login/Register
// (warm-stone canvas, one emerald accent, Inter, calm/enterprise) — a hero that
// states the accounting-first promise, then concrete proof, then the CTAs again.
const PROOF = [
  {
    icon: Scale,
    title: "A real double-entry ledger",
    body: "Every invoice, payment, and refund posts balanced legs. Your trial balance always ties.",
  },
  {
    icon: Search,
    title: "Explain any number",
    body: "Drill from any figure straight to its exact postings and the source document behind it.",
  },
  {
    icon: RotateCcw,
    title: "Reversible by design",
    body: "Refunds, write-offs, disputes, chargebacks — each a first-class, auditable event.",
  },
  {
    icon: Landmark,
    title: "Tax & revenue recognition",
    body: "GST, VAT, and US nexus, with cash or accrual (ASC 606) recognition — built in, not bolted on.",
  },
];

const CAPABILITIES = [
  "Subscriptions",
  "Invoicing",
  "Payments",
  "Dunning",
  "Wallets",
  "Quotes",
  "Tax",
  "Revenue recognition",
  "Reconciliation",
  "Multi-entity",
];

export default function Landing() {
  return (
    <div className="flex min-h-screen flex-col bg-stone-50 text-foreground">
      {/* Top bar */}
      <header className="border-b border-border/70 bg-background/70 backdrop-blur">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
              <Layers className="h-4 w-4" aria-hidden="true" />
            </span>
            <span className="text-base font-semibold tracking-tight">Recurso</span>
          </div>
          <nav className="flex items-center gap-2">
            <Button variant="ghost" size="sm" asChild>
              <Link to="/login">Log in</Link>
            </Button>
            <Button size="sm" asChild>
              <Link to="/register">Create account</Link>
            </Button>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <main className="flex-1">
        <section className="mx-auto w-full max-w-6xl px-4 pt-20 pb-16 sm:px-6 sm:pt-28">
          <div className="mx-auto max-w-3xl text-center">
            <p className="text-xs font-semibold uppercase tracking-wider text-emerald-700">
              Accounting-first subscription billing
            </p>
            <h1 className="mt-4 text-balance text-4xl font-semibold tracking-tight sm:text-5xl">
              Billing your accountant can trust.
            </h1>
            <p className="mx-auto mt-5 max-w-2xl text-pretty text-lg text-muted-foreground">
              Recurso runs subscriptions, payments, tax, and revenue recognition on a real
              double-entry ledger — so every number is explainable, every event is
              reversible, and the books always reconcile.
            </p>
            <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Button size="lg" asChild>
                <Link to="/register">
                  Create your workspace
                  <ArrowRight className="h-4 w-4" aria-hidden="true" />
                </Link>
              </Button>
              <Button size="lg" variant="outline" asChild>
                <Link to="/login">Log in</Link>
              </Button>
            </div>
            <p className="mt-4 text-sm text-muted-foreground">
              Set up in minutes — connect Stripe, Razorpay, ACH, or GoCardless.
            </p>
          </div>

          {/* Capability strip */}
          <ul className="mx-auto mt-14 flex max-w-4xl flex-wrap items-center justify-center gap-x-3 gap-y-2">
            {CAPABILITIES.map((c) => (
              <li
                key={c}
                className="rounded-full border border-border bg-background px-3 py-1 text-xs font-medium text-muted-foreground"
              >
                {c}
              </li>
            ))}
          </ul>
        </section>

        {/* Proof */}
        <section className="mx-auto w-full max-w-6xl px-4 pb-24 sm:px-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {PROOF.map(({ icon: Icon, title, body }) => (
              <Card key={title}>
                <CardContent className="p-5">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-muted">
                    <Icon className="h-5 w-5 text-emerald-700" aria-hidden="true" />
                  </span>
                  <h2 className="mt-4 text-sm font-semibold text-foreground">{title}</h2>
                  <p className="mt-1.5 text-sm text-muted-foreground">{body}</p>
                </CardContent>
              </Card>
            ))}
          </div>

          {/* Closing line */}
          <div className="mx-auto mt-16 max-w-2xl text-center">
            <p className="text-pretty text-base text-muted-foreground">
              The through-line: commercial operations you run day to day, backed by
              financial proof an auditor would accept.
            </p>
            <div className="mt-6">
              <Button asChild>
                <Link to="/register">
                  Get started
                  <ArrowRight className="h-4 w-4" aria-hidden="true" />
                </Link>
              </Button>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-border">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-center justify-between gap-3 px-4 py-6 text-sm text-muted-foreground sm:flex-row sm:px-6">
          <span>© {new Date().getFullYear()} Recurso</span>
          <div className="flex items-center gap-5">
            <a
              href="https://docs.recurso.dev"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 transition-colors hover:text-foreground"
            >
              Documentation
              <ArrowUpRight className="h-3.5 w-3.5" aria-hidden="true" />
            </a>
            <Link to="/login" className="transition-colors hover:text-foreground">
              Log in
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
