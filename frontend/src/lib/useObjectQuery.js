import { useQuery } from "@tanstack/react-query";

import { isNotFound } from "@/lib/httpError";

/**
 * useObjectQuery — the canonical read lifecycle for a single object page. It is
 * a thin classifier over ONE object GET (it does NOT replace react-query and is
 * not a general-purpose fetch hook): it runs the query and buckets the result
 * into the four states every object page needs, so each page renders one
 * skeleton, one not-found, and one error via the shared components.
 *
 *   const { object, loading, notFound, isError, error, refetch } =
 *     useObjectQuery(["invoice", id], () => endpoints.getInvoice(id).then(r => r.data.data), {
 *       enabled: Boolean(id),
 *     });
 *
 * Classification (mutually exclusive, in order):
 *   - loading  — the initial load (an in-flight fetch, or a disabled/idle query
 *                with no data yet); NEVER a paused query (that would hang)
 *   - notFound — the object doesn't exist: a real 404, the API `not_found` code,
 *                OR a resolved-but-null object (the fix for the live 404-copy bug)
 *   - isError  — the request genuinely failed (network / 5xx / other), OR the
 *                fetch is paused offline (a retryable, non-hanging error)
 *   - object   — the resolved object (only when none of the above)
 *
 * `object` is null in every non-success state so a page can render its states
 * before touching object fields.
 */
export function useObjectQuery(queryKey, queryFn, options = {}) {
  const query = useQuery({ queryKey, queryFn, ...options });

  // A paused fetch (react-query `networkMode: "online"` + the onlineManager
  // reporting offline) holds the query at status "pending" with no error object
  // and never settles. We must NOT treat that as "loading" — that renders an
  // object-page skeleton forever. It's a failed/blocked load, so surface it as a
  // retryable error (react-query auto-resumes, and Retry refetches, once online).
  const paused = query.fetchStatus === "paused";

  // loading = genuinely working toward a first result: an in-flight fetch, or a
  // still-disabled/idle query with no data yet (`enabled: Boolean(id)` before an
  // id arrives — isLoading would be false there and leak an undefined object).
  // Excludes the paused case above.
  const loading = query.isPending && !paused;
  const notFound =
    !loading &&
    isNotFound({ error: query.error, data: query.data, resolved: query.isSuccess });
  const isError = !loading && !notFound && (Boolean(query.error) || paused);

  return {
    object: loading || notFound || isError ? null : query.data,
    loading,
    notFound,
    isError,
    error: query.error,
    refetch: query.refetch,
    // Escape hatch for pages that need more of the underlying query (e.g.
    // isFetching); use sparingly.
    query,
  };
}

export default useObjectQuery;
