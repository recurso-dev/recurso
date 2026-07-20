import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "./api";

// useCustomers centralizes the "resolve customer ids to names" pattern used
// across list pages and create dialogs. Backed by react-query: every page in
// a session shares ONE cached fetch instead of refiring
// getCustomers({limit:1000}) per mount. Best-effort: on failure the caller
// keeps working with raw ids.
export function useCustomers() {
  const { data } = useQuery({
    queryKey: ["customers", "all"],
    queryFn: async () => {
      const res = await endpoints.getCustomers({ limit: 1000 });
      return res?.data?.data || [];
    },
  });
  const customers = useMemo(() => data || [], [data]);

  const names = useMemo(() => {
    const map = {};
    customers.forEach((c) => {
      map[c.id] = c.name;
    });
    return map;
  }, [customers]);

  return { customers, names };
}

// usePlans returns {plans, names} with the same shared-cache semantics.
export function usePlans() {
  const { data } = useQuery({
    queryKey: ["plans", "all"],
    queryFn: async () => {
      // The API defaults to limit=10 — ask for everything or name
      // resolution silently truncates past the first page of plans.
      const res = await endpoints.getPlans({ limit: 1000 });
      return res?.data?.data || [];
    },
  });
  const plans = useMemo(() => data || [], [data]);
  const names = useMemo(() => {
    const map = {};
    plans.forEach((p) => {
      map[p.id] = p.name;
    });
    return map;
  }, [plans]);
  return { plans, names };
}

// useSubscriptions returns the tenant's subscriptions from the shared cache.
export function useSubscriptions() {
  const { data } = useQuery({
    queryKey: ["subscriptions", "all"],
    queryFn: async () => {
      // Same limit=10 API default as plans — fetch the full set.
      const res = await endpoints.getSubscriptions({ limit: 1000 });
      return res?.data?.data || [];
    },
  });
  return useMemo(() => data || [], [data]);
}
