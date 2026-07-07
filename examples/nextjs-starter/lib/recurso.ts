// Server-side Recurso API client.
//
// IMPORTANT: This module reads RECURSO_API_KEY and must only ever run on the
// server (server components, route handlers, server actions). Do not import it
// into a "use client" component — that would ship the key to the browser.
//
// We use plain `fetch` here deliberately: a typed Node SDK exists at
// `sdk/node` in the main repo, but it is not yet published to npm, so this
// starter stays dependency-free. Swap in the SDK once it ships.

const BASE_URL = process.env.RECURSO_API_URL ?? "http://localhost:8080";
const API_KEY = process.env.RECURSO_API_KEY ?? "";

export interface Price {
  id: string;
  currency: string;
  amount: number; // in the currency's lowest unit (e.g. paise for INR)
  type: string; // "recurring" | "one_time"
}

export interface Plan {
  id: string;
  name: string;
  code: string;
  interval_unit: string;
  interval_count: number;
  active: boolean;
  prices?: Price[];
}

export interface Customer {
  id: string;
  email: string;
  name: string;
}

export interface Subscription {
  id: string;
  customer_id: string;
  plan_id: string;
  status: string;
}

export interface DimensionUsage {
  dimension: string;
  period_quantity: number;
  lifetime_quantity: number;
  limit_value: number | null;
  remaining: number | null;
}

export interface SubscriptionUsage {
  subscription_id: string;
  customer_id: string;
  current_period_start: string;
  current_period_end: string;
  dimensions: DimensionUsage[];
}

export interface EntitlementCheck {
  feature_key: string;
  granted: boolean;
  limit_value: number | null;
}

class RecursoError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "RecursoError";
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${API_KEY}`,
      ...(init.headers ?? {}),
    },
    // Billing data is dynamic; never cache it.
    cache: "no-store",
  });

  if (!res.ok) {
    const body = await res.text();
    throw new RecursoError(res.status, `Recurso ${res.status}: ${body}`);
  }
  return (await res.json()) as T;
}

// GET /v1/plans — returns { data: Plan[] }
export async function listPlans(): Promise<Plan[]> {
  const json = await request<{ data: Plan[] }>("/v1/plans");
  return json.data ?? [];
}

// POST /v1/customers
export async function createCustomer(input: {
  email: string;
  name: string;
  country?: string;
}): Promise<Customer> {
  return request<Customer>("/v1/customers", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

// POST /v1/subscriptions
export async function createSubscription(input: {
  customer_id: string;
  plan_id: string;
}): Promise<Subscription> {
  return request<Subscription>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

// GET /v1/subscriptions/{id}/usage
export async function getSubscriptionUsage(
  subscriptionId: string,
): Promise<SubscriptionUsage> {
  return request<SubscriptionUsage>(
    `/v1/subscriptions/${subscriptionId}/usage`,
  );
}

// GET /v1/entitlements/check?customer_id=&feature=
export async function checkEntitlement(
  customerId: string,
  feature: string,
): Promise<EntitlementCheck> {
  const qs = new URLSearchParams({ customer_id: customerId, feature });
  return request<EntitlementCheck>(`/v1/entitlements/check?${qs.toString()}`);
}

export { RecursoError };
