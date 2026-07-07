import { NextRequest, NextResponse } from "next/server";
import { createCustomer, createSubscription } from "@/lib/recurso";

// POST /api/signup — the "fake signup" backend.
//
// Runs entirely on the server: creates a Recurso customer, subscribes them to
// the chosen plan, then stores the resulting ids in httpOnly cookies so the
// /account and /feature pages can identify the current user. A real app would
// tie these ids to its own authenticated user record instead.
export async function POST(req: NextRequest) {
  const form = await req.formData();
  const name = String(form.get("name") ?? "").trim();
  const email = String(form.get("email") ?? "").trim();
  const planId = String(form.get("plan_id") ?? "").trim();

  if (!name || !email || !planId) {
    return NextResponse.json(
      { error: "name, email and plan_id are required" },
      { status: 400 },
    );
  }

  try {
    const customer = await createCustomer({ email, name, country: "IN" });
    const subscription = await createSubscription({
      customer_id: customer.id,
      plan_id: planId,
    });

    const res = NextResponse.redirect(new URL("/account", req.url), 303);
    const cookieOpts = { httpOnly: true, sameSite: "lax" as const, path: "/" };
    res.cookies.set("recurso_customer_id", customer.id, cookieOpts);
    res.cookies.set("recurso_subscription_id", subscription.id, cookieOpts);
    return res;
  } catch (e) {
    const message = e instanceof Error ? e.message : "signup failed";
    return NextResponse.json({ error: message }, { status: 502 });
  }
}
