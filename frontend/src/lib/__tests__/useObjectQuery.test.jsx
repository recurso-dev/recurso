import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect } from "vitest";
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

  it("exposes a refetch to retry", async () => {
    const { result } = render(async () => ({ id: "1" }));
    await waitFor(() => expect(result.current.object).toBeTruthy());
    expect(typeof result.current.refetch).toBe("function");
  });
});
