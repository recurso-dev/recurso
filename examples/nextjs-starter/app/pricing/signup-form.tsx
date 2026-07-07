import type { Plan } from "@/lib/recurso";

// A plain HTML form that posts to the /api/signup route handler. No client
// JavaScript required — the handler creates the customer + subscription on the
// server and redirects to /account.
export default function SignupForm({ plans }: { plans: Plan[] }) {
  return (
    <form action="/api/signup" method="post" className="card">
      <div className="field">
        <label htmlFor="name">Name</label>
        <input id="name" name="name" required placeholder="Ada Lovelace" />
      </div>
      <div className="field">
        <label htmlFor="email">Email</label>
        <input
          id="email"
          name="email"
          type="email"
          required
          placeholder="ada@example.com"
        />
      </div>
      <div className="field">
        <label htmlFor="plan_id">Plan</label>
        <select id="plan_id" name="plan_id" required>
          {plans.map((plan) => (
            <option key={plan.id} value={plan.id}>
              {plan.name}
            </option>
          ))}
        </select>
      </div>
      <button className="btn" type="submit">
        Create account &amp; subscribe
      </button>
    </form>
  );
}
