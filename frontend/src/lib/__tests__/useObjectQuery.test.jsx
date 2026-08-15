import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider, onlineManager } from "@tanstack/react-query";
import { describe, it, expect, afterEach } from "vitest";
import { useObjectQuery } from "../useObjectQuery";

function wrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  const Wrapper = ({ children }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return Wrapper;
}

const render = (queryFn, key = ["obj", "1"]) =>
  renderHook(() => useObjectQuery(key, queryFn), { wrapper: wrapper() });

describe("useObjectQuery", () => {
  afterEach(() => onlineManager.setOnline(true));

  it("reports loading, then the resolved object on success", async () => {
    const { result } = render(async () => ({ id: "1", name: "Acme" }));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.object).toEqual({ id: "1", name: "Acme" });
    expect(result.current.notFound).toBe(false);
    expect(result.current.isError).toBe(false);
  });

  it("classifies a resolved-null object as notFound (not error)", async () => {
    const { result } = render(async () => null, ["obj", "null"]);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.notFound).toBe(true);
    expect(result.current.isError).toBe(false);
    expect(result.current.object).toBeNull();
  });

  it("classifies a real HTTP 404 as notFound", async () => {
    const { result } = render(async () => {
      throw { response: { status: 404, data: { error: { code: "not_found" } } } };
    }, ["obj", "404"]);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.notFound).toBe(true);
    expect(result.current.isError).toBe(false);
    expect(result.current.object).toBeNull();
  });

  it("classifies a genuine (non-404) failure as isError", async () => {
    const { result } = render(async () => {
      throw { response: { status: 500, data: { error: { code: "internal" } } } };
    }, ["obj", "500"]);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.isError).toBe(true);
    expect(result.current.notFound).toBe(false);
    expect(result.current.object).toBeNull();
    expect(result.current.error).toBeTruthy();
  });

  it("treats a paused (offline) fetch as a retryable error, never an endless load", async () => {
    // networkMode "online" + onlineManager offline → the fetch pauses at
    // status "pending" / fetchStatus "paused" with no error object. Regression
    // guard: this must NOT read as loading (that hangs the object-page skeleton
    // forever) — it must surface as a retryable error.
    onlineManager.setOnline(false);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
    const Wrapper = ({ children }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(
      () => useObjectQuery(["obj", "paused"], async () => ({ id: "1" })),
      { wrapper: Wrapper }
    );
    await waitFor(() => expect(result.current.query.fetchStatus).toBe("paused"));
    expect(result.current.loading).toBe(false);
    expect(result.current.isError).toBe(true);
    expect(result.current.notFound).toBe(false);
    expect(result.current.object).toBeNull();
  });

  it("keeps a still-disabled query in the loading state (no undefined object leak)", async () => {
    const { result } = renderHook(
      () => useObjectQuery(["obj", "disabled"], async () => ({ id: "1" }), { enabled: false }),
      { wrapper: wrapper() }
    );
    expect(result.current.loading).toBe(true);
    expect(result.current.isError).toBe(false);
    expect(result.current.object).toBeNull();
  });

  it("exposes a refetch to retry", async () => {
    const { result } = render(async () => ({ id: "1" }));
    await waitFor(() => expect(result.current.object).toBeTruthy());
    expect(typeof result.current.refetch).toBe("function");
  });
});
