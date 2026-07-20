import { useEffect, useMemo, useState } from "react";

import { endpoints } from "./api";

// useCustomers centralizes the "resolve customer ids to names" pattern used
// across list pages and create dialogs. Best-effort: on failure the caller
// keeps working with raw ids.
export function useCustomers() {
  const [customers, setCustomers] = useState([]);

  useEffect(() => {
    let active = true;
    endpoints
      .getCustomers({ limit: 1000 })
      .then((res) => {
        if (active) setCustomers(res?.data?.data || []);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, []);

  const names = useMemo(() => {
    const map = {};
    customers.forEach((c) => {
      map[c.id] = c.name;
    });
    return map;
  }, [customers]);

  return { customers, names };
}
