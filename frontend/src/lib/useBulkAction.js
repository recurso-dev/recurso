import { useCallback, useState } from "react";

const errMsg = (e) =>
  e?.response?.data?.error?.message || e?.message || "Failed";

/**
 * useBulkAction — run a per-record action across many records as an OBSERVABLE,
 * partial-failure-aware operation. Deliberately NOT a generic workflow engine:
 * it runs one async `fn(id)` per id, in order, and reports total / succeeded /
 * failed / processing so the UI can show real progress and keep the failed rows
 * identifiable and retryable.
 *
 * `fn(id)` resolves on success and throws on failure. A partial result is never
 * success — the caller inspects `status`:
 *   "running" | "all_succeeded" | "partial" | "all_failed".
 *
 * Retry re-runs ONLY the still-failed ids (the caller passes them back in). The
 * caller reuses the same idempotency key per id across a retry, so a retry can
 * never double-act on a record that actually succeeded server-side.
 */
export function useBulkAction() {
  const [state, setState] = useState(null);
  const running = state?.status === "running";

  const run = useCallback(async (ids, fn) => {
    const succeeded = [];
    const failed = [];
    setState({ total: ids.length, succeeded: [], failed: [], processing: null, status: "running" });
    for (const id of ids) {
      setState((s) => ({ ...s, processing: id }));
      try {
        await fn(id);
        succeeded.push(id);
      } catch (e) {
        failed.push({ id, error: errMsg(e) });
      }
      setState((s) => ({ ...s, succeeded: [...succeeded], failed: [...failed], processing: null }));
    }
    const status =
      failed.length === 0 ? "all_succeeded" : succeeded.length === 0 ? "all_failed" : "partial";
    setState((s) => ({ ...s, processing: null, status }));
    return { succeeded, failed };
  }, []);

  const reset = useCallback(() => setState(null), []);

  return { state, running, run, reset };
}
