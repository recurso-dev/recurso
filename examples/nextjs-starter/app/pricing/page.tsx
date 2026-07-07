import { listPlans, type Plan } from "@/lib/recurso";
import { formatMoney } from "@/lib/format";
import SignupForm from "./signup-form";

// Always render fresh — plans are fetched from the API on each request.
export const dynamic = "force-dynamic";

function primaryPrice(plan: Plan) {
  if (!plan.prices || plan.prices.length === 0) return null;
  return (
    plan.prices.find((p) => p.type === "recurring") ?? plan.prices[0]
  );
}

export default async function PricingPage() {
  let plans: Plan[] = [];
  let error: string | null = null;

  try {
    plans = await listPlans();
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load plans";
  }

  return (
    <section>
      <h1>Pricing</h1>
      <p className="muted">
        Plans are fetched live from <code>GET /v1/plans</code>.
      </p>

      {error && (
        <div className="notice" style={{ borderColor: "#b42318" }}>
          Could not reach Recurso: {error}
          <br />
          Is the stack running (<code>make demo</code>) and are{" "}
          <code>RECURSO_API_URL</code> / <code>RECURSO_API_KEY</code> set?
        </div>
      )}

      {plans.length > 0 && (
        <>
          <div className="grid" style={{ margin: "20px 0" }}>
            {plans.map((plan) => {
              const price = primaryPrice(plan);
              return (
                <div key={plan.id} className="card">
                  <h3 style={{ margin: 0 }}>{plan.name}</h3>
                  <div className="price">
                    {price
                      ? formatMoney(price.amount, price.currency)
                      : "Custom"}
                  </div>
                  <div className="muted">
                    per {plan.interval_count > 1 ? plan.interval_count : ""}{" "}
                    {plan.interval_unit}
                  </div>
                </div>
              );
            })}
          </div>

          <h2>Start a subscription</h2>
          <p className="muted">
            A fake signup: creates a customer (<code>POST /v1/customers</code>)
            then subscribes them (<code>POST /v1/subscriptions</code>).
          </p>
          <SignupForm plans={plans} />
        </>
      )}
    </section>
  );
}
