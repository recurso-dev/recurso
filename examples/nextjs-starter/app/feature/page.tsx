import Link from "next/link";
import { cookies } from "next/headers";
import { checkEntitlement, type EntitlementCheck } from "@/lib/recurso";

export const dynamic = "force-dynamic";

// The feature key this page is gated on. Grant it to a plan with
// PUT /v1/plans/{id}/entitlements to unlock the page for subscribers.
const FEATURE_KEY = "advanced_reports";

export default async function FeaturePage() {
  const customerId = (await cookies()).get("recurso_customer_id")?.value;

  if (!customerId) {
    return (
      <section>
        <h1>Advanced Reports</h1>
        <div className="notice">
          No customer on this session. <Link href="/pricing">Sign up</Link>{" "}
          first.
        </div>
      </section>
    );
  }

  let check: EntitlementCheck | null = null;
  let error: string | null = null;
  try {
    check = await checkEntitlement(customerId, FEATURE_KEY);
  } catch (e) {
    error = e instanceof Error ? e.message : "entitlement check failed";
  }

  if (error || !check) {
    return (
      <section>
        <h1>Advanced Reports</h1>
        <div className="notice" style={{ borderColor: "#b42318" }}>
          Could not check entitlement: {error}
        </div>
      </section>
    );
  }

  return (
    <section>
      <h1>Advanced Reports</h1>
      <p className="muted">
        Gated on feature <code>{FEATURE_KEY}</code> via{" "}
        <code>GET /v1/entitlements/check</code>.
      </p>

      {check.granted ? (
        <>
          <p>
            <span className="badge ok">Unlocked</span>
          </p>
          <div className="card">
            <h3>📈 Your advanced analytics</h3>
            <p className="muted">
              Real content would render here. This customer&apos;s plan grants{" "}
              <code>{FEATURE_KEY}</code>
              {check.limit_value !== null && (
                <> (limit: {check.limit_value.toLocaleString()})</>
              )}
              .
            </p>
          </div>
        </>
      ) : (
        <>
          <p>
            <span className="badge locked">Locked</span>
          </p>
          <div className="notice">
            This customer&apos;s current plan does not include{" "}
            <code>{FEATURE_KEY}</code>. Grant it to their plan with{" "}
            <code>PUT /v1/plans/{"{id}"}/entitlements</code> (or upgrade them to
            a plan that has it), then reload.
          </div>
        </>
      )}
    </section>
  );
}
