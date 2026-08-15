import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import { useBulkAction } from "../useBulkAction";

describe("useBulkAction", () => {
  it("reports all-succeeded when every record succeeds", async () => {
    const { result } = renderHook(() => useBulkAction());
    const fn = vi.fn().mockResolvedValue({});
    let res;
    await act(async () => {
      res = await result.current.run(["a", "b", "c"], fn);
    });
    expect(fn).toHaveBeenCalledTimes(3);
    expect(res.succeeded).toEqual(["a", "b", "c"]);
    expect(res.failed).toEqual([]);
    expect(result.current.state.status).toBe("all_succeeded");
  });

  it("records a partial failure as its own state, not success", async () => {
    const { result } = renderHook(() => useBulkAction());
    const fn = vi.fn((id) =>
      id === "b" ? Promise.reject({ message: "nope" }) : Promise.resolve({})
    );
    let res;
    await act(async () => {
      res = await result.current.run(["a", "b", "c"], fn);
    });
    expect(res.succeeded).toEqual(["a", "c"]);
    expect(res.failed).toEqual([{ id: "b", error: "nope" }]);
    expect(result.current.state.status).toBe("partial");
  });

  it("reports all-failed when every record fails", async () => {
    const { result } = renderHook(() => useBulkAction());
    const fn = vi.fn().mockRejectedValue({
      response: { data: { error: { message: "bad" } } },
    });
    let res;
    await act(async () => {
      res = await result.current.run(["a", "b"], fn);
    });
    expect(res.failed.map((f) => f.error)).toEqual(["bad", "bad"]);
    expect(result.current.state.status).toBe("all_failed");
  });

  it("retry runs only the failed ids", async () => {
    const { result } = renderHook(() => useBulkAction());
    const calls = [];
    const fn = vi.fn((id) => {
      calls.push(id);
      return id === "b" && calls.filter((c) => c === "b").length === 1
        ? Promise.reject({ message: "first-fail" })
        : Promise.resolve({});
    });
    await act(async () => {
      await result.current.run(["a", "b", "c"], fn);
    });
    expect(result.current.state.status).toBe("partial");
    // Retry only "b".
    await act(async () => {
      await result.current.run(["b"], fn);
    });
    expect(result.current.state.status).toBe("all_succeeded");
    // "a" and "c" ran exactly once (never re-run).
    expect(calls.filter((c) => c === "a")).toHaveLength(1);
    expect(calls.filter((c) => c === "c")).toHaveLength(1);
    expect(calls.filter((c) => c === "b")).toHaveLength(2);
  });

  it("exposes running + processing during the run", async () => {
    let resolveFirst;
    const fn = vi.fn(
      () =>
        new Promise((r) => {
          resolveFirst = r;
        })
    );
    const { result } = renderHook(() => useBulkAction());
    let runPromise;
    act(() => {
      runPromise = result.current.run(["a"], fn);
    });
    await waitFor(() => expect(result.current.running).toBe(true));
    expect(result.current.state.processing).toBe("a");
    await act(async () => {
      resolveFirst({});
      await runPromise;
    });
    expect(result.current.running).toBe(false);
  });

  it("reset clears the state", async () => {
    const { result } = renderHook(() => useBulkAction());
    await act(async () => {
      await result.current.run(["a"], vi.fn().mockResolvedValue({}));
    });
    expect(result.current.state).not.toBeNull();
    act(() => result.current.reset());
    expect(result.current.state).toBeNull();
  });
});
