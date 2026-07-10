import { render, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, vi, afterEach } from "vitest";

import { AuthProvider } from "../../auth/AuthProvider";
import { ToastProvider } from "../../components/Toast";

// jsdom in this config doesn't expose localStorage; AuthProvider reads it on init.
const store = {};
vi.stubGlobal("localStorage", {
  getItem: (k) => (k in store ? store[k] : null),
  setItem: (k, v) => { store[k] = String(v); },
  removeItem: (k) => { delete store[k]; },
  clear: () => { for (const k in store) delete store[k]; },
});

// Every page renders behind a Proxy'd api client whose every method returns a
// promise that never settles — so pages mount, fire their effects, and show
// loading states without a real network call. This catches the class of bug a
// build can't: a missing import or an undefined reference that only throws when
// the component actually renders (e.g. the ConfirmDialog/setPmError crashes
// introduced during the dashboard migration). Robustness guardrail — every
// route is proven to at least mount.

vi.mock("../../lib/api", () => {
  const pending = () => new Promise(() => {});
  const apiProxy = new Proxy(
    {},
    {
      get: (_t, prop) => {
        // Axios-instance shape used by a few pages (api.get/post/...).
        if (["get", "post", "put", "patch", "delete"].includes(prop)) return pending;
        // endpoints.* — any accessed name is a callable returning a pending promise.
        return () => pending();
      },
    }
  );
  return {
    __esModule: true,
    default: apiProxy,
    endpoints: apiProxy,
    API_BASE: "/v1",
    API_ROOT: "",
  };
});

// Tremor charts need layout the jsdom environment doesn't provide; stub the
// pieces pages import so mounting a chart page doesn't warn/throw.
vi.mock("@tremor/react", () => {
  const Stub = ({ children }) => <div>{children}</div>;
  return {
    __esModule: true,
    AreaChart: Stub,
    BarChart: Stub,
    LineChart: Stub,
    DonutChart: Stub,
    Card: Stub,
    Title: Stub,
    Text: Stub,
  };
});

// Dashboard pages under App's routes (portal pages have their own auth shell).
const PAGES = import.meta.glob("../*.jsx");

const wrap = (ui) => (
  <MemoryRouter>
    <AuthProvider>
      <ToastProvider>{ui}</ToastProvider>
    </AuthProvider>
  </MemoryRouter>
);

describe("Dashboard pages mount without crashing", () => {
  afterEach(cleanup);

  for (const path of Object.keys(PAGES)) {
    // Skip test files and index barrels.
    if (path.includes("__tests__") || path.endsWith("/index.jsx")) continue;
    const name = path.replace("../", "").replace(".jsx", "");

    it(`renders ${name}`, async () => {
      const mod = await PAGES[path]();
      const Page = mod.default;
      if (typeof Page !== "function") return; // not a component module
      expect(() => render(wrap(<Page />))).not.toThrow();
    });
  }
});
