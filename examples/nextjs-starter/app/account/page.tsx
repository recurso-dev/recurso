import Link from "next/link";
import { cookies } from "next/headers";
import { getSubscriptionUsage, type SubscriptionUsage } from "@/lib/recurso";

export const dynamic = "force-dynamic";

function EmptyState() {
  return (
    <section>
      <h1>Account</h1>
      <div className="notice">
        No subscription on this session yet. Head to{" "}
        <Link href="/pricing">Pricing</Link> and complete the signup flow first.
      </div>
    </section>
  );
}

export default async function AccountPage() {
  const subscriptionId = cookies().get("recurso_subscription_id")?.value;
  if (!subscriptionId) return <EmptyState />;

  let usage: SubscriptionUsage | null = null;
  let error: string | null = null;
  try {
    usage = await getSubscriptionUsage(subscriptionId);
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load usage";
  }

  if (error || !usage) {
    return (
      <section>
        <h1>Account</h1>
        <div className="notice" style={{ borderColor: "#b42318" }}>
          Could not load usage: {error}
        </div>
      </section>
    );
  }

  const period = `${new Date(
    usage.current_period_start,
  ).toLocaleDateString()} – ${new Date(
    usage.current_period_end,
  ).toLocaleDateString()}`;

  return (
    <section>
      <h1>Account</h1>
      <p className="muted">
        Subscription <code>{usage.subscription_id}</code>
      </p>
      <p>
        <strong>Current billing period:</strong> {period}
      </p>

      <h2>Usage &amp; entitlement headroom</h2>
      <p className="muted">
        From <code>GET /v1/subscriptions/{"{id}"}/usage</code>. Limits are
        joined in from the customer&apos;s entitlements; &quot;remaining&quot;
        is the headroom left this period.
      </p>

      {usage.dimensions.length === 0 ? (
        <div className="notice">
          No usage recorded for this subscription yet. Report some with{" "}
          <code>POST /v1/usage/events</code> and refresh.
        </div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Dimension</th>
              <th>This period</th>
              <th>Lifetime</th>
              <th>Limit</th>
              <th>Remaining</th>
            </tr>
          </thead>
          <tbody>
            {usage.dimensions.map((d) => (
              <tr key={d.dimension}>
                <td>{d.dimension}</td>
                <td>{d.period_quantity.toLocaleString()}</td>
                <td>{d.lifetime_quantity.toLocaleString()}</td>
                <td>
                  {d.limit_value === null
                    ? "unlimited"
                    : d.limit_value.toLocaleString()}
                </td>
                <td>
                  {d.remaining === null ? (
                    "—"
                  ) : (
                    <span
                      className={`badge ${d.remaining >= 0 ? "ok" : "locked"}`}
                    >
                      {d.remaining.toLocaleString()}
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <p style={{ marginTop: 24 }}>
        <Link href="/feature">Try a feature-gated page →</Link>
      </p>
    </section>
  );
}
