import Link from "next/link";

export default function HomePage() {
  return (
    <section>
      <h1>Recurso Next.js Starter</h1>
      <p className="muted">
        A minimal but real SaaS starter that talks to the Recurso billing
        engine. Every Recurso call runs server-side, so your API key never
        reaches the browser.
      </p>

      <div className="notice" style={{ margin: "20px 0" }}>
        Point this app at a running Recurso stack (<code>make demo</code> in the
        main repo) and set <code>RECURSO_API_URL</code> /{" "}
        <code>RECURSO_API_KEY</code> in <code>.env.local</code>. See the README.
      </div>

      <h2>What&apos;s included</h2>
      <ul>
        <li>
          <Link href="/pricing">Pricing</Link> — lists live plans from{" "}
          <code>/v1/plans</code>
        </li>
        <li>
          <Link href="/pricing">Signup</Link> — creates a customer +
          subscription
        </li>
        <li>
          <Link href="/account">Account</Link> — subscription + usage with
          entitlement headroom
        </li>
        <li>
          <Link href="/feature">Reports</Link> — a feature-gated page using{" "}
          <code>/v1/entitlements/check</code>
        </li>
      </ul>
    </section>
  );
}
